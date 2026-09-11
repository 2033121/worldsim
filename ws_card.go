package main

// ws_card.go — 世界卡（World Card）：把"一个世界"打包成单文件带走/分享（v1.9.0）。
//
// 内容：worldbook.md + art/plan.json + art/sheets/*.png + art/sprites/*.png
//      （可选 with_game=1 附 game.json 游玩进度）
// 不含：编年史/记忆/小说（隐私边界——那些是玩出来的，不是世界本身）。
//
// 端点：
//   GET  /api/world/card[?with_game=1]  → zip 下载（Content-Disposition）
//   POST /api/worlds/import             → multipart 上传 zip，落盘为新世界
//
// 全部 stdlib archive/zip，零依赖。

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// GET /api/world/card
func (ws *worldServer) handleWorldCardExport(w http.ResponseWriter, r *http.Request) {
	inst := ws.inst()
	if inst == nil {
		ws.writeJSON(w, 400, map[string]any{"ok": false, "error": "没有可用世界"})
		return
	}
	name := inst.name
	if name == "" {
		ws.writeJSON(w, 400, map[string]any{"ok": false, "error": "世界名未知"})
		return
	}
	withGame := r.URL.Query().Get("with_game") == "1"

	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)
	addFile := func(arcPath, diskPath string) bool {
		data, err := os.ReadFile(diskPath)
		if err != nil {
			return false
		}
		h := &zip.FileHeader{Name: arcPath, Method: zip.Deflate, Modified: time.Now()}
		f, err := zw.CreateHeader(h)
		if err != nil {
			return false
		}
		_, _ = f.Write(data)
		return true
	}
	count := 0
	// 世界书（worldbooks/ 池；自建世界书的副本可能在世界目录）
	wbCandidates := []string{
		filepath.Join(ws.baseDir, "..", "worldbooks", name+".md"),
		filepath.Join(inst.dir, "worldbook.md"),
	}
	for _, p := range wbCandidates {
		if addFile("worldbook.md", p) {
			count++
			break
		}
	}
	// 美术
	count += addTreeToZip(zw, filepath.Join(inst.dir, "art", "plan.json"), "art/plan.json")
	count += addTreeToZip(zw, filepath.Join(inst.dir, "art", "sheets"), "art/sheets")
	count += addTreeToZip(zw, filepath.Join(inst.dir, "art", "sprites"), "art/sprites")
	// 可选游玩进度
	if withGame {
		if addFile("game.json", filepath.Join(inst.dir, "game.json")) {
			count++
		}
	}
	// 元数据
	meta := fmt.Sprintf(`{"name":%q,"exported":%q,"worldsim":"v1.9.0"}`, name, time.Now().Format(time.RFC3339))
	if f, err := zw.Create("card.json"); err == nil {
		_, _ = f.Write([]byte(meta))
		count++
	}
	if count <= 1 { // 只有 card.json = 空卡
		ws.writeJSON(w, 400, map[string]any{"ok": false, "error": "世界卡内容为空（世界书与美术都不存在）"})
		return
	}
	_ = zw.Close()
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s.worldcard.zip", urlEscape(name)))
	_, _ = w.Write(buf.Bytes())
}

// addTreeToZip 把单个文件或整个目录追加进 zip，返回追加数量
func addTreeToZip(zw *zip.Writer, diskPath, arcPrefix string) int {
	st, err := os.Stat(diskPath)
	if err != nil {
		return 0
	}
	n := 0
	if !st.IsDir() {
		data, err := os.ReadFile(diskPath)
		if err == nil {
			if f, cerr := zw.Create(arcPrefix); cerr == nil {
				_, _ = f.Write(data)
				n++
			}
		}
		return n
	}
	_ = filepath.Walk(diskPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, rerr := filepath.Rel(diskPath, path)
		if rerr != nil {
			return nil
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil
		}
		if f, cerr := zw.Create(filepath.ToSlash(filepath.Join(arcPrefix, rel))); cerr == nil {
			_, _ = f.Write(data)
			n++
		}
		return nil
	})
	return n
}

