package art

// plan.go — 素材规划层：世界书 → plan.json（素材名单 + 每条英文提示词 + 调色板）。
//
// 提示词工程分工（与 Play Mode 同哲学）：
//   - 代码层持有：风格契约 STYLE_ROLE（像素风/厚描边/统一底色/禁文字）、
//     网格布局、调色板注入——这些是"要求"，不靠 LLM 记住；
//   - LLM 层产出：本世界专属的素材名单（人物 12/怪物 8/场景 3/地图块 16/物品 24）
//     与每条内容的英文描述——这些是"该画什么"，随世界书走。
//
// plan.json 落 <world>/art/plan.json，UI 可编辑后再生成。

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"worldsim/internal/config"
	"worldsim/internal/llm"
	"worldsim/internal/worldbook"
)

// PlanAsset 单个素材条目
type PlanAsset struct {
	Name   string `json:"name"`             // 中文名（林九 / 储物袋）
	Role   string `json:"role,omitempty"`   // 人物/怪物角色定位
	Look   string `json:"look,omitempty"`   // 中文外观速写（UI 展示/用户改写用）
	Kind   string `json:"kind,omitempty"`   // tile/item 功能分类（safe_ground / consumable）
	Time   string `json:"time,omitempty"`   // scene 专用：morning|dusk|night
	Prompt string `json:"prompt"`           // 英文提示词正文（粘贴即用）
}

// Plan 素材规划（plan.json 结构）
type Plan struct {
	Version    int         `json:"version"`
	Theme      string      `json:"theme"`
	Title      string      `json:"title,omitempty"`
	Palette    string      `json:"palette"`              // 调色板文字描述
	PaletteHex []string    `json:"palette_hex,omitempty"` // 锁一致性的十六进制色板
	Characters []PlanAsset `json:"characters"`
	Monsters   []PlanAsset `json:"monsters"`
	Scenes     []PlanAsset `json:"scenes"`
	Tiles      []PlanAsset `json:"tiles"`
	Items      []PlanAsset `json:"items"`
}

// PlanFileName plan.json 文件名（存世界目录 art/ 下）
const PlanFileName = "plan.json"

// PlanDirPath 世界 art 目录
func PlanDirPath(worldDir string) string {
	return worldDir + string(os.PathSeparator) + "art"
}

// SavePlan 落盘 plan.json（原子写）
func SavePlan(worldDir string, plan *Plan) error {
	if err := os.MkdirAll(PlanDirPath(worldDir), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return err
	}
	path := PlanDirPath(worldDir) + string(os.PathSeparator) + PlanFileName
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// LoadPlan 读 plan.json；无规划返回 nil,nil
func LoadPlan(worldDir string) (*Plan, error) {
	data, err := os.ReadFile(PlanDirPath(worldDir) + string(os.PathSeparator) + PlanFileName)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var plan Plan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("解析 plan.json 失败: %w", err)
	}
	return &plan, nil
}

// ---------- 风格契约（代码层持有，注入每条 prompt） ----------

// StyleRole 全局风格契约（移植 scripts/gen_pixel_suite.py 的 STYLE_ROLE）
const StyleRole = "16-bit console pixel art, crisp hard pixels, thick dark outlines, " +
	"consistent palette, no text/letters/numbers/logos/watermarks/grid lines/shadows outside the figure, " +
	"plain solid uniform very-light-grey (#e8e8e8) background covering the whole canvas edge-to-edge, " +
	"single centered subject, original artwork only."

// 视角契约：按资产类型附加（prompt 工程的"景别术语"）
const (
	viewCharacter = "full-body front view standing pose, feet on identical baseline"
	viewMonster   = "menacing full-body pose, readable silhouette"
	viewTile      = "top-down bird's-eye view, seamless tileable"
	viewItem      = "clean icon sprite with 1-pixel dark rim"
	viewScene     = "wide banner scene, landscape composition"
)

