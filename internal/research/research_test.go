package research

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"worldsim/internal/config"
	"worldsim/internal/prompt"
	"worldsim/internal/themes"
)

func newTestAgent(t *testing.T, mock func(system, user string) string) *Agent {
	t.Helper()
	ag := NewAgent(
		&config.APIConfig{BaseURL: "http://mock", Model: "mock", APIKey: "x"},
		nil, // 无联网搜索（测试降级路径）
		themes.Load(filepath.Join(t.TempDir(), "themes")),
		prompt.New(""),
		filepath.Join(t.TempDir(), "research"),
	)
	ag.MockLLM = mock
	return ag
}

func TestRunComparison_ParsesProposal(t *testing.T) {
	ag := newTestAgent(t, func(system, user string) string {
		// 断言：附件素材注入到用户输入
		if !strings.Contains(user, "【用户想法】") {
			t.Errorf("用户输入缺少想法标记")
		}
		return `{"candidates":[{"id":"c1","title":"都市修仙·职场","positioning":"修仙+职场","selling_point":"打工人修仙","risks":"同质化","audience":"都市白领","fit":"贴合想法","recommend":true,"reason":"卖点清晰"},{"id":"c2","title":"末世基建","positioning":"末世+基建","selling_point":"种田基建","risks":"节奏慢","audience":"末世爱好者","fit":"中等"}],"recommended_id":"c1"}`
	})
	p, err := ag.RunComparison(context.Background(), "我想写个修仙+职场的文", "附件：一份市场报告")
	if err != nil {
		t.Fatalf("RunComparison 失败: %v", err)
	}
	if len(p.Candidates) != 2 {
		t.Fatalf("候选数应为 2，得到 %d", len(p.Candidates))
	}
	if p.RecommendedID != "c1" {
		t.Fatalf("推荐 id 应为 c1，得到 %s", p.RecommendedID)
	}
	if rec := p.Recommended(); rec == nil || rec.ID != "c1" {
		t.Fatalf("Recommended() 应返回 c1")
	}
}

func TestRunComparison_TooFewCandidates(t *testing.T) {
	ag := newTestAgent(t, func(system, user string) string {
		return `{"candidates":[{"id":"c1","title":"单个","positioning":"p","selling_point":"s"}],"recommended_id":"c1"}`
	})
	if _, err := ag.RunComparison(context.Background(), "想法", ""); err == nil {
		t.Fatalf("候选不足应报错")
	}
}

func TestRunComparison_NoJSON(t *testing.T) {
	ag := newTestAgent(t, func(system, user string) string {
		return "对不起，我无法研究。"
	})
	if _, err := ag.RunComparison(context.Background(), "想法", ""); err == nil {
		t.Fatalf("无 JSON 应报错")
	}
}

func TestBuildThemeCard_ParsesCard(t *testing.T) {
	ag := newTestAgent(t, func(system, user string) string {
		return `{"id":"xianzhi","name":"现代仙侠职场","aliases":["都市修仙","职场修仙"],"dimensions":{"assets":["职位","人脉","功德"],"body":["灵力","颈椎"],"self":["境界","工龄"]},"locations":["写字楼","道观"],"time_scale":"按天","life_texture":["通勤","摸鱼"],"events_pool":["灵气复苏","裁员"],"taboos":["不得借用西幻的中世纪维度"]}`
	})
	card, err := ag.BuildThemeCard(context.Background(), Candidate{ID: "c1", Title: "现代仙侠职场"})
	if err != nil {
		t.Fatalf("BuildThemeCard 失败: %v", err)
	}
	if card.ID != "xianzhi" || len(card.Dim.Assets) == 0 {
		t.Fatalf("卡片解析错误: %+v", card)
	}
}

func TestBuildDirection_ReturnsMarkdown(t *testing.T) {
	ag := newTestAgent(t, func(system, user string) string {
		return "# 世界书方向：测试\n## 题材定位\n...\n"
	})
	d, err := ag.BuildDirection(context.Background(), Candidate{ID: "c1", Title: "测试", Positioning: "p"}, "")
	if err != nil {
		t.Fatalf("BuildDirection 失败: %v", err)
	}
	if !strings.HasPrefix(strings.TrimSpace(d), "# 世界书方向") {
		t.Fatalf("世界书方向应以标题开头: %q", d)
	}
}

func TestSaveProposalAndList(t *testing.T) {
	ag := newTestAgent(t, func(system, user string) string { return "{}" })
	rec := &ProposalRecord{Input: "想法", Proposal: &Proposal{
		Candidates: []Candidate{{ID: "c1", Title: "A"}, {ID: "c2", Title: "B"}},
	}}
	if err := ag.SaveProposal(rec); err != nil {
		t.Fatalf("SaveProposal 失败: %v", err)
	}
	if rec.ID == "" {
		t.Fatalf("SaveProposal 应生成 id")
	}
	list := ag.ListProposals()
	if len(list) != 1 {
		t.Fatalf("历史方案应为 1，得到 %d", len(list))
	}
	loaded, err := ag.LoadProposal(rec.ID)
	if err != nil {
		t.Fatalf("LoadProposal 失败: %v", err)
	}
	if len(loaded.Proposal.Candidates) != 2 {
		t.Fatalf("加载方案候选数应为 2")
	}
}

// 验证 theme_card 可被 JSON 序列化并回读（供 HTTP 层直接透传）。
func TestProposalRecordJSONRoundTrip(t *testing.T) {
	rec := ProposalRecord{
		ID: "rp_1", Input: "i", CreatedAt: "t",
		Proposal:  &Proposal{Candidates: []Candidate{{ID: "c1"}}},
		Direction: "dir",
		ThemeCard: &themes.Theme{ID: "t1", Name: "测试"},
	}
	b, _ := json.Marshal(rec)
	var back ProposalRecord
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatalf("JSON 回读失败: %v", err)
	}
	if back.ThemeCard == nil || back.ThemeCard.ID != "t1" {
		t.Fatalf("ThemeCard 回读失败")
	}
	if back.Proposal == nil || len(back.Proposal.Candidates) != 1 {
		t.Fatalf("Proposal 回读失败")
	}
}
