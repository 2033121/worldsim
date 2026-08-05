package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Tavily 是基于 Tavily Search API 的搜索后端（内置，无需额外容器）。
// 需要外网可达；Go 的 http.Transport 默认读取 HTTP_PROXY/HTTPS_PROXY 系统代理，
// 因此受限网络下可通过代理环境变量正常访问。
// API 文档：https://docs.tavily.com/
type Tavily struct {
	APIKey string
	Client *http.Client
}

// NewTavily 创建 Tavily 搜索后端。
func NewTavily(apiKey string, timeout time.Duration) *Tavily {
	return &Tavily{
		APIKey: strings.TrimSpace(apiKey),
		Client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{ForceAttemptHTTP2: true},
		},
	}
}

// Name 返回提供方名称。
func (t *Tavily) Name() string { return "tavily" }

// Search 调用 Tavily Search API 搜索网页。
// endpoint 使用 /search 接口，通过 Authorization: Bearer 鉴权（Tavily 推荐方式）。
func (t *Tavily) Search(ctx context.Context, query string, max int, language string) ([]Result, error) {
	if strings.TrimSpace(t.APIKey) == "" {
		return nil, fmt.Errorf("tavily 搜索未配置 API key（search.json 的 tavily_api_key）")
	}
	if max <= 0 {
		max = 5
	}
	if max > 10 {
		max = 10 // Tavily max_results 上限为 20，这里保守限制为 10
	}

	body := map[string]any{
		"query":       query,
		"max_results": max,
		"search_depth": "basic",
	}
	if language != "" {
		body["topic"] = "general"
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.tavily.com/search", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+t.APIKey)

	resp, err := t.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tavily 请求失败: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("tavily API key 无效或已过期（HTTP %d）", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tavily 响应错误，状态码: %d，内容: %.200s", resp.StatusCode, string(respBody))
	}

	var payloadOut struct {
		Results []struct {
			Title   string  `json:"title"`
			URL     string  `json:"url"`
			Content string  `json:"content"`
			Score   float64 `json:"score"`
		} `json:"results"`
	}
	if err := json.Unmarshal(respBody, &payloadOut); err != nil {
		return nil, fmt.Errorf("tavily JSON 解析失败: %w", err)
	}

	out := make([]Result, 0, max)
	for _, r := range payloadOut.Results {
		if len(out) >= max {
			break
		}
		out = append(out, Result{
			Title:   strings.TrimSpace(r.Title),
			URL:     strings.TrimSpace(r.URL),
			Content: strings.TrimSpace(r.Content),
			Engine:  "tavily",
			Score:   r.Score,
		})
	}
	return out, nil
}