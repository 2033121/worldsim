package sim

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"

	"worldsim/internal/engine"
)

// ---------- 开放式关系系统（P0-1：无预设单/多女主，感情自由演化） ----------
// 设计原则：
//   · 不预设谁成为女主——新角色由世界自然产生，靠互动积累好感，演化为恋人/朋友/路人/对手
//   · 感情可升温（在一起）也可破裂（分开/淡出），状态机双向可走
//   · 双向视角：主角对TA的好感 + TA对主角的好感各自独立演化

// 关系状态（正向推进 + 破裂路径）
const (
	RelStranger     = "stranger"     // 陌生人
	RelAcquaintance = "acquaintance" // 认识
	RelFriend       = "friend"       // 朋友
	RelCloseFriend  = "close_friend" // 挚友
	RelCrush        = "crush"        // 心动
	RelDating       = "dating"       // 交往
	RelLover        = "lover"        // 恋人
	RelPartner      = "partner"      // 伴侣/道侣
	RelConflict     = "conflict"     // 冲突
	RelEstranged    = "estranged"    // 疏远
	RelEx           = "ex"           // 已分开/决裂
)

// RelStatusLabel 由好感+信任映射关系状态（纯函数，无预设上限）
func RelStatusLabel(affinity, trust float64) string {
	switch {
	case affinity >= 0.92 && trust >= 0.75:
		return RelPartner
	case affinity >= 0.83:
		return RelLover
	case affinity >= 0.72 && trust >= 0.5:
		return RelDating
	case affinity >= 0.62:
		return RelCrush
	case affinity >= 0.48:
		return RelCloseFriend
	case affinity >= 0.32:
		return RelFriend
	case affinity >= 0.12:
		return RelAcquaintance
	default:
		return RelStranger
	}
}

// RelationshipView 某角色视角下的一段关系（用于查询/展示）
type RelationshipView struct {
	Other     string   `json:"other"`
	Affinity  float64  `json:"affinity"`  // 好感 -1~1（负=厌恶）
	Trust     float64  `json:"trust"`     // 信任 0~1
	Status    string   `json:"status"`    // 关系状态
	SinceDay  int      `json:"since_day"` // 初遇日
	Events    []string `json:"events"`    // 关系大事记
	Role      string   `json:"role"`      // 对方在剧情中的角色定位
}

// ---------- 关系读写（走 State Engine 提案，引擎是唯一事实源） ----------

// ReadRelation 读主角对 other 的关系视角（从当前状态）
func (s *Simulator) ReadRelation(hero, other string) RelationshipView {
	st := s.engine.State()
	v := RelationshipView{Other: other, Status: RelStranger, Role: "npc"}
	he := st.Entities[hero]
	oe := st.Entities[other]
	if a, ok := he.Relationship[other]; ok {
		v.Affinity = a
	}
	if t, ok := he.Extra["rel_trust_"+other].(float64); ok {
		v.Trust = t
	} else if t, ok := he.Extra["rel_trust_"+other].(int); ok {
		v.Trust = float64(t)
	}
	if stt, ok := he.Extra["rel_status_"+other].(string); ok {
		v.Status = stt
	} else {
		v.Status = RelStatusLabel(v.Affinity, v.Trust)
	}
	if d, ok := he.Extra["rel_since_"+other].(float64); ok {
		v.SinceDay = int(d)
	}
	if evs, ok := he.Extra["rel_events_"+other].(string); ok && evs != "" {
		v.Events = strings.Split(evs, "；")
	}
	if r, ok := oe.Extra["role"].(string); ok {
		v.Role = r
	}
	return v
}

