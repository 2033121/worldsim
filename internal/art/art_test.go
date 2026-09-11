package art

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"
	"testing"

	"worldsim/internal/worldbook"
)

// synthCell 画一格：纯色主体块（居中）+ 指定背景
func synthCell(w, h int, bg color.RGBA, fg color.RGBA, inset int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if x >= inset && x < w-inset && y >= inset && y < h-inset {
				img.SetRGBA(x, y, fg)
			} else {
				img.SetRGBA(x, y, bg)
			}
		}
	}
	return img
}

func encodeT(t *testing.T, img *image.RGBA) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// 合成一张 4x2 sheet（灰底 #e8e8e8、红/蓝主体交替，最后一格故意留空）
func synthSheet(t *testing.T, cols, rows, cw, ch int) []byte {
	t.Helper()
	bg := color.RGBA{232, 232, 232, 255}
	sheet := image.NewRGBA(image.Rect(0, 0, cols*cw, rows*ch))
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			fg := color.RGBA{200, 30, 30, 255}
			if row%2 == 1 {
				fg = color.RGBA{30, 30, 200, 255}
			}
			cell := synthCell(cw, ch, bg, fg, 20)
			// 最后一格留空（纯背景）
			if row == rows-1 && col == cols-1 {
				cell = synthCell(cw, ch, bg, bg, 0)
			}
			b := cell.Bounds()
			for y := 0; y < b.Dy(); y++ {
				for x := 0; x < b.Dx(); x++ {
					sheet.SetRGBA(col*cw+x, row*ch+y, cell.RGBAAt(x, y))
				}
			}
		}
	}
	return encodeT(t, sheet)
}

func TestCropSheet(t *testing.T) {
	spec := &GridSpec{Label: "monsters", FileLabel: "monster", Cols: 4, Rows: 2, Alpha: true, Size: "320x160"}
	sheet := synthSheet(t, 4, 2, 80, 80)
	cells, err := CropSheet(sheet, spec)
	if err != nil {
		t.Fatal(err)
	}
	if len(cells) != 8 {
		t.Fatalf("cells=%d, want 8", len(cells))
	}
	// 每个非空格应抠底成功且角落透明
	for i, c := range cells {
		if i == 7 {
			if !c.Empty {
				t.Fatalf("cell 8 应判空格")
			}
			continue
		}
		if c.Empty {
			t.Fatalf("cell %d 不应为空", i+1)
		}
		img, err := png.Decode(bytes.NewReader(c.PNG))
		if err != nil {
			t.Fatal(err)
		}
		b := img.Bounds()
		if b.Dx() >= 80 || b.Dy() >= 80 {
			t.Fatalf("cell %d bbox 未收紧：%dx%d", i+1, b.Dx(), b.Dy())
		}
		// 角落应透明（洪泛抠底生效）
		r, g, bb, a := img.At(b.Min.X, b.Min.Y).RGBA()
		_ = r
		_ = g
		_ = bb
		if a != 0 {
			t.Fatalf("cell %d 角落未透明 alpha=%d", i+1, a)
		}
		// 中心应不透明（主体保留）
		_, _, _, ca := img.At(b.Min.X+b.Dx()/2, b.Min.Y+b.Dy()/2).RGBA()
		if ca != 0xffff {
			t.Fatalf("cell %d 中心 alpha 丢失=%d", i+1, ca)
		}
	}
}

func TestCropTileFullCell(t *testing.T) {
	spec := &GridSpec{Label: "maptiles", FileLabel: "tile", Cols: 2, Rows: 2, Alpha: false, Size: "64x64"}
	sheet := synthSheet(t, 2, 2, 32, 32)
	cells, err := CropSheet(sheet, spec)
	if err != nil {
		t.Fatal(err)
	}
	if len(cells) != 4 || cells[0].Empty {
		t.Fatalf("tile 应整格输出 %v", cells)
	}
	img, _ := png.Decode(bytes.NewReader(cells[0].PNG))
	if img.Bounds().Dx() != 32 || img.Bounds().Dy() != 32 {
		t.Fatalf("tile 应整格 32x32，got %v", img.Bounds())
	}
}

func TestSpecByLabel(t *testing.T) {
	if s := SpecByLabel("characters"); s == nil || s.Cols != 4 || s.Rows != 3 || s.Count != 12 {
		t.Fatalf("characters spec 异常: %+v", s)
	}
	if s := SpecByLabel("items"); s == nil || s.Count != 24 || !s.Alpha {
		t.Fatalf("items spec 异常: %+v", s)
	}
	if SpecByLabel("nope") != nil {
		t.Fatal("未知类型应返回 nil")
	}
}

func TestLoadSaveConfig(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/img.json"
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(cfg.Provider, "openai") {
		t.Fatalf("默认 provider=%s", cfg.Provider)
	}
	if cfg.Enabled() {
		t.Fatal("无密钥时不应 Enabled")
	}
	cfg.BaseURL = "http://127.0.0.1:1"
	cfg.APIKey = "sk-test-123456789"
	if err := SaveConfig(path, cfg); err != nil {
		t.Fatal(err)
	}
	cfg2, err := LoadConfig(path)
	if err != nil || !cfg2.Enabled() {
		t.Fatalf("落盘后应可用 err=%v", err)
	}
	masked := cfg2.MaskedKey()
	if strings.Contains(masked, "123456789") {
		t.Fatalf("掩码泄漏完整 key: %s", masked)
	}
	_ = os.Remove(path)
}

func TestGuessTheme(t *testing.T) {
	cases := map[string]string{
		"# 世界书：九州·凡尘仙途\n## A1 世界观\n修仙世界": "xianxia",
		"# 世界书：浮城·异能打工人\n## A1 世界观\n都市异能": "urban",
	}
	for raw, want := range cases {
		got := GuessTheme(worldbook.Parse(raw))
		if got != want {
			t.Errorf("%q → %s, want %s", raw, got, want)
		}
	}
}
