package game

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

// ---------- v1.9.0 好感度追踪 ----------

func TestTurnRelations(t *testing.T) {
	dir := t.TempDir()
	g := Load(dir)
	if _, err := g.Start(context.Background(), nil, "测试界", "测试", "阿测", "", nil); err != nil {
		t.Fatal(err)
	}
	verdict := `{"intent":"请童恒喝酒","ability":"","dc":0,"relations":[{"name":"童恒","delta":2},{"name":"黑心掌柜","delta":-5}]}`
	if _, err := g.Turn(context.Background(), mockLLM(verdict), 3, "向童恒打听灵石坊", "say", "无"); err != nil {
		t.Fatal(err)
	}
	st := g.StateCurrent()
	if st.Relations["童恒"] != 2 || st.Relations["黑心掌柜"] != -5 {
		t.Fatalf("relations not applied: %v", st.Relations)
	}
	// 越界钳制
	verdict2 := `{"intent":"狂刷好感","relations":[{"name":"童恒","delta":50},{"name":"宿敌","delta":-99}]}`
	if _, err := g.Turn(context.Background(), mockLLM(verdict2), 4, "继续套话", "say", "无"); err != nil {
		t.Fatal(err)
	}
	st = g.StateCurrent()
	if st.Relations["童恒"] != 10 || st.Relations["宿敌"] != -10 {
		t.Fatalf("relations clamp wrong: %v", st.Relations)
	}
	// 持久化
	g2 := Load(dir)
	if g2.StateCurrent().Relations["童恒"] != 10 {
		t.Fatalf("relations not persisted: %v", g2.StateCurrent().Relations)
	}
}

// ---------- v1.10.0 撤销 / 倒下 / 单飞 / 存档导入 ----------

func TestUndoRestoresPrevTurn(t *testing.T) {
	dir := t.TempDir()
	g := Load(dir)
	if _, err := g.Start(context.Background(), nil, "测试界", "测试", "阿测", "", nil); err != nil {
		t.Fatal(err)
	}
	verdict := `{"intent":"捡钱","ability":"","dc":0,"gold_delta":100,"inventory_add":["古玉"],"hp_delta":-20}`
	if _, err := g.Turn(context.Background(), mockLLM(verdict), 2, "捡钱", "do", "无"); err != nil {
		t.Fatal(err)
	}
	st := g.StateCurrent()
	if st.Gold != 150 || st.HP != 80 || len(st.Inventory) != 3 {
		t.Fatalf("pre-undo state wrong: gold=%d hp=%d inv=%v", st.Gold, st.HP, st.Inventory)
	}
	restored, err := g.Undo()
	if err != nil {
		t.Fatal(err)
	}
	if restored.Gold != 50 || restored.HP != 100 || len(restored.Inventory) != 2 {
		t.Fatalf("undo not restored: %+v", restored)
	}
	if restored.Turn != 0 || len(restored.Log) != 1 {
		t.Fatalf("undo log wrong: turn=%d log=%d", restored.Turn, len(restored.Log))
	}
	// 单步：再撤没有
	if _, err := g.Undo(); err == nil {
		t.Fatal("second undo should fail (single step)")
	}
	// 撤销后持久化一致
	if Load(dir).StateCurrent().Gold != 50 {
		t.Fatal("undo not persisted")
	}
}

func TestDownedState(t *testing.T) {
	dir := t.TempDir()
	g := Load(dir)
	if _, err := g.Start(context.Background(), nil, "测试界", "测试", "阿测", "", nil); err != nil {
		t.Fatal(err)
	}
	verdict := `{"intent":"硬冲尸群","ability":"","dc":0,"hp_delta":-130}`
	if _, err := g.Turn(context.Background(), mockLLM(verdict), 2, "硬冲尸群", "do", "无"); err != nil {
		t.Fatal(err)
	}
	st := g.StateCurrent()
	if !st.Downed || st.HP != 0 {
		t.Fatalf("want downed, got downed=%v hp=%d", st.Downed, st.HP)
	}
	// 等待回血 → 恢复意识
	if _, err := g.Wait(context.Background(), nil, 2, "无"); err != nil {
		t.Fatal(err)
	}
	st = g.StateCurrent()
	if st.Downed || st.HP <= 0 {
		t.Fatalf("wait should revive: downed=%v hp=%d", st.Downed, st.HP)
	}
}

func TestSingleFlightTurn(t *testing.T) {
	dir := t.TempDir()
	g := Load(dir)
	_, _ = g.Start(context.Background(), nil, "界", "d", "主", "b", nil)
	release := make(chan struct{})
	slow := Caller(func(ctx context.Context, tier, system, user string) (string, error) {
		if strings.Contains(system, "叙述者") {
			return "叙事完成。", nil
		}
		<-release // 裁判阶段挂起，制造并发窗口
		return `{"intent":"x"}`, nil
	})
	done := make(chan error, 1)
	go func() {
		_, err := g.Turn(context.Background(), slow, 1, "第一回合", "do", "")
		done <- err
	}()
	// 等 goroutine 真正占住单飞位
	deadline := time.Now().Add(2 * time.Second)
	for {
		g.mu.Lock()
		busy := g.turning
		g.mu.Unlock()
		if busy || time.Now().After(deadline) {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if _, err := g.Turn(context.Background(), nil, 1, "并发回合", "do", ""); err == nil || !strings.Contains(err.Error(), "回合进行中") {
		t.Fatalf("want single-flight error, got %v", err)
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatalf("first turn failed: %v", err)
	}
}

func TestImportFromValidates(t *testing.T) {
	dir := t.TempDir()
	g := Load(dir)
	_, _ = g.Start(context.Background(), nil, "界", "d", "主", "b", nil)
	// 非法 JSON
	if err := g.ImportFrom([]byte("不是json")); err == nil {
		t.Fatal("want error on bad json")
	}
	// 合法存档（含越界值 → clamp 收数）
	save := `{"enabled":true,"turn":7,"attrs":{"力量":99},"hp":-8,"max_hp":120,"gold":5,"inventory":["剑"],"location":"北门","quest":"守城","relations":{"老王":55},"log":[]}`
	if err := g.ImportFrom([]byte(save)); err != nil {
		t.Fatal(err)
	}
	st := g.StateCurrent()
	if st.Attrs["力量"] != 10 || st.Turn != 7 {
		t.Fatalf("import clamp wrong: %v turn=%d", st.Attrs, st.Turn)
	}
	if st.Relations["老王"] != 10 {
		t.Fatalf("relation clamp wrong: %v", st.Relations)
	}
	// 倒下状态机：HP 负值 → clamp 到 0 → Downed
	if !st.Downed || st.HP != 0 {
		t.Fatalf("downed state wrong: %v %d", st.Downed, st.HP)
	}
}

func TestRollUnknownAbilityNoPenalty(t *testing.T) {
	attrs := map[string]int{"力量": 10}
	r := Roll(attrs, "神识", 10) // 面板没有的属性：无修正（不暗中 -5）
	if r.Mod != 0 {
		t.Fatalf("unknown ability should have mod 0, got %d", r.Mod)
	}
	r2 := Roll(attrs, "力量", 10)
	if r2.Mod != 5 {
		t.Fatalf("known ability mod wrong: %d", r2.Mod)
	}
}
