// Package attach 提供"世界参考资料"附件管理：上传/列表/删除 + 文本提取 + 聚合注入。
// 附件是用户上传给某个世界的设定/事实补充（txt/md/json/csv 等纯文本），
// 会被聚合为"世界参考资料"注入到世界/GM/事件 Agent 的 LLM 上下文，让剧情严格遵循用户提供的设定。
// 设计保持零外部依赖（与项目单二进制风格一致）：仅直接支持纯文本格式提取；
// pdf/docx 等二进制格式仅保存文件（extractable=false），供日后扩展或外部工具转换。
package attach

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Attachment 一条已上传的世界参考资料。
type Attachment struct {
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	Ext         string `json:"ext"`
	Extractable bool   `json:"extractable"` // 是否成功提取出可直接给 LLM 读的文本
	Text        string `json:"text,omitempty"` // 提取的文本（上传/详情时返回；列表时不返回全文）
	Uploaded    string `json:"uploaded"`    // 上传时间
}

// extractableExts 支持直接提取文本的扩展名（纯 UTF-8 文本，零依赖）。
var extractableExts = map[string]bool{
	".txt": true, ".md": true, ".json": true, ".csv": true,
	".xml": true, ".html": true, ".htm": true, ".log": true,
	".yaml": true, ".yml": true, ".ini": true, ".toml": true,
}

// MaxBytes 单文件大小上限（8MB）。
const MaxBytes = 8 << 20

// MaxAggregateBytes 聚合注入的文本总量上限（约 16K 字符，控制上下文）。
const MaxAggregateBytes = 16000

// Store 管理某世界的附件目录（worlds/{世界名}/attachments/）。
type Store struct {
	dir string
}

// NewStore 创建附件存储（目录不存在则创建）。
func NewStore(dir string) *Store {
	os.MkdirAll(dir, 0755)
	return &Store{dir: dir}
}

// extractText 从文件内容提取纯文本；二进制/不支持格式或空文本返回 ("", false)。
func extractText(name string, data []byte) (string, bool) {
	ext := strings.ToLower(filepath.Ext(name))
	if !extractableExts[ext] {
		return "", false
	}
	// 去除 UTF-8 BOM
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		data = data[3:]
	}
	s := strings.TrimSpace(string(data))
	if s == "" {
		return "", false
	}
	return s, true
}

// Save 保存一个附件，返回其元数据（含提取文本）。
// 文件名会被 filepath.Base 归一化以防御路径穿越。
func (s *Store) Save(filename string, data []byte) (*Attachment, error) {
	name := filepath.Base(strings.ReplaceAll(filename, "\\", "/"))
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "/\\") {
		return nil, fmt.Errorf("非法文件名")
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("空文件")
	}
	if len(data) > MaxBytes {
		return nil, fmt.Errorf("文件过大（上限 8MB）")
	}
	if err := os.WriteFile(filepath.Join(s.dir, name), data, 0644); err != nil {
		return nil, err
	}
	text, ok := extractText(name, data)
	return &Attachment{
		Name: name, Size: int64(len(data)), Ext: filepath.Ext(name),
		Extractable: ok, Text: text, Uploaded: time.Now().Format("2006-01-02 15:04:05"),
	}, nil
}

// List 返回附件元数据列表（不含全文 text，避免接口过大；按文件名排序）。
func (s *Store) List() []Attachment {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return []Attachment{}
	}
	out := []Attachment{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		att := Attachment{
			Name: e.Name(), Size: info.Size(), Ext: filepath.Ext(e.Name()),
			Extractable: extractableExts[strings.ToLower(filepath.Ext(e.Name()))],
			Uploaded:    info.ModTime().Format("2006-01-02 15:04:05"),
		}
		out = append(out, att)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Delete 删除一个附件（文件名归一化防御穿越）。
func (s *Store) Delete(name string) error {
	name = filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "/\\") {
		return fmt.Errorf("非法文件名")
	}
	return os.Remove(filepath.Join(s.dir, name))
}

// Aggregate 聚合所有可提取附件文本为"世界参考资料"字符串（注入 LLM 上下文用）。
// 按文件名排序，总量受 MaxAggregateBytes 限制；无可提取内容返回空串。
func (s *Store) Aggregate() string {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return ""
	}
	var b strings.Builder
	total := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if !extractableExts[ext] {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.dir, e.Name()))
		if err != nil {
			continue
		}
		text, ok := extractText(e.Name(), data)
		if !ok {
			continue
		}
		if total+len(text) > MaxAggregateBytes {
			text = text[:MaxAggregateBytes-total]
		}
		b.WriteString("◆ 附件《" + e.Name() + "》：\n" + text + "\n")
		total += len(text) + len(e.Name())
		if total >= MaxAggregateBytes {
			break
		}
	}
	return strings.TrimRight(b.String(), "\n")
}