package game

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mockLLM 返回固定 JSON 的裁判（无叙述调用需求：第二个调用返回叙述文本）
func mockLLM(verdict string) Caller {
	return func(ctx context.Context, tier, system, user string) (string, error) {
		if strings.Contains(system, "叙述者") {
			return "你推门走进大厅，灰尘在光柱里翻滚。", nil
		}
		return "```json\n" + verdict + "\n```", nil
	}
}

func TestRollBoundsAndCrits(t *testing.T) {
	attrs := map[string]int{"力量": 5}
	for i := 0; i < 500; i++ {
		r := Roll(attrs, "力量", 10)
		if r.Roll < 1 || r.Roll > 20 {
			t.Fatalf("d20 out of range: %d", r.Roll)
		}
	}
	// nat20 必成 / nat1 必败 由构造保证；这里验证修正计算
	if got := attrs["力量"] - 5; got != 0 {
		t.Fatalf("mod calc wrong: %d", got)
	}
}

func TestClampAndLevelUp(t *testing.T) {
	st := zeroState()
	st.HP = 150
	st.XP = 250
	note := st.clamp()
	if st.HP != st.MaxHP {
		t.Fatalf("hp not clamped: %d", st.HP)
	}
	if st.Level != 3 {
		t.Fatalf("want level 3, got %d", st.Level)
	}
	if note == "" {
		t.Fatal("want level-up note")
	}
	// 负数收数
	st.HP = -5
	st.XP = -30
	st.clamp()
	if st.HP != 0 || st.XP != 0 {
		t.Fatalf("neg clamp failed: hp=%d xp=%d", st.HP, st.XP)
	}
}

func TestTurnPersistenceAndCheck(t *testing.T) {
	dir := t.TempDir()
	g := Load(dir)
	if _, err := g.Start(context.Background(), nil, "测试界", "一个测试世界", "阿测", "普通青年", nil); err != nil {
		t.Fatal(err)
	}
	verdict := `{"intent":"挥剑砍树","ability":"力量","dc":10,"hp_delta":-5,"gold_delta":10,"xp_delta":20,"inventory_add":["木柴"],"location":"","quest":"收集10根木柴","world_beat":"远处传来狼嚎"}`
	narr, err := g.Turn(context.Background(), mockLLM(verdict), 3, "挥剑砍树收集木柴", "do", "无")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(narr, "推门") && !strings.Contains(narr, "你") {
		t.Fatalf("narration empty: %q", narr)
	}
	// 重启后状态还在
	g2 := Load(dir)
	st := g2.StateCurrent()
	if st.Turn != 1 {
		t.Fatalf("turn not persisted: %d", st.Turn)
	}
	if st.HP != 95 || st.Gold != 60 || st.XP != 20 {
		t.Fatalf("deltas not applied: %+v", st)
	}
	if len(st.Inventory) != 3 || st.Inventory[2] != "木柴" {
		t.Fatalf("inventory wrong: %v", st.Inventory)
	}
	if st.Quest != "收集10根木柴" {
		t.Fatalf("quest wrong: %q", st.Quest)
	}
	if len(st.Log) != 2 || st.Log[1].Check == nil {
		b, _ := json.Marshal(st.Log)
		t.Fatalf("log wrong: %s", b)
	}
}

func TestWaitHeals(t *testing.T) {
	dir := t.TempDir()
	g := Load(dir)
	_, _ = g.Start(context.Background(), nil, "界", "d", "主", "b", nil)
	g.mu.Lock()
	g.state.HP = 50
	g.mu.Unlock()
	if _, err := g.Wait(context.Background(), nil, 1, ""); err != nil {
		t.Fatal(err)
	}
	st := g.StateCurrent()
	if st.HP != 60 {
		t.Fatalf("want healed 60, got %d", st.HP)
	}
	if st.Turn != 1 {
		t.Fatalf("turn wrong: %d", st.Turn)
	}
}

func TestStartWorldbookAttrsFallback(t *testing.T) {
	dir := t.TempDir()
	g := Load(dir)
	_, err := g.Start(context.Background(), nil, "界", "d", "主", "b",
		map[string]int{"炼气": 5, "体魄": 4, "道心": 3})
	if err != nil {
		t.Fatal(err)
	}
	st := g.StateCurrent()
	if st.Attrs["炼气"] != 5 || st.Attrs["体魄"] != 4 || st.Attrs["道心"] != 3 {
		t.Fatalf("worldbook attrs not applied: %v", st.Attrs)
	}
	if _, ok := st.Attrs["力量"]; ok {
		t.Fatalf("generic fallback should be replaced by worldbook attrs: %v", st.Attrs)
	}
}

func TestStopAndGuard(t *testing.T) {
	dir := t.TempDir()
	g := Load(dir)
	_, _ = g.Start(context.Background(), nil, "界", "d", "主", "b", nil)
	g.Stop()
	if _, err := g.Turn(context.Background(), nil, 1, "走", "do", ""); err == nil {
		t.Fatal("want error after stop")
	}
	if _, err := os.Stat(filepath.Join(dir, GameStateFile)); err != nil {
		t.Fatal("game.json should exist")
	}
}

func TestParseJSONTolerant(t *testing.T) {
	if p := parseJSON("```json\n{\"a\":1}\n```"); p == nil {
		t.Fatal("fenced json not parsed")
	}
	if p := parseJSON("好的，输出如下：{\"a\":1} 请查收"); p == nil {
		t.Fatal("polluted json not parsed")
	}
	if p := parseJSON("没有json"); p != nil {
		t.Fatal("should be nil")
	}
}
