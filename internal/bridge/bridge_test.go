package bridge

import (
	"os"
	"path/filepath"
	"testing"

	"worldsim/internal/i18n"
	"worldsim/internal/story"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// TestSeedProjectFromWorld 验证：从真实世界数据（世界书/世界状态/编年史）
// 播种小说项目，且 progress.json / settings.json 的关键字段被世界数据填充。
func TestSeedProjectFromWorld(t *testing.T) {
	progDir := t.TempDir()
	worldName := "测试都市"
	worldDir := filepath.Join(progDir, "worlds", worldName)

	// 世界书（A1/A6/A8/C0 段）
	wb := `# 测试都市
## A1 世界观
都市背后的神仙世界，主角给神仙发工资。
## A6 主角目标链
短期：签下嫦娥入职；长期：成立诸神人事部。
## A8 反派行动线
天庭裁员办持续施压。
## C0 题材基调
喜剧、职场、打脸。
`
	writeFile(t, filepath.Join(progDir, "worldbooks", worldName+".md"), wb)

	// 世界状态（角色 + 势力 + 天数）
	state := `{
  "day": 3,
  "entities": {
    "沈听眠": {"job": "HR", "assets": {"现金": 3000, "功德值": 80},
      "extra": {"role": "protagonist", "persona": "打工人",
        "persona_sheet": "{\"name\":\"沈听眠\",\"identity\":\"主角HR\",\"personality\":[\"毒舌\",\"热心\"],\"motives\":[\"给神仙发工资\"],\"secret\":\"看得见神仙\"}"}},
    "嫦娥": {"job": "文员", "extra": {"role": "love_interest"}}
  },
  "world_level": {
    "factions": {
      "天庭人事部": {"visibility": "public", "stance": "支持主角", "recent_actions": ["批准新合同"]}
    }
  }
}`
	writeFile(t, filepath.Join(worldDir, "world_state.json"), state)

	// 近期编年史事件（开篇章节钩子）
	chronicle := `{"day":1,"kind":"FACT","content":"嫦娥投递简历"}`
	chronicle += "\n" + `{"day":2,"kind":"FACT","content":"沈听眠约见嫦娥"}`
	chronicle += "\n" + `{"day":3,"kind":"FACT","content":"签订试用期合同"}`
	writeFile(t, filepath.Join(worldDir, "chronicle.jsonl"), chronicle)

	// 读取世界数据
	data, err := ReadWorld(progDir, worldName)
	if err != nil {
		t.Fatalf("ReadWorld: %v", err)
	}
	if data.Day != 3 {
		t.Errorf("Day = %d, want 3", data.Day)
	}
	if len(data.Characters) != 2 {
		t.Errorf("characters = %d, want 2", len(data.Characters))
	}
	if len(data.RecentEvents) != 3 {
		t.Errorf("recent events = %d, want 3", len(data.RecentEvents))
	}

	// 播种
	projectDir := filepath.Join(progDir, "storys", "测试小说")
	res, err := SeedProjectFromWorld(projectDir, i18n.LangZH, data)
	if err != nil {
		t.Fatalf("SeedProjectFromWorld: %v", err)
	}
	if !res.Reused {
		t.Error("expected Reused=true")
	}
	if res.CharacterCount != 2 {
		t.Errorf("character count = %d, want 2", res.CharacterCount)
	}

	// settings.json：角色 / 世界观 / 组织 被填充且来自世界数据
	settings, err := story.LoadProjectSettings(filepath.Join(projectDir, "settings.json"))
	if err != nil {
		t.Fatalf("load settings: %v", err)
	}
	if len(settings.Characters) != 2 {
		t.Errorf("seeded characters = %d, want 2", len(settings.Characters))
	}
	var foundPersonality bool
	for _, c := range settings.Characters {
		if c.Name == "沈听眠" && c.Personality == "毒舌、热心" {
			foundPersonality = true
		}
	}
	if !foundPersonality {
		t.Error("expected 沈听眠 personality from world persona_sheet")
	}
	if len(settings.Worldview) == 0 {
		t.Error("seeded worldview empty, want entries from worldbook")
	}
	if len(settings.Organizations) != 1 {
		t.Errorf("seeded organizations = %d, want 1", len(settings.Organizations))
	}

	// progress.json：大纲章节来自近期事件，梗概来自世界书
	progress, err := story.LoadProgress(filepath.Join(projectDir, "progress.json"))
	if err != nil {
		t.Fatalf("load progress: %v", err)
	}
	if len(progress.Chapters) != 3 {
		t.Errorf("outline chapters = %d, want 3", len(progress.Chapters))
	}
	if progress.StorySynopsis == "" {
		t.Error("expected story synopsis from worldbook")
	}
}

// TestReadWorldMissing 验证无世界时返回合理错误。
func TestReadWorldMissing(t *testing.T) {
	progDir := t.TempDir()
	if _, err := ReadWorld(progDir, "不存在"); err == nil {
		t.Error("expected error for missing world")
	}
}