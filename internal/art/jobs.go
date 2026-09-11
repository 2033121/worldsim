package art

// jobs.go — 美术生成任务编排。
//
// 任务模型（与 /api/world/sim/thinking 轮询风格一致，不引 SSE）：
//   batch  — 按类批量：出 sheet → 裁剪 → sprites → 空格自动单品补齐
//   single — 单品：改某条 prompt 后只重生成一个 sprite（用户改人物的主路径）
//
// 单品失败不中断批次；已完成的 sheet 续跑时跳过（断点续生成）。

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// StepResult 任务里每类资产的结果
type StepResult struct {
	Label  string `json:"label"`
	Status string `json:"status"` // running|done|failed|skipped
	Error  string `json:"error,omitempty"`
	Count  int    `json:"count,omitempty"` // 产出 sprite 数
	Healed int    `json:"healed,omitempty"`
}

// Job 任务
type Job struct {
	ID      string       `json:"id"`
	World   string       `json:"world"`
	Kind    string       `json:"kind"` // batch|single
	Assets  []string     `json:"assets,omitempty"`
	Label   string       `json:"label,omitempty"` // single 用
	Index   int          `json:"index,omitempty"` // single 用（1 起）
	Status  string       `json:"status"`          // queued|running|done|failed
	Steps   []StepResult `json:"steps"`
	Error   string       `json:"error,omitempty"`
	Created time.Time    `json:"created"`
	Updated time.Time    `json:"updated"`
}

// HistoryEntry 生成记录（追溯/重掷）
type HistoryEntry struct {
	Time   string `json:"time"`
	World  string `json:"world"`
	Kind   string `json:"kind"` // sheet|single|heal
	Label  string `json:"label"`
	Index  int    `json:"index,omitempty"`
	Prompt string `json:"prompt"`
	Size   string `json:"size"`
	Bytes  int    `json:"bytes"`
	Note   string `json:"note,omitempty"`
}

// Manager 美术任务管理器（挂在 worldServer 上，全局一个）
type Manager struct {
	mu    sync.Mutex
	jobs  map[string]*Job
	sem   chan struct{}
	cfgFn func() *Config // 每次任务热读配置（改配置不用重启）
	seq   int
}

// NewManager 构造（maxConcurrent<=0 取默认 2）
func NewManager(cfgFn func() *Config) *Manager {
	max := 2
	if cfgFn != nil {
		if c := cfgFn(); c != nil && c.MaxConcurrent > 0 {
			max = c.MaxConcurrent
		}
	}
	return &Manager{jobs: map[string]*Job{}, sem: make(chan struct{}, max), cfgFn: cfgFn}
}

// Submit 提交任务并异步执行
func (m *Manager) Submit(world, kind string, assets []string, label string, index int) (*Job, error) {
	if m.cfgFn == nil {
		return nil, fmt.Errorf("美术模块未配置")
	}
	cfg := m.cfgFn()
	if !cfg.Enabled() {
		return nil, fmt.Errorf("图片生成未启用：请在美术工坊配置 base_url 与 api_key（或设环境变量 %s）", orDefault(cfg.APIKeyEnv, "GPTIMG_KEY"))
	}
	if assets == nil && kind == "batch" {
		assets = []string{"characters", "monsters", "scenes", "maptiles", "items"}
	}
	m.mu.Lock()
	m.seq++
	job := &Job{
		ID:      fmt.Sprintf("art-%d-%d", time.Now().Unix(), m.seq),
		World:   world,
		Kind:    kind,
		Assets:  assets,
		Label:   label,
		Index:   index,
		Status:  "queued",
		Created: time.Now(),
		Updated: time.Now(),
	}
	m.jobs[job.ID] = job
	m.mu.Unlock()
	go m.run(job)
	return job, nil
}

// Get 查任务
func (m *Manager) Get(id string) *Job {
	m.mu.Lock()
	defer m.mu.Unlock()
	j := m.jobs[id]
	if j == nil {
		return nil
	}
	cp := *j
	cp.Steps = append([]StepResult(nil), j.Steps...)
	return &cp
}

