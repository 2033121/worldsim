// Package research 提供"题材研究 / 主题规划"智能体。
//
// 用途：建世界前的立项研究。用户往往只有模糊想法（一句"想做个修仙+职场"），
// 本智能体用联网搜索（web_search）研究当下热门题材/卖点/竞品/读者偏好，
// 读取用户上传的附件（市场报告/竞品/设定稿）作为权威背景，产出结构化研究结果：
//   - 题材对比方案（2~3 个候选题材 + 推荐）
//   - 世界书方向文档（据此建世界的蓝本）
//   - 题材知识卡片（可选沉淀到 wsdata/themes/{id}.json，供同类题材复用）
//
// 设计遵循"零叙事示例"：系统提示词只给维度与 JSON Schema 约束，
// 不给具体题材对照示例，避免题材内容雷同（去硬编码）。
package research

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"worldsim/internal/config"
	"worldsim/internal/llm"
	"worldsim/internal/prompt"
	"worldsim/internal/sim"
	"worldsim/internal/themes"
)

// Candidate 一个候选题材/主题的对比项。
type Candidate struct {
	ID          string `json:"id"`
	Title       string `json:"title"`        // 题材/主题名（具体有辨识度）
	Positioning string `json:"positioning"`  // 题材定位（一句话）
	SellingPoint string `json:"selling_point"` // 核心卖点
	Risks       string `json:"risks"`        // 风险
	Audience    string `json:"audience"`     // 目标读者
	Fit         string `json:"fit"`          // 与用户想法/附件的适配性
	Recommend   bool   `json:"recommend"`    // 是否推荐
	Reason      string `json:"reason"`       // 推荐理由（仅推荐项填）
}

// Proposal 题材对比方案（研究第一步产出）。
type Proposal struct {
	Candidates    []Candidate `json:"candidates"`
	RecommendedID string      `json:"recommended_id"`
}

// Recommended 返回推荐候选（无则返回 nil）。
func (p *Proposal) Recommended() *Candidate {
	for i := range p.Candidates {
		if p.Candidates[i].ID == p.RecommendedID || p.Candidates[i].Recommend {
			return &p.Candidates[i]
		}
	}
	return nil
}

// ProposalRecord 一份已保存的研究方案（含时间戳，供历史列表）。
type ProposalRecord struct {
	ID        string     `json:"id"`
	Input     string     `json:"input"`
	CreatedAt string     `json:"created_at"`
	Proposal  *Proposal  `json:"proposal"`
	Direction string     `json:"direction,omitempty"` // 世界书方向（选定候选后生成）
	ThemeCard *themes.Theme `json:"theme_card,omitempty"`
}

// Agent 题材研究智能体：持有 LLM 配置、联网搜索工具、题材卡片存储、提示词加载器。
type Agent struct {
	apiCfg  *config.APIConfig
	tools   *llm.ToolRegistry // 联网搜索工具（web_search）；可为 nil（未启用）
	themes  *themes.Store
	prompts *prompt.Loader
	baseDir string            // 研究方案存档目录（wsdata/research）
	MockLLM func(system, user string) string // 测试钩子：非 nil 时走本地模拟（无 API 调用）
}

// NewAgent 构造研究智能体。
// tools 为已注册 web_search 的 ToolRegistry（可为 nil/空，则只靠附件+背景知识研究）。
// themeStore 用于 Save 沉淀题材卡片；promptLoader 用于外置提示词。
func NewAgent(apiCfg *config.APIConfig, tools *llm.ToolRegistry, themeStore *themes.Store, promptLoader *prompt.Loader, baseDir string) *Agent {
	return &Agent{apiCfg: apiCfg, tools: tools, themes: themeStore, prompts: promptLoader, baseDir: baseDir}
}

// client 构造一个复用 sim.LLMClient 的调用客户端（处理 WorldRefs 注入 + 工具循环 + nil 降级 + 超时）。
func (a *Agent) client(refs string) *sim.LLMClient {
	return &sim.LLMClient{Cfg: a.apiCfg, Tools: a.tools, WorldRefs: refs, Mock: a.MockLLM}
}

// SearchEnabled 报告联网搜索是否可用（供前端 UI 提示）。
func (a *Agent) SearchEnabled() bool {
	return a.tools != nil && len(a.tools.Schemas()) > 0
}

// RunComparison 运行热门题材研究，产出对比方案。
// userInput 为用户想法；refs 为附件聚合文本（世界/题材素材，可为空）。
func (a *Agent) RunComparison(ctx context.Context, userInput, refs string) (*Proposal, error) {
	if a.apiCfg == nil || a.apiCfg.Model == "" {
		return nil, fmt.Errorf("LLM 配置不可用，无法研究")
	}
	sys, err := a.prompts.Render("research_compare", nil)
	if err != nil {
		return nil, err
	}
	user := buildUserInput(userInput, refs)
	raw, err := a.client(refs).CompleteTier(ctx, "normal", sys, user)
	if err != nil {
		return nil, err
	}
	jsonStr := llm.ExtractJSON(raw)
	if jsonStr == "" {
		return nil, fmt.Errorf("研究智能体输出无 JSON: %s", truncate(raw, 160))
	}
	var p Proposal
	if err := json.Unmarshal([]byte(jsonStr), &p); err != nil {
		return nil, fmt.Errorf("研究方案 JSON 解析失败: %v", err)
	}
	if len(p.Candidates) < 2 {
		return nil, fmt.Errorf("候选题材不足 2 个（得到 %d 个），请重试", len(p.Candidates))
	}
	if p.RecommendedID == "" {
		if rec := p.Recommended(); rec != nil {
			p.RecommendedID = rec.ID
		}
	}
	return &p, nil
}

