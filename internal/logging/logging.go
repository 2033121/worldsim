// Package logging provides a complete, persistent structured logging system
// for WorldSim. Every log entry is:
//   - written to stdout (mirrors previous behavior, keeps `docker logs` working)
//   - appended to a daily-rotated JSONL file under <progDir>/logs/
//   - retained in an in-memory ring buffer for fast querying via the HTTP API
//
// Levels: debug < info < warn < error. Categories categorize entries (llm,
// sim, event, character, loop, server, init, ...) so problems can be filtered.
package logging

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Levels (ordered by severity).
const (
	LevelDebug = "debug"
	LevelInfo  = "info"
	LevelWarn  = "warn"
	LevelError = "error"
)

// Reusable category constants (callers are free to use their own strings too).
const (
	CatLLM         = "llm"          // LLM 调用（成功/失败/超时）
	CatSim         = "sim"          // 模拟主流程
	CatEvent       = "event"        // 事件生成
	CatChar        = "character"    // 角色注册 / 生命周期
	CatLoop        = "loop"         // 后台模拟循环
	CatInit        = "init"         // 世界初始化
	CatServer      = "server"       // HTTP 服务 / 世界操作
	CatRewind      = "rewind"       // 回档 / 快照
	CatNovel       = "novel"        // 小说写手
	CatResearch    = "research"     // 题材研究
	CatWorldCreate = "world_create" // 世界创建 / 世界书生成
)

// Entry 单条结构化日志。
type Entry struct {
	TS     string         `json:"ts"`               // 2006-01-02 15:04:05
	Level  string         `json:"level"`            // debug|info|warn|error
	Cat    string         `json:"cat"`              // 分类
	World  string         `json:"world,omitempty"`  // 所属世界
	Msg    string         `json:"msg"`              // 人类可读消息
	Fields map[string]any `json:"fields,omitempty"` // 附加结构化字段
}

// Store 日志存储：每日轮转 JSONL 文件 + 内存环形缓冲 + stdout 镜像。
type Store struct {
	mu       sync.Mutex
	dir      string   // 落盘目录（<progDir>/logs）
	entries  []*Entry // 环形缓冲（最新在前）
	capacity int      // 缓冲上限
	file     *os.File // 当前天文件句柄
	curDay   string   // 当前文件日期键（YYYY-MM-DD）
	stdout   bool     // 是否镜像到 stdout
}

// std 全局默认日志存储（单例）。
var std = &Store{capacity: 50000, stdout: true}

// Init 初始化全局日志存储并加载最近的既有日志（跨重启保留上下文）。
// dir 为空则不落盘（仅内存 + stdout），便于测试。
func Init(dir string) error {
	std.mu.Lock()
	defer std.mu.Unlock()
	if strings.TrimSpace(dir) != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		std.dir = dir
	}
	std.loadRecentLocked()
	return nil
}

// SetStdout 控制是否镜像到 stdout（默认开启）。
func SetStdout(on bool) {
	std.mu.Lock()
	std.stdout = on
	std.mu.Unlock()
}

// ringKey 内存缓冲的键（降序：最新在前）。
func (s *Store) ringKey(t time.Time) string {
	return t.Format("2006-01-02 15:04:05.000")
}

// loadRecentLocked 启动时把最近的日志文件加载进缓冲（已持锁）。
func (s *Store) loadRecentLocked() {
	if s.dir == "" {
		return
	}
	matches, err := filepath.Glob(filepath.Join(s.dir, "*.jsonl"))
	if err != nil || len(matches) == 0 {
		return
	}
	sort.Strings(matches)
	// 从最新文件倒序加载，直到填满 bufffer 或处理完所有文件。
	loaded := 0
	for i := len(matches) - 1; i >= 0 && loaded < s.capacity; i-- {
		data, err := os.ReadFile(matches[i])
		if err != nil {
			continue
		}
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		// 跳过最后一行（可能是不完整的进行中写入）
		for j := len(lines) - 1; j >= 0 && loaded < s.capacity; j-- {
			line := strings.TrimSpace(lines[j])
			if line == "" {
				continue
			}
			var e Entry
			if json.Unmarshal([]byte(line), &e) != nil {
				continue
			}
			if e.TS == "" {
				continue
			}
			s.entries = append(s.entries, &e)
			loaded++
		}
	}
}

// rotateLocked 按需轮转当前天文件（已持锁）。
func (s *Store) rotateLocked(now time.Time) {
	if s.dir == "" {
		return
	}
	day := now.Format("2006-01-02")
	if s.file != nil && s.curDay == day {
		return
	}
	if s.file != nil {
		s.file.Close()
		s.file = nil
	}
	f, err := os.OpenFile(filepath.Join(s.dir, day+".jsonl"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	s.file = f
	s.curDay = day
}

// Write 记录一条日志。level/cat 为空时给默认值。
func Write(level, cat, world, msg string, fields map[string]any) {
	std.write(time.Now(), level, cat, world, msg, fields)
}

// WriteAt 带显式时间戳写入（用于回放历史等场景）。
func WriteAt(t time.Time, level, cat, world, msg string, fields map[string]any) {
	std.write(t, level, cat, world, msg, fields)
}

func (s *Store) write(now time.Time, level, cat, world, msg string, fields map[string]any) {
	if level == "" {
		level = LevelInfo
	}
	if cat == "" {
		cat = "general"
	}
	if msg == "" {
		msg = "(空消息)"
	}
	e := &Entry{
		TS:     now.Format("2006-01-02 15:04:05"),
		Level:  level,
		Cat:    cat,
		World:  world,
		Msg:    msg,
		Fields: fields,
	}

	s.mu.Lock()
	// 内存环形缓冲（最新在前）
	s.entries = append([]*Entry{e}, s.entries...)
	if len(s.entries) > s.capacity {
		s.entries = s.entries[:s.capacity]
	}
	// 按天轮转 + 追加落盘（JSONL）
	s.rotateLocked(now)
	if s.file != nil {
		if data, err := json.Marshal(e); err == nil {
			s.file.Write(append(data, '\n'))
		}
	}
	stdout := s.stdout
	s.mu.Unlock()

	if stdout {
		fmt.Printf(" [%s][%s] %s%s\n", level, cat, worldPrefix(world), msg)
	}
}

func worldPrefix(w string) string {
	if w == "" {
		return ""
	}
	return "[" + w + "] "
}

// ---------- 便捷分级写入 ----------

func Debug(cat, msg string, fields map[string]any) { Write(LevelDebug, cat, "", msg, fields) }
func Info(cat, msg string, fields map[string]any)  { Write(LevelInfo, cat, "", msg, fields) }
func Warn(cat, msg string, fields map[string]any)  { Write(LevelWarn, cat, "", msg, fields) }
func Error(cat, msg string, fields map[string]any) { Write(LevelError, cat, "", msg, fields) }

// 带世界归属的便捷写入。
func DebugW(world, cat, msg string, fields map[string]any) {
	Write(LevelDebug, cat, world, msg, fields)
}
func InfoW(world, cat, msg string, fields map[string]any) { Write(LevelInfo, cat, world, msg, fields) }
func WarnW(world, cat, msg string, fields map[string]any) { Write(LevelWarn, cat, world, msg, fields) }
func ErrorW(world, cat, msg string, fields map[string]any) {
	Write(LevelError, cat, world, msg, fields)
}

// ---------- 查询 ----------

// Query 过滤条件。
type Query struct {
	Level string // 精确级别；空=全部
	Cat   string // 精确分类；空=全部
	World string // 精确世界；空=全部
	Kw    string // 关键词子串匹配（msg + fields 序列化）；空=不过滤
	Max   int    // 返回条数上限
}

// Query 返回满足条件的日志（最新在前）。
func List(q Query) []Entry {
	std.mu.Lock()
	defer std.mu.Unlock()
	if q.Max <= 0 {
		q.Max = 200
	}
	out := make([]Entry, 0, q.Max)
	for _, e := range std.entries {
		if len(out) >= q.Max {
			break
		}
		if !match(e, q) {
			continue
		}
		out = append(out, *e)
	}
	return out
}

func match(e *Entry, q Query) bool {
	if q.Level != "" && e.Level != q.Level {
		return false
	}
	if q.Cat != "" && e.Cat != q.Cat {
		return false
	}
	if q.World != "" && e.World != q.World {
		return false
	}
	if q.Kw != "" {
		if strings.Contains(e.Msg, q.Kw) {
			return true
		}
		if e.Fields != nil {
			if b, err := json.Marshal(e.Fields); err == nil && strings.Contains(string(b), q.Kw) {
				return true
			}
		}
		return false
	}
	return true
}

// Stats 按级别/分类聚合计数（用于前端筛选面板）。
type Stats struct {
	Total    int            `json:"total"`
	ByLevel  map[string]int `json:"by_level"`
	ByCat    map[string]int `json:"by_cat"`
	Errors   int            `json:"errors"`
	Warnings int            `json:"warnings"`
}

func Counts() Stats {
	std.mu.Lock()
	defer std.mu.Unlock()
	st := Stats{ByLevel: map[string]int{}, ByCat: map[string]int{}}
	for _, e := range std.entries {
		st.Total++
		st.ByLevel[e.Level]++
		st.ByCat[e.Cat]++
		if e.Level == LevelError {
			st.Errors++
		}
		if e.Level == LevelWarn {
			st.Warnings++
		}
	}
	return st
}

// Close 关闭全局日志存储（落盘句柄）。
func Close() {
	std.mu.Lock()
	defer std.mu.Unlock()
	if std.file != nil {
		std.file.Close()
		std.file = nil
	}
}
