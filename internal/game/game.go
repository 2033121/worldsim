// Package game 实现「世界模拟 → 文字游戏」的游玩层（Play Mode）。
//
// 设计原则（对标 AI Dungeon / KoboldAI adventure mode / Inform7 rulebook）：
//   - 数值/背包/检定/等级全部在代码层持有（game.json）；LLM 不靠 prompt 记住数值；
//   - LLM 只做两件事：①裁判——把玩家自由输入映射为意图+难度+申报变更；②叙述者——
//     把既定掷骰结果讲成第二人称游戏叙事；
//   - 回合管线：输入 → 裁判 → 代码掷骰(d20+属性修正 vs DC) → 叙述 → 落账；
//   - wait 回合 = 玩家挂机：世界继续转、休息回血，制造补给/压力张力。
//
// 本包不直接写世界状态机：数值变化由 main.go 的 HTTP 层映射回 engine.Changes
// （entities.<主角>.health/money/location/…），保证小说播种链路继续可用。
package game

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Caller 是 LLM 调用抽象（main.go 注入 sim.LLMClient.CompleteTier；Mock 模式天然可用）
type Caller func(ctx context.Context, tier, system, user string) (string, error)

// TurnLog 单回合记录（游戏账本）
type TurnLog struct {
	Turn      int          `json:"turn"`
	Day       int          `json:"day"`
	Input     string       `json:"input"`
	Mode      string       `json:"mode"` // do | say | story | start
	Narration string       `json:"narration"`
	Check     *CheckResult `json:"check,omitempty"`
}

// CheckResult 技能检定结果（纯代码掷骰，LLM 只解释不掷骰）
type CheckResult struct {
	Ability string `json:"ability"`
	DC      int    `json:"dc"`
	Roll    int    `json:"roll"`
	Mod     int    `json:"mod"`
	Success bool   `json:"success"`
}

// GameState 游戏会话状态（世界目录 game.json；数值层唯一可信来源）
type GameState struct {
	Enabled bool           `json:"enabled"`
	Turn    int            `json:"turn"`
	Mode    string         `json:"mode"`
	Attrs   map[string]int `json:"attrs,omitempty"` // key 由开局按主题生成（1~10），引擎不硬编码
	HP      int            `json:"hp"`
	MaxHP   int            `json:"max_hp"`
	XP      int            `json:"xp"`
	Level   int            `json:"level"`
	Gold    int            `json:"gold"`
	// Inventory 背包（worldbook/主题包驱动的资产语义保留在引擎 Assets 侧；此处是玩家随身直观背包）
	Inventory []string       `json:"inventory,omitempty"`
	Location  string         `json:"location,omitempty"`
	Quest     string         `json:"quest,omitempty"`
	Relations map[string]int `json:"relations,omitempty"` // 好感度（-10..10，裁判申报、代码收数钳制）
	Downed    bool           `json:"downed,omitempty"`    // 倒下（HP 0）：等待回血恢复或重开
	Log       []TurnLog      `json:"log"`                 // 最近 200 条
	// Prev 单步撤销快照（上一回合前的状态；不含日志与 prev 自身，避免体积膨胀）
	Prev *GameState `json:"prev,omitempty"`
}

// 出生/兜底/上限常量
const (
	GameStateFile = "game.json"
	maxLog        = 200
	xpPerLevel    = 100
	maxAttr       = 10
)

// Game 一个世界的游玩会话
type Game struct {
	mu      sync.Mutex
	dir     string
	state   GameState
	turning bool // 回合单飞标志（mu 保护）：LLM 期间不持锁，但同一世界不并发两回合
}

// cloneState 深拷贝（map/slice 独立；Prev 置空防递归膨胀）
func cloneState(st GameState) GameState {
	c := st
	if st.Attrs != nil {
		c.Attrs = make(map[string]int, len(st.Attrs))
		for k, v := range st.Attrs {
			c.Attrs[k] = v
		}
	}
	if st.Relations != nil {
		c.Relations = make(map[string]int, len(st.Relations))
		for k, v := range st.Relations {
			c.Relations[k] = v
		}
	}
	if st.Inventory != nil {
		c.Inventory = append([]string(nil), st.Inventory...)
	}
	if st.Log != nil {
		c.Log = append([]TurnLog(nil), st.Log...)
	}
	c.Prev = nil
	return c
}

