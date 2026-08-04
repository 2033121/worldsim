package search

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Result 是归一化的搜索结果条目。
type Result struct {
	Title   string  `json:"title"`
	URL     string  `json:"url"`
	Content string  `json:"content"` // 摘要/正文片段
	Engine  string  `json:"engine,omitempty"`
	Score   float64 `json:"score,omitempty"`
}

// Provider 抽象一个联网搜索后端。
type Provider interface {
	Name() string
	// Search 按 query 搜索，返回最多 max 条结果；language 为语言偏好（可为空）。
	Search(ctx context.Context, query string, max int, language string) ([]Result, error)
}

// Config 描述搜索后端配置（对应 wsdata/search.json）。
type Config struct {
	Enabled        bool   `json:"enabled"`
	Provider       string `json:"provider"` // 当前仅支持 searxng
	SearxngURL     string `json:"searxng_url"`
	MaxResults     int    `json:"max_results"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

// DefaultConfig 返回默认配置（SearXNG 自托管、本地回环、无鉴权）。
func DefaultConfig() *Config {
	return &Config{
		Enabled:        true,
		Provider:       "searxng",
		SearxngURL:     "http://localhost:8888",
		MaxResults:     5,
		TimeoutSeconds: 20,
	}
}

// LoadConfig 从 path 读取配置；文件不存在返回 nil（搜索不启用，不报错）。
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析搜索配置失败: %w", err)
	}
	// 补齐默认值
	def := DefaultConfig()
	if cfg.Provider == "" {
		cfg.Provider = def.Provider
	}
	if cfg.SearxngURL == "" {
		cfg.SearxngURL = def.SearxngURL
	}
	if cfg.MaxResults <= 0 {
		cfg.MaxResults = def.MaxResults
	}
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = def.TimeoutSeconds
	}
	return &cfg, nil
}

// NewProvider 按配置创建搜索后端 Provider。
func NewProvider(cfg *Config) (Provider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("搜索配置为空")
	}
	switch cfg.Provider {
	case "searxng":
		return NewSearXNG(cfg.SearxngURL, time.Duration(cfg.TimeoutSeconds)*time.Second), nil
	default:
		return nil, fmt.Errorf("不支持的搜索后端: %s", cfg.Provider)
	}
}