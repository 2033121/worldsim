package story

import (
	"strings"
	"testing"

	"worldsim/internal/config"
	"worldsim/internal/i18n"
)

// totalBuiltinSkillsThreshold 内置技能卡数量下限（当前约 105+ 张，宽松阈值防新卡误删导致回归）。
const totalBuiltinSkillsThreshold = 100

// TestLoadBuiltinSkillsAllValid 验证内置技能全部可解析，且每张卡 ID/Name/Category 非空、无解析失败。
func TestLoadBuiltinSkillsAllValid(t *testing.T) {
	skills := LoadBuiltinSkills()
	if len(skills) <= totalBuiltinSkillsThreshold {
		t.Fatalf("内置技能加载数量过少：期望 > %d，实际 %d（可能存在解析失败被跳过）", totalBuiltinSkillsThreshold, len(skills))
	}

	seen := map[string]bool{}
	for _, s := range skills {
		if s.ID == "" {
			t.Fatalf("存在缺失 id 的技能卡：name=%q", s.Name)
		}
		if s.Name == "" {
			t.Fatalf("技能 %s 缺失 name", s.ID)
		}
		if s.Category == "" {
			t.Fatalf("技能 %s(%s) 缺失 category", s.ID, s.Name)
		}
		if s.Content == "" {
			t.Fatalf("技能 %s(%s) 缺失正文内容", s.ID, s.Name)
		}
		if seen[s.ID] {
			t.Fatalf("技能 id 重复：%s", s.ID)
		}
		seen[s.ID] = true
	}
}

// TestFilterSkillsByLang 验证按项目语言过滤正确：空 lang 始终返回，lang 不匹配则剔除。
func TestFilterSkillsByLang(t *testing.T) {
	skills := []Skill{
		{ID: "zh-only", Lang: i18n.LangZH},
		{ID: "en-only", Lang: i18n.LangEN},
		{ID: "agnostic", Lang: ""},
	}

	zh := FilterSkillsByLang(skills, i18n.LangZH)
	if len(zh) != 2 {
		t.Fatalf("zh 过滤期望 2 条（zh-only + agnostic），实际 %d", len(zh))
	}
	for _, want := range []string{"zh-only", "agnostic"} {
		if !containsID(zh, want) {
			t.Fatalf("zh 过滤结果缺少 %s", want)
		}
	}

	en := FilterSkillsByLang(skills, i18n.LangEN)
	if len(en) != 2 {
		t.Fatalf("en 过滤期望 2 条（en-only + agnostic），实际 %d", len(en))
	}
	for _, want := range []string{"en-only", "agnostic"} {
		if !containsID(en, want) {
			t.Fatalf("en 过滤结果缺少 %s", want)
		}
	}
}

// TestGetEnabledSkillsByCategoryPolish 验证润色环节按 category=polish 过滤能命中启用卡。
// kc-flow-008 修改润色SOP 应在润色注入逻辑中生效（category 已调整为 polish）。
func TestGetEnabledSkillsByCategoryPolish(t *testing.T) {
	sc := &config.SkillConfig{
		EnabledSkills: map[string]bool{
			"kc-flow-008": true,
			"kc-flow-007": true, // flow 类，不应被 polish 过滤命中
			"humanizer-zh": true,
		},
	}
	skills := LoadBuiltinSkills()

	polish := GetEnabledSkillsByCategory(skills, sc, "polish")
	if len(polish) == 0 {
		t.Fatal("category=polish 过滤结果为空，润色环节将无规则可注入")
	}
	for _, s := range polish {
		if s.Category != "polish" {
			t.Fatalf("polish 过滤返回了非 polish 分类的技能：%s(%s)", s.ID, s.Category)
		}
	}
	if !containsID(polish, "kc-flow-008") {
		t.Fatal("修改润色SOP(kc-flow-008) 未进入 polish 过滤结果，润色漏注入")
	}
	if containsID(polish, "kc-flow-007") {
		t.Fatal("章节写作SOP(kc-flow-007) 不应被 polish 过滤误命中")
	}
}

// TestFormatSkillsContentNonEmpty 验证启用技能经格式化后生成非空提示，且包含技能名。
func TestFormatSkillsContentNonEmpty(t *testing.T) {
	sc := &config.SkillConfig{
		EnabledSkills: map[string]bool{
			"kc-flow-007": true,
			"kc-skill-016": true, // 朱雀去痕技法
		},
	}
	skills := GetEnabledSkills(LoadBuiltinSkills(), sc)
	if len(skills) == 0 {
		t.Fatal("启用技能过滤结果为空")
	}

	out := FormatSkillsContent(skills)
	if strings.TrimSpace(out) == "" {
		t.Fatal("FormatSkillsContent 返回空提示")
	}
	if !strings.Contains(out, "章节写作SOP") || !strings.Contains(out, "朱雀去痕技法") {
		t.Fatalf("格式化结果未包含已启用技能名：%q", minrune(out, 80))
	}

	// 空技能列表应返回空串（不会 panic）
	if got := FormatSkillsContent(nil); got != "" {
		t.Fatalf("空技能列表应返回空串，实际 %q", got)
	}
}

func containsID(skills []Skill, id string) bool {
	for _, s := range skills {
		if s.ID == id {
			return true
		}
	}
	return false
}

func minrune(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}