// Load 从世界目录读取游戏状态（缺失/损坏 → 空会话）
func Load(dir string) *Game {
	g := &Game{dir: dir}
	if data, err := os.ReadFile(filepath.Join(dir, GameStateFile)); err == nil {
		var st GameState
		if json.Unmarshal(data, &st) == nil {
			g.state = st
		}
	}
	return g
}

// StateCurrent 返回状态快照
func (g *Game) StateCurrent() GameState {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.state
}

func (g *Game) save() {
	if data, err := json.MarshalIndent(g.state, "", "  "); err == nil {
		_ = os.WriteFile(filepath.Join(g.dir, GameStateFile), data, 0644)
	}
}

func (g *Game) appendLog(e TurnLog) {
	g.state.Log = append(g.state.Log, e)
	if len(g.state.Log) > maxLog {
		g.state.Log = g.state.Log[len(g.state.Log)-maxLog:]
	}
}

// Stop 退出游玩模式
func (g *Game) Stop() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.state.Enabled = false
	g.save()
}

// Roll 掷骰：d20 + 属性修正 vs DC（nat20 强成 / nat1 强败）。
// 属性修正=值-5（属性域 1~10）；裁判点名了面板中不存在的属性按无修正处理（不暗中罚分）。
func Roll(attrs map[string]int, ability string, dc int) CheckResult {
	mod := 0
	if v, ok := attrs[ability]; ok {
		mod = v - 5
	}
	d20 := 1 + rand.Intn(20)
	res := CheckResult{Ability: ability, DC: dc, Roll: d20, Mod: mod}
	switch {
	case d20 == 20:
		res.Success = true
	case d20 == 1:
		res.Success = false
	default:
		res.Success = d20+mod >= dc
	}
	return res
}

// clamp 收数并返回过程附注（升级时满血；HP 0=倒下，回血后自动恢复意识）
func (st *GameState) clamp() string {
	var notes []string
	if st.MaxHP <= 0 {
		st.MaxHP = 100
	}
	if st.Attrs != nil {
		for k, v := range st.Attrs {
			if v < 1 || v > maxAttr {
				if v < 1 {
					st.Attrs[k] = 1
				} else {
					st.Attrs[k] = maxAttr
				}
			}
		}
	}
	if st.HP > st.MaxHP {
		st.HP = st.MaxHP
	}
	if st.HP < 0 {
		st.HP = 0
	}
	if st.XP < 0 {
		st.XP = 0
	}
	ups := 0
	for st.XP >= xpPerLevel {
		st.XP -= xpPerLevel
		st.Level++
		st.MaxHP += 10
		st.HP = st.MaxHP
		ups++
	}
	if ups > 0 {
		notes = append(notes, fmt.Sprintf("声望与实力获得认可：Level up → %d（满血）", st.Level))
	}
	// 倒下状态机：HP 触底 → 倒下；重新有血 → 恢复意识（等待回合回血可触发）
	if st.HP <= 0 && !st.Downed {
		st.Downed = true
		notes = append(notes, "你倒下了（HP 0）——等待回合回血可恢复意识，或重新开始")
	} else if st.Downed && st.HP > 0 {
		st.Downed = false
		notes = append(notes, "你恢复了意识（HP 已回升）")
	}
	return strings.Join(notes, "；")
}

// clampAttrs 收进 1~10（外观层：数值由数据/裁判给出，代码只保证合法域）
func clampAttrs(m map[string]int) map[string]int {
	out := map[string]int{}
	for k, v := range m {
		if v < 1 {
			v = 1
		}
		if v > maxAttr {
			v = maxAttr
		}
		out[k] = v
	}
	return out
}

func zeroState() GameState {
	return GameState{
		Enabled:   true,
		Attrs:     map[string]int{"力量": 5, "敏捷": 5, "心志": 5},
		HP:        100,
		MaxHP:     100,
		Level:     1,
		Gold:      50,
		Inventory: []string{"水壶", "干粮"},
	}
}

