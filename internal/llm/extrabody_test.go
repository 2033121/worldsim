package llm

import (
	"encoding/json"
	"testing"
)

// v1.10.1：api.json 的 extra_body 原样并入请求体顶层
func TestMergeExtraBody(t *testing.T) {
	base := []byte(`{"model":"m","messages":[{"role":"user","content":"hi"}],"max_tokens":8192}`)

	// 空 extra：原样返回（同一底层数组即可）
	if got := mergeExtraBody(base, nil); string(got) != string(base) {
		t.Fatalf("空 extra 应原样返回: %s", got)
	}

	// 新增键
	got := mergeExtraBody(base, map[string]any{"enable_thinking": false})
	var m map[string]any
	if err := json.Unmarshal(got, &m); err != nil {
		t.Fatalf("合并结果不是合法 JSON: %v (%s)", err, got)
	}
	if v, ok := m["enable_thinking"]; !ok || v != false {
		t.Fatalf("enable_thinking 未并入: %v", m["enable_thinking"])
	}
	if m["model"] != "m" || m["max_tokens"] != float64(8192) {
		t.Fatalf("原有字段被破坏: %v", m)
	}

	// 覆盖同名键（extra_body 优先，便于临时改参）
	got = mergeExtraBody(base, map[string]any{"max_tokens": float64(32000)})
	_ = json.Unmarshal(got, &m)
	if m["max_tokens"] != float64(32000) {
		t.Fatalf("extra_body 应覆盖同名键: %v", m["max_tokens"])
	}

	// 非法输入不 panic、不破坏主链路
	if got := mergeExtraBody([]byte("not json"), map[string]any{"a": 1}); string(got) != "not json" {
		t.Fatalf("非法 JSON 应原样返回: %s", got)
	}
}

// 提示文案必须给出可操作信息（用户排障靠它）
func TestReasoningBudgetHintActionable(t *testing.T) {
	h := reasoningBudgetHint()
	for _, want := range []string{"max_tokens", "enable_thinking", "extra_body"} {
		if !hintContains(h, want) {
			t.Fatalf("提示缺少 %q：%s", want, h)
		}
	}
}

func hintContains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
