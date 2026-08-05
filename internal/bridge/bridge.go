// Package bridge 实现"世界模拟 → 小说创作"的数据流整合：
//
//	把已模拟世界的世界书 / 角色 / 势力 / 近期编年史事件，
//	直接播种成小说项目的大纲（progress.json）、角色与世界观（settings.json），
//	避免让 LLM 重新从零推导世界信息 —— 省一次（甚至多次）大纲生成调用，
//	也降低后续章节生成的输入 token 与前缀缓存未命中。
//
// 关键复用点：播种后 role/worldview 写入 settings.json，而小说章节生成
// （story.GenerateChapterAction）已通过 buildCharacterContextForLang /
// buildWorldviewContextForLang 注入 settings 中的角色与世界观 —— 因此世界设定
// 自动成为章节生成上下文，无需额外的 LLM 推导。
package bridge

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"worldsim/internal/config"
	"worldsim/internal/engine"
	"worldsim/internal/i18n"
	"worldsim/internal/sim"
	"worldsim/internal/story"
	"worldsim/internal/worldbook"
)

// maxOpeningChapters 播种的大纲开篇章节数（取世界近期至多这么多天）。
const maxOpeningChapters = 8

// CharacterInfo 从世界实体提取的角色信息（供映射为小说角色）。
type CharacterInfo struct {
	Name        string
	Role        string // protagonist | love_interest | important_npc | rival | npc
	Identity    string // 职业/身份
	Personality string // 性格特质
	Background  string // 背景/一句话人设
	Motivation  string // 目标/动机
	Secret      string // 秘密
	Abilities   string // 能力/资产/身体状态简述
}

// WorldData 聚合一个世界的可播种数据（世界书 / 角色 / 势力 / 近期事件）。
type WorldData struct {
	WorldName     string
	Worldbook     *worldbook.Worldbook
	Characters    []CharacterInfo
	Organizations []story.Organization
	RecentEvents  []string // 近期（至多 maxOpeningChapters 天）编年史事件，用于大纲开篇钩子
	Day           int      // 世界当前模拟天数
}

// SeedResult 播种结果摘要（供 HTTP 端点返回）。
type SeedResult struct {
	ProjectName       string `json:"project_name"`
	WorldName         string `json:"world_name"`
	Language          string `json:"language"`
	CharacterCount    int    `json:"character_count"`
	WorldviewCount    int    `json:"worldview_count"`
	OrganizationCount int    `json:"organization_count"`
	OutlineChapterCount int `json:"outline_chapter_count"` // 大纲开篇章节数
	Day               int    `json:"day"`
	Reused            bool   `json:"reused"` // 是否复用了世界数据（恒为 true）
}

// ReadWorld 从 progDir 读取指定世界的可播种数据。
//
//	世界书：progDir/worldbooks/{worldName}.md（兜底 worldDir/worldbook.md）
//	世界状态：progDir/worlds/{worldName}/world_state.json
//	近期事件：progDir/worlds/{worldName}/chronicle.jsonl
func ReadWorld(progDir, worldName string) (*WorldData, error) {
	worldDir := filepath.Join(progDir, "worlds", worldName)
	if info, err := os.Stat(worldDir); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("世界不存在或无法访问: %s", worldName)
	}

	data := &WorldData{WorldName: worldName}

	// 世界书
	data.Worldbook = loadWorldbook(progDir, worldDir, worldName)

	// 世界状态：角色 + 势力 + 天数
	state, err := loadWorldState(worldDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if state != nil {
		data.Day = state.Day
		data.Characters = collectCharacters(state)
		data.Organizations = collectOrganizations(state)
	}

	// 近期编年史事件 → 大纲开篇钩子
	data.RecentEvents, err = recentEvents(worldDir, maxOpeningChapters)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return data, nil
}

func loadWorldbook(progDir, worldDir, worldName string) *worldbook.Worldbook {
	paths := []string{
		filepath.Join(progDir, "worldbooks", worldName+".md"),
		filepath.Join(worldDir, "worldbook.md"),
	}
	for _, p := range paths {
		if wb, err := worldbook.Load(p); err == nil {
			return wb
		}
	}
	return nil
}

func loadWorldState(worldDir string) (*engine.WorldState, error) {
	p := filepath.Join(worldDir, "world_state.json")
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var st engine.WorldState
	if err := json.Unmarshal(b, &st); err != nil {
		return nil, fmt.Errorf("解析世界状态失败: %w", err)
	}
	return &st, nil
}

// collectCharacters 从世界实体提取角色档案（persona_sheet / identity / persona）。
func collectCharacters(st *engine.WorldState) []CharacterInfo {
	var out []CharacterInfo
	names := make([]string, 0, len(st.Entities))
	for name := range st.Entities {
		names = append(names, name)
	}
	sortStrings(names)
	for _, name := range names {
		ent := st.Entities[name]
		ci := CharacterInfo{Name: name, Role: entityRole(ent)}
		if v, ok := ent.Extra["identity"].(string); ok {
			ci.Identity = v
		}
		if v, ok := ent.Extra["persona"].(string); ok {
			ci.Background = v
		}
		if v, ok := ent.Extra["persona_sheet"].(string); ok {
			var cs sim.CharacterSheet
			if json.Unmarshal([]byte(v), &cs) == nil && cs.Name != "" {
				if len(cs.Personality) > 0 {
					ci.Personality = strings.Join(cs.Personality, "、")
				}
				if len(cs.Motives) > 0 {
					ci.Motivation = strings.Join(cs.Motives, "；")
				}
				if cs.Secret != "" {
					ci.Secret = cs.Secret
				}
				if ci.Identity == "" {
					ci.Identity = cs.Identity
				}
			}
		}
		// 能力 / 资产 / 身体状态简述（作为 abilities 提示词种子）
		if ent.Job != "" {
			ci.Abilities = "职业：" + ent.Job
		}
		if len(ent.Assets) > 0 {
			for k, v := range ent.Assets {
				ci.Abilities += fmt.Sprintf("；%s=%.0f", k, v)
			}
		}
		out = append(out, ci)
	}
	return out
}

func entityRole(ent engine.Entity) string {
	if r, ok := ent.Extra["role"].(string); ok && r != "" {
		return r
	}
	return "npc"
}

// collectOrganizations 从世界势力（world_level.factions）提取组织。
func collectOrganizations(st *engine.WorldState) []story.Organization {
	var out []story.Organization
	names := make([]string, 0, len(st.WorldLevel.Factions))
	for name := range st.WorldLevel.Factions {
		names = append(names, name)
	}
	sortStrings(names)
	for _, name := range names {
		fac := st.WorldLevel.Factions[name]
		typ := fac.Visibility
		if typ == "" {
			typ = "势力"
		}
		desc := ""
		if fac.Stance != "" {
			desc = "立场：" + fac.Stance
		}
		if len(fac.RecentActions) > 0 {
			desc += "；近期行动：" + strings.Join(fac.RecentActions, "；")
		}
		out = append(out, story.Organization{
			Name: name, Type: typ, Description: strings.TrimSpace(desc),
		})
	}
	return out
}

// recentEvents 读取 chronicle.jsonl 并按天分组，返回最近至多 maxDays 天的内容。
func recentEvents(worldDir string, maxDays int) ([]string, error) {
	f, err := os.Open(filepath.Join(worldDir, "chronicle.jsonl"))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var days []struct {
		day   int
		parts []string
	}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var e sim.ChronicleEntry
		if json.Unmarshal([]byte(line), &e) != nil || strings.TrimSpace(e.Content) == "" {
			continue
		}
		if len(days) == 0 || days[len(days)-1].day != e.Day {
			days = append(days, struct {
				day   int
				parts []string
			}{day: e.Day})
		}
		days[len(days)-1].parts = append(days[len(days)-1].parts, e.Content)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}

	start := len(days) - maxDays
	if start < 0 {
		start = 0
	}
	var out []string
	for i := start; i < len(days); i++ {
		out = append(out, strings.Join(days[i].parts, "\n"))
	}
	return out, nil
}

// SeedProjectFromWorld 把世界数据播种成小说项目：
//   - 创建项目目录（若不存在）+ 写入 config.json（设定语言）
//   - settings.json：角色（世界角色→小说角色）、世界观（世界书→世界观条目）、组织（势力→组织）
//   - progress.json：大纲（近期事件→开篇章节）+ 故事梗概（世界书世界观/目标链/反派线）
//
// 全程零 LLM 调用 —— 世界数据直接作为种子复用。
func SeedProjectFromWorld(projectDir, language string, data *WorldData) (*SeedResult, error) {
	lang := i18n.NormalizeLanguage(language)
	if lang == "" {
		lang = i18n.LangZH
	}

	// 1. 项目目录
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return nil, fmt.Errorf("创建项目目录失败: %w", err)
	}
	os.MkdirAll(filepath.Join(projectDir, "sessions"), 0755)

	// 2. config.json（语言）
	cfg := config.DefaultConfigForLang(lang)
	cfg.Language = lang
	if err := config.SaveConfig(filepath.Join(projectDir, "config.json"), cfg); err != nil {
		return nil, fmt.Errorf("初始化项目配置失败: %w", err)
	}

	// 3. settings.json：角色 + 世界观 + 组织
	settings := buildSettings(data)

	// 4. progress.json：大纲 + 梗概
	progress := buildProgress(data, lang)

	if err := story.SaveProjectSettings(filepath.Join(projectDir, "settings.json"), settings); err != nil {
		return nil, fmt.Errorf("保存项目设定失败: %w", err)
	}
	if err := story.SaveProgress(filepath.Join(projectDir, "progress.json"), progress); err != nil {
		return nil, fmt.Errorf("保存项目进度失败: %w", err)
	}

	projectName := filepath.Base(projectDir)
	return &SeedResult{
		ProjectName:         projectName,
		WorldName:           data.WorldName,
		Language:            lang,
		CharacterCount:      len(settings.Characters),
		WorldviewCount:      len(settings.Worldview),
		OrganizationCount:   len(settings.Organizations),
		OutlineChapterCount: len(progress.Chapters),
		Day:                 data.Day,
		Reused:              true,
	}, nil
}