// Start 开局：优先解码 LLM 一次调用生成的"主题自适应面板"（attrs 由裁判按世界主题生成，
// 不硬编码）；无 LLM/失败时降级 worldbook 游玩属性段（PlayAttrs 数据层）→ 通用兜底。
func (g *Game) Start(ctx context.Context, llm Caller, worldName, worldDesc, heroName, heroBrief string, worldAttrs map[string]int) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.turning {
		return "", fmt.Errorf("回合进行中，请稍后再重开")
	}
	// 注意：Start 会清空上一局（含撤销快照）——调用方 UI 应在已开局时先确认

	sys := "你是文字游戏的初始化裁判。根据世界设定与主角档案，输出纯 JSON（无 markdown 代码块）：\n" +
		`{"scene":"开场场景文字(第二人称,100-250字,写清此刻在哪/周围有什么/眼前正发生什么)",` +
		`"attrs":{"<3-4个主题适配的属性名>":1-10的整数},` +
		`"hp":<合理初始血量,默认100>,"gold":<初始资源数>,"level":1,` +
		`"inventory":["随身3-6件东西"],"location":"当前地点","quest":"当前最直接的目标(一句话)"}`
	user := fmt.Sprintf("世界名：%s\n世界设定：%s\n主角：%s\n主角档案：\n%s", worldName, worldDesc, heroName, heroBrief)
	if llm != nil {
		if out, err := llm(ctx, "fast", sys, user); err == nil {
			if j := parseJSON(out); j != nil {
				var ini struct {
					Scene     string         `json:"scene"`
					Attrs     map[string]int `json:"attrs"`
					HP        int            `json:"hp"`
					Gold      int            `json:"gold"`
					Level     int            `json:"level"`
					Inventory []string       `json:"inventory"`
					Location  string         `json:"location"`
					Quest     string         `json:"quest"`
				}
				if json.Unmarshal(j, &ini) == nil && ini.Scene != "" {
					st := zeroState()
					if len(ini.Attrs) > 0 {
						st.Attrs = ini.Attrs
					}
					if ini.HP > 0 {
						st.HP = ini.HP
					}
					if ini.Gold >= 0 {
						st.Gold = ini.Gold
					}
					if ini.Level > 0 {
						st.Level = ini.Level
					}
					if len(ini.Inventory) > 0 {
						st.Inventory = ini.Inventory
					}
					if ini.Location != "" {
						st.Location = ini.Location
					}
					if ini.Quest != "" {
						st.Quest = ini.Quest
					}
					st.clamp()
					g.state = st
					g.state.Turn = 0
					g.appendLog(TurnLog{Mode: "start", Narration: ini.Scene})
					g.save()
					return ini.Scene, nil
				}
			}
		}
	}
	// 无 LLM / 解析失败：worldbook 游玩属性段（数据层）> 通用兜底
	g.state = zeroState()
	if len(worldAttrs) > 0 {
		g.state.Attrs = clampAttrs(worldAttrs)
	}
	g.state.Location = "出发点"
	g.state.Quest = "活下去，弄清这个世界"
	scene := fmt.Sprintf("【%s】%s 睁开眼。世界书里的设定成为你眼前的现实——细节要靠你自己去碰。（未接入 LLM，这是占位开场：直接输入做什么/说什么/看哪里即可。）", worldName, heroName)
	g.state.Log = append(g.state.Log, TurnLog{Mode: "start", Narration: scene})
	g.save()
	return scene, nil
}

// RelDelta 好感度申报（在场 NPC 与玩家互动引起的情绪变化）
type RelDelta struct {
	Name  string `json:"name"`
	Delta int    `json:"delta"`
}

// refereeOutput 裁判返回的结构
type refereeOutput struct {
	Intent          string         `json:"intent"`
	Relations       []RelDelta     `json:"relations,omitempty"`
	Ability         string         `json:"ability"`
	DC              int            `json:"dc"`
	HPDelta         int            `json:"hp_delta"`
	GoldDelta       int            `json:"gold_delta"`
	XPDelta         int            `json:"xp_delta"`
	AttrsDelta      map[string]int `json:"attrs_delta"`
	InventoryAdd    []string       `json:"inventory_add"`
	InventoryRemove []string       `json:"inventory_remove"`
	Location        string         `json:"location"`
	Quest           string         `json:"quest"`
	WorldBeat       string         `json:"world_beat"`
}

