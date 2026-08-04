// Package worldbook 实现世界书的加载与分层注入（设计文档 §3.3 / §4.5）
//
// 世界书 = 世界的"宪法"。按 Part A-D 组织，按 L1-L5 知识层分发：
//   - 主角/NPC 只能拿到"角色认知视角"的知识（不能全知）
//   - 世界 Agent / GM 持有全部（含 L5 系统秘密）
//   - 小说化 Agent 额外持有叙事约束（Part C）与伏笔清单（B4）
package worldbook

import (
	"fmt"
	"os"
	"strings"
)

type Worldbook struct {
	Title         string
	A1Worldview   string // 世界观（含隐藏真相部分）
	A2Physics     string // 物理/超自然规则（L1）
	A3Society     string // 社会结构（L2）
	A4Geography   string // 地理（L3）
	A5Factions    string // 势力速览（明面部分）
	A6GoalChain   string // 主角目标链（长期/阶段/即时，网文引擎）
	A7PowerSys    string // 能力成长体系（等级/解锁/升级时刻，网文引擎）
	A8Villain     string // 反派行动线（谁在动/怎么压迫，网文引擎）
	B1Secrets     string // 世界秘密（L5）
	B2EventPool   string // 事件类型池（导演内部）
	B3ArcPlan     string // 全书弧线建议（导演内部）
	B4Foreshadows string // 隐藏伏笔清单（导演内部）
	B5EventPool   string // 事件谱（本世界会发生的事，事件生成器的弹药库）
	C0Tone        string // 题材基调（本世界的讲事风格：题材类型/主角人设/核心爽点/感情线定位/禁忌——所有Agent遵守的灵魂）
	CNarrative    string // 叙事约束（小说化专属）
	DSafety       string // 内容安全边界
	Raw           string
	// 深层世界观层（E段：世界一开始就很大，随时间渐进揭示——冰山理论）
	DeferredLayers []DeferredLayer
	pendingMarker  string // 解析中的E段标题触发标记（临时）
}

// DeferredLayer 深层世界观层：世界一开始就很大，只透露一部分
// Trigger: "day"=日期到点自动浮出（保底）| "event"=靠事件/探索触发（更自然）
type DeferredLayer struct {
	Title     string `json:"title"`
	RevealDay int    `json:"reveal_day"` // 最早解锁天数（0=仅事件触发）
	Trigger   string `json:"trigger"`    // day | event
	EventHint string `json:"event_hint"` // 事件型的触发线索（注入事件Agent）
	Content   string `json:"content"`
	Revealed  bool   `json:"revealed"`
}

// Load 从 Markdown 文件加载世界书（按 "## A1 标题" 等二级标题切分）
func Load(path string) (*Worldbook, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(string(raw)), nil
}

