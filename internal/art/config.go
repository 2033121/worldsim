// Package art 实现「美术工坊」：世界书驱动的像素素材生成系统。
//
// 三层架构（对标 Play Mode 的"代码层数值 + LLM 裁判"分工）：
//   - 规划层（plan.go）：LLM 读世界书 → plan.json（素材名单+每条英文提示词+调色板）
//   - 生成层（imagegen.go）：可插拔 Provider（openai_images 中转站 / pixellab 像素专用 API）
//   - 加工层（crop.go）：sheet 网格切分 → 非背景紧致 bbox → 边缘洪泛抠底 → sprite
//
// 设计原则：
//   - 零第三方依赖（go.mod 只有 stdlib）；算法手写，与 scripts/crop_grid.py 同构
//   - 密钥只存程序数据目录 img.json 或环境变量，仓库内零密钥
//   - 单品生成是一等公民（用户改某个角色时不重跑整张 sheet）
package art

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config 图片生成配置（存 progDir/img.json；结构对齐 api.json 惯例）
type Config struct {
	Provider       string `json:"provider"`                // openai_images（默认）| pixellab
	BaseURL        string `json:"base_url"`                // 如 http://ai.jiuqingyunapi.top 或 https://api.pixellab.ai/v1
	Model          string `json:"model"`                   // gpt-image-2 / pixflux 等；openai_images 用
	APIKey         string `json:"api_key,omitempty"`       // 直接写（本地部署场景）；优先级高于环境变量
	APIKeyEnv      string `json:"api_key_env,omitempty"`   // 从该环境变量读 key（默认 GPTIMG_KEY，兜底 GPT_IMAGE_API_KEY）
	Size           string `json:"size"`                    // 默认 1536x1024（sheet 主尺寸）
	Quality        string `json:"quality,omitempty"`       // openai_images 可选 quality 字段
	UserAgent      string `json:"user_agent,omitempty"`    // 默认 curl/8.5.0（中转站 WAF 拦常见程序 UA，实测需要）
	TimeoutSeconds int    `json:"timeout_seconds"`         // 单张生成超时，默认 300
	MaxRetries     int    `json:"max_retries"`             // 失败重试次数，默认 3（退避 1s/3s/9s）
	MaxConcurrent  int    `json:"max_concurrent"`          // 并发生成上限，默认 2
	Quantize       bool   `json:"quantize,omitempty"`      // 预留：调色板量化（v1.8 未实现，默认关）
	ExtraOutline   string `json:"extra_outline,omitempty"` // pixellab outline 参数（默认 single color black outline）
	ExtraShading   string `json:"extra_shading,omitempty"` // pixellab shading（默认 basic shading）
}

// DefaultConfig 返回骨架配置（未配密钥时模块软禁用，服务不受影响）
func DefaultConfig() *Config {
	return &Config{
		Provider:       "openai_images",
		BaseURL:        "",
		Model:          "gpt-image-2",
		APIKeyEnv:      "GPTIMG_KEY",
		Size:           "1536x1024",
		UserAgent:      "curl/8.5.0",
		TimeoutSeconds: 300,
		MaxRetries:     3,
		MaxConcurrent:  2,
		ExtraOutline:   "single color black outline",
		ExtraShading:   "basic shading",
	}
}

// LoadConfig 读取 img.json；不存在则写出默认骨架并返回（Enabled=false 由调用方据 APIKey 判定）
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := DefaultConfig()
			_ = os.MkdirAll(filepath.Dir(path), 0755)
			if saveErr := SaveConfig(path, cfg); saveErr != nil {
				return nil, fmt.Errorf("创建默认图片配置失败: %w", saveErr)
			}
			return cfg, nil
		}
		return nil, fmt.Errorf("读取图片配置失败: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析图片配置失败: %w", err)
	}
	fillDefaults(&cfg)
	return &cfg, nil
}

// SaveConfig 原子落盘（tmp+rename）
func SaveConfig(path string, cfg *Config) error {
	fillDefaults(cfg)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// fillDefaults 补齐零值字段（老配置文件兼容）
func fillDefaults(cfg *Config) {
	if cfg.Provider == "" {
		cfg.Provider = "openai_images"
	}
	if cfg.Model == "" {
		cfg.Model = "gpt-image-2"
	}
	if cfg.APIKeyEnv == "" {
		cfg.APIKeyEnv = "GPTIMG_KEY"
	}
	if cfg.Size == "" {
		cfg.Size = "1536x1024"
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = "curl/8.5.0"
	}
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = 300
	}
	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = 0
	}
	if cfg.MaxConcurrent <= 0 {
		cfg.MaxConcurrent = 2
	}
	if cfg.ExtraOutline == "" {
		cfg.ExtraOutline = "single color black outline"
	}
	if cfg.ExtraShading == "" {
		cfg.ExtraShading = "basic shading"
	}
}

// ResolveAPIKey 密钥解析：api_key > 环境变量（APIKeyEnv，兜底 GPT_IMAGE_API_KEY）> 空
func (c *Config) ResolveAPIKey() string {
	if c == nil {
		return ""
	}
	if c.APIKey != "" {
		return c.APIKey
	}
	if env := c.APIKeyEnv; env != "" {
		if v := os.Getenv(env); v != "" {
			return v
		}
	}
	return os.Getenv("GPT_IMAGE_API_KEY")
}

// Enabled 密钥+base_url 就绪才算启用
func (c *Config) Enabled() bool {
	return c != nil && c.ResolveAPIKey() != "" && strings.TrimSpace(c.BaseURL) != ""
}

// MaskedKey 返回掩码（sk-abc1****wxyz），API 读取永远用这个
func (c *Config) MaskedKey() string {
	key := c.ResolveAPIKey()
	if key == "" {
		return ""
	}
	head, tail := key, key
	if len(key) > 6 {
		head = key[:6]
	}
	if len(key) > 4 {
		tail = key[len(key)-4:]
	}
	if len(key) <= 10 {
		return head + "****"
	}
	return head + "****" + tail
}