// UpdateRelation 更新双向关系（好感/信任/状态/大事记），返回变更提案
func (s *Simulator) UpdateRelation(hero, other string, affDelta, trustDelta float64, event string) []engine.Change {
	cur := s.ReadRelation(hero, other)
	aff := clamp(cur.Affinity+affDelta, -1, 1)
	trust := clamp(cur.Trust+trustDelta, 0, 1)
	status := RelStatusLabel(aff, trust)
	// 关系大事记
	events := cur.Events
	if event != "" {
		events = append(events, fmt.Sprintf("Day%d·%s", s.day, event))
		if len(events) > 12 {
			events = events[len(events)-12:]
		}
	}
	since := cur.SinceDay
	if since == 0 {
		since = s.day
	}

	var changes []engine.Change
	add := func(path string, v any) {
		changes = append(changes, engine.Change{Path: path, Op: "set", Value: v})
	}
	add(fmt.Sprintf("entities.%s.relationship.%s", hero, other), round2(aff))
	add(fmt.Sprintf("entities.%s.extra.rel_trust_%s", hero, other), round2(trust))
	add(fmt.Sprintf("entities.%s.extra.rel_status_%s", hero, other), status)
	add(fmt.Sprintf("entities.%s.extra.rel_since_%s", hero, other), since)
	add(fmt.Sprintf("entities.%s.extra.rel_events_%s", hero, other), strings.Join(events, "；"))
	// 对方对主角的好感同步演化（对方感受略滞后、幅度略小）
	oc := s.ReadRelation(other, hero)
	oaff := clamp(oc.Affinity+affDelta*0.7, -1, 1)
	ostatus := RelStatusLabel(oaff, oc.Trust)
	add(fmt.Sprintf("entities.%s.relationship.%s", other, hero), round2(oaff))
	add(fmt.Sprintf("entities.%s.extra.rel_status_%s", other, hero), ostatus)
	add(fmt.Sprintf("entities.%s.extra.rel_trust_%s", other, hero), round2(clamp(oc.Trust+trustDelta*0.6, 0, 1)))
	return changes
}

// BreakRelation 感情破裂（分手/决裂/决裂后疏远）：好感与信任骤降
func (s *Simulator) BreakRelation(hero, other string, reason string) []engine.Change {
	return s.UpdateRelation(hero, other, -0.5, -0.4, "关系破裂："+reason)
}

