package sim

import (
	"context"
	"testing"

	"worldsim/internal/engine"
)

// 验证异常1修复：每个带 options 的事件独立 AI 代决，ai_choice 为选项 ID 而非同一句全局文本
func TestCaptureDecisionsPerEvent(t *testing.T) {
	se := engine.NewStateEngine(engine.Rules{}, "")
	st := se.State()
	st.Entities["周明远"] = engine.Entity{Location: "临江市", Job: "跑腿店员"}
	st.Entities["玛丽·坎贝尔"] = engine.Entity{Location: "临江市", Job: "占卜师"}

	s := NewSimulator(se, t.TempDir())
	s.heroName = "周明远"
	s.llm = NewMockLLM()

	// 两个不同的事件，各带不同选项
	s.events = []EventCard{
		{Type: "wonder", Title: "怪事A", Frame: "巷口有人低声念叨你的名字", Options: []string{"上前搭话", "假装没听见走开"}},
		{Type: "conflict", Title: "冲突B", Frame: "有人拦住你去路要钱", Options: []string{"给钱息事", "硬闯过去"}},
	}

	s.day = 4
	heroAction := "主角今天决定先打探消息再行动"
	s.captureDecisions(context.Background(), heroAction)

	all := s.decisions.All()
	if len(all) != 2 {
		t.Fatalf("期望 2 个决策，实际 %d", len(all))
	}
	for _, d := range all {
		if d.AIChoice == "" {
			t.Errorf("决策 [%s] AIChoice 为空", d.Title)
		}
		// ai_choice 必须是选项 ID（A/B…），前端依赖 ai_choice === o.id 高亮；
		// 原 bug 是把全局行动长文本塞给所有岔口，导致与任何选项 ID 都不匹配
		if d.AIChoice != "A" && d.AIChoice != "B" {
			t.Errorf("决策 [%s] AIChoice=%q 不是合法选项 ID", d.Title, d.AIChoice)
		}
		if d.AIChoice == heroAction {
			t.Errorf("决策 [%s] AIChoice 仍被塞入了全局行动文本", d.Title)
		}
	}
}

// 验证异常2修复：新注册角色 identity 不再为空
func TestRegisterCharacterSetsIdentity(t *testing.T) {
	se := engine.NewStateEngine(engine.Rules{}, "")
	st := se.State()
	st.Entities["周明远"] = engine.Entity{Location: "临江市", Job: "跑腿店员"}

	s := NewSimulator(se, t.TempDir())
	s.heroName = "周明远"

	nc := NewCharacter{Name: "玛丽·坎贝尔", Identity: "占卜师", Persona: "神秘寡言", Location: "教堂", RoleHint: "npc"}
	changes := s.RegisterCharacter(nc)
	if len(changes) == 0 {
		t.Fatal("RegisterCharacter 未产生变更")
	}
	prop := &engine.Proposal{
		CommandID: "test-reg-1", ActorID: "world_agent", BaseRevision: 0, Type: "state_change",
		Changes: changes,
	}
	if err := se.Submit(context.Background(), prop); err != nil {
		t.Fatalf("Submit 失败: %v", err)
	}
	ent := st.Entities["玛丽·坎贝尔"]
	if ent.Job != "占卜师" {
		t.Errorf("job=%q，期望 占卜师", ent.Job)
	}
	if id, _ := ent.Extra["identity"].(string); id != "占卜师" {
		t.Errorf("extra.identity=%q，期望 占卜师（曾为空）", id)
	}
}