const refereeSys = `你是文字游戏的裁判（数值层管理者，规则先判后写）。根据玩家输入与当前世界上下文，输出纯 JSON（无 markdown）：
{"intent":"行动意图一句话",
 "ability":"若本回合有检定，使用的属性key（必须是玩家属性表里已有的）",
 "dc":<难度5-25；无风险动作填0=免检定>,
 "hp_delta":<整数的增减，负值=受伤>, 
 "gold_delta":<整数增减>,
 "xp_delta":<正=获得经验>,
 "attrs_delta":{"属性key":<增减整数>},
 "inventory_add":["新增物品"],"inventory_remove":["移除物品"],
 "location":"若玩家移动，新地点名；否则空串",
 "quest":"任务/目标更新一句话，否则空串",
 "world_beat":"世界时钟推进：非玩家角色/环境在本回合发生了什么（1-2句，没有就空串）",
 "relations":[{"name":"本回合互动过的在场NPC名","delta":<-10~10的情绪变化，无互动则不传>}]}
纪律：玩家 HP 触底 0 会由代码判定为「倒下」（可等待恢复），你不要宣布死亡或跳过回合；
玩家血量≤15% 时危机后果要真实（负伤、被迫撤退），但也给玩家留活路。`

// Turn 一回合。ctxText=世界上下文包（main.go 拼装：主角引擎快照/在场NPC/最近叙事）。
//
// 并发纪律（v1.10.0）：单飞 + 短锁——
//   - LLM 两次调用期间不持锁（此前整回合持锁会让 /api/game/status 最长挂 300s）；
//   - turning 标志保证同一世界不并发两回合（第二个请求立即拿到明确错误）；
//   - 撤销快照在开局即取（prev），失败/停止期间回合并入提交。
func (g *Game) Turn(ctx context.Context, llm Caller, day int, input, mode, ctxText string) (string, error) {
	// ---------- 短锁①：校验 + 单飞占位 + 回血（wait）+ 撤销快照 ----------
	g.mu.Lock()
	if !g.state.Enabled {
		g.mu.Unlock()
		return "", fmt.Errorf("游戏未开启：先 POST /api/game/start")
	}
	if g.turning {
		g.mu.Unlock()
		return "", fmt.Errorf("回合进行中，请等当前回合结算")
	}
	g.turning = true
	wait := mode == "wait"
	if wait {
		// 等待回合：先代码层回血（10% 上限），世界继续自转
		if g.state.HP < g.state.MaxHP {
			g.state.HP += g.state.MaxHP / 10
			if g.state.HP > g.state.MaxHP {
				g.state.HP = g.state.MaxHP
			}
		}
		mode = "do"
	} else if mode != "do" && mode != "say" && mode != "story" {
		mode = "do"
	}
	input = strings.TrimSpace(input)
	if input == "" {
		input = "（原地等待，观察四周）"
	}
	prev := cloneState(g.state) // 撤销快照（回滚点=本回合落账前）
	prev.Log = nil
	work := cloneState(g.state)
	g.mu.Unlock()

	// ---------- 1. 裁判：意图 / 难度 / 申报变更（无锁调用 LLM） ----------
	verdict := refereeOutput{Intent: input}
	if llm != nil {
		user := fmt.Sprintf("=== 玩家面板 ===\n%s\n=== 世界上下文 ===\n%s\n=== 玩家输入(mode=%s) ===\n%s",
			sheetJSON(work), ctxText, mode, input)
		if out, err := llm(ctx, "fast", refereeSys, user); err == nil {
			if j := parseJSON(out); j != nil {
				var v refereeOutput
				if json.Unmarshal(j, &v) == nil && v.Intent != "" {
					verdict = v
				}
			}
		}
	}

	// ---------- 2. 代码掷骰（命运在骰子上，不在 prompt 里） ----------
	var chk *CheckResult
	if verdict.DC > 0 {
		r := Roll(work.Attrs, verdict.Ability, verdict.DC)
		chk = &r
	}

	// ---------- 3. 落账（对工作副本先扣后收，统一收数） ----------
	work.Turn++
	work.Mode = mode
	work.HP += verdict.HPDelta
	work.Gold += verdict.GoldDelta
	work.XP += verdict.XPDelta
	for k, v := range verdict.AttrsDelta {
		if work.Attrs == nil {
			work.Attrs = map[string]int{}
		}
		work.Attrs[k] += v
	}
	for _, it := range verdict.InventoryAdd {
		if strings.TrimSpace(it) != "" {
			work.Inventory = append(work.Inventory, it)
		}
	}
	if len(verdict.InventoryRemove) > 0 {
		next := make([]string, 0, len(work.Inventory))
		for _, it := range work.Inventory {
			dropped := false
			for _, rm := range verdict.InventoryRemove {
				if it == rm {
					dropped = true
					break
				}
			}
			if !dropped {
				next = append(next, it)
			}
		}
		work.Inventory = next
	}
	if verdict.Location != "" {
		work.Location = verdict.Location
	}
	if verdict.Quest != "" {
		work.Quest = verdict.Quest
	}
	relNote := ""
	for _, rd := range verdict.Relations {
		name := strings.TrimSpace(rd.Name)
		if name == "" {
			continue
		}
		if work.Relations == nil {
			work.Relations = map[string]int{}
		}
		cur := work.Relations[name] + rd.Delta
		if cur > 10 {
			cur = 10
		}
		if cur < -10 {
			cur = -10
		}
		work.Relations[name] = cur
		relNote += fmt.Sprintf("；好感度[%s]%+d→%d", name, rd.Delta, cur)
	}
	levelNote := work.clamp()

	// ---------- 4. 叙述者：既定结果 → 第二人称叙事（无锁调用 LLM） ----------
	outcome := "免检定，顺利"
	if chk != nil {
		verdictWord := "失败"
		if chk.Success {
			verdictWord = "成功"
		}
		outcome = fmt.Sprintf("掷d20=%d(%s修正%+d) vs 难度%d → %s", chk.Roll, chk.Ability, chk.Mod, chk.DC, verdictWord)
	}
	settle := fmt.Sprintf("血量%+d 资源%+d 经验%+d", verdict.HPDelta, verdict.GoldDelta, verdict.XPDelta)
	if verdict.Location != "" {
		settle += fmt.Sprintf("；地理转移→%q", verdict.Location)
	}
	if verdict.Quest != "" {
		settle += fmt.Sprintf("；任务→%q", verdict.Quest)
	}
	if relNote != "" {
		settle += relNote
	}
	narrSys := "你是文字游戏的叙述者。把裁判的既定结果写成第二人称游戏叙事，90-220字：直接回应玩家输入；" +
		"然后写世界反应（把 NPC/环境动态融进叙事或以「与此同时——」带出）；失败就写失败的代价；" +
		"不许修改数值结果，不许替玩家做新决定，结尾不列选项（玩家自由输入）。纯文本，不要 JSON/markdown。" +
		"若结算注明玩家已倒下，写出倒下的瞬间与周围反应（玩家仍可等待恢复或重开）。"
	narrUser := fmt.Sprintf("玩家面板(含最近叙事)：%s\n检定：%s\n世界动态：%s\n数值结算：%s\n等级结算:%s\n玩家输入(mode=%s)：%s\n裁判意图:%s",
		narrSheet(work), outcome, orDefault(verdict.WorldBeat, "无"), settle, orDefault(levelNote, "无"), mode, input, verdict.Intent)
	narr := ""
	if llm != nil {
		if out, err := llm(ctx, "normal", narrSys, narrUser); err == nil {
			narr = strings.TrimSpace(cleanNarration(out))
		}
	}
	if narr == "" { // 离线兜底
		narr = fmt.Sprintf("你尝试：%s\n（结算）%s → %s", input, verdict.Intent, outcome)
		if verdict.WorldBeat != "" {
			narr += " 与此同时——" + verdict.WorldBeat
		}
		if levelNote != "" {
			narr += "\n（" + levelNote + "）"
		}
	}

	// ---------- 短锁②：提交（唯一出口；回合期间被 stop 则尊重退出丢弃回合） ----------
	g.mu.Lock()
	defer g.mu.Unlock()
	g.turning = false
	if !g.state.Enabled {
		return narr, nil
	}
	work.Prev = &prev
	g.state = work
	g.appendLog(TurnLog{Turn: work.Turn, Day: day, Input: input, Mode: mode, Narration: narr, Check: chk})
	g.save()
	return narr, nil
}

