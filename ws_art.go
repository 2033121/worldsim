package main

// ws_art.go — 「美术工坊」（Art Studio）的 HTTP 层。
//
// 端点（挂 :48091，统一网关 48092 的 /api/* 自动转发；/studio /art 另加反代）：
//   GET  /api/art/config        — 读配置（key 掩码）
//   POST /api/art/config        — 写配置
//   POST /api/art/config/test   — 连通性试生成
//   POST /api/art/plan          — 从世界书生成素材规划（异步）
//   GET  /api/art/plan          — 当前世界规划
//   PUT  /api/art/plan          — 保存用户编辑后的规划
//   POST /api/art/generate      — 批量 {assets:[...]} 或单品 {label,index}（异步）
//   GET  /api/art/jobs[/{id}]   — 任务进度
//   GET  /api/art/assets        — 素材清单
//   GET  /api/art/history       — 生成记录
//   GET  /art/{file}            — 当前世界 sprite 直出（打包套件兜底）

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	art "worldsim/internal/art"
)

// artCfgPath 图片生成配置路径（progDir/img.json，与 api.json 同目录惯例）
func (ws *worldServer) artCfgPath() string {
	return filepath.Join(filepath.Dir(ws.baseDir), "img.json")
}

// artCfg 热读配置（每次调用读盘，改配置即生效）
func (ws *worldServer) artCfg() *art.Config {
	cfg, err := art.LoadConfig(ws.artCfgPath())
	if err != nil {
		cfg = art.DefaultConfig()
	}
	return cfg
}

// artMgr 懒加载任务管理器
func (ws *worldServer) artMgr() *art.Manager {
	ws.artOnce.Do(func() {
		ws.artManager = art.NewManager(func() *art.Config { return ws.artCfg() })
	})
	return ws.artManager
}

// GET /api/art/config
func (ws *worldServer) handleArtConfigGet(w http.ResponseWriter, r *http.Request) {
	cfg := ws.artCfg()
	ws.writeJSON(w, 200, map[string]any{
		"ok":      true,
		"enabled": cfg.Enabled(),
		"config": map[string]any{
			"provider":        cfg.Provider,
			"base_url":        cfg.BaseURL,
			"model":           cfg.Model,
			"api_key_masked":  cfg.MaskedKey(),
			"api_key_env":     cfg.APIKeyEnv,
			"size":            cfg.Size,
			"quality":         cfg.Quality,
			"timeout_seconds": cfg.TimeoutSeconds,
			"max_retries":     cfg.MaxRetries,
			"max_concurrent":  cfg.MaxConcurrent,
			"user_agent":      cfg.UserAgent,
			"quantize":        cfg.Quantize,
		},
	})
}

// POST /api/art/config — 写配置（api_key 为空串时保留旧 key）
func (ws *worldServer) handleArtConfigSet(w http.ResponseWriter, r *http.Request) {
	var req map[string]any
	if err := json.NewDecoder(io.LimitReader(r.Body, 64<<10)).Decode(&req); err != nil {
		ws.writeJSON(w, 400, map[string]any{"ok": false, "error": "请求体解析失败"})
		return
	}
	cfg := ws.artCfg()
	setStr := func(field string, dst *string) {
		if v, ok := req[field].(string); ok && v != "" {
			*dst = v
		}
	}
	setInt := func(field string, dst *int) {
		if v, ok := req[field].(float64); ok && v > 0 {
			*dst = int(v)
		}
	}
	setStr("provider", &cfg.Provider)
	setStr("base_url", &cfg.BaseURL)
	setStr("model", &cfg.Model)
	setStr("size", &cfg.Size)
	setStr("quality", &cfg.Quality)
	setStr("api_key_env", &cfg.APIKeyEnv)
	setStr("user_agent", &cfg.UserAgent)
	setInt("timeout_seconds", &cfg.TimeoutSeconds)
	setInt("max_retries", &cfg.MaxRetries)
	setInt("max_concurrent", &cfg.MaxConcurrent)
	if v, ok := req["api_key"].(string); ok && v != "" {
		cfg.APIKey = v
	}
	if v, ok := req["quantize"].(bool); ok {
		cfg.Quantize = v
	}
	if err := art.SaveConfig(ws.artCfgPath(), cfg); err != nil {
		ws.writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	ws.writeJSON(w, 200, map[string]any{"ok": true, "enabled": cfg.Enabled(), "api_key_masked": cfg.MaskedKey()})
}

// POST /api/art/config/test — 连通性试生成（64x64 最小成本）
func (ws *worldServer) handleArtConfigTest(w http.ResponseWriter, r *http.Request) {
	cfg := ws.artCfg()
	if !cfg.Enabled() {
		ws.writeJSON(w, 400, map[string]any{"ok": false, "error": "未配置密钥或 base_url"})
		return
	}
	gen := art.NewGenerator(cfg)
	if gen == nil {
		ws.writeJSON(w, 400, map[string]any{"ok": false, "error": "provider 配置无效"})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.TimeoutSeconds)*time.Second)
	defer cancel()
	data, err := gen.Generate(ctx, "one small red pixel heart icon, plain background, no text", "64x64")
	if err != nil {
		ws.writeJSON(w, 502, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	ws.writeJSON(w, 200, map[string]any{"ok": true, "bytes": len(data)})
}

// POST /api/art/plan — 从世界书生成规划（异步：LLM 生成完自动落盘）
func (ws *worldServer) handleArtPlanGenerate(w http.ResponseWriter, r *http.Request) {
	inst := ws.inst()
	if inst == nil || inst.dir == "" {
		ws.writeJSON(w, 400, map[string]any{"ok": false, "error": "没有可用世界"})
		return
	}
	if inst.wb == nil {
		ws.writeJSON(w, 400, map[string]any{"ok": false, "error": "世界书未加载"})
		return
	}
	wb := inst.wb
	worldDir := inst.dir
	entities := []string{}
	if inst.engine != nil {
		st := inst.engine.State()
		for name, e := range st.Entities {
			if e.Status == "active" && name != inst.heroName {
				entities = append(entities, name)
			}
		}
		if hero := inst.heroName; hero != "" {
			if inst.sim != nil && inst.sim.HeroName() != "" {
				hero = inst.sim.HeroName()
			}
			entities = append([]string{hero}, entities...)
		}
	}
	worldName := inst.name
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		plan, err := art.PlanFromWorldbook(ctx, ws.apiCfg, wb, entities)
		if err != nil {
			fmt.Printf(" [美术] 规划生成失败：%v\n", err)
			return
		}
		if err := art.SavePlan(worldDir, plan); err != nil {
			fmt.Printf(" [美术] 规划落盘失败：%v\n", err)
			return
		}
		fmt.Printf(" [美术] 规划已生成：theme=%s 世界=%s\n", plan.Theme, worldName)
	}()
	ws.writeJSON(w, 200, map[string]any{"ok": true, "note": "规划生成中，稍后用 GET /api/art/plan 轮询"})
}

// GET /api/art/plan
func (ws *worldServer) handleArtPlanGet(w http.ResponseWriter, r *http.Request) {
	inst := ws.inst()
	if inst == nil {
		ws.writeJSON(w, 400, map[string]any{"ok": false, "error": "没有可用世界"})
		return
	}
	plan, err := art.LoadPlan(inst.dir)
	if err != nil {
		ws.writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	if plan == nil {
		ws.writeJSON(w, 200, map[string]any{"ok": true, "plan": nil})
		return
	}
	ws.writeJSON(w, 200, map[string]any{"ok": true, "plan": plan, "warns": plan.ValidatePlan()})
}

// PUT /api/art/plan — 保存编辑（前端传完整 JSON）
func (ws *worldServer) handleArtPlanSet(w http.ResponseWriter, r *http.Request) {
	inst := ws.inst()
	if inst == nil {
		ws.writeJSON(w, 400, map[string]any{"ok": false, "error": "没有可用世界"})
		return
	}
	var plan art.Plan
	if err := json.NewDecoder(io.LimitReader(r.Body, 4<<20)).Decode(&plan); err != nil {
		ws.writeJSON(w, 400, map[string]any{"ok": false, "error": "规划 JSON 解析失败: " + err.Error()})
		return
	}
	if plan.Theme == "" {
		plan.Theme = "custom"
	}
	if plan.Version == 0 {
		plan.Version = 1
	}
	if err := art.SavePlan(inst.dir, &plan); err != nil {
		ws.writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	ws.writeJSON(w, 200, map[string]any{"ok": true, "warns": plan.ValidatePlan()})
}

// POST /api/art/generate — {assets:["characters",…]} 批量 | {label:"character",index:5} 单品
func (ws *worldServer) handleArtGenerate(w http.ResponseWriter, r *http.Request) {
	inst := ws.inst()
	if inst == nil || inst.dir == "" {
		ws.writeJSON(w, 400, map[string]any{"ok": false, "error": "没有可用世界"})
		return
	}
	var req struct {
		Assets []string `json:"assets"`
		Label  string   `json:"label"`
		Index  int      `json:"index"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 64<<10)).Decode(&req); err != nil {
		ws.writeJSON(w, 400, map[string]any{"ok": false, "error": "请求体解析失败"})
		return
	}
	job, err := func() (*art.Job, error) {
		if req.Label != "" {
			return ws.artMgr().Submit(inst.dir, "single", nil, req.Label, req.Index)
		}
		assets := req.Assets
		if len(assets) == 0 {
			assets = []string{"characters", "monsters", "scenes", "maptiles", "items"}
		}
		for _, a := range assets {
			if art.SpecByLabel(a) == nil {
				return nil, fmt.Errorf("未知资产类型：%s", a)
			}
		}
		return ws.artMgr().Submit(inst.dir, "batch", assets, "", 0)
	}()
	if err != nil {
		ws.writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	ws.writeJSON(w, 200, map[string]any{"ok": true, "job": job})
}

// GET /api/art/jobs[/{id}]
func (ws *worldServer) handleArtJobs(w http.ResponseWriter, r *http.Request) {
	if id := r.PathValue("id"); id != "" {
		job := ws.artMgr().Get(id)
		if job == nil {
			ws.writeJSON(w, 404, map[string]any{"ok": false, "error": "任务不存在"})
			return
		}
		ws.writeJSON(w, 200, map[string]any{"ok": true, "job": job})
		return
	}
	ws.writeJSON(w, 200, map[string]any{"ok": true, "jobs": ws.artMgr().List(20)})
}

// GET /api/art/assets — 素材清单（存在与否 + URL）
func (ws *worldServer) handleArtAssets(w http.ResponseWriter, r *http.Request) {
	inst := ws.inst()
	if inst == nil {
		ws.writeJSON(w, 400, map[string]any{"ok": false, "error": "没有可用世界"})
		return
	}
	artDir := filepath.Join(inst.dir, "art")
	assets := map[string]any{}
	for _, spec := range art.SheetSpecs() {
		sheetPath := filepath.Join(artDir, "sheets", spec.Label+".png")
		sheetInfo := map[string]any{"exists": false}
		if st, err := os.Stat(sheetPath); err == nil {
			sheetInfo = map[string]any{"exists": true, "mtime": st.ModTime().Unix(), "size": st.Size()}
		}
		sprites := make([]map[string]any, 0, spec.Count)
		for i := 1; i <= spec.Count; i++ {
			name := spec.SpriteName(i)
			info := map[string]any{"url": "/art/" + name, "exists": false}
			if _, err := os.Stat(filepath.Join(artDir, "sprites", name)); err == nil {
				info["exists"] = true
			}
			sprites = append(sprites, info)
		}
		assets[spec.Label] = map[string]any{"sheet": sheetInfo, "sprites": sprites, "count": spec.Count}
	}
	ws.writeJSON(w, 200, map[string]any{"ok": true, "assets": assets})
}

// GET /api/art/history
func (ws *worldServer) handleArtHistory(w http.ResponseWriter, r *http.Request) {
	inst := ws.inst()
	if inst == nil {
		ws.writeJSON(w, 400, map[string]any{"ok": false, "error": "没有可用世界"})
		return
	}
	ws.writeJSON(w, 200, map[string]any{"ok": true, "history": art.LoadHistory(inst.dir)})
}

// GET /art/{file} — 当前世界 sprite 直出：
//
//	<world>/art/sprites/{file} → docs/art/pixel/{detectTheme}/sprites/{file} → docs/art/pixel/{theme}/{file}
func (ws *worldServer) handleArtFile(w http.ResponseWriter, r *http.Request) {
	file := r.PathValue("file")
	if file == "" || !strings.HasSuffix(file, ".png") || strings.Contains(file, "/") {
		http.NotFound(w, r)
		return
	}
	candidates := []string{}
	if inst := ws.inst(); inst != nil && inst.dir != "" {
		candidates = append(candidates, filepath.Join(inst.dir, "art", "sprites", file))
	}
	if theme := func() string {
		if inst := ws.inst(); inst != nil {
			return detectPixelTheme(inst)
		}
		return ""
	}(); theme != "" {
		progRoot := filepath.Dir(ws.baseDir)
		candidates = append(candidates,
			filepath.Join(progRoot, "docs", "art", "pixel", theme, "sprites", file),
			filepath.Join(progRoot, "docs", "art", "pixel", theme, file),
			filepath.Join(progRoot, "art", "pixel", theme, "sprites", file),
		)
	}
	for _, p := range candidates {
		if data, err := os.ReadFile(p); err == nil {
			w.Header().Set("Content-Type", "image/png")
			w.Header().Set("Cache-Control", "public, max-age=300")
			_, _ = w.Write(data)
			return
		}
	}
	http.NotFound(w, r)
}

// GET /studio — 美术工坊页（自包含 HTML，embed 进二进制，同 game.html 模式）
func (ws *worldServer) handleStudioPage(w http.ResponseWriter, r *http.Request) {
	data, err := wsWeb.ReadFile("wsweb/studio.html")
	if err != nil {
		http.Error(w, "美术工坊页未安装", 404)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

// artPixelPayload 组装"本世界专属"pixel 载荷（plan 存在时优先）
// 返回 nil 表示无本世界规划（回落 detectPixelTheme 打包套件）
func artPixelPayload(inst *worldInstance) map[string]any {
	if inst == nil || inst.dir == "" {
		return nil
	}
	plan, err := art.LoadPlan(inst.dir)
	if err != nil || plan == nil {
		return nil
	}
	spriteURL := func(fileLabel string, n int) []string {
		urls := []string{}
		for i := 1; i <= n; i++ {
			urls = append(urls, fmt.Sprintf("/art/%s-%d.png", fileLabel, i))
		}
		return urls
	}
	payload := map[string]any{
		"theme":      plan.Theme,
		"palette":    plan.Palette,
		"hero":       "/art/character-1.png",
		"monster":    "/art/monster-1.png",
		"characters": spriteURL("character", len(plan.Characters)),
		"monsters":   spriteURL("monster", len(plan.Monsters)),
		"scenes":     spriteURL("scene", len(plan.Scenes)),
		"custom":     true,
	}
	if len(plan.PaletteHex) > 0 {
		payload["palette_hex"] = plan.PaletteHex
	}
	return payload
}