// ViewFor 按前缀给视角契约
func ViewFor(fileLabel string) string {
	switch fileLabel {
	case "character":
		return viewCharacter
	case "monster":
		return viewMonster
	case "tile":
		return viewTile
	case "item":
		return viewItem
	case "scene":
		return viewScene
	}
	return ""
}

// paletteBlock 调色板注入文本（plan.palette_hex 逐条色值锁一致性）
func (p *Plan) paletteBlock() string {
	b := &strings.Builder{}
	if p.Palette != "" {
		b.WriteString("Palette: " + p.Palette + ".")
	}
	if len(p.PaletteHex) > 0 {
		b.WriteString(" Exact palette colors: " + strings.Join(p.PaletteHex, ", ") + ".")
	}
	return b.String()
}

// AssetPrompt 组装某条素材的完整生成 prompt（单品生成用）
func (p *Plan) AssetPrompt(fileLabel string, a PlanAsset) string {
	content := a.Prompt
	if content == "" {
		content = a.Name + " — " + orDash(a.Look)
	}
	return fmt.Sprintf("PROMPT GOAL: one pixel-art asset for the %q theme of a world-simulation game.\n\n"+
		"Subject (%s): %s\n"+
		"View: %s.\n"+
		"%s\n\n"+
		"Constraints: %s",
		p.Theme, a.Name, content, ViewFor(fileLabel), p.paletteBlock(), StyleRole)
}

// SheetPrompt 组装整张 sheet 的 prompt（批量生成用：N 项进网格，一格一项）
func (p *Plan) SheetPrompt(spec *GridSpec, assets []PlanAsset) string {
	names := make([]string, 0, len(assets))
	for _, a := range assets {
		line := a.Name
		if a.Look != "" {
			line += " (" + a.Look + ")"
		}
		names = append(names, line)
	}
	return fmt.Sprintf("PROMPT GOAL: a pixel-art asset sheet for the %q theme of a world-simulation game.\n\n"+
		"Sheet layout: %d items in a %dx%d (%d columns x %d rows) grid, each cell exactly one %s with identical framing.\n"+
		"Contents, in reading order:\n%s\n\n"+
		"View per cell: %s.\n"+
		"%s\n\n"+
		"Constraints: %s",
		p.Theme, len(assets), spec.Cols, spec.Rows, spec.Cols, spec.Rows, spec.FileLabel,
		strings.Join(numbered(names), "\n"),
		ViewFor(spec.FileLabel), p.paletteBlock(), StyleRole)
}

// SingleSheetLayout 单独生成一个 sprite 时的画布规格
const SingleSheetLayout = "1024x1024"