// Parse 解析世界书文本（Markdown 二级标题切分）
func Parse(raw string) *Worldbook {
	w := &Worldbook{Raw: raw}
	lines := strings.Split(raw, "\n")
	var current []string
	var eSec, eMarker string
	var eBody []string
	// flushE 把已收集的E段正文压入 DeferredLayers
	flushE := func() {
		if eSec == "" || len(eBody) == 0 {
			eSec, eMarker, eBody = "", "", nil
			return
		}
		layer := DeferredLayer{Title: eSec, Content: strings.TrimSpace(strings.Join(eBody, "\n")), Trigger: "day"}
		marker := eMarker
		if strings.HasPrefix(marker, "事件触发") {
			layer.Trigger = "event"
			if c := strings.Index(marker, "："); c >= 0 {
				layer.EventHint = strings.TrimSpace(marker[c+len("："):])
			} else if c := strings.Index(marker, ":"); c >= 0 {
				layer.EventHint = strings.TrimSpace(marker[c+1:])
			}
		} else {
			var day int
			if _, err := fmt.Sscanf(marker, "Day%d", &day); err == nil {
				layer.RevealDay = day
			}
		}
		// 未指定天数：按 E 序号递增（E1=Day10, E2=Day20...）
		if layer.RevealDay == 0 && layer.Trigger == "day" {
			n := 0
			fmt.Sscanf(eSec[1:], "%d", &n)
			if n > 0 {
				layer.RevealDay = n * 10
			}
		}
		w.DeferredLayers = append(w.DeferredLayers, layer)
		eSec, eMarker, eBody = "", "", nil
	}
	collect := func(section string) {
		if len(current) == 0 {
			return
		}
		body := strings.TrimSpace(strings.Join(current, "\n"))
		switch section {
		case "A1":
			w.A1Worldview = body
		case "A2":
			w.A2Physics = body
		case "A3":
			w.A3Society = body
		case "A4":
			w.A4Geography = body
		case "A5":
			w.A5Factions = body
		case "A6":
			w.A6GoalChain = body
		case "A7":
			w.A7PowerSys = body
		case "A8":
			w.A8Villain = body
		case "B1":
			w.B1Secrets = body
		case "B2":
			w.B2EventPool = body
		case "B3":
			w.B3ArcPlan = body
		case "B4":
			w.B4Foreshadows = body
		case "B5":
			w.B5EventPool = body
		case "C0":
			w.C0Tone = body
		case "C1":
			// C1 文风：兼容旧世界书（含"题材基调"），映射到基调字段。
			// 若已显式定义 C0（更精确的基调守则），则 C1 作为增量补充。
			if w.C0Tone == "" {
				w.C0Tone = body
			} else {
				w.C0Tone = w.C0Tone + "\n\n" + body
			}
		case "C":
			w.CNarrative = body
		case "D":
			w.DSafety = body
		}
	}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// 匹配 "## A1 世界观" / "## A2 物理与超自然规则" / "## B1 ..." / "## E1 ..." 等
		if strings.HasPrefix(trimmed, "## ") {
			parts := strings.SplitN(strings.TrimPrefix(trimmed, "## "), " ", 2)
			sec := strings.ToUpper(strings.TrimSpace(parts[0]))
			if len(sec) == 2 && (sec[0] == 'A' || sec[0] == 'B' || sec[0] == 'C' || sec[0] == 'D' || sec[0] == 'E') {
				if sec[0] == 'E' {
					// E段独立收集：flush上一个E段，开始新E段
					flushE()
					eSec = sec
					eMarker = ""
					if len(parts) > 1 {
						head := parts[1]
						if idx := strings.Index(head, "【"); idx >= 0 {
							rest := head[idx:]
							if end := strings.Index(rest, "】"); end > 0 {
								eMarker = strings.TrimSpace(rest[len("【"):end])
							}
						}
					}
					continue
				}
				flushE()
				collect(sec)
				current = []string{}
				if w.Title == "" && sec == "A1" {
					w.Title = strings.TrimSpace(parts[1])
				}
				continue
			}
		}
		if eSec != "" {
			eBody = append(eBody, line)
			continue
		}
		if strings.HasPrefix(trimmed, "# ") && w.Title == "" {
			w.Title = strings.TrimPrefix(trimmed, "# ")
		}
		if current != nil {
			current = append(current, line)
		}
	}
	// 文件末尾：flush残留的E段 + 最后一个普通段
	flushE()
	if len(lines) > 0 {
		sec := ""
		for _, l := range lines {
			t := strings.TrimSpace(l)
			if strings.HasPrefix(t, "## ") {
				p := strings.SplitN(strings.TrimPrefix(t, "## "), " ", 2)
				s := strings.ToUpper(strings.TrimSpace(p[0]))
				if len(s) == 2 && (s[0] == 'A' || s[0] == 'B' || s[0] == 'C' || s[0] == 'D') {
					sec = s
				}
			}
		}
		if sec != "" {
			collect(sec)
		}
	}
	return w
}

