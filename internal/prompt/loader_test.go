package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 内置提示词：用于测试随包分发（testdata 同名占位）。
// 正式提示词放 builtin/ 由 go:embed 打包；这里仅验证插值逻辑与外部覆盖。
func TestInterpolate(t *testing.T) {
	raw := "你是{{ role }}。主题：{{ theme }}。{{ missing }}结束"
	got := interpolate(raw, map[string]string{"role": "导演", "theme": "修仙"})
	if strings.Contains(got, "{{ role }}") || strings.Contains(got, "{{ theme }}") {
		t.Fatalf("应替换已提供占位符: %s", got)
	}
	if !strings.Contains(got, "修仙") {
		t.Fatalf("应注入值: %s", got)
	}
	// 未提供的 key 应替换为空串而非保留
	if strings.Contains(got, "{{ missing }}") {
		t.Fatalf("未提供 key 应替换为空串: %s", got)
	}
}

func TestExternalOverride(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "prompts"), 0755)
	ext := filepath.Join(dir, "prompts", "demo.md")
	if err := os.WriteFile(ext, []byte("外部版 {{x}}"), 0644); err != nil {
		t.Fatal(err)
	}
	l := New(filepath.Join(dir, "prompts"))
	got, err := l.Render("demo", map[string]string{"x": "OK"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "外部版") {
		t.Fatalf("应优先读外部覆盖: %s", got)
	}
}

func TestMissingPrompt(t *testing.T) {
	l := New(t.TempDir())
	if _, err := l.Render("no_such_prompt_xyz", nil); err == nil {
		t.Fatal("应返回错误")
	}
}
