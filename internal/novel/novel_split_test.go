package novel

import (
	"testing"

	"worldsim/internal/sim"
)

// TestNormalizeChapterBreaks 验证 LLM 输出乱序/重叠/漏天区间时，能被规整为
// 首尾相接、覆盖全部天数、不引入无记录天（day6）的合法章节。
func TestNormalizeChapterBreaks(t *testing.T) {
	w := &Writer{}
	chronicle := []sim.ChronicleEntry{
		{Day: 1, Kind: "FACT", Source: "事件", Content: "灵根觉醒"},
		{Day: 2, Kind: "FACT", Source: "事件", Content: "遭遇山贼"},
		{Day: 3, Kind: "SAID", Content: "与师父对话"},
		{Day: 4, Kind: "FACT", Source: "事件", Content: "离开小村"},
		{Day: 5, Kind: "FACT", Source: "事件", Content: "入城"},
		{Day: 7, Kind: "FACT", Source: "事件", Content: "试炼"},
		{Day: 8, Kind: "SAID", Content: "邀请入宗"},
	}
	// 注意：day6 无记录，不应出现在任何章节的 Days 里
	days := []int{1, 2, 3, 4, 5, 7, 8}

	cases := []struct {
		name   string
		breaks []chapterBreak
		want   [][2]int // [day_start, day_end]
	}{
		{
			name:   "正常连续区间",
			breaks: []chapterBreak{{Title: "觉醒", DayStart: 1, DayEnd: 2}, {Title: "入城", DayStart: 3, DayEnd: 5}, {Title: "入宗", DayStart: 7, DayEnd: 8}},
			want:   [][2]int{{1, 2}, {3, 5}, {7, 8}},
		},
		{
			name:   "乱序区间自动排序",
			breaks: []chapterBreak{{Title: "入宗", DayStart: 7, DayEnd: 8}, {Title: "觉醒", DayStart: 1, DayEnd: 2}, {Title: "入城", DayStart: 3, DayEnd: 5}},
			want:   [][2]int{{1, 2}, {3, 5}, {7, 8}},
		},
		{
			name:   "区间重叠并入上一章",
			breaks: []chapterBreak{{Title: "觉醒", DayStart: 1, DayEnd: 4}, {Title: "入城", DayStart: 2, DayEnd: 5}, {Title: "入宗", DayStart: 7, DayEnd: 8}},
			want:   [][2]int{{1, 5}, {7, 8}}, // {2,5}与{1,4}重叠→并入，覆盖扩展到5
		},
		{
			name:   "中间漏天自动并入相邻空档",
			breaks: []chapterBreak{{Title: "觉醒", DayStart: 1, DayEnd: 2}, {Title: "入宗", DayStart: 7, DayEnd: 8}},
			want:   [][2]int{{1, 5}, {7, 8}}, // 3~5并入首章，day6无记录自动跳过
		},
		{
			name:   "首部漏天自动补齐",
			breaks: []chapterBreak{{Title: "入城", DayStart: 3, DayEnd: 5}, {Title: "入宗", DayStart: 7, DayEnd: 8}},
			want:   [][2]int{{1, 2}, {3, 5}, {7, 8}},
		},
		{
			name:   "超尾自动截断",
			breaks: []chapterBreak{{Title: "觉醒", DayStart: 1, DayEnd: 2}, {Title: "入城", DayStart: 3, DayEnd: 99}},
			want:   [][2]int{{1, 2}, {3, 8}},
		},
		{
			name:   "区间起点落在无记录天上自动就近",
			breaks: []chapterBreak{{Title: "觉醒", DayStart: 1, DayEnd: 2}, {Title: "入城", DayStart: 6, DayEnd: 8}}, // day6无记录→就近到7
			want:   [][2]int{{1, 5}, {7, 8}},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := w.normalizeChapterBreaks(chronicle, days, c.breaks)
			if len(got) != len(c.want) {
				t.Fatalf("章节数=%d, 期望%d: %+v", len(got), len(c.want), got)
			}
			// 校验不以无记录天为 DayStart/DayEnd
			for i, g := range got {
				wv := c.want[i]
				if g.DayStart != wv[0] || g.DayEnd != wv[1] {
					t.Fatalf("第%d章边界=%d~%d, 期望%d~%d", i+1, g.DayStart, g.DayEnd, wv[0], wv[1])
				}
				for _, d := range g.Days {
					if d == 6 {
						t.Fatalf("第%d章包含无记录天 day6", i+1)
					}
				}
			}
		})
	}
}