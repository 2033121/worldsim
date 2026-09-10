package worldbook

import (
	"context"
	"fmt"
	"strings"

	"worldsim/internal/config"
	"worldsim/internal/llm"
)

// ---------- 世界书生成 Agent（选主题包 → LLM 生成完整世界书） ----------
// 用法：用户选一个主题包（经典修仙/都市异能/克苏鲁…）+ 一句话设定，
// LLM 按主题包的要素池 + 通用模板结构，生成可直接用的世界书 md。
// 模板只给维度，内容按用户设定生成——不会千人一面。

// GenWorldbookLLM 生成世界书（1次 normal 调用），返回世界书 md 全文
func GenWorldbookLLM(ctx context.Context, apiCfg *config.APIConfig, themeContent, userDesc string) (string, error) {
	if apiCfg == nil {
		return "", fmt.Errorf("LLM 配置不可用")
	}
	ctx = llm.WithSpan(ctx, "世界书生成")
	system := `你是世界书架构师。根据"主题包要素池"和用户的设定，生成一份完整的世界书（Markdown）。
世界书结构（严格按此章节，每个字段都要填，内容必须贴合用户设定，不得套别的世界的现成设定）：
# 世界书：{世界名}
## A1 世界观（核心图景：时代/场景/特殊设定/普通人怎么看待）
## A2 规则（3~6条铁律：力量怎么获得/资源怎么流通/代价/禁忌）
## A3 势力（3~4个：官方/暗面/商业/散人，各带立场与手段）
## A4 地理（主角生活圈：出生地/主城/危险区/特殊地点）
## A5 势力速览（普通人眼里的世界）
## A6 主角目标链（核心欲望+阶段目标，主角必须有目标）
## A7 能力成长体系（升级路径+实感）
## A8 反派行动线（反派暗处做什么：压迫→对抗→打脸）
## B1 秘密（3条：主角身世/世界真相/NPC隐藏身份）
## B2 组织（关键组织内部结构）
## B3 弧线（主线4~6步：起点→第一次蜕变）
## B4 伏笔（清单，标注长度：长线/中线/短线；长线3~5个够）
## B5 事件谱（从主题包事件池里选20~30种，落到本世界的具体地点和样子：测灵根在青牛镇广场/裁员在周五下班前HR办公室）
## C1 文风（题材基调+视角+语言质感，参考主题包的文风基调）
## E1-E3 深层设定（世界真相，格式：## EX 标题【事件触发：关键词|关键词】——用事件触发不用天数）
## D 安全（内容安全边界）
要求：
1. 主题包的要素池是"该想哪些维度"，不是照抄——要按用户设定重新组织成具体内容。
2. 时间感必须贴合主题包的时间尺度（修仙按年/末世按天/星际按标准时），禁止用"Day20"这类天数触发。
3. 生活质感要具体（杂役的糙米饭/便利店的冷柜嗡鸣/营地的限时供水），让世界像真有人在过。
4. 只输出世界书 Markdown，不要其他文字。`
	user := fmt.Sprintf("主题包要素池：\n%s\n\n用户设定：\n%s\n\n请生成世界书。", themeContent, userDesc)
	// 用流式优先（CallAPITier → CallAPIMessages 流式优先）：同步模式下中转站网关
	// 对长时间生成会返回 504/EOF（完整世界书常需 5+ 分钟），流式持续收 chunk 可避免。
	return llm.CallAPITier(ctx, apiCfg, "normal", system, user)
}

// TrimWorldbook 清洗 LLM 输出：去掉 markdown 代码块包裹（若 LLM 画蛇添足）
func TrimWorldbook(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```markdown")
	raw = strings.TrimPrefix(raw, "```md")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	return strings.TrimSpace(raw)
}
