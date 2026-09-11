package art

// crop.go — Go 原生裁剪管线（stdlib image/png，与 scripts/crop_grid.py 同构）。
//
// 三步：
//  1. 均匀网格切分 sheet → cells
//  2. 每格求非背景紧致 bbox（dr>60，PAD=8 外扩）
//  3. 边缘 BFS 洪泛抠底转透明（tol=78）——只吃与边界连通的背景色，
//     不伤主体内部同色（实战坑：全局阈值扫描会留下灰框）
//
// tile 类（Alpha=false）不抠底，整格输出（tileset 本来就要满格）。
// 严格 RGBA convert 后再写 alpha（PIL RGB 模式写 alpha 是静默无操作的同款坑）。

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
)

const (
	bboxDrThreshold = 60 // 非背景判定：|Δr|+|Δg|+|Δb| > 60
	bboxPad         = 8
	alphaTolerance  = 78 // 洪泛背景判定阈值
	// emptyRatioThreshold：格内前景占比低于此判"空格"（触发单品自动补齐）
	emptyRatioThreshold = 0.02
)

// GridSpec sheet 网格规格（Go 版 GRIDS manifest）
type GridSpec struct {
	Label     string // sheet 文件名：characters.png
	FileLabel string // sprite 命名前缀：character-1.png
	Cols      int
	Rows      int
	Alpha     bool   // 是否抠底（tiles=false）
	Size      string // 生成尺寸 "1536x1024"
	Count     int    // 期望素材数（cols*rows）
}

// SheetSpecs 五类资产规格表（与 v1.7.0 打包套件同口径）
func SheetSpecs() []GridSpec {
	return []GridSpec{
		{Label: "characters", FileLabel: "character", Cols: 4, Rows: 3, Alpha: true, Size: "1536x1024", Count: 12},
		{Label: "monsters", FileLabel: "monster", Cols: 4, Rows: 2, Alpha: true, Size: "1536x1024", Count: 8},
		{Label: "scenes", FileLabel: "scene", Cols: 3, Rows: 1, Alpha: true, Size: "1792x1024", Count: 3},
		{Label: "maptiles", FileLabel: "tile", Cols: 4, Rows: 4, Alpha: false, Size: "1024x1024", Count: 16},
		{Label: "items", FileLabel: "item", Cols: 6, Rows: 4, Alpha: true, Size: "1536x1024", Count: 24},
	}
}

// SpecByLabel 按资产类型名取规格
func SpecByLabel(label string) *GridSpec {
	for i := range SheetSpecs() {
		if SheetSpecs()[i].Label == label {
			return &SheetSpecs()[i]
		}
	}
	return nil
}

// SpriteName sprite 文件名（character-5.png）
func (s *GridSpec) SpriteName(index int) string { // index 从 1 起
	return fmt.Sprintf("%s-%d.png", s.FileLabel, index)
}

// CellResult 单格裁剪结果
type CellResult struct {
	PNG        []byte
	Foreground float64 // 前景像素占比（空格校验用）
	Empty      bool
}

// CropSheet 把一张 sheet PNG 切分裁剪为逐格 sprite PNG
func CropSheet(sheetPNG []byte, spec *GridSpec) ([]CellResult, error) {
	src, err := png.Decode(bytes.NewReader(sheetPNG))
	if err != nil {
		return nil, fmt.Errorf("sheet 解码失败: %w", err)
	}
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("sheet 尺寸异常 %dx%d", w, h)
	}
	cw, ch := w/spec.Cols, h/spec.Rows
	if cw <= 0 || ch <= 0 {
		return nil, fmt.Errorf("sheet 网格切分异常（%dx%d / %dx%d）", w, h, spec.Cols, spec.Rows)
	}

	results := make([]CellResult, 0, spec.Cols*spec.Rows)
	for row := 0; row < spec.Rows; row++ {
		for col := 0; col < spec.Cols; col++ {
			cell := cropRect(src, image.Rect(col*cw, row*ch, (col+1)*cw, (row+1)*ch))
			res := CellResult{}
			if spec.Alpha {
				bg := cellBackgroundColor(cell)
				bb := bboxOfNonBg(cell, bg)
				if bb == (image.Rectangle{}) {
					results = append(results, CellResult{Empty: true})
					continue
				}
				fg := cropRect(cell, bb)
				fg = floodFillAlpha(fg, bg)
				res.Foreground = fgRatio(fg, bg)
				res.Empty = res.Foreground < emptyRatioThreshold
				if res.Empty {
					continue // 保留判定但不输出废图
				}
				res.PNG = encodePNG(fg)
			} else {
				res.PNG = encodePNG(cell)
				res.Foreground = 1.0
			}
			results = append(results, res)
		}
	}
	return results, nil
}

