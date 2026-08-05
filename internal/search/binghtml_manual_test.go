package search

import (
	"context"
	"testing"
	"time"
)

// TestBingHTMLRealSearch 真实联网验证 cn.bing.com 抓取（需外网）。失败时跳过不阻塞 CI。
func TestBingHTMLRealSearch(t *testing.T) {
	b := NewBingHTML("", 15*time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 18*time.Second)
	defer cancel()

	results, err := b.Search(ctx, "Python 爬虫 入门教程", 5, "zh-CN")
	if err != nil {
		t.Skipf("联网抓取暂时不可用，跳过: %v", err)
	}
	if len(results) == 0 {
		t.Skip("cn.bing.com 未返回结果")
	}
	t.Logf("返回 %d 条结果", len(results))
	for i, r := range results {
		t.Logf("%d. %s | %s | %s", i+1, r.Title, r.URL, truncate(r.Content, 40))
		if r.Title == "" || r.URL == "" {
			t.Errorf("结果 %d 缺少标题或链接", i)
		}
	}
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