// CheckDayReveals 日期型层：到点自动浮出（保底机制），返回新揭示内容
func (w *Worldbook) CheckDayReveals(currentDay int) string {
	if w == nil || len(w.DeferredLayers) == 0 {
		return ""
	}
	var newSb strings.Builder
	for i := range w.DeferredLayers {
		l := &w.DeferredLayers[i]
		if l.Trigger == "day" && l.RevealDay > 0 && currentDay >= l.RevealDay && !l.Revealed {
			l.Revealed = true
			newSb.WriteString(fmt.Sprintf("【世界深层·%s】\n%s\n", l.Title, l.Content))
		}
	}
	return strings.TrimSpace(newSb.String())
}

// TriggerByEvent 事件型层：被事件/探索命中时触发揭示（返回新揭示内容）
func (w *Worldbook) TriggerByEvent(match string) string {
	if w == nil || len(w.DeferredLayers) == 0 || match == "" {
		return ""
	}
	var newSb strings.Builder
	for i := range w.DeferredLayers {
		l := &w.DeferredLayers[i]
		if l.Trigger != "event" || l.Revealed {
			continue
		}
		hit := strings.Contains(match, l.Title)
		if !hit {
			// hint 关键词拆解：支持 "|" 或 "、" 分隔的多关键词，任一命中即触发
			sep := func(s string) []string {
				for _, sp := range []string{"|", "、", "，", ",", "；", ";", " "} {
					if strings.Contains(s, sp) {
						return strings.Split(s, sp)
					}
				}
				return []string{s}
			}
			for _, kw := range sep(l.EventHint) {
				kw = strings.TrimSpace(kw)
				if kw != "" && strings.Contains(match, kw) {
					hit = true
					break
				}
			}
		}
		if hit {
			l.Revealed = true
			newSb.WriteString(fmt.Sprintf("【世界深层·%s】\n%s\n", l.Title, l.Content))
		}
	}
	return strings.TrimSpace(newSb.String())
}

// UnrevealedHints 尚未揭示的事件型层的线索（存在感：世界很大，有些东西还没浮出水面）
func (w *Worldbook) UnrevealedHints() string {
	if w == nil || len(w.DeferredLayers) == 0 {
		return ""
	}
	var sb strings.Builder
	for i := range w.DeferredLayers {
		l := &w.DeferredLayers[i]
		if !l.Revealed {
			if l.Trigger == "event" && l.EventHint != "" {
				sb.WriteString(fmt.Sprintf("· %s（线索：%s）\n", l.Title, l.EventHint))
			} else if l.Trigger == "day" {
				sb.WriteString(fmt.Sprintf("· %s（有风声，但真相还没浮出水面）\n", l.Title))
			}
		}
	}
	return strings.TrimSpace(sb.String())
}

// RevealedAll 返回所有已揭示的深层层（给事件Agent当全貌背景，主角只能接触冰山一角）
func (w *Worldbook) RevealedAll() string {
	if w == nil || len(w.DeferredLayers) == 0 {
		return ""
	}
	var sb strings.Builder
	for i := range w.DeferredLayers {
		l := &w.DeferredLayers[i]
		if l.Revealed {
			sb.WriteString(fmt.Sprintf("【%s】%s\n", l.Title, l.Content))
		}
	}
	return strings.TrimSpace(sb.String())
}

// ForProtagonist 主角视角：社会常识+地区常识+明面势力+主角人格与基调；物理规则用"角色认知视角"
func (w *Worldbook) ForProtagonist(heroName, heroProfile string) string {
	var sb strings.Builder
	sb.WriteString("【世界·你眼中的样子（普通人的认知）】\n")
	if w.A3Society != "" {
		sb.WriteString("· 社会：\n" + w.A3Society + "\n")
	}
	if w.A4Geography != "" {
		sb.WriteString("· 地理：\n" + w.A4Geography + "\n")
	}
	if w.A5Factions != "" {
		sb.WriteString("· 你听过的势力：\n" + w.A5Factions + "\n")
	}
	// L1 角色认知视角：你确信什么、见过什么，由你的亲身经历决定（不再硬编码"从未见过超自然"，否则会与带超自然设定的世界矛盾）
	if heroProfile != "" {
		sb.WriteString("· 你自己（你是谁、经历过什么、性格如何）：" + heroProfile + "\n")
	}
	sb.WriteString(w.ForTone())
	return sb.String()
}