// CropSingle 单张独立生成图（1024x1024 单 sprite）→ 紧致抠底 PNG；返回 nil 表示空图
func CropSingle(spritePNG []byte) ([]byte, bool, error) {
	src, err := png.Decode(bytes.NewReader(spritePNG))
	if err != nil {
		return nil, false, fmt.Errorf("sprite 解码失败: %w", err)
	}
	// 已带透明底（pixellab noBackground / 用户上传）：跳过抠底只做 bbox
	if hasTransparency(src) {
		bb := bboxOfAlphaContent(toRGBA(src))
		if bb == (image.Rectangle{}) {
			return nil, true, nil
		}
		out := cropRect(src, bb)
		return encodePNG(out), false, nil
	}
	rgbaSrc := toRGBA(src)
	bg := cellBackgroundColor(rgbaSrc)
	bb := bboxOfNonBg(rgbaSrc, bg)
	if bb == (image.Rectangle{}) {
		return nil, true, nil
	}
	out := cropRect(src, bb)
	out = floodFillAlpha(out, bg)
	ratio := fgRatio(out, bg)
	if ratio < emptyRatioThreshold {
		return nil, true, nil
	}
	return encodePNG(out), false, nil
}

// ---------- 内部实现 ----------

// cropRect 从图像中截取矩形（转 RGBA 拷贝）
func cropRect(src image.Image, rect image.Rectangle) *image.RGBA {
	rect = rect.Intersect(src.Bounds())
	w, h := rect.Dx(), rect.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dst.Set(x, y, src.At(rect.Min.X+x, rect.Min.Y+y))
		}
	}
	return dst
}

// cellBackgroundColor 取四角平均色作为背景色（生成契约固定 #e8e8e8，但抗渐变）
func cellBackgroundColor(img image.Image) color.RGBA {
	b := img.Bounds()
	corners := [4][2]int{{b.Min.X, b.Min.Y}, {b.Max.X - 1, b.Min.Y}, {b.Min.X, b.Max.Y - 1}, {b.Max.X - 1, b.Max.Y - 1}}
	var r, g, bl, a int
	for _, c := range corners {
		cr, cg, cb, ca := img.At(c[0], c[1]).RGBA()
		r += int(cr >> 8)
		g += int(cg >> 8)
		bl += int(cb >> 8)
		a += int(ca >> 8)
	}
	return color.RGBA{uint8(r / 4), uint8(g / 4), uint8(bl / 4), uint8(a / 4)}
}

func dr(p, bg color.RGBA) int {
	return abs(int(p.R)-int(bg.R)) + abs(int(p.G)-int(bg.G)) + abs(int(p.B)-int(bg.B))
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// bboxOfNonBg 全格扫描非背景像素的紧致 bbox（外扩 PAD）
func bboxOfNonBg(img *image.RGBA, bg color.RGBA) image.Rectangle {
	b := img.Bounds()
	minx, miny, maxx, maxy := b.Max.X, b.Max.Y, b.Min.X-1, b.Min.Y-1
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			p := img.RGBAAt(x, y)
			if dr(p, bg) > bboxDrThreshold {
				if x < minx {
					minx = x
				}
				if x > maxx {
					maxx = x
				}
				if y < miny {
					miny = y
				}
				if y > maxy {
					maxy = y
				}
			}
		}
	}
	if maxx < minx {
		return image.Rectangle{}
	}
	return image.Rect(max(b.Min.X, minx-bboxPad), max(b.Min.Y, miny-bboxPad),
		min(b.Max.X, maxx+bboxPad+1), min(b.Max.Y, maxy+bboxPad+1))
}

