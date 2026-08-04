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

// SearXNG 是基于自托管 searxng 的搜索后端（免费、无限、无鉴权）。
// 调用 /search?q=...&format=json 返回 JSON；聚合多引擎，结果干净。
type SearXNG struct {
	BaseURL string
	Client  *http.Client
}

// NewSearXNG 创建 SearXNG 后端。
func NewSearXNG(baseURL string, timeout time.Duration) *SearXNG {
	return &SearXNG{
		BaseURL: strings.TrimSuffix(baseURL, "/"),
		Client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				ForceAttemptHTTP2:  false,
				DisableKeepAlives:  true,
				MaxIdleConns:       1,
				MaxIdleConnsPerHost: 1,
			},
		},
	}
}

// Name 返回提供方名称。
func (s *SearXNG) Name() string { return "searxng" }

// Search 调用 searxng 的 JSON API 搜索。
func (s *SearXNG) Search(ctx context.Context, query string, max int, language string) ([]Result, error) {
	u := s.BaseURL + "/search?q=" + url.QueryEscape(query) + "&format=json"
	if language != "" {
		u += "&language=" + url.QueryEscape(language)
	}
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("searxng 请求失败: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("searxng 响应错误，状态码: %d，内容: %.200s", resp.StatusCode, string(body))
	}

	var payload struct {
		Results []struct {
			URL     string  `json:"url"`
			Title   string  `json:"title"`
			Content string  `json:"content"`
			Engine  string  `json:"engine"`
			Score   float64 `json:"score"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("searxng JSON 解析失败: %w", err)
	}

	out := make([]Result, 0, max)
	for _, r := range payload.Results {
		if len(out) >= max {
			break
		}
		out = append(out, Result{
			Title:   strings.TrimSpace(r.Title),
			URL:     strings.TrimSpace(r.URL),
			Content: strings.TrimSpace(r.Content),
			Engine:  r.Engine,
			Score:   r.Score,
		})
	}
	return out, nil
}