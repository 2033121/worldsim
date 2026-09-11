package art

// imagegen.go — 可插拔图片生成客户端。
//
// Provider 协议：
//   openai_images（默认，中转站）：OpenAI images 协议 POST {base}/v1/images/generations，
//     模型 gpt-image-2，返回 data[0].url（b64_json 兜底）。注意：中转站 WAF 会 403
//     拒绝 python-urllib 等程序 UA，请求必须带 curl/8.5.0 伪装头（实测结论）。
//   pixellab（预留）：PixelLab Pixflux 端点，noBackground 直出透明底可跳过抠底；
//     Bitforge 可带风格参考图锁调色板。见 https://api.pixellab.ai/v1/docs。

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Generator 一次文生图调用，返回 PNG 字节
type Generator interface {
	Generate(ctx context.Context, prompt, size string) ([]byte, error)
}

// NewGenerator 按配置构造（未启用返回 nil，调用方降级）
func NewGenerator(cfg *Config) Generator {
	if cfg == nil || !cfg.Enabled() {
		return nil
	}
	switch cfg.Provider {
	case "pixellab":
		return &retryGen{cfg: cfg, inner: &pixellabProvider{cfg: cfg}}
	default:
		return &retryGen{cfg: cfg, inner: &openaiImagesProvider{cfg: cfg}}
	}
}

// ---------- openai_images Provider ----------

type openaiImagesProvider struct {
	cfg *Config
}

type imageGenRequest struct {
	Model   string `json:"model"`
	Prompt  string `json:"prompt"`
	Size    string `json:"size,omitempty"`
	N       int    `json:"n"`
	Quality string `json:"quality,omitempty"`
}

type imageGenResponse struct {
	Data []struct {
		URL     string `json:"url"`
		B64JSON string `json:"b64_json"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (p *openaiImagesProvider) Generate(ctx context.Context, prompt, size string) ([]byte, error) {
	base := strings.TrimRight(p.cfg.BaseURL, "/")
	if !strings.Contains(base, "/v1") {
		base += "/v1"
	}
	endpoint := base + "/images/generations"

	body := imageGenRequest{
		Model:   p.cfg.Model,
		Prompt:  prompt,
		Size:    orDefault(size, p.cfg.Size),
		N:       1,
		Quality: p.cfg.Quality,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.cfg.ResolveAPIKey())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", p.cfg.UserAgent)

	client := &http.Client{Timeout: time.Duration(p.cfg.TimeoutSeconds) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("图片生成请求失败: %w", err)
	}
	defer resp.Body.Close()

	var parsed imageGenResponse
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("图片生成返回非 JSON（HTTP %d）", resp.StatusCode)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return nil, fmt.Errorf("图片生成失败（HTTP %d）: %s", resp.StatusCode, parsed.Error.Message)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("图片生成失败（HTTP %d）", resp.StatusCode)
	}
	if len(parsed.Data) == 0 {
		return nil, fmt.Errorf("图片生成未返回图片")
	}
	item := parsed.Data[0]
	if item.B64JSON != "" {
		if data, err := base64.StdEncoding.DecodeString(item.B64JSON); err == nil {
			return data, nil
		}
	}
	if item.URL == "" {
		return nil, fmt.Errorf("图片生成未返回图片")
	}
	return p.download(ctx, item.URL)
}

// download 下载生成图（同样带伪装 UA，中转站图床同域拦截）
func (p *openaiImagesProvider) download(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", p.cfg.UserAgent)
	client := &http.Client{Timeout: time.Duration(p.cfg.TimeoutSeconds) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("下载生成图失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("下载生成图失败（HTTP %d）", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 32<<20))
}

// ---------- pixellab Provider（预留，Pixflux 文生图） ----------

type pixellabProvider struct {
	cfg *Config
}

type pixellabRequest struct {
	Description          string `json:"description"`
	ImageSize            struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"image_size"`
	NegativeDescription string `json:"negative_description,omitempty"`
	NoBackground        bool   `json:"no_background"`
	Outline             string `json:"outline,omitempty"`
	Shading             string `json:"shading,omitempty"`
	Detail              string `json:"detail,omitempty"`
}

type pixellabResponse struct {
	Image string `json:"image"` // data:image/png;base64,… 或纯 base64
	Error string `json:"error,omitempty"`
}

func (p *pixellabProvider) Generate(ctx context.Context, prompt, size string) ([]byte, error) {
	w, h := parseSize(orDefault(size, "64x64"))
	body := pixellabRequest{
		Description:         prompt,
		NegativeDescription: "text, letters, numbers, watermark, grid lines, blurry",
		NoBackground:        true, // 直出透明底：crop 管线自动跳过抠底
		Outline:             p.cfg.ExtraOutline,
		Shading:             p.cfg.ExtraShading,
		Detail:              "medium detail",
	}
	body.ImageSize.Width, body.ImageSize.Height = w, h
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	endpoint := strings.TrimRight(p.cfg.BaseURL, "/") + "/generate-image-pixflux"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.cfg.ResolveAPIKey())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", p.cfg.UserAgent)
	client := &http.Client{Timeout: time.Duration(p.cfg.TimeoutSeconds) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pixellab 请求失败: %w", err)
	}
	defer resp.Body.Close()
	var parsed pixellabResponse
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("pixellab 返回非 JSON（HTTP %d）", resp.StatusCode)
	}
	if parsed.Error != "" {
		return nil, fmt.Errorf("pixellab 失败: %s", parsed.Error)
	}
	b64 := parsed.Image
	if i := strings.Index(b64, "base64,"); i >= 0 {
		b64 = b64[i+len("base64,"):]
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
	if err != nil {
		return nil, fmt.Errorf("pixellab 图片解码失败: %w", err)
	}
	return data, nil
}

// ---------- 重试包装 ----------

type retryGen struct {
	cfg   *Config
	inner Generator
}

func (r *retryGen) Generate(ctx context.Context, prompt, size string) ([]byte, error) {
	retries := r.cfg.MaxRetries
	var lastErr error
	for attempt := 0; attempt <= retries; attempt++ {
		if attempt > 0 {
			// 退避 1s/3s/9s；ctx 取消立即退出
			backoff := time.Duration(1<<(attempt-1)*2) * time.Second
			if backoff > 9*time.Second {
				backoff = 9 * time.Second
			}
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		data, err := r.inner.Generate(ctx, prompt, size)
		if err == nil {
			return data, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("图片生成失败")
	}
	return nil, fmt.Errorf("重试 %d 次后仍失败: %w", retries, lastErr)
}

// ---------- 小工具 ----------

func orDefault(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}

// parseSize "1536x1024" → (1536,1024)；解析失败退 64x64
func parseSize(s string) (int, int) {
	parts := strings.SplitN(strings.ToLower(strings.TrimSpace(s)), "x", 2)
	if len(parts) != 2 {
		return 64, 64
	}
	var w, h int
	if _, err := fmt.Sscanf(parts[0], "%d", &w); err != nil {
		return 64, 64
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &h); err != nil {
		return 64, 64
	}
	if w <= 0 || h <= 0 || w > 4096 || h > 4096 {
		return 64, 64
	}
	return w, h
}