// bboxOfAlphaContent 已透明图像的内容 bbox
func bboxOfAlphaContent(img *image.RGBA) image.Rectangle {
	b := img.Bounds()
	minx, miny, maxx, maxy := b.Max.X, b.Max.Y, b.Min.X-1, b.Min.Y-1
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if img.RGBAAt(x, y).A > 16 {
				if x < minx {
					minx = x
				}
				if x > maxx {
					maxx = x
				}
				if y < miny {
					miny = y
				}
				if y > maxy {
					maxy = y
				}
			}
		}
	}
	if maxx < minx {
		return image.Rectangle{}
	}
	return image.Rect(max(b.Min.X, minx-bboxPad), max(b.Min.Y, miny-bboxPad),
		min(b.Max.X, maxx+bboxPad+1), min(b.Max.Y, maxy+bboxPad+1))
}

// floodFillAlpha 边缘 BFS 洪泛：只把与图像边界连通的背景色置为透明
func floodFillAlpha(img *image.RGBA, bg color.RGBA) *image.RGBA {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w == 0 || h == 0 {
		return img
	}
	seen := make([]bool, w*h)
	type pt struct{ x, y int }
	queue := make([]pt, 0, w*h/4)
	push := func(x, y int) {
		idx := y*w + x
		if seen[idx] {
			return
		}
		p := img.RGBAAt(b.Min.X+x, b.Min.Y+y)
		if dr(p, bg) < alphaTolerance {
			seen[idx] = true
			queue = append(queue, pt{x, y})
		}
	}
	for x := 0; x < w; x++ {
		push(x, 0)
		push(x, h-1)
	}
	for y := 0; y < h; y++ {
		push(0, y)
		push(w-1, y)
	}
	for len(queue) > 0 {
		cur := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		img.SetRGBA(b.Min.X+cur.x, b.Min.Y+cur.y, color.RGBA{0, 0, 0, 0})
		if cur.x > 0 {
			push(cur.x-1, cur.y)
		}
		if cur.x < w-1 {
			push(cur.x+1, cur.y)
		}
		if cur.y > 0 {
			push(cur.x, cur.y-1)
		}
		if cur.y < h-1 {
			push(cur.x, cur.y+1)
		}
	}
	return img
}

// fgRatio 前景像素占比（洪泛后：非零 alpha 即前景）
func fgRatio(img *image.RGBA, bg color.RGBA) float64 {
	b := img.Bounds()
	total := b.Dx() * b.Dy()
	if total == 0 {
		return 0
	}
	fg := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if img.RGBAAt(x, y).A > 16 {
				fg++
			}
		}
	}
	return float64(fg) / float64(total)
}

// hasTransparency 图像是否已带透明通道内容
func hasTransparency(img image.Image) bool {
	b := img.Bounds()
	step := 7 // 抽样即可
	n := 0
	for y := b.Min.Y; y < b.Max.Y; y += step {
		for x := b.Min.X; x < b.Max.X; x += step {
			if _, _, _, a := img.At(x, y).RGBA(); a < 4000 { // a<~0.06
				return true
			}
			n++
			if n > 4096 {
				return false
			}
		}
	}
	return false
}

// toRGBA 任意图像转 RGBA 拷贝
func toRGBA(src image.Image) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			dst.Set(x, y, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}

func encodePNG(img image.Image) []byte {
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