// BuildDirection 基于选定候选生成"世界书方向" markdown 文档（建世界蓝本）。
func (a *Agent) BuildDirection(ctx context.Context, c Candidate, refs string) (string, error) {
	if a.apiCfg == nil || a.apiCfg.Model == "" {
		return "", fmt.Errorf("LLM 配置不可用，无法生成世界书方向")
	}
	// 若该题材已有卡片，注入维度提示，避免方向跑偏
	themeCard := ""
	if a.themes != nil {
		if t := a.themes.Detect(c.Title+ " " + c.ID); t != nil {
			themeCard = t.InjectText()
		}
	}
	sys, err := a.prompts.Render("research_direction", map[string]string{"themeCard": themeCard})
	if err != nil {
		return "", err
	}
	user := buildUserInput(fmt.Sprintf("选定候选题材：%s（%s）。\n卖点：%s\n请据此生成世界书方向。", c.Title, c.Positioning, c.SellingPoint), refs)
	raw, err := a.client(refs).CompleteTier(ctx, "normal", sys, user)
	if err != nil {
		return "", err
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("世界书方向生成失败（空输出）")
	}
	return raw, nil
}

// BuildThemeCard 基于候选提炼题材知识卡片（维度参考，无叙事示例）。
func (a *Agent) BuildThemeCard(ctx context.Context, c Candidate) (*themes.Theme, error) {
	if a.apiCfg == nil || a.apiCfg.Model == "" {
		return nil, fmt.Errorf("LLM 配置不可用，无法提炼题材卡片")
	}
	sys, err := a.prompts.Render("research_card", nil)
	if err != nil {
		return nil, err
	}
	user := fmt.Sprintf("请把以下候选题材提炼成题材知识卡片：\n题材名：%s\n定位：%s\n卖点：%s\n目标读者：%s",
		c.Title, c.Positioning, c.SellingPoint, c.Audience)
	raw, err := a.client("").CompleteTier(ctx, "normal", sys, user)
	if err != nil {
		return nil, err
	}
	jsonStr := llm.ExtractJSON(raw)
	if jsonStr == "" {
		return nil, fmt.Errorf("题材卡片智能体输出无 JSON: %s", truncate(raw, 160))
	}
	var card themes.Theme
	if err := json.Unmarshal([]byte(jsonStr), &card); err != nil {
		return nil, fmt.Errorf("题材卡片 JSON 解析失败: %v", err)
	}
	if card.ID == "" {
		return nil, fmt.Errorf("题材卡片缺少 id")
	}
	return &card, nil
}

// SaveCard 把题材卡片落盘（写入 wsdata/themes/{id}.json）。
func (a *Agent) SaveCard(card *themes.Theme) error {
	if a.themes == nil {
		return fmt.Errorf("题材卡片存储未配置")
	}
	return a.themes.Save(*card)
}

// SaveProposal 把研究方案存档到 wsdata/research/{id}.json（供历史/建世界引用）。
func (a *Agent) SaveProposal(rec *ProposalRecord) error {
	if a.baseDir == "" {
		return nil // 未配置目录则跳过存档
	}
	if err := os.MkdirAll(a.baseDir, 0755); err != nil {
		return err
	}
	if rec.ID == "" {
		rec.ID = fmt.Sprintf("rp_%d", time.Now().Unix())
	}
	if rec.CreatedAt == "" {
		rec.CreatedAt = time.Now().Format("2006-01-02 15:04:05")
	}
	b, _ := json.MarshalIndent(rec, "", "  ")
	return os.WriteFile(filepath.Join(a.baseDir, rec.ID+".json"), b, 0644)
}

// LoadProposal 按 id 读取已存档研究方案。
func (a *Agent) LoadProposal(id string) (*ProposalRecord, error) {
	if a.baseDir == "" {
		return nil, fmt.Errorf("未配置研究存档目录")
	}
	b, err := os.ReadFile(filepath.Join(a.baseDir, id+".json"))
	if err != nil {
		return nil, err
	}
	var rec ProposalRecord
	if err := json.Unmarshal(b, &rec); err != nil {
		return nil, err
	}
	return &rec, nil
}

// ListProposals 列出历史研究方案（最新在前）。
func (a *Agent) ListProposals() []ProposalRecord {
	if a.baseDir == "" {
		return []ProposalRecord{}
	}
	entries, err := os.ReadDir(a.baseDir)
	if err != nil {
		return []ProposalRecord{}
	}
	out := []ProposalRecord{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(a.baseDir, e.Name()))
		if err != nil {
			continue
		}
		var rec ProposalRecord
		if json.Unmarshal(b, &rec) != nil {
			continue
		}
		out = append(out, rec)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt > out[j].CreatedAt })
	return out
}

// buildUserInput 组装用户研究输入：附件素材 + 用户想法。
func buildUserInput(userInput, refs string) string {
	var b strings.Builder
	if strings.TrimSpace(refs) != "" {
		b.WriteString("【题材素材 / 附件（用户上传，视为权威背景，研究结论必须与其一致，并可从其提炼维度要素）】\n")
		b.WriteString(strings.TrimSpace(refs))
		b.WriteString("\n\n")
	}
	b.WriteString("【用户想法】\n")
	b.WriteString(strings.TrimSpace(userInput))
	return b.String()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}