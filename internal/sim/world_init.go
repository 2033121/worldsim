package sim

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"worldsim/internal/engine"
	"worldsim/internal/llm"
	"worldsim/internal/worldbook"
)

// ---------- 世界初始化 Agent（按世界书生成初始主角/NPC/地点，杜绝"套都市模板"） ----------
// 每个新世界都应该有自己的起点：修仙=山村砍柴少年，末世=避难所幸存者，都市=夜班店员。
// 模板只给"维度"（主角要有人设/位置/职业/现状），内容由 LLM 按世界书推导。

type WorldInitPlan struct {
	Protagonist struct {
		Name     string `json:"name"`
		Location string `json:"location"`
		Job      string `json:"job"`
		Money    int    `json:"money"`
		Health   int    `json:"health"`
		Profile  string `json:"profile"` // 一句话现状（注入 extra.profile）
		Memory   string `json:"memory"`  // 初始记忆（他记得什么）
	} `json:"protagonist"`
	NPCs []struct {
		Name     string `json:"name"`
		Location string `json:"location"`
		Job      string `json:"job"`
		Profile  string `json:"profile"` // 一句话身份（他清楚自己，主角不知道）
		Memory   string `json:"memory"`  // 初始记忆
	} `json:"npcs"`
	SeedLocations []struct {
		Name string `json:"name"`
		Type string `json:"type"` // 村镇/宗门/坊市/城区/自然/建筑
		Note string `json:"note"`
	} `json:"seed_locations"`
	WorldEvents []string `json:"world_events"` // 初始世界事件（背景设定）
}

// WorldInitPlanLLM 用 LLM 按世界书生成世界初始方案（1次 fast 调用）
func WorldInitPlanLLM(ctx context.Context, c *LLMClient, wb *worldbook.Worldbook) *WorldInitPlan {
	if c == nil || wb == nil {
		return nil
	}
	ctx = llm.WithSpan(ctx, "世界初始化")
	worldCtx := wb.ForWorldAgent()
	system := `你是这方世界的"创世者"。根据世界书，生成这个世界的**初始状态**——主角是谁、他在哪、干什么，常驻NPC，种子地点，初始背景事件。
输出严格 JSON，格式：
{"protagonist":{"name":"主角名","location":"初始位置（贴合世界：修仙=山村/宗门，末世=营地/废墟，都市=出租屋/便利店）","job":"职业/身份","money":0,"health":90,"profile":"一句话现状（贴合世界质感，如'山村砍柴少年，家中只有寡母'）","memory":"主角的初始记忆（他记得的最近的事）"},
"npcs":[{"name":"常驻NPC名","location":"位置","job":"身份","profile":"一句话身份（他清楚自己是谁，主角不知道全貌）","memory":"他记得什么"}],
"seed_locations":[{"name":"地点名","type":"村镇|宗门|坊市|城区|自然|建筑|交通","note":"地点简述"}],
"world_events":["初始背景事件（世界正在发生的事，如'青云宗开山门测灵根的日子近了'）"]}
要求：
1. **一切按世界书来**：主角的职业/处境/目标必须从世界书的弧线（B3）和世界观（A1）推导，NPC 从势力（A3）和秘密（B1）推导——绝不允许套用"都市待业青年/便利店老板"这类别的世界的模板。
2. 主角要有"生活质感"：他的日常（砍柴/烧火/数灵石/排队领饭）是他那个世界真实的生活，不是抽象设定。
3. NPC 2~3个：1个与主角有日常接触的（同村/同门/邻居），1个是"知道内情的人"（从世界书秘密里选，如陈伯），1个是势力代表（从世界书势力里选）。
4. money 用该世界的货币单位数值（修仙用铜钱/灵石数量，末世用物资点，都市用人民币）。
5. 只输出 JSON，不要其他文字。`
	raw, err := c.CompleteTier(ctx, "fast", system, "请生成这个世界的初始方案。世界书内容：\n"+worldCtx)
	if err != nil {
		return nil
	}
	jsonStr := llm.ExtractJSON(raw)
	if jsonStr == "" {
		return nil
	}
	var plan WorldInitPlan
	if json.Unmarshal([]byte(jsonStr), &plan) != nil {
		return nil
	}
	if strings.TrimSpace(plan.Protagonist.Name) == "" {
		return nil
	}
	if len(plan.SeedLocations) == 0 {
		return nil
	}
	return &plan
}

// ApplyPlan 把初始方案转成 engine.Change 列表（供 init 提交）
func (p *WorldInitPlan) Changes(hero string) []engine.Change {
	var ch []engine.Change
	ch = append(ch,
		engine.Change{Path: "world_level.tension", Op: "set", Value: 0.2},
	)
	// 主角
	ch = append(ch,
		engine.Change{Path: "entities." + hero + ".location", Op: "set", Value: p.Protagonist.Location},
		engine.Change{Path: "entities." + hero + ".money", Op: "set", Value: p.Protagonist.Money},
		engine.Change{Path: "entities." + hero + ".health", Op: "set", Value: p.Protagonist.Health},
		engine.Change{Path: "entities." + hero + ".job", Op: "set", Value: p.Protagonist.Job},
		engine.Change{Path: "entities." + hero + ".alive", Op: "set", Value: true},
		engine.Change{Path: "entities." + hero + ".status", Op: "set", Value: "active"},
		engine.Change{Path: "entities." + hero + ".extra.role", Op: "set", Value: "protagonist"},
		engine.Change{Path: "entities." + hero + ".extra.profile", Op: "set", Value: p.Protagonist.Profile},
		engine.Change{Path: "entities." + hero + ".extra.memory", Op: "set", Value: p.Protagonist.Memory},
	)
	// 常驻 NPC
	for _, n := range p.NPCs {
		ch = append(ch,
			engine.Change{Path: "entities." + n.Name + ".location", Op: "set", Value: n.Location},
			engine.Change{Path: "entities." + n.Name + ".job", Op: "set", Value: n.Job},
			engine.Change{Path: "entities." + n.Name + ".alive", Op: "set", Value: true},
			engine.Change{Path: "entities." + n.Name + ".status", Op: "set", Value: "active"},
			engine.Change{Path: "entities." + n.Name + ".extra.role", Op: "set", Value: "important_npc"},
			engine.Change{Path: "entities." + n.Name + ".extra.profile", Op: "set", Value: n.Profile},
			engine.Change{Path: "entities." + n.Name + ".extra.memory", Op: "set", Value: n.Memory},
		)
	}
	// 种子地点（写进去后 SeedLocations 检测到非空就不会套都市默认池）
	for _, l := range p.SeedLocations {
		ch = append(ch,
			engine.Change{Path: "world_level.locations." + l.Name + ".type", Op: "set", Value: l.Type},
			engine.Change{Path: "world_level.locations." + l.Name + ".state", Op: "set", Value: "正常"},
			engine.Change{Path: "world_level.locations." + l.Name + ".note", Op: "set", Value: l.Note},
		)
	}
	// 初始背景事件
	for _, e := range p.WorldEvents {
		ch = append(ch, engine.Change{Path: "world_level.global_events", Op: "add", Value: e})
	}
	return ch
}

func (p *WorldInitPlan) String() string {
	return fmt.Sprintf("主角=%s(%s@%s) NPC=%d 地点=%d 事件=%d",
		p.Protagonist.Name, p.Protagonist.Job, p.Protagonist.Location, len(p.NPCs), len(p.SeedLocations), len(p.WorldEvents))
}