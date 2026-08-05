package llm

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"worldsim/internal/fsutil"
)

// ---------- 全局 Token 消耗历史（按时间窗 hour/day 聚合 + 原子落盘） ----------
// 与内存态 SpanTracker 互补：SpanTracker 是"当前进程内"的实时统计，
// 这里把每次调用增量计入当前小时窗/当天窗并持久化到磁盘，供跨重启查看趋势。

// Record 单条时间窗聚合记录。
type Record struct {
	TS         string `json:"ts"`         // 窗口键（hour: 2006-01-02T15；day: 2006-01-02）
	Window     string `json:"window"`     // hour | day
	Prompt     int    `json:"prompt"`     // 输入 token
	Completion int    `json:"completion"` // 输出 token
	Cached     int    `json:"cached"`     // 前缀缓存命中 token
	Calls      int    `json:"calls"`      // 调用次数（含失败）
	Failures   int    `json:"failures"`   // 失败/超时次数
}

// Total 便捷合计（输入+输出）。
func (r *Record) Total() int {
	if r == nil {
		return 0
	}
	return r.Prompt + r.Completion
}

// StatsStore 按时间窗持久化 token 消耗（全局单例，mutex 保护）。
type StatsStore struct {
	mu   sync.Mutex
	dir  string // 落盘目录（如 <progDir>/llm_stats）
	hour map[string]*Record
	day  map[string]*Record
}

var statsStore = &StatsStore{hour: map[string]*Record{}, day: map[string]*Record{}}

// InitStatsStore 设置落盘目录并从磁盘加载既有历史（启动时调用一次）。
// dir 为空则仅内存累计、不落盘（便于无数据目录环境的测试）。
func InitStatsStore(dir string) error {
	statsStore.mu.Lock()
	defer statsStore.mu.Unlock()
	if dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	statsStore.dir = dir
	statsStore.hour = loadWindowFile(filepath.Join(dir, "hour.json"))
	statsStore.day = loadWindowFile(filepath.Join(dir, "day.json"))
	return nil
}

// loadWindowFile 读取窗口 JSON 文件并重建为 map（文件缺失/损坏则返回空）。
func loadWindowFile(path string) map[string]*Record {
	out := map[string]*Record{}
	data, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	var list []*Record
	if json.Unmarshal(data, &list) != nil {
		return out
	}
	for _, r := range list {
		if r != nil && r.TS != "" {
			out[r.TS] = r
		}
	}
	return out
}

// filePath 返回指定窗口的落盘文件路径。
func (s *StatsStore) filePath(window string) string {
	return filepath.Join(s.dir, window+".json")
}

// RecordWindow 把一次调用的用量增量计入当前小时窗与当天窗，并原子落盘。
// 仅在成功调用时 prompt/completion/cached 非零；失败时 calls=1、failures=1。
func RecordWindow(now time.Time, prompt, completion, cached, calls, failures int) {
	statsStore.mu.Lock()
	defer statsStore.mu.Unlock()
	if statsStore.dir == "" {
		return // 未初始化：仅内存累计（当前无意义），跳过落盘
	}
	hourKey := now.Format("2006-01-02T15")
	dayKey := now.Format("2006-01-02")
	statsStore.add(hourKey, "hour", prompt, completion, cached, calls, failures)
	statsStore.add(dayKey, "day", prompt, completion, cached, calls, failures)
	_ = statsStore.persist("hour")
	_ = statsStore.persist("day")
}

func (s *StatsStore) add(ts, window string, prompt, completion, cached, calls, failures int) {
	m := s.hour
	if window == "day" {
		m = s.day
	}
	r := m[ts]
	if r == nil {
		r = &Record{TS: ts, Window: window}
		m[ts] = r
	}
	r.Prompt += prompt
	r.Completion += completion
	r.Cached += cached
	r.Calls += calls
	r.Failures += failures
}

// persist 把指定窗口的内存 map 原子写盘（按 TS 升序）。
func (s *StatsStore) persist(window string) error {
	m := s.hour
	if window == "day" {
		m = s.day
	}
	list := make([]*Record, 0, len(m))
	for _, r := range m {
		list = append(list, r)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].TS < list[j].TS })
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(s.filePath(window), data)
}

// LoadHistory 返回指定窗口（hour|day）的历史记录（按时间升序）。
// 优先返回内存态（与磁盘始终保持一致），未初始化/无数据时返回空切片。
func LoadHistory(window string) []Record {
	statsStore.mu.Lock()
	defer statsStore.mu.Unlock()
	m := statsStore.hour
	if window == "day" {
		m = statsStore.day
	}
	list := make([]Record, 0, len(m))
	for _, r := range m {
		list = append(list, *r)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].TS < list[j].TS })
	return list
}