// POST /api/worlds/import — multipart 字段 file=zip，可选 name=新世界名
func (ws *worldServer) handleWorldCardImport(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(200 << 20); err != nil {
		ws.writeJSON(w, 400, map[string]any{"ok": false, "error": "请用 multipart/form-data 上传 worldcard zip（字段 file）"})
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		ws.writeJSON(w, 400, map[string]any{"ok": false, "error": "缺少文件字段 file"})
		return
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		ws.writeJSON(w, 400, map[string]any{"ok": false, "error": "读取上传失败"})
		return
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		ws.writeJSON(w, 400, map[string]any{"ok": false, "error": "不是合法的 worldcard zip"})
		return
	}
	// 元数据：优先用 card.json 的世界名
	cardName := ""
	for _, f := range zr.File {
		if f.Name == "card.json" {
			rc, oerr := f.Open()
			if oerr == nil {
				raw, _ := io.ReadAll(rc)
				_ = rc.Close()
				var meta struct {
					Name string `json:"name"`
				}
				if err := json.Unmarshal(raw, &meta); err == nil {
					cardName = strings.TrimSpace(meta.Name)
				}
			}
		}
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		name = cardName
	}
	if name == "" {
		ws.writeJSON(w, 400, map[string]any{"ok": false, "error": "zip 里没有 card.json 世界名，请在表单里填 name"})
		return
	}
	name = sanitizeWorldName(name)
	// 重名自动 -2/-3…
	finalName := name
	for i := 2; ; i++ {
		if _, exists := ws.worlds[finalName]; !exists {
			if _, err := os.Stat(filepath.Join(ws.baseDir, finalName)); err != nil {
				break
			}
		}
		finalName = fmt.Sprintf("%s-%d", name, i)
	}
	worldDir := filepath.Join(ws.baseDir, finalName)
	if err := os.MkdirAll(worldDir, 0755); err != nil {
		ws.writeJSON(w, 500, map[string]any{"ok": false, "error": "创建世界目录失败: " + err.Error()})
		return
	}
	imported := 0
	for _, f := range zr.File {
		if f.Name == "card.json" {
			continue
		}
		// 防路径遍历
		clean := filepath.Clean(f.Name)
		if strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
			continue
		}
		target := filepath.Join(worldDir, clean)
		if strings.HasSuffix(clean, "worldbook.md") {
			// 世界书进全局池（worldbooks/<名>.md），供世界加载链路使用
			target = filepath.Join(ws.baseDir, "..", "worldbooks", finalName+".md")
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			continue
		}
		rc, oerr := f.Open()
		if oerr != nil {
			continue
		}
		out, werr := os.Create(target)
		if werr != nil {
			_ = rc.Close()
			continue
		}
		_, _ = io.Copy(out, rc)
		_ = out.Close()
		_ = rc.Close()
		imported++
	}
	if imported == 0 {
		ws.writeJSON(w, 400, map[string]any{"ok": false, "error": "zip 里没有可导入内容"})
		return
	}
	// 注册为新世界并选中
	if inst := ws.newInstance(finalName); inst.ready() {
		ws.worlds[finalName] = inst
		ws.current = finalName
	}
	ws.writeJSON(w, 200, map[string]any{"ok": true, "world": finalName, "files": imported})
}

// sanitizeWorldName 世界名清理（禁路径分隔符/保留字符）
func sanitizeWorldName(name string) string {
	name = strings.TrimSpace(name)
	repl := strings.NewReplacer("/", "／", "\\", "＼", ":", "：", "*", "＊", "?", "？", "\"", "＂", "<", "＜", ">", "＞", "|", "｜", "..", "…")
	return strings.TrimSpace(repl.Replace(name))
}

// urlEscape Content-Disposition 用的最小转义
func urlEscape(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}