// DecayRelations 时间衰减：久不联系，好感/信任缓慢下降（"只见过几年"的疏远）
func (s *Simulator) DecayRelations(hero string, everyNDays int) []engine.Change {
	st := s.engine.State()
	he := st.Entities[hero]
	if he.Relationship == nil {
		return nil
	}
	var names []string
	for n := range he.Relationship {
		if n != hero {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		return nil
	}
	if s.day%everyNDays != 0 {
		return nil
	}
	var changes []engine.Change
	for _, n := range names {
		v := s.ReadRelation(hero, n)
		if v.Status == RelPartner || v.Status == RelLover || v.Status == RelDating {
			continue // 亲密关系不衰减（用心维系）
		}
		decay := -0.015
		changes = append(changes, engine.Change{Path: fmt.Sprintf("entities.%s.relationship.%s", hero, n), Op: "add", Value: decay})
		// 长时间低好感 → 状态跌回陌生人
		if v.Affinity+decay < 0.12 && v.Status != RelStranger {
			changes = append(changes, engine.Change{Path: fmt.Sprintf("entities.%s.extra.rel_status_%s", hero, n), Op: "set", Value: RelStranger})
		}
	}
	return changes
}

// ---------- 自然产生角色（女主/配角由世界演化，不预设） ----------

// NewCharacter 事件引入的新角色
type NewCharacter struct {
	Name     string `json:"name"`
	Gender   string `json:"gender"`    // 男/女/未知
	Identity string `json:"identity"`  // 职业/身份
	Persona  string `json:"persona"`   // 一句话人设（性格/背景）
	Location string `json:"location"`  // 首次出场地点
	RoleHint string `json:"role_hint"` // 剧情定位建议：love_interest(潜在女主)/important_npc(重要配角)/rival(对手)/minor_npc(普通配角)/temporary_npc(临时龙套)
	Tier     string `json:"tier"`      // 配角层级：core=核心(建档+记忆) | support=重要(轻量) | walkon=龙套(不建档不记忆)
}

// 角色影响力分级（决定出场频率/记忆深度/是否生成完整人设卡）
const (
	roleProtagonist  = "protagonist"
	roleLoveInterest = "love_interest"
	roleImportant    = "important_npc" // 重要配角：贯穿主线，出场频繁，完整记忆与人设卡
	roleRival        = "rival"         // 对手
	roleMinor        = "minor_npc"     // 普通配角：偶尔出场，随剧情可能淡出，轻记忆
	roleTemporary    = "temporary_npc" // 临时龙套：出场一两次即退场，无需记忆/人设卡
)

// isTransient 是否临时龙套（只出场一两次、不生成人设卡、短生命周期）
func isTransient(role string) bool { return role == roleTemporary }

// isMajorRole 是否值得常驻（重要配角/潜在女主/对手）——决定是否长期维护
func isMajorRole(role string) bool {
	switch role {
	case roleLoveInterest, roleImportant, roleRival:
		return true
	}
	return false
}

// RegisterCharacter 注册新角色实体（谁登场由世界决定，是否成为女主由互动决定）
// 分层：core=核心配角（完整人设+记忆）| support=重要配角（轻量）| walkon=龙套（不建档不占记忆）
func (s *Simulator) RegisterCharacter(c NewCharacter) []engine.Change {
	if c.Name == "" {
		return nil
	}
	// 已存在则跳过
	if _, ok := s.engine.State().Entities[c.Name]; ok {
		return nil
	}
	role := firstNonEmpty(c.RoleHint, "minor_npc")
	tier := strings.TrimSpace(c.Tier)
	if tier == "" {
		// 未标注：按 role_hint 推断（love_interest/rival/important_npc=core，其余=support）
		switch c.RoleHint {
		case "love_interest", "rival", "important_npc":
			tier = "core"
		default:
			tier = "support"
		}
	}
	var changes []engine.Change
	// 兜底地点：主角所在地优先（任何世界通用）
	fallbackLoc := ""
	if h, ok := s.engine.State().Entities[s.heroName]; ok {
		fallbackLoc = h.Location
	}
	if fallbackLoc == "" {
		fallbackLoc = "本地"
	}
	changes = append(changes,
		engine.Change{Path: "entities." + c.Name + ".location", Op: "set", Value: firstNonEmpty(c.Location, fallbackLoc)},
		engine.Change{Path: "entities." + c.Name + ".job", Op: "set", Value: firstNonEmpty(c.Identity, "本地人")},
		engine.Change{Path: "entities." + c.Name + ".extra.identity", Op: "set", Value: firstNonEmpty(c.Identity, "本地人")},
		engine.Change{Path: "entities." + c.Name + ".alive", Op: "set", Value: true},
		engine.Change{Path: "entities." + c.Name + ".status", Op: "set", Value: "active"},
		engine.Change{Path: "entities." + c.Name + ".health", Op: "set", Value: 90},
		engine.Change{Path: "entities." + c.Name + ".extra.persona", Op: "set", Value: firstNonEmpty(c.Persona, "一个普通市民")},
		engine.Change{Path: "entities." + c.Name + ".extra.gender", Op: "set", Value: firstNonEmpty(c.Gender, "未知")},
		engine.Change{Path: "entities." + c.Name + ".extra.role", Op: "set", Value: role},
		engine.Change{Path: "entities." + c.Name + ".extra.tier", Op: "set", Value: tier},
		engine.Change{Path: "entities." + c.Name + ".extra.debut_day", Op: "set", Value: s.day},
		engine.Change{Path: "entities." + c.Name + ".extra.last_active_day", Op: "set", Value: s.day},
	)
	// 临时龙套：短生命周期（出场后 1~3 天自动退场），不生成完整人设卡
	if role == roleTemporary {
		changes = append(changes,
			engine.Change{Path: "entities." + c.Name + ".extra.transient", Op: "set", Value: true},
			engine.Change{Path: "entities." + c.Name + ".extra.exit_day", Op: "set", Value: s.day + 1 + rand.Intn(2)},
		)
	}
	// 龙套（walkon）：轻量注册，不建档不占记忆——只留一句话人设，出场即走
	if tier == "walkon" {
		return changes
	}
	// 主角认识TA（核心/重要配角才有关系值）
	changes = append(changes, engine.Change{Path: "entities." + s.heroName + ".relationship." + c.Name, Op: "set", Value: 0.08})
	changes = append(changes, engine.Change{Path: "entities." + s.heroName + ".extra.rel_since_" + c.Name, Op: "set", Value: s.day})
	// 若背景人物晋升为配角：从背景池移除（不再只是"背景里的人"）
	if s.engine.State().WorldLevel.Background != nil {
		if _, isBg := s.engine.State().WorldLevel.Background[c.Name]; isBg {
			changes = append(changes, engine.Change{Path: "world_level.background." + c.Name, Op: "del"})
		}
	}
	return changes
}

// ---------- 角色生命周期（"只见过几年"：到点离开/远去/死亡） ----------

// CheckLifecycle 检查角色生命周期：活跃期结束 → 触发离开（由世界Agent写理由）
// 包括两类：
//
//	· 临时龙套（mitransient）：出场后自动到期退场（status→departed）
//	· 普通配角淡出：久未出场（>fadeThresholdDays）→ 标记 dormant（"背景化"），不再作为常驻活跃角色，
//	  但仍在名册中标注"已淡出"，供事件 Agent 偶尔提及/召回（符合"随剧情淡出视野、偶尔提起"）
func (s *Simulator) CheckLifecycle() []engine.Change {
	st := s.engine.State()
	var changes []engine.Change
	for name, ent := range st.Entities {
		if name == s.heroName || ent.Status == "departed" {
			continue
		}
		role, _ := ent.Extra["role"].(string)
		// 1) 临时龙套：到点退场
		if isTransient(role) {
			exitDay, _ := ent.Extra["exit_day"].(float64)
			if int(exitDay) > 0 && int(exitDay) <= s.day {
				changes = append(changes,
					engine.Change{Path: "entities." + name + ".status", Op: "set", Value: "departed"},
					engine.Change{Path: "entities." + name + ".extra.exit_reason", Op: "set", Value: "临时出场后淡出（Day" + fmt.Sprint(s.day) + "）"},
				)
				continue
			}
		}
		// 2) 普通配角淡出：非临时、非重要角色，久未出场 → 背景化（dormant）
		if !isMajorRole(role) && !isTransient(role) && ent.Status == "active" {
			lastDay, _ := ent.Extra["last_active_day"].(float64)
			if int(lastDay) > 0 && s.day-int(lastDay) >= fadeThresholdDays {
				changes = append(changes,
					engine.Change{Path: "entities." + name + ".status", Op: "set", Value: "dormant"},
					engine.Change{Path: "entities." + name + ".extra.dormant_day", Op: "set", Value: s.day},
				)
			}
		}
	}
	return changes
}

// fadeThresholdDays 普通配角淡出阈值：连续这么多天未出场即背景化
const fadeThresholdDays = 21

// FadeOutCheck 配角淡出机制：小说里配角不是永远在场的——
// 长时间没出场的 support/core 配角自动降级为 "mentioned"（只在编年史/事件里被提起，不再主动登场），
// 之后若再次与主角互动则恢复 active。龙套（walkon）不参与（本来就轻量）。
func (s *Simulator) FadeOutCheck() []engine.Change {
	st := s.engine.State()
	if s.llm == nil {
		return nil // 无 LLM 时不淡出（避免 dry-run 误伤）
	}
	var changes []engine.Change
	for name, ent := range st.Entities {
		if name == s.heroName || ent.Status != "active" {
			continue
		}
		tier, _ := ent.Extra["tier"].(string)
		if tier == "walkon" || tier == "" {
			continue // 龙套/未知层级不参与淡出
		}
		lastSeen, _ := ent.Extra["last_seen_day"].(float64)
		// 出场不足 15 天（还在新鲜期）或 30 天内见过 → 不淡出
		if int(lastSeen) <= 0 || s.day-int(lastSeen) < 30 {
			continue
		}
		changes = append(changes,
			engine.Change{Path: "entities." + name + ".status", Op: "set", Value: "mentioned"},
			engine.Change{Path: "entities." + name + ".extra.faded_day", Op: "set", Value: s.day},
			engine.Change{Path: "entities." + name + ".extra.faded_reason", Op: "set", Value: "淡出视野（Day" + fmt.Sprint(s.day) + "后仅偶尔被提起）"},
		)
		// 主角记忆沉淀："好久没见过TA了"
		s.mem.AddDay(s.heroName, fmt.Sprintf("%s已经很久没出现在生活里了，只在别人嘴里偶尔听到", name), "event", 0.7, s.day)
	}
	return changes
}

// ---------- 工具 ----------

// castNames 返回当前世界角色名单（主角 + 活跃NPC），用于生成/维护人设档案
// 分层：core/support 参与建档；walkon（龙套）/临时龙套 不建档不占记忆
func (s *Simulator) castNames() []string {
	var names []string
	seen := map[string]bool{}
	if s.heroName != "" {
		names = append(names, s.heroName)
		seen[s.heroName] = true
	}
	for name, ent := range s.engine.State().Entities {
		if seen[name] {
			continue
		}
		if ent.Status != "active" {
			continue
		}
		if role, _ := ent.Extra["role"].(string); isTransient(role) {
			continue
		}
		if tier, _ := ent.Extra["tier"].(string); tier == "walkon" {
			continue // 龙套不建档
		}
		names = append(names, name)
	}
	return names
}

// castRoster 生成角色名册（供事件Agent编排出场/淡出）：名字+定位+地点+最近出场+活跃状态
// 含 active 与 dormant（淡出）角色，dormant 标注以供"偶尔提及/召回"。
func (s *Simulator) castRoster() string {
	st := s.engine.State()
	var lines []string
	var names []string
	for name := range st.Entities {
		if name != s.heroName {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	roleLabel := map[string]string{
		roleLoveInterest: "潜在女主", roleImportant: "重要配角", roleRival: "对手",
		roleMinor: "普通配角", roleTemporary: "临时龙套",
	}
	for _, name := range names {
		ent := st.Entities[name]
		if ent.Status == "departed" {
			continue
		}
		role, _ := ent.Extra["role"].(string)
		label := roleLabel[role]
		if label == "" {
			label = "配角"
		}
		lastDay, _ := ent.Extra["last_active_day"].(float64)
		statusMark := "活跃"
		if ent.Status == "dormant" {
			statusMark = "已淡出(偶尔提及即可)"
		} else if isTransient(role) {
			statusMark = "临时(出场1~2次)"
		}
		lines = append(lines, fmt.Sprintf("· %s【%s】%s，地点%s，最近出场Day%d（%s）",
			name, label, statusMark, firstNonEmpty(ent.Location, "?"), int(lastDay), statusMark))
	}
	if len(lines) == 0 {
		return "（暂无已出场配角）"
	}
	return strings.Join(lines, "\n")
}

// formatBackground 生成背景人物池文本（供事件Agent决定是否晋升为配角）
func (s *Simulator) formatBackground() string {
	bg := s.engine.State().WorldLevel.Background
	if len(bg) == 0 {
		return "（暂无背景人物）"
	}
	var names []string
	for n := range bg {
		names = append(names, n)
	}
	sort.Strings(names)
	var lines []string
	for _, n := range names {
		lines = append(lines, fmt.Sprintf("· %s：%s", n, bg[n]))
	}
	return "背景人物池（还不是NPC，只在主角生活圈里被远远看到/听说过；当某个背景人物变得重要时，可把它晋升为正式配角/对手/潜在女主）：\n" + strings.Join(lines, "\n")
}

// touchLastActive 记录某角色今天出场（出现在事件NPC或新角色）
func (s *Simulator) touchLastActive(name string) []engine.Change {
	if name == "" || name == s.heroName {
		return nil
	}
	ent, ok := s.engine.State().Entities[name]
	if !ok {
		return nil
	}
	if last, _ := ent.Extra["last_active_day"].(float64); int(last) == s.day {
		return nil
	}
	// 淡出后被召回：重新激活
	ret := []engine.Change{engine.Change{Path: "entities." + name + ".extra.last_active_day", Op: "set", Value: s.day}}
	if ent.Status == "dormant" {
		ret = append(ret, engine.Change{Path: "entities." + name + ".status", Op: "set", Value: "active"})
	}
	return ret
}

func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