// ForWorldAgent 世界/GM 视角：全部知识（含 L5 秘密）
func (w *Worldbook) ForWorldAgent() string {
	var sb strings.Builder
	sb.WriteString("【世界真相（你是唯一知情人，严格保密）】\n")
	sb.WriteString("A1 世界观：\n" + w.A1Worldview + "\n")
	sb.WriteString("A2 物理/超自然规则：\n" + w.A2Physics + "\n")
	if w.B1Secrets != "" {
		sb.WriteString("B1 世界秘密（绝不泄露给角色）：\n" + w.B1Secrets + "\n")
	}
	if w.B3ArcPlan != "" {
		sb.WriteString("B3 弧线建议（导演意图）：\n" + w.B3ArcPlan + "\n")
	}
	if w.B4Foreshadows != "" {
		sb.WriteString("B4 伏笔清单：\n" + w.B4Foreshadows + "\n")
	}
	sb.WriteString(w.ForTone())
	return sb.String()
}

// ForTone 题材基调执行守则：本世界的"讲事灵魂"，所有 Agent 必须遵守。
// 决定事件怎么发生、角色怎么行动、剧情朝哪走——防止"设定对但演出来跑偏"（如把喜剧写成正剧/言情）。
func (w *Worldbook) ForTone() string {
	var sb strings.Builder
	sb.WriteString("【题材基调（本世界的讲事风格与灵魂，所有 Agent 必须严格遵守——它决定事件怎么发生、主角怎么行动、剧情朝哪走）】\n")
	if strings.TrimSpace(w.C0Tone) != "" {
		sb.WriteString(w.C0Tone + "\n")
	} else {
		sb.WriteString("（本世界未显式定义题材基调——请根据 A1 世界观的题材类型、A6 主角目标与主角身份，推断本世界的讲事风格：是喜剧/乐子人、还是正剧/热血/悬疑/末世求生？并在事件生成与角色行动中贯彻到底，不要写得四平八稳。）\n")
	}
	sb.WriteString("执行总则：\n")
	sb.WriteString("· 事件与角色行动必须贴合基调：基调是喜剧/乐子人时，事件要有爽点、梗、脑洞反转，主角主动搞事，节奏轻快，杜绝拖沓言情；基调是正剧/热血时再走厚重压迫。\n")
	sb.WriteString("· 主角人格必须符合基调（乐子人/热血/腹黑/惜字如金…），行动要带出这种人格，不能把主角写成四平八稳的正剧工具人。\n")
	sb.WriteString("· 感情线（romance）在本世界是\"调味\"还是\"主线\"，由基调决定：基调未强调感情主线时，感情线只能作为轻松点缀，绝不能反客为主主导剧情。\n")
	return sb.String()
}

// ForGM 总导演视角：完整世界背景（GM 是唯一知情人，规划剧情段落需掌握全量设定）
func (w *Worldbook) ForGM() string {
	var sb strings.Builder
	sb.WriteString("【世界背景（你是总导演，唯一知情人，规划剧情段落用）】\n")
	if w.A1Worldview != "" {
		sb.WriteString("A1 世界观：\n" + w.A1Worldview + "\n")
	}
	if w.A2Physics != "" {
		sb.WriteString("A2 规则：\n" + w.A2Physics + "\n")
	}
	if w.A3Society != "" {
		sb.WriteString("A3 社会结构：\n" + w.A3Society + "\n")
	}
	if w.A5Factions != "" {
		sb.WriteString("A5 势力：\n" + w.A5Factions + "\n")
	}
	if w.A6GoalChain != "" {
		sb.WriteString("A6 主角目标链（驱动主角行动）：\n" + w.A6GoalChain + "\n")
	}
	if w.A7PowerSys != "" {
		sb.WriteString("A7 能力成长体系（金手指节奏规划用）：\n" + w.A7PowerSys + "\n")
	}
	if w.A8Villain != "" {
		sb.WriteString("A8 反派行动线（反派要持续施压）：\n" + w.A8Villain + "\n")
	}
	if w.B1Secrets != "" {
		sb.WriteString("B1 世界秘密（绝不泄露给角色）：\n" + w.B1Secrets + "\n")
	}
	if w.B3ArcPlan != "" {
		sb.WriteString("B3 全书弧线建议（导演意图）：\n" + w.B3ArcPlan + "\n")
	}
	if w.B4Foreshadows != "" {
		sb.WriteString("B4 伏笔清单（规划段落时推进/回收）：\n" + w.B4Foreshadows + "\n")
	}
	sb.WriteString(w.ForTone())
	return sb.String()
}

