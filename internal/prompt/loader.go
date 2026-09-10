// Package prompt 提供 Agent 提示词的外置化加载：提示词作为"数据"而非"代码"。
//
// 所有 Agent 的 system prompt 从 Go 源码迁到物理 Markdown 文件：
//   - 外部覆盖：优先读 <dataDir>/<name>.md（免编译热调优）
//   - 内置回退：go:embed 的 builtin/<name>.md（随二进制分发）
//
// 支持 {{key}} 占位符插值，未提供的 key 替换为空串且不报错。
// 参考 inkOS 的 PromptLoader 设计：提示词与代码解耦，可复用、可版本化、可热调优。
package prompt

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

//go:embed builtin/*.md
var builtinFS embed.FS

var placeholderRe = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_]+)\s*\}\}`)

// Loader 加载并渲染 Agent 提示词。
type Loader struct {
	dataDir string // 外部提示词目录（wsdata/prompts），可为空
}

// New 创建提示词加载器。dataDir 为外部覆盖目录（可为空，仅走内置）。
func New(dataDir string) *Loader {
	return &Loader{dataDir: dataDir}
}

// Render 渲染 name 对应的提示词，用 data 填充 {{key}} 占位符。
// 未提供某 key 时替换为空串（不报错）；提示词在外部与内置都不存在时返回错误。
func (l *Loader) Render(name string, data map[string]string) (string, error) {
	raw, err := l.raw(name)
	if err != nil {
		return "", err
	}
	return interpolate(raw, data), nil
}

// raw 读取提示词原文：优先外部，缺失回退内置。
func (l *Loader) raw(name string) (string, error) {
	if l != nil && l.dataDir != "" {
		p := filepath.Join(l.dataDir, name+".md")
		if b, err := os.ReadFile(p); err == nil {
			return string(b), nil
		}
	}
	b, err := builtinFS.ReadFile("builtin/" + name + ".md")
	if err != nil {
		return "", fmt.Errorf("提示词 %q 不存在（外部与内置均未找到）", name)
	}
	return string(b), nil
}

// interpolate 做 {{key}} 占位符替换。
func interpolate(raw string, data map[string]string) string {
	return placeholderRe.ReplaceAllStringFunc(raw, func(m string) string {
		sub := placeholderRe.FindStringSubmatch(m)
		if len(sub) < 2 {
			return ""
		}
		key := strings.TrimSpace(sub[1])
		if data == nil {
			return ""
		}
		return data[key]
	})
}