// buildSettings 世界角色/势力 → 小说角色/组织，世界书 → 世界观条目。
func buildSettings(data *WorldData) *story.ProjectSettings {
	ps := &story.ProjectSettings{}

	// 角色
	for _, c := range data.Characters {
		notes := ""
		if c.Role != "" {
			notes = "世界定位：" + c.Role
		}
		if c.Secret != "" {
			if notes != "" {
				notes += "；"
			}
			notes += "秘密：" + c.Secret
		}
		ps.Characters = append(ps.Characters, story.Character{
			ID:          ps.NextCharacterID(),
			Name:        c.Name,
			Personality: c.Personality,
			Background:  c.Background,
			Motivation:  c.Motivation,
			Abilities:   c.Abilities,
			Notes:       notes,
		})
	}

	// 世界观（来自世界书结构化字段）
	if wb := data.Worldbook; wb != nil {
		appendWorldview := func(category, name, desc string) {
			if desc = strings.TrimSpace(desc); desc == "" {
				return
			}
			ps.Worldview = append(ps.Worldview, story.WorldviewEntry{
				ID: ps.NextWorldviewID(), Category: category, Name: name, Description: desc,
			})
		}
		appendWorldview("世界观", "世界观总纲", wb.A1Worldview)
		appendWorldview("规则", "物理与超自然规则", wb.A2Physics)
		appendWorldview("社会", "社会结构", wb.A3Society)
		appendWorldview("地理", "地理", wb.A4Geography)
		appendWorldview("势力", "势力速览", wb.A5Factions)
		appendWorldview("目标", "主角目标链", wb.A6GoalChain)
		appendWorldview("力量体系", "能力成长体系", wb.A7PowerSys)
		appendWorldview("反派", "反派行动线", wb.A8Villain)
		appendWorldview("秘密", "世界深层秘密", wb.B1Secrets)
		appendWorldview("基调", "题材基调", wb.C0Tone)
		appendWorldview("事件谱", "本世界事件谱", wb.B5EventPool)
		for _, layer := range wb.DeferredLayers {
			appendWorldview("深层世界", layer.Title, layer.Content)
		}
	}

	// 组织（来自世界势力）
	ps.Organizations = data.Organizations

	return ps
}

// buildProgress 近期事件 → 开篇章节；世界书 → 故事梗概。
func buildProgress(data *WorldData, lang string) *story.Progress {
	p := &story.Progress{
		Phase:  "outline",
		Title:  data.WorldName,
		Chapters: []story.ChapterState{},
	}

	// 故事梗概：世界观 + 目标链 + 反派线 + 基调（作为后续大纲/章节生成的既有设定）
	var synopsis []string
	if wb := data.Worldbook; wb != nil {
		if strings.TrimSpace(wb.A1Worldview) != "" {
			synopsis = append(synopsis, "【世界观】"+wb.A1Worldview)
		}
		if strings.TrimSpace(wb.A6GoalChain) != "" {
			synopsis = append(synopsis, "【主角目标】"+wb.A6GoalChain)
		}
		if strings.TrimSpace(wb.A8Villain) != "" {
			synopsis = append(synopsis, "【反派行动线】"+wb.A8Villain)
		}
		if strings.TrimSpace(wb.C0Tone) != "" {
			synopsis = append(synopsis, "【题材基调】"+wb.C0Tone)
		}
	}
	p.StorySynopsis = strings.Join(synopsis, "\n\n")

	// 开篇章节：近期事件 → 钩子
	for i, ev := range data.RecentEvents {
		day := data.Day - (len(data.RecentEvents) - 1 - i)
		if day < 1 {
			day = 1
		}
		title := fmt.Sprintf("第%d日", day)
		if lang == i18n.LangEN {
			title = fmt.Sprintf("Day %d", day)
		}
		p.Chapters = append(p.Chapters, story.ChapterState{
			Num:     i + 1,
			Title:   title,
			Outline: ev,
			Status:  story.StatusPending,
		})
	}

	return p
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}