package worldbook

import "testing"

func TestParseGameAttrs(t *testing.T) {
	wb := Parse(`# 测试界 世界书

## A2 规则
物理规则。

## A3 势力
无势力。

## 游玩属性（Play Mode 数据层默认）
- 炼气: 5
- 体魄：4
"心志": "6"（不该解析成属性）

## D 安全
无。
`)
	if wb.GameAttrsRaw == "" {
		t.Fatal("游玩属性段未收集")
	}
	attrs := wb.GameAttrs()
	if attrs["炼气"] != 5 || attrs["体魄"] != 4 {
		t.Fatalf("attrs parse wrong: %v", attrs)
	}
	if _, junk := attrs["6"]; junk || len(attrs) != 2 {
		t.Fatalf("junk parsed or wrong count: %v", attrs)
	}
	// A2/A3 不受影响
	if wb.A2Physics == "" || wb.A3Society == "" {
		t.Fatal("原有段落被游玩属性段吞掉")
	}
}

func TestGameAttrsNilAndEmpty(t *testing.T) {
	var wb *Worldbook
	if wb.GameAttrs() != nil {
		t.Fatal("nil receiver should return nil")
	}
	if Parse("# 空\n").GameAttrs() != nil {
		t.Fatal("missing section should return nil")
	}
}

func TestGameAttrsClamped(t *testing.T) {
	wb := Parse("# 压界\n\n## 游玩属性\n- 力量: 99\n- 弱: 0\n- 另: -3\n")
	a := wb.GameAttrs()
	if a["力量"] != 10 || a["弱"] != 1 || a["另"] != 1 {
		t.Fatalf("clamp wrong: %v", a)
	}
}