// Undo 撤销最近一步：状态恢复到上一回合落账前（数值/背包/好感/位置/倒下位全部回滚），
// 日志去掉该回合记录。单步撤销——快照用完即弃，不链式回退。
func (g *Game) Undo() (GameState, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.state.Enabled {
		return GameState{}, fmt.Errorf("游戏未开启")
	}
	if g.turning {
		return GameState{}, fmt.Errorf("回合进行中，稍后再撤销")
	}
	if g.state.Prev == nil {
		return GameState{}, fmt.Errorf("没有可撤销的回合")
	}
	restored := cloneState(*g.state.Prev)
	if n := len(g.state.Log); n > 0 {
		restored.Log = g.state.Log[:n-1]
	} else {
		restored.Log = nil
	}
	g.state = restored
	g.save()
	return g.state, nil
}

// ImportFrom 覆盖导入存档（校验+统一收数；enabled 以存档为准；撤销快照清空）。
func (g *Game) ImportFrom(data []byte) error {
	var st GameState
	if err := json.Unmarshal(data, &st); err != nil {
		return fmt.Errorf("不是合法的存档 JSON：%w", err)
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.turning {
		return fmt.Errorf("回合进行中，稍后再导入")
	}
	if st.Turn < 0 {
		st.Turn = 0
	}
	if len(st.Log) > maxLog {
		st.Log = st.Log[len(st.Log)-maxLog:]
	}
	st.Attrs = clampAttrs(st.Attrs)
	for k, v := range st.Relations { // 好感度合法域
		if v > 10 {
			st.Relations[k] = 10
		} else if v < -10 {
			st.Relations[k] = -10
		}
	}
	st.Prev = nil
	st.clamp() // HP/属性/等级合法域 + 倒下状态机
	g.state = st
	g.save()
	return nil
}

// Wait 等待回合（世界自转+休息回血）：数值与单飞纪律统一走 Turn（mode=wait 触发回血）。
func (g *Game) Wait(ctx context.Context, llm Caller, day int, ctxText string) (string, error) {
	return g.Turn(ctx, llm, day+1,
		"（玩家选择等待/休息，观察世界按自己的节奏继续运转；留意窗口期与逼近的危机）", "wait", ctxText)
}

// ---------- helpers ----------

// parseJSON 从模型输出打捞 JSON（容忍 ``` 围栏 / 前后废话）
func parseJSON(s string) []byte {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if strings.HasPrefix(s, "{") {
		return []byte(s)
	}
	startIdx := strings.Index(s, "{")
	endIdx := strings.LastIndex(s, "}")
	if startIdx >= 0 && endIdx > startIdx {
		candidate := s[startIdx : endIdx+1]
		if json.Valid([]byte(candidate)) {
			return []byte(candidate)
		}
	}
	return nil
}

// cleanNarration 去掉模型可能的垃圾前缀
func cleanNarration(s string) string {
	s = strings.TrimSpace(s)
	for _, p := range []string{"```", "旁白：", "叙事：", "旁白:", "叙事:"} {
		if strings.HasPrefix(s, p) {
			s = strings.TrimPrefix(s, p)
		}
	}
	s = strings.TrimSuffix(strings.TrimSpace(s), "```")
	return s
}

func orDefault(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}

// sheetJSON 裁判上下文面板
func sheetJSON(st GameState) string {
	b, _ := json.Marshal(map[string]any{
		"turn": st.Turn, "level": st.Level, "hp": st.HP, "max_hp": st.MaxHP,
		"gold": st.Gold, "xp": st.XP, "attrs": st.Attrs,
		"inventory": st.Inventory, "location": st.Location, "quest": st.Quest,
		"downed": st.Downed,
	})
	return string(b)
}

// narrSheet 叙述者面板（带最近 2 条叙事，保语感连贯）
func narrSheet(st GameState) string {
	recent := make([]string, 0, 2)
	for i := len(st.Log) - 2; i < len(st.Log); i++ {
		if i >= 0 && st.Log[i].Narration != "" {
			recent = append(recent, st.Log[i].Narration)
		}
	}
	b, _ := json.Marshal(map[string]any{
		"level": st.Level, "hp": st.HP, "max_hp": st.MaxHP, "gold": st.Gold,
		"attrs": st.Attrs, "inventory": st.Inventory,
		"location": st.Location, "quest": st.Quest,
		"last_narrations": recent,
	})
	return string(b)
}