func numbered(items []string) []string {
	out := make([]string, len(items))
	for i, s := range items {
		out[i] = fmt.Sprintf("%d. %s", i+1, s)
	}
	return out
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// ---------- 规划 Agent ----------

// 规划 Agent 的 system prompt：世界书架构师 → 素材导演。
// 代码层只给"维度与要求"，内容必须贴合世界书。
const plannerSystem = `你是游戏美术导演。根据"世界书摘要"，为一个世界模拟游戏的像素美术套件产出素材规划，输出严格 JSON。

输出格式（只输出 JSON，不要其他文字）：
{
  "theme": "英文主题短名（如 xianxia/apocalypse/cosmic，按题材自拟）",
  "palette": "调色板英文描述（一句话：主色+辅色+点缀）",
  "palette_hex": ["#2c3e50", ...]（6~10 个十六进制色值，覆盖主色/辅色/点缀/肤色/深描边）,
  "characters": [{"name":"名","role":"主角|同伴|导师|对手|路人","look":"中文外观速写一句话","prompt":"英文"}] 恰好 12 个,
  "monsters":   [{"name":"名","role":"定位","look":"中文速写","prompt":"英文"}] 恰好 8 个,
  "scenes":     [{"time":"morning|dusk|night","place":"中文地点","look":"中文速写","prompt":"英文"}] 恰好 3 个（同一标志性地点的晨/暮/夜）,
  "tiles":      [{"name":"名","kind":"safe_ground|road|dangerous|water|forest|rock|wall|ruin|camp|bridge|gate|pit|treasure|transport|landmark","look":"中文速写","prompt":"英文"}] 恰好 16 个,
  "items":      [{"name":"名","kind":"weapon|tool|consumable|currency|valuable|quest|curio|transport","look":"中文速写","prompt":"英文"}] 恰好 24 个
}

硬性要求：
1. 数量严格：characters 12 / monsters 8 / scenes 3 / tiles 16 / items 24，不多不少。
2. 人物梯度：必须有主角（可参考世界书主角名）、核心同伴、导师、对手、若干路人；名字用世界书/题材语境的中文名。
3. 怪物按本世界的威胁谱自拟（从小喽啰到小头目），禁止套用别的题材的形象。
4. tiles 覆盖功能谱（安全地/道路/危险地/水/林/岩/墙/遗迹/营地/桥/门/坑/宝点/传送点/地标）。
5. items 覆盖经济谱（武器/工具/消耗品/货币/贵重品/任务物/奇物/代步）。
6. 每条 prompt 用英文，写"画什么"：主体+关键特征+材质/色彩意象（如 ink-blue robe, jade hairpin, antique gold trim）。不要写风格要求（统一底色/描边/禁文字等由系统注入），不要写"pixel art"以外的风格词。
7. 世界书里的美术设定段（若提供）是最高优先：外观速写与调色板优先遵循它。
8. 所有内容必须来自世界书的题材与设定，不得出现现代都市词混入修仙世界之类的串味。`

// PlanFromWorldbook 规划 Agent：1 次 normal 档调用
func PlanFromWorldbook(ctx context.Context, apiCfg *config.APIConfig, wb *worldbook.Worldbook, entities []string) (*Plan, error) {
	if apiCfg == nil {
		return nil, fmt.Errorf("LLM 配置不可用")
	}
	if wb == nil {
		return nil, fmt.Errorf("世界书未加载")
	}
	ctx = llm.WithSpan(ctx, "美术素材规划")
	user := fmt.Sprintf("世界书摘要：\n%s\n\n美术设定（可空）：\n%s\n\n实体名单（可空）：\n%s\n\n请生成素材规划 JSON。",
		worldbookDigest(wb), wb.ArtSection, strings.Join(entities, "、"))
	raw, err := llm.CallAPITier(ctx, apiCfg, "normal", plannerSystem, user)
	if err != nil {
		return nil, err
	}
	return ParsePlan(raw, wb)
}

// worldbookDigest 世界书摘要（喂规划 Agent 的裁剪版，控上下文）
func worldbookDigest(wb *worldbook.Worldbook) string {
	sec := func(label, body string, limit int) string {
		if body == "" {
			return ""
		}
		if len(body) > limit {
			body = body[:limit]
		}
		return "## " + label + "\n" + body + "\n"
	}
	b := &strings.Builder{}
	b.WriteString("# 世界书：" + wb.Title + "\n")
	b.WriteString(sec("A1 世界观", wb.A1Worldview, 600))
	b.WriteString(sec("A2 规则", wb.A2Physics, 400))
	b.WriteString(sec("A3 势力", wb.A3Society, 400))
	b.WriteString(sec("A4 地理", wb.A4Geography, 500))
	b.WriteString(sec("A6 主角目标", wb.A6GoalChain, 300))
	b.WriteString(sec("A7 成长体系", wb.A7PowerSys, 300))
	b.WriteString(sec("B5 事件谱", wb.B5EventPool, 800))
	b.WriteString(sec("C1 文风", wb.CNarrative, 200))
	return b.String()
}

// ParsePlan 解析规划 JSON（容错：代码块包裹/数量不足自动补）
func ParsePlan(raw string, wb *worldbook.Worldbook) (*Plan, error) {
	cleaned := llm.ExtractJSON(raw)
	var plan Plan
	if err := json.Unmarshal([]byte(cleaned), &plan); err != nil {
		return nil, fmt.Errorf("规划 JSON 解析失败: %w", err)
	}
	if plan.Version == 0 {
		plan.Version = 1
	}
	if plan.Theme == "" && wb != nil {
		plan.Theme = GuessTheme(wb)
	}
	if plan.Title == "" && wb != nil {
		plan.Title = wb.Title
	}
	// 数量校验：不足补占位（UI 可见可改），超出截断
	plan.Characters = clampAssets(plan.Characters, 12, "角色")
	plan.Monsters = clampAssets(plan.Monsters, 8, "怪物")
	plan.Scenes = clampScenes(plan.Scenes)
	plan.Tiles = clampAssets(plan.Tiles, 16, "地图块")
	plan.Items = clampAssets(plan.Items, 24, "物品")
	if plan.Theme == "" {
		plan.Theme = "custom"
	}
	return &plan, nil
}

func clampAssets(list []PlanAsset, n int, kindZh string) []PlanAsset {
	out := make([]PlanAsset, 0, n)
	for i := 0; i < n; i++ {
		if i < len(list) {
			out = append(out, list[i])
		} else {
			out = append(out, PlanAsset{Name: fmt.Sprintf("%s %d（待补）", kindZh, i+1)})
		}
	}
	return out
}

func clampScenes(list []PlanAsset) []PlanAsset {
	times := []string{"morning", "dusk", "night"}
	out := make([]PlanAsset, 0, 3)
	for i := 0; i < 3; i++ {
		if i < len(list) {
			if list[i].Time == "" {
				list[i].Time = times[i]
			}
			out = append(out, list[i])
		} else {
			out = append(out, PlanAsset{Time: times[i], Name: fmt.Sprintf("场景 %d（待补）", i+1)})
		}
	}
	return out
}

// GuessTheme 从世界书推断英文主题短名（与 detectPixelTheme 同映射，独立于 main 包避免循环依赖）
func GuessTheme(wb *worldbook.Worldbook) string {
	if wb == nil {
		return ""
	}
	text := strings.ToLower(wb.Title + " " + firstN(wb.Raw, 600))
	switch {
	case containsAny(text, "修仙", "仙侠", "仙", "xianxia", "cultivation"):
		return "xianxia"
	case containsAny(text, "末世", "废土", "apocalypse", "wasteland"):
		return "apocalypse"
	case containsAny(text, "西幻", "骑士", "奇幻", "fantasy", "knight"):
		return "western"
	case containsAny(text, "克苏鲁", "异界", "cosmic", "cthulhu"):
		return "cosmic"
	case containsAny(text, "星际", "科幻", "interstellar", "star"):
		return "interstellar"
	case containsAny(text, "都市", "异能", "urban", "city"):
		return "urban"
	case containsAny(text, "历史", "王朝", "history", "dynasty"):
		return "history"
	case containsAny(text, "武侠", "玄幻", "wuxia"):
		return "wuxia"
	case containsAny(text, "洪荒", "神话", "primordial", "myth"):
		return "primordial"
	case containsAny(text, "恐怖", "灵异", "horror"):
		return "horror"
	case containsAny(text, "无限", "诸天", "infinite"):
		return "infinite"
	case containsAny(text, "系统", "system"):
		return "system"
	case containsAny(text, "美食", "种田", "farm"):
		return "farm"
	case containsAny(text, "军旅", "战争", "military", "war"):
		return "war"
	case containsAny(text, "原始", "primitive"):
		return "primitive"
	}
	return ""
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func firstN(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// ValidatePlan 校验规划可生成性（所有条目都有非空 prompt）
func (p *Plan) ValidatePlan() []string {
	var warns []string
	check := func(label string, list []PlanAsset) {
		for i, a := range list {
			if a.Prompt == "" {
				warns = append(warns, fmt.Sprintf("%s#%d 缺英文 prompt", label, i+1))
			}
		}
	}
	check("character", p.Characters)
	check("monster", p.Monsters)
	check("scene", p.Scenes)
	check("tile", p.Tiles)
	check("item", p.Items)
	return warns
}
