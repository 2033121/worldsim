package search

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// BingHTML 是基于必应中国大陆版（cn.bing.com）HTML 搜索页的纯抓取后端。
// 国内直连、免 API key、免浏览器、零第三方依赖，仅做 HTML 正则解析。
// 注意：抓取类接口可能受反爬影响，失败时返回错误由上层降级处理。
type BingHTML struct {
	BaseURL string
	Client  *http.Client
	// userAgent 模拟常见浏览器，降低被反爬拦截的概率。
	userAgent string
}

// NewBingHTML 创建 cn.bing.com HTML 抓取后端。baseURL 为空时使用官方国内地址。
func NewBingHTML(baseURL string, timeout time.Duration) *BingHTML {
	if baseURL == "" {
		baseURL = "https://cn.bing.com/search"
	}
	return &BingHTML{
		BaseURL: baseURL,
		Client: &http.Client{
			Timeout: timeout,
			// 保留默认 Transport（含系统代理、连接池）。
			Transport: &http.Transport{ForceAttemptHTTP2: true},
		},
		userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	}
}

// Name 返回提供方名称。
func (b *BingHTML) Name() string { return "binghtml" }

// 解析用正则：匹配 <li class="b_algo">…</li> 结果块。
var (
	reAlgoBlock = regexp.MustCompile(`<li class="b_algo"[\s\S]*?</li>`)
	reResult    = regexp.MustCompile(`<h2[^>]*><a[^>]*href="([^"]+)"[^>]*>([\s\S]*?)</a></h2>`)
	reSnippet   = regexp.MustCompile(`class="b_caption[\s\S]*?<p[^>]*>([\s\S]*?)</p>`)
	reTag       = regexp.MustCompile(`<[^>]+>`)
	reEntity    = regexp.MustCompile(`&[a-zA-Z#0-9]+;`)
)

// Search 抓取 cn.bing.com 搜索页并解析结果。
func (b *BingHTML) Search(ctx context.Context, query string, max int, language string) ([]Result, error) {
	if max <= 0 {
		max = 5
	}
	if max > 20 {
		max = 20
	}

	u := b.BaseURL + "?q=" + url.QueryEscape(query)
	if language != "" {
		u += "&setlang=" + url.QueryEscape(strings.ReplaceAll(language, "_", "-"))
	}

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", b.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")

	resp, err := b.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("binghtml 请求失败: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("binghtml 响应错误，状态码: %d", resp.StatusCode)
	}

	html := string(body)
	blocks := reAlgoBlock.FindAllString(html, max)
	out := make([]Result, 0, len(blocks))
	for _, block := range blocks {
		m := reResult.FindStringSubmatch(block)
		if len(m) < 3 {
			continue
		}
		link := strings.TrimSpace(m[1])
		title := cleanHTML(m[2])
		if title == "" || link == "" || strings.HasPrefix(link, "javascript:") {
			continue
		}
		snippet := ""
		if sm := reSnippet.FindStringSubmatch(block); len(sm) > 1 {
			snippet = cleanHTML(sm[1])
		}
		out = append(out, Result{
			Title:   title,
			URL:     link,
			Content: snippet,
			Engine:  "bing(cn)",
		})
	}
	return out, nil
}

// cleanHTML 去除 HTML 标签与实体，返回纯文本。
func cleanHTML(s string) string {
	s = reTag.ReplaceAllString(s, "")
	s = reEntity.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}