// List 最近任务（新→旧，限量）
func (m *Manager) List(limit int) []*Job {
	m.mu.Lock()
	defer m.mu.Unlock()
	all := make([]*Job, 0, len(m.jobs))
	for _, j := range m.jobs {
		all = append(all, j)
	}
	sort.Slice(all, func(i, k int) bool { return all[i].Created.After(all[k].Created) })
	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}
	return all
}

// ---------- 执行 ----------

func (m *Manager) run(job *Job) {
	m.sem <- struct{}{}
	defer func() { <-m.sem }()
	m.setStatus(job, "running")

	worldDir, ok := m.resolveWorldDir(job.World)
	if !ok {
		m.fail(job, "找不到世界目录："+job.World)
		return
	}
	plan, err := LoadPlan(worldDir)
	if err != nil || plan == nil {
		m.fail(job, "素材规划不存在，请先生成规划")
		return
	}
	gen := NewGenerator(m.cfgFn())
	if gen == nil {
		m.fail(job, "图片生成未启用")
		return
	}

	switch job.Kind {
	case "batch":
		for _, label := range job.Assets {
			spec := SpecByLabel(label)
			if spec == nil {
				m.step(job, StepResult{Label: label, Status: "failed", Error: "未知资产类型"})
				continue
			}
			step := m.generateSheet(context.Background(), gen, worldDir, plan, spec)
			m.step(job, step)
		}
	case "single":
		spec := specByFileLabel(job.Label)
		if spec == nil {
			m.fail(job, "未知素材类型："+job.Label)
			return
		}
		assets, _ := planAssetsByLabel(plan, spec.FileLabel)
		if job.Index < 1 || job.Index-1 >= len(assets) {
			m.fail(job, "素材序号越界")
			return
		}
		step := m.generateSingle(context.Background(), gen, worldDir, plan, spec.FileLabel, assets[job.Index-1], job.Index, "single")
		if step.Status == "failed" {
			m.fail(job, step.Error)
		} else {
			m.setStatus(job, "done")
		}
	default:
		m.fail(job, "未知任务类型："+job.Kind)
	}
}

// resolveWorldDir 从 baseDir（worlds/）找世界目录；Manager 不知道 baseDir，靠 ws 层注入 worldDir 直传。
// 这里用 job.World 存的是世界目录绝对路径（Submit 前由 ws 层解析好），世界名只做展示。
func (m *Manager) resolveWorldDir(world string) (string, bool) {
	p := world
	if st, err := os.Stat(p); err == nil && st.IsDir() {
		return p, true
	}
	return "", false
}

// generateSheet 生成一张 sheet 并裁剪；空格自动单品补齐
func (m *Manager) generateSheet(ctx context.Context, gen Generator, worldDir string, plan *Plan, spec *GridSpec) StepResult {
	assets, _ := planAssetsByLabel(plan, spec.FileLabel)
	prompt := plan.SheetPrompt(spec, assets)
	data, err := gen.Generate(ctx, prompt, spec.Size)
	if err != nil {
		return StepResult{Label: spec.Label, Status: "failed", Error: err.Error()}
	}
	if err := writeFile(filepath.Join(worldDir, "art", "sheets", spec.Label+".png"), data); err != nil {
		return StepResult{Label: spec.Label, Status: "failed", Error: err.Error()}
	}
	_ = m.addHistory(worldDir, HistoryEntry{World: worldDir, Kind: "sheet", Label: spec.Label, Prompt: prompt, Size: spec.Size, Bytes: len(data)})

	cells, cerr := CropSheet(data, spec)
	if cerr != nil {
		return StepResult{Label: spec.Label, Status: "failed", Error: cerr.Error()}
	}
	count, healed := 0, 0
	spritesDir := filepath.Join(worldDir, "art", "sprites")
	_ = os.MkdirAll(spritesDir, 0755)
	for i, cell := range cells {
		if cell.Empty {
			// 空格自动补齐：单品流程重生成（最多 2 轮）
			index := i + 1
			asset := assetAt(assets, i)
			ok := false
			for attempt := 0; attempt < 2; attempt++ {
				if out, empty, serr := m.generateOneBytes(ctx, gen, plan, spec.FileLabel, asset, SingleSheetLayout); serr == nil && !empty && out != nil {
					_ = writeFile(filepath.Join(spritesDir, spec.SpriteName(index)), out)
					count++
					healed++
					ok = true
					break
				}
			}
			if !ok {
				_ = m.addHistory(worldDir, HistoryEntry{World: worldDir, Kind: "heal", Label: spec.FileLabel, Index: index, Note: "空格补齐失败"})
			}
			continue
		}
		if err := writeFile(filepath.Join(spritesDir, spec.SpriteName(i+1)), cell.PNG); err != nil {
			continue
		}
		count++
	}
	if count == 0 {
		return StepResult{Label: spec.Label, Status: "failed", Error: "所有格子均为空"}
	}
	return StepResult{Label: spec.Label, Status: "done", Count: count, Healed: healed}
}

// generateSingle 生成并落盘单个 sprite
func (m *Manager) generateSingle(ctx context.Context, gen Generator, worldDir string, plan *Plan, fileLabel string, asset PlanAsset, index int, note string) StepResult {
	out, empty, err := m.generateOneBytes(ctx, gen, plan, fileLabel, asset, SingleSheetLayout)
	if err != nil {
		return StepResult{Label: fileLabel, Status: "failed", Error: err.Error()}
	}
	if empty || out == nil {
		return StepResult{Label: fileLabel, Status: "failed", Error: "生成结果为空图"}
	}
	spritesDir := filepath.Join(worldDir, "art", "sprites")
	if err := writeFile(filepath.Join(spritesDir, fmt.Sprintf("%s-%d.png", fileLabel, index)), out); err != nil {
		return StepResult{Label: fileLabel, Status: "failed", Error: err.Error()}
	}
	_ = m.addHistory(worldDir, HistoryEntry{World: worldDir, Kind: note, Label: fileLabel, Index: index, Bytes: len(out)})
	return StepResult{Label: fileLabel, Status: "done", Count: 1}
}

func (m *Manager) generateOneBytes(ctx context.Context, gen Generator, plan *Plan, fileLabel string, asset PlanAsset, size string) ([]byte, bool, error) {
	data, err := gen.Generate(ctx, plan.AssetPrompt(fileLabel, asset), size)
	if err != nil {
		return nil, false, err
	}
	out, empty, err := CropSingle(data)
	if err != nil {
		return nil, false, err
	}
	return out, empty, nil
}

// ---------- history ----------

func (m *Manager) addHistory(worldDir string, entry HistoryEntry) error {
	entry.Time = time.Now().Format(time.RFC3339)
	path := filepath.Join(worldDir, "art", "history.json")
	list := []HistoryEntry{}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &list)
	}
	list = append(list, entry)
	if len(list) > 500 {
		list = list[len(list)-500:]
	}
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return writeFile(path, data)
}

// LoadHistory 供 HTTP 层读取
func LoadHistory(worldDir string) []HistoryEntry {
	data, err := os.ReadFile(filepath.Join(worldDir, "art", "history.json"))
	if err != nil {
		return []HistoryEntry{}
	}
	list := []HistoryEntry{}
	_ = json.Unmarshal(data, &list)
	sort.Slice(list, func(i, k int) bool { return list[i].Time > list[k].Time })
	return list
}

// ---------- 文件与状态工具 ----------

func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (m *Manager) setStatus(job *Job, status string) {
	m.mu.Lock()
	job.Status = status
	job.Updated = time.Now()
	m.mu.Unlock()
	m.persist(job)
}

func (m *Manager) fail(job *Job, msg string) {
	m.mu.Lock()
	job.Status = "failed"
	job.Error = msg
	job.Updated = time.Now()
	m.mu.Unlock()
	m.persist(job)
}

func (m *Manager) step(job *Job, step StepResult) {
	m.mu.Lock()
	job.Steps = append(job.Steps, step)
	job.Updated = time.Now()
	m.mu.Unlock()
	m.persist(job)
}

// persist 任务落账到 <world>/art/jobs.json（尽力而为）
func (m *Manager) persist(job *Job) {
	worldDir := job.World
	list := []*Job{}
	if data, err := os.ReadFile(filepath.Join(worldDir, "art", "jobs.json")); err == nil {
		_ = json.Unmarshal(data, &list)
	}
	// 替换同 ID
	replaced := false
	for i, j := range list {
		if j.ID == job.ID {
			cp := *job
			cp.Steps = append([]StepResult(nil), job.Steps...)
			list[i] = &cp
			replaced = true
			break
		}
	}
	if !replaced {
		cp := *job
		cp.Steps = append([]StepResult(nil), job.Steps...)
		list = append(list, &cp)
	}
	if len(list) > 50 {
		list = list[len(list)-50:]
	}
	if data, err := json.MarshalIndent(list, "", "  "); err == nil {
		_ = writeFile(filepath.Join(worldDir, "art", "jobs.json"), data)
	}
}

// specByFileLabel 按命名前缀反查规格（single 任务只给 fileLabel）
func specByFileLabel(fileLabel string) *GridSpec {
	for _, s := range SheetSpecs() {
		if s.FileLabel == fileLabel {
			cp := s
			return &cp
		}
	}
	return nil
}

// planAssetsByLabel 从 plan 里取某类条目（副本）
func planAssetsByLabel(plan *Plan, fileLabel string) ([]PlanAsset, string) {
	switch fileLabel {
	case "character":
		return append([]PlanAsset(nil), plan.Characters...), "characters"
	case "monster":
		return append([]PlanAsset(nil), plan.Monsters...), "monsters"
	case "scene":
		return append([]PlanAsset(nil), plan.Scenes...), "scenes"
	case "tile":
		return append([]PlanAsset(nil), plan.Tiles...), "maptiles"
	case "item":
		return append([]PlanAsset(nil), plan.Items...), "items"
	}
	return nil, ""
}

func assetAt(assets []PlanAsset, i int) PlanAsset {
	if i < len(assets) {
		return assets[i]
	}
	return PlanAsset{Name: fmt.Sprintf("素材 %d", i+1)}
}

// ContactSheet 把 sprite 拼成联络表（浅灰底、定长格）
func ContactSheet(spritePNGs [][]byte, cols int) ([]byte, error) {
	if len(spritePNGs) == 0 {
		return nil, fmt.Errorf("无 sprite")
	}
	if cols <= 0 {
		cols = 8
	}
	const cell = 160
	rows := (len(spritePNGs) + cols - 1) / cols
	canvas := image.NewRGBA(image.Rect(0, 0, cols*cell, rows*cell))
	bg := color.RGBA{232, 232, 232, 255} // #e8e8e8 与生成契约同底
	for y := 0; y < canvas.Bounds().Dy(); y++ {
		for x := 0; x < canvas.Bounds().Dx(); x++ {
			canvas.SetRGBA(x, y, bg)
		}
	}
	for i, raw := range spritePNGs {
		src, derr := png.Decode(bytes.NewReader(raw))
		if derr != nil {
			continue
		}
		b := src.Bounds()
		// 等比缩放进 cell（最近邻，保持像素感）
		scale := float64(cell-16) / float64(max(b.Dx(), b.Dy()))
		if scale > 1 {
			scale = 1
		}
		dw, dh := int(float64(b.Dx())*scale), int(float64(b.Dy())*scale)
		if dw <= 0 || dh <= 0 {
			continue
		}
		cx, cy := (i%cols)*cell, (i/cols)*cell
		ox, oy := cx+(cell-dw)/2, cy+(cell-dh)/2
		for y := 0; y < dh; y++ {
			sy := b.Min.Y + y*b.Dy()/dh
			for x := 0; x < dw; x++ {
				sx := b.Min.X + x*b.Dx()/dw
				canvas.SetRGBA(ox+x, oy+y, colorToRGBA(src.At(sx, sy)))
			}
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, canvas); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// colorToRGBA 颜色转 RGBA
func colorToRGBA(c color.Color) color.RGBA {
	r, g, b, a := c.RGBA()
	return color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
}