// ForEventAgent 事件 Agent 视角：A + 事件类型池 + 事件谱（本世界弹药库）
func (w *Worldbook) ForEventAgent() string {
	var sb strings.Builder
	sb.WriteString("【世界背景与事件类型】\n")
	sb.WriteString("A1 世界观：\n" + w.A1Worldview + "\n")
	if w.A2Physics != "" {
		sb.WriteString("A2 世界规则：\n" + w.A2Physics + "\n")
	}
	if w.B2EventPool != "" {
		sb.WriteString("B2 事件类型池（你的弹药库）：\n" + w.B2EventPool + "\n")
	}
	if w.B5EventPool != "" {
		sb.WriteString("B5 本世界事件谱（优先从这里挑事件，落到本世界的具体样子）：\n" + w.B5EventPool + "\n")
	}
	sb.WriteString(w.ForTone())
	return sb.String()
}

// ForNovelist 小说化 Agent 视角：A + 伏笔 + 事件谱 + 叙事约束
func (w *Worldbook) ForNovelist() string {
	var sb strings.Builder
	sb.WriteString("【小说化用设定】\n")
	sb.WriteString("A1 世界观：\n" + w.A1Worldview + "\n")
	if w.A6GoalChain != "" {
		sb.WriteString("A6 主角目标链（主角的欲望驱动，写手必须让主角为目标行动，不能被动挨打）：\n" + w.A6GoalChain + "\n")
	}
	if w.A7PowerSys != "" {
		sb.WriteString("A7 能力成长体系（主角能力要有等级感和成长节奏，本章该不该升级、怎么解锁）：\n" + w.A7PowerSys + "\n")
	}
	if w.A8Villain != "" {
		sb.WriteString("A8 反派行动线（反派要主动出击，给主角持续压力，压迫→对抗→打脸）：\n" + w.A8Villain + "\n")
	}
	if w.B4Foreshadows != "" {
		sb.WriteString("B4 伏笔清单（小说层编排）：\n" + w.B4Foreshadows + "\n")
	}
	if w.B5EventPool != "" {
		sb.WriteString("B5 本世界事件谱（这个世界会发生的事——写手的弹药库：日常/冲突/奇遇都从这里取质感，闲笔和伏笔才有方向）：\n" + w.B5EventPool + "\n")
	}
	if w.CNarrative != "" {
		sb.WriteString("C 叙事约束：\n" + w.CNarrative + "\n")
	}
	sb.WriteString(w.ForTone())
	return sb.String()
}

// ForWorldBrief 精简世界背景（角色人设生成/生活质感推导用：世界观+时代+生活气息）
func (w *Worldbook) ForWorldBrief() string {
	var sb strings.Builder
	if w.A1Worldview != "" {
		sb.WriteString(w.A1Worldview)
	}
	if w.A2Physics != "" {
		sb.WriteString("\n规则：" + w.A2Physics)
	}
	return strings.TrimSpace(sb.String())
}

// WorldRule 物理规则（L1 真值）——GM 软规则裁决用
func (w *Worldbook) WorldRule() string {
	if w.A2Physics == "" {
		return "现实都市规则；存在超自然现象但普通人看不见；身体有极限；法律与现实社会规则有效。"
	}
	return w.A2Physics
}

// Safety 内容安全边界（所有 Agent 的行为约束）
func (w *Worldbook) Safety() string {
	return w.DSafety
}
