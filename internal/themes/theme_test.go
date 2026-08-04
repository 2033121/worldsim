package themes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadBuiltin(t *testing.T) {
	s := Load("")
	if len(s.List()) == 0 {
		t.Fatal("内置题材卡片不应为空")
	}
	if s.Find("xianxia") == nil {
		t.Fatal("应包含 xianxia 卡片")
	}
}

func TestDetect(t *testing.T) {
	s := Load("")
	if s.Detect("一个修仙世界，山村少年") == nil {
		t.Fatal("应识别修仙")
	}
	if s.Detect("都市异能，现代都市") == nil {
		t.Fatal("应识别都市")
	}
	if s.Detect("完全无关的杂谈") != nil {
		t.Fatal("无匹配应返回 nil")
	}
}

func TestInjectNoExample(t *testing.T) {
	s := Load("")
	text := s.Inject("xianxia")
	if text == "" {
		t.Fatal("Inject 不应为空")
	}
	// 维度提示词不应含具体叙事示例（如"山村砍柴少年"这种人物示例）
	if strings.Contains(text, "山村砍柴少年") {
		t.Fatal("卡片维度提示不应含叙事示例")
	}
}

func TestSaveOverride(t *testing.T) {
	dir := t.TempDir()
	s := Load(dir)
	card := Theme{ID: "custom", Name: "自定义题材",
		Dim: Dimensions{Assets: []string{"甲", "乙"}, Body: []string{"丙"}},
	}
	if err := s.Save(card); err != nil {
		t.Fatal(err)
	}
	if s.Find("custom") == nil {
		t.Fatal("保存后应可查")
	}
	// 覆盖
	card2 := Theme{ID: "custom", Name: "自定义题材2"}
	if err := s.Save(card2); err != nil {
		t.Fatal(err)
	}
	if got := s.Find("custom"); got == nil || got.Name != "自定义题材2" {
		t.Fatal("同 id 应覆盖")
	}
	// 落盘文件存在
	if _, err := os.Stat(filepath.Join(dir, "custom.json")); err != nil {
		t.Fatal("卡片文件应写入:", err)
	}
	// 重新加载外部目录应读到
	s2 := Load(dir)
	if s2.Find("custom") == nil {
		t.Fatal("重新加载应读到外部卡片")
	}
}

func TestSaveEmptyID(t *testing.T) {
	s := Load(t.TempDir())
	if err := s.Save(Theme{Name: "无id"}); err == nil {
		t.Fatal("空 id 应报错")
	}
}