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

// BodyInit 身体状态初始结构：多维数值 vitals + 自由描述 desc
type BodyInit struct {
	Vitals map[string]float64 `json:"vitals,omitempty"` // 维度与数值，如 {"体力":80,"精神":70,"健康":90}
	Desc   string             `json:"desc,omitempty"`   // 状态描述，如 "轻度感冒·通宵后很疲惫"
}

// ---------- 世界初始化 Agent（按世界书生成初始主角/NPC/地点，杜绝"套都市模板"） ----------
// 每个新世界都应该有自己的起点：修仙=山村砍柴少年，末世=避难所幸存者，都市=夜班店员。
// 模板只给"维度"（主角要有人设/位置/职业/现状），内容由 LLM 按世界书推导。

type WorldInitPlan struct {
	Protagonist struct {
		Name     string             `json:"name"`
		Location string             `json:"location"`
		Job      string             `json:"job"`
		Money    int                `json:"money"`            // 兼容旧字段（保留回退）
		Health   int                `json:"health"`           // 兼容旧字段（保留回退）
		Assets   map[string]float64 `json:"assets,omitempty"` // 资产表：通用键值（按世界书定维度，如 {"现金":3386,"功德值":0}）
		Body     BodyInit           `json:"body,omitempty"`   // 身体状态：多维数值 + 描述
		Profile  string             `json:"profile"`          // 一句话现状（注入 extra.profile）
		Memory   string             `json:"memory"`           // 初始记忆（他记得什么）
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
{"protagonist":{"name":"主角名","location":"初始位置（贴合世界：修仙=山村/宗门，末世=营地/废墟，都市=出租屋/便利店）","job":"职业/身份","assets":{"资产名":数值}, "body":{"vitals":{"维度":数值},"desc":"描述"}, "profile":"一句话现状（贴合世界质感，如'山村砍柴少年，家中只有寡母'）","memory":"主角的初始记忆（他记得的最近的事）"},
"npcs":[{"name":"常驻NPC名","location":"位置","job":"身份","profile":"一句话身份（他清楚自己是谁，主角不知道全貌）","memory":"他记得什么"}],
"seed_locations":[{"name":"地点名","type":"村镇|宗门|坊市|城区|自然|建筑|交通","note":"地点简述"}],
"world_events":["初始背景事件（世界正在发生的事，如'青云宗开山门测灵根的日子近了'）"]}
要求：
1. **一切按世界书来**：主角的职业/处境/目标必须从世界书的弧线（B3）和世界观（A1）推导，NPC 从势力（A3）和秘密（B1）推导——绝不允许套用"都市待业青年/便利店老板"这类别的世界的模板。
2. 主角要有"生活质感"：他的日常（砍柴/烧火/数灵石/排队领饭）是他那个世界真实的生活，不是抽象设定。
3. NPC 2~3个：1个与主角有日常接触的（同村/同门/邻居），1个是"知道内情的人"（从世界书秘密里选），1个是势力代表（从世界书势力里选）。
4. **assets（资产表）**：按世界书和主角处境，用这个世界特有的资产维度做键值表。都市世界可有"现金/存款/欠款/功德值"，修仙世界可有"铜钱/灵石/丹药/粮袋"，末世世界可有"物资点/食物/弹药"。值用该世界能理解的数值。**不要照抄例子，按世界书定维度。**
5. **body（身体状态）**：vitals 是主角当前的多维身体/精神数值（都市常见：体力/精神/健康；修仙常见：灵力/伤势/心境；按世界和处境选合理维度，2~4个即可）。desc 用一句话描述当前状态（如"吃泡面度日，前途未卜"）。不要写成空洞的"良好"。
6. 只输出 JSON，不要其他文字。`
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
	base := "entities." + hero
	ch = append(ch,
		engine.Change{Path: base + ".location", Op: "set", Value: p.Protagonist.Location},
		engine.Change{Path: base + ".job", Op: "set", Value: p.Protagonist.Job},
		engine.Change{Path: base + ".alive", Op: "set", Value: true},
		engine.Change{Path: base + ".status", Op: "set", Value: "active"},
		engine.Change{Path: base + ".extra.role", Op: "set", Value: "protagonist"},
		engine.Change{Path: base + ".extra.profile", Op: "set", Value: p.Protagonist.Profile},
		engine.Change{Path: base + ".extra.memory", Op: "set", Value: p.Protagonist.Memory},
	)
	// 资产表（新结构优先；LLM 未给 assets 时回退到 money）
	if len(p.Protagonist.Assets) > 0 {
		for name, val := range p.Protagonist.Assets {
			ch = append(ch, engine.Change{Path: base + ".assets." + name, Op: "set", Value: val})
		}
	} else {
		ch = append(ch, engine.Change{Path: base + ".money", Op: "set", Value: p.Protagonist.Money})
	}
	// 身体状态（新结构优先；LLM 未给 body 时回退到 health）
	if len(p.Protagonist.Body.Vitals) > 0 || strings.TrimSpace(p.Protagonist.Body.Desc) != "" {
		for dim, val := range p.Protagonist.Body.Vitals {
			ch = append(ch, engine.Change{Path: base + ".body.vitals." + dim, Op: "set", Value: val})
		}
		if d := strings.TrimSpace(p.Protagonist.Body.Desc); d != "" {
			ch = append(ch, engine.Change{Path: base + ".body.desc", Op: "set", Value: d})
		}
	} else {
		ch = append(ch, engine.Change{Path: base + ".health", Op: "set", Value: p.Protagonist.Health})
	}
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
