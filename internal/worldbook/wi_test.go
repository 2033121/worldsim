package worldbook

import (
	"strings"
	"testing"
)

func TestParseWIEntries(t *testing.T) {
	body := `说明行（会被忽略）
- 灵石,灵石坊 => 灵石坊是青牛镇唯一的灵石交易点，掌柜童恒深不可测。
- 妖狼|狼嚎 -> 妖狼群在黑水岭出没，狼嚎近在三里之内。`
	entries := ParseWIEntries(body)
	if len(entries) != 2 {
		t.Fatalf("entries=%d want 2", len(entries))
	}
	if entries[0].Keys[0] != "灵石" || entries[0].Keys[1] != "灵石坊" {
		t.Fatalf("keys=%v", entries[0].Keys)
	}
	if !strings.Contains(entries[1].Content, "黑水岭") {
		t.Fatalf("content=%q", entries[1].Content)
	}
	if ParseWIEntries("没有分隔符的行") != nil {
		t.Fatal("无分隔符行应被忽略")
	}
}

func TestActivateWI(t *testing.T) {
	entries := []WIEntry{
		{Keys: []string{"灵石坊", "童恒"}, Content: "灵石坊掌柜童恒是宗门外门执事的远亲，交易价虚高两成。"},
		{Keys: []string{"妖狼", "狼嚎"}, Content: "妖狼群栖黑水岭，狼嚎近三里必袭营。"},
		{Keys: []string{"测灵根"}, Content: "测灵根在青牛镇广场，每旬一次，灵根决定入门资格。"},
	}
	// 1) 中文子串命中（whole-word 会毁中文——用子串）
	sticky := map[string]int{}
	hits := ActivateWI(entries, "你说要去灵石坊看看童恒的成色", sticky, 1200, 3)
	if len(hits) != 1 || !strings.Contains(hits[0].Content, "童恒") {
		t.Fatalf("hits=%v", hits)
	}
	// 2) sticky：下一回合没有命中也应保持
	hits = ActivateWI(entries, "今天天气不错", sticky, 1200, 3)
	if len(hits) != 1 {
		t.Fatalf("sticky 应保持命中: %v", hits)
	}
	// 3) sticky 到期衰减（3 回合后消失）
	for i := 0; i < 3; i++ {
		hits = ActivateWI(entries, "风平浪静", sticky, 1200, 3)
	}
	if len(hits) != 0 {
		t.Fatalf("sticky 应到期: %v", hits)
	}
	// 4) 预算约束：只装得下小条目
	big := []WIEntry{
		{Keys: []string{"a"}, Content: strings.Repeat("长", 1500)},
		{Keys: []string{"b"}, Content: "短条目"},
	}
	hits = ActivateWI(big, "ab", sticky, 1200, 3)
	if len(hits) != 1 || hits[0].Content != "短条目" {
		t.Fatalf("budget 过滤失败: %v", hits)
	}
}
