package search

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Bing 是基于微软 Bing Web Search API v7 的搜索后端（内置，无需额外容器）。
// 需要外网可达；Go 的 http.Transport 默认读取 HTTP_PROXY/HTTPS_PROXY 系统代理，
// 因此受限网络下可通过代理环境变量正常访问。
// API 文档：https://learn.microsoft.com/bing/search-apis/bing-web-search
type Bing struct {
	APIKey   string
	BaseURL  string // 默认 https://api.bing.microsoft.com/v7.0/search
	Client   *http.Client
	maxQuery int
}

// NewBing 创建 Bing 搜索后端。baseURL 为空时使用官方默认地址。
func NewBing(apiKey, baseURL string, timeout time.Duration) *Bing {
	if baseURL == "" {
		baseURL = "https://api.bing.microsoft.com/v7.0/search"
	}
	return &Bing{
		APIKey:  strings.TrimSpace(apiKey),
		BaseURL: strings.TrimSuffix(baseURL, "/"),
		Client: &http.Client{
			Timeout: timeout,
			// 保留默认 Transport（含系统代理、连接池），不手动禁用代理。
			Transport: &http.Transport{ForceAttemptHTTP2: true},
		},
		maxQuery: 32,
	}
}

// Name 返回提供方名称。
func (b *Bing) Name() string { return "bing" }

// Search 调用 Bing Web Search API 搜索网页。
func (b *Bing) Search(ctx context.Context, query string, max int, language string) ([]Result, error) {
	if strings.TrimSpace(b.APIKey) == "" {
		return nil, fmt.Errorf("bing 搜索未配置 API key（search.json 的 bing_api_key）")
	}
	if max <= 0 {
		max = 5
	}
	if max > 50 {
		max = 50
	}

	u := b.BaseURL + "?q=" + url.QueryEscape(query) + "&count=" + itoa(max) + "&responseFilter=WebPages&mkt=zh-CN"
	if language != "" {
		u += "&mkt=" + url.QueryEscape(strings.ReplaceAll(language, "_", "-"))
	}

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Ocp-Apim-Subscription-Key", b.APIKey)
	req.Header.Set("Accept", "application/json")

	resp, err := b.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("bing 请求失败: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("bing API key 无效或已过期（HTTP %d）", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bing 响应错误，状态码: %d，内容: %.200s", resp.StatusCode, string(body))
	}

	var payload struct {
		WebPages struct {
			Value []struct {
				Name    string `json:"name"`
				URL     string `json:"url"`
				Snippet string `json:"snippet"`
			} `json:"value"`
		} `json:"webPages"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("bing JSON 解析失败: %w", err)
	}

	out := make([]Result, 0, max)
	for _, p := range payload.WebPages.Value {
		if len(out) >= max {
			break
		}
		out = append(out, Result{
			Title:   strings.TrimSpace(p.Name),
			URL:     strings.TrimSpace(p.URL),
			Content: strings.TrimSpace(p.Snippet),
			Engine:  "bing",
		})
	}
	return out, nil
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
