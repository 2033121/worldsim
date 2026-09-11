package art

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"worldsim/internal/worldbook"
)

// ---------- 规划解析 ----------

func planJSON() string {
	return "```json\n" + `{
  "theme": "xianxia",
  "palette": "ink-blue + jade",
  "palette_hex": ["#2c3e50", "#7d9d6c"],
  "characters": [{"name":"林九","role":"主角","prompt":"young cultivator in ink-blue robe"}],
  "monsters": [],
  "scenes": [{"name":"青牛镇","prompt":"market street at dawn"}],
  "tiles": [],
  "items": []
}` + "\n```"
}

func TestParsePlanClamp(t *testing.T) {
	wb := worldbook.Parse("# 世界书：测试\n## A1 世界观\n修仙世界")
	plan, err := ParsePlan(planJSON(), wb)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Theme != "xianxia" {
		t.Fatalf("theme=%s", plan.Theme)
	}
	if len(plan.Characters) != 12 {
		t.Fatalf("characters=%d, want 12", len(plan.Characters))
	}
	if plan.Characters[0].Name != "林九" {
		t.Fatalf("首条丢失")
	}
	if plan.Characters[1].Name == "" {
		t.Fatalf("占位缺失")
	}
	if len(plan.Monsters) != 8 || len(plan.Tiles) != 16 || len(plan.Items) != 24 {
		t.Fatalf("数量校验失败 m=%d t=%d i=%d", len(plan.Monsters), len(plan.Tiles), len(plan.Items))
	}
	if len(plan.Scenes) != 3 || plan.Scenes[1].Time != "dusk" {
		t.Fatalf("scenes=%v", plan.Scenes)
	}
	if warns := plan.ValidatePlan(); len(warns) == 0 {
		t.Fatal("缺 prompt 应有告警")
	}
}

// prompt 组装：契约与色板必须注入
func TestPromptAssembly(t *testing.T) {
	plan, _ := ParsePlan(planJSON(), worldbook.Parse("# t\n## A1\n修仙"))
	ap := plan.AssetPrompt("character", plan.Characters[0])
	for _, want := range []string{"young cultivator", "#2c3e50", "full-body front view", "#e8e8e8"} {
		if !strings.Contains(ap, want) {
			t.Errorf("AssetPrompt 缺 %q", want)
		}
	}
	spec := SpecByLabel("characters")
	sp := plan.SheetPrompt(spec, plan.Characters)
	for _, want := range []string{"4x3", "1. 林九", "full-body front view"} {
		if !strings.Contains(sp, want) {
			t.Errorf("SheetPrompt 缺 %q", want)
		}
	}
}

// ---------- openai_images Provider（httptest 伪中转站） ----------

// encodeImg 无 t 版本编码（httptest handler 里用）
func encodeImg(img *image.RGBA) []byte {
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// srvURL 请求自身的 scheme://host（伪图床下载地址）
func srvURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

func TestOpenAIProviderFlow(t *testing.T) {
	fakePNG := encodeImg(synthCell(4, 4, color.RGBA{0, 0, 0, 255}, color.RGBA{255, 0, 0, 255}, 1))

	var sawUA, sawAuth, sawPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawUA = r.Header.Get("User-Agent")
		if strings.HasSuffix(r.URL.Path, "/images/generations") {
			sawPath = r.URL.Path
			sawAuth = r.Header.Get("Authorization")
		}
		if strings.HasSuffix(r.URL.Path, "/images/generations") {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]string{{"url": srvURL(r) + "/img.png"}},
			})
			return
		}
		w.Write(fakePNG)
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL // 无 /v1，应自动补
	cfg.APIKey = "sk-fake"
	gen := NewGenerator(cfg)
	if gen == nil {
		t.Fatal("应可用")
	}
	data, err := gen.Generate(context.Background(), "a red knight", "64x64")
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != len(fakePNG) {
		t.Fatalf("下载图长度 %d != %d", len(data), len(fakePNG))
	}
	if sawUA != "curl/8.5.0" {
		t.Fatalf("UA=%s（WAF 伪装头必须生效）", sawUA)
	}
	if sawAuth != "Bearer sk-fake" {
		t.Fatalf("auth=%s", sawAuth)
	}
	if !strings.HasSuffix(sawPath, "/v1/images/generations") {
		t.Fatalf("path=%s（应自动补 /v1）", sawPath)
	}
}

// b64_json 分支
func TestOpenAIProviderB64(t *testing.T) {
	fakePNG := encodeImg(synthCell(4, 4, color.RGBA{0, 0, 0, 255}, color.RGBA{0, 255, 0, 255}, 1))
	b64 := base64.StdEncoding.EncodeToString(fakePNG)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{"b64_json": b64}}})
	}))
	defer srv.Close()
	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL + "/v1" // 已带 /v1，不应重复
	cfg.APIKey = "k"
	gen := NewGenerator(cfg)
	data, err := gen.Generate(context.Background(), "p", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != len(fakePNG) {
		t.Fatal("b64 解码结果不对")
	}
}

// 错误透传 + 重试退避
func TestOpenAIProviderRetry(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":{"message":"upstream busy"}}`))
	}))
	defer srv.Close()
	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.APIKey = "k"
	cfg.MaxRetries = 1
	gen := NewGenerator(cfg)
	_, err := gen.Generate(context.Background(), "p", "")
	if err == nil || !strings.Contains(err.Error(), "upstream busy") {
		t.Fatalf("错误应透传: %v", err)
	}
	if hits != 2 { // 1 次初始 + 1 次重试
		t.Fatalf("hits=%d", hits)
	}
}
