package main

import (
	"fmt"
	"testing"

	"worldsim/internal/worldbook"
)

func TestMatchAssetIndex(t *testing.T) {
	names := []string{"储物袋", "灵石", "木剑"}
	if got := matchAssetIndex(names, "灵石", 24); got != 2 {
		t.Fatalf("exact=%d", got)
	}
	if got := matchAssetIndex(names, "从废墟捡到的储物袋", 24); got != 1 {
		t.Fatalf("substring=%d", got)
	}
	// 未命中 → 稳定 hash：同输入必同输出，且在范围内
	a := matchAssetIndex(names, "神秘物品", 24)
	b := matchAssetIndex(names, "神秘物品", 24)
	if a != b || a < 1 || a > 24 {
		t.Fatalf("hash 不稳定/越界: %d %d", a, b)
	}
}

func TestHashIndexStable(t *testing.T) {
	for _, s := range []string{"青牛镇", "黑水岭", "妖狼巢穴", ""} {
		if hashIndex(s, 16) != hashIndex(s, 16) {
			t.Fatalf("hash 不稳定: %q", s)
		}
		if v := hashIndex(s, 16); v < 0 || v >= 16 {
			t.Fatalf("越界: %d", v)
		}
	}
}

func TestStdTileKeyword(t *testing.T) {
	cases := map[string]int{"青牛镇市集": 2, "黑水湖": 4, "迷雾森林": 6, "断魂崖": 7, "佣兵营地": 10, "灵石坊": 0, "unknown-place": 0}
	for name, want := range cases {
		if got := stdTileKeyword(name); got != want {
			t.Errorf("%s → %d want %d", name, got, want)
		}
	}
}

func TestSceneURLDeterministic(t *testing.T) {
	// 打包套件兜底路径（inst.wb 有标题即可触发 detectPixelTheme）
	inst := &worldInstance{}
	inst.wb = worldbook.Parse("# 世界书：测试\n## A1 世界观\n修仙世界灵石坊")
	url1 := sceneURLFor(inst, 3)
	url2 := sceneURLFor(inst, 3)
	if url1 != url2 || url1 == "" {
		t.Fatalf("scene 不确定或为空: %q %q", url1, url2)
	}
	if url1 != fmt.Sprintf("/pixel-art/%s/scene-%d.png", detectPixelTheme(inst), 3%3+1) {
		t.Fatalf("scene URL 语义错误: %q", url1)
	}
}
