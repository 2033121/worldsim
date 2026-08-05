package search

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"worldsim/internal/llm"
)

// RegisterWebSearch 把 web_search 工具注册进给定的 ToolRegistry，绑定到 prov 搜索后端，
// 执行结果格式化为纯文本（标题/链接/摘要），作为 role:"tool" 消息回传给 LLM。
// maxResults 为默认返回条数上限（单次不超过 10）。
func RegisterWebSearch(reg *llm.ToolRegistry, prov Provider, maxResults int) {
	if reg == nil || prov == nil {
		return
	}
	if maxResults <= 0 {
		maxResults = 5
	}
	reg.Register("web_search", llm.WebSearchToolSchema(),
		func(ctx context.Context, args json.RawMessage) (string, error) {
			var a llm.WebSearchArgs
			if err := json.Unmarshal(args, &a); err != nil {
				return "", fmt.Errorf("web_search 参数解析失败: %w", err)
			}
			if strings.TrimSpace(a.Query) == "" {
				return "", fmt.Errorf("web_search 缺少 query 参数")
			}
			n := a.MaxResult
			if n <= 0 {
				n = maxResults
			}
			if n > 10 {
				n = 10
			}
			results, err := prov.Search(ctx, a.Query, n, a.Language)
			if err != nil {
				return "", err
			}
			if len(results) == 0 {
				return "搜索【" + a.Query + "】没有找到相关结果。", nil
			}
			var b strings.Builder
			fmt.Fprintf(&b, "搜索【%s】共 %d 条结果：\n", a.Query, len(results))
			for i, r := range results {
				fmt.Fprintf(&b, "%d. %s\n   链接: %s\n   摘要: %s\n",
					i+1, firstNonEmpty(r.Title, "(无标题)"), firstNonEmpty(r.URL, "(无链接)"), firstNonEmpty(r.Content, "(无摘要)"))
			}
			return b.String(), nil
		})
}

func firstNonEmpty(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}
