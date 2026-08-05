package selfheal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Manager 是 WorldSim 的内嵌"持续监测 + 自动修复"模块。
//
// 职责：
//  1. 日志落盘：把运行过程/错误日志写入 selfheal/runtime.log（同时保留 stdout）。
//  2. 周期性监测四类异常：模拟异常 / 服务进程 / 数据一致性 / LLM-API 可用性。
//  3. 检测到异常 → 自动诊断根源 → 触发修复回调（healer）→ 记录 Incident。
//  4. 暴露 /api/selfheal/status 与 /api/selfheal/incidents 供前端监测面板展示。
//
// Manager 与运行中的服务解耦：所有需要触达运行态（停循环/重建实例/回退快照）的修复，
// 由外部（main.go）通过 RegisterHealer 注入回调完成，避免包间循环依赖。
type Manager struct {
	baseDir      string // 数据根目录（wsdata）
	apiCfgPath   string // api.json 路径
	logPath      string // selfheal/runtime.log
	incidentPath string // selfheal/incidents.jsonl

	mu        sync.Mutex
	checks    map[string]HealthCheck
	incidents []Incident
	startTime time.Time
	lastTick  time.Time
	repairs   int

	// 注入的修复回调（由 main.go 注册，用于触达运行态）
	healers map[string]HealFunc

	// 注入的运行时状态读取（由 main.go 注册）
	loopStateFn func() LoopState

	// 日志 writer（幂等创建）
	logFile *os.File
}

// HealFunc 一个修复动作：返回是否修复成功 + 描述。
type HealFunc func() (string, error)

// LoopState 世界模拟循环的运行时快照（由 main.go handleLoopSet 附近注册读取）。
type LoopState struct {
	Running     bool   `json:"running"`
	World       string `json:"world"`
	Day         int    `json:"day"`
	TargetDay   int    `json:"target_day"`
	LastErr     string `json:"last_err"`
	ConsecFail  int    `json:"consec_fail"` // 连续 RunDay 失败次数
	LLMRunning  bool   `json:"llm_running"` // 是否启用了 LLM（false=仅 dry-run）
}

// New 创建自愈 Manager，并初始化 selfheal 目录与日志文件。
// baseDir：数据根目录；apiCfgPath：api.json 完整路径。
func New(baseDir, apiCfgPath string) (*Manager, error) {
	dir := filepath.Join(baseDir, "selfheal")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	m := &Manager{
		baseDir:      baseDir,
		apiCfgPath:   apiCfgPath,
		logPath:      filepath.Join(dir, "runtime.log"),
		incidentPath: filepath.Join(dir, "incidents.jsonl"),
		checks:       make(map[string]HealthCheck),
		healers:      make(map[string]HealFunc),
		startTime:    time.Now(),
	}
	// 打开日志文件（追加）
	if f, err := os.OpenFile(m.logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		m.logFile = f
	}
	// 恢复历史 incidents
	m.loadIncidents()
	m.Record("info", fmt.Sprintf("自愈模块已启动（数据目录 %s）", baseDir))
	return m, nil
}

func (m *Manager) Close() {
	if m.logFile != nil {
		_ = m.logFile.Close()
	}
}

// Record 记录一条运行日志：写日志文件 + 控制台。
func (m *Manager) Record(level, msg string) {
	line := fmt.Sprintf("[%s] %s\n", strings.ToUpper(level), msg)
	m.mu.Lock()
	if m.logFile != nil {
		_, _ = m.logFile.WriteString(line)
	}
	m.mu.Unlock()
	if level == "error" || level == "warn" {
		fmt.Printf(" [自愈:%s] %s\n", level, msg)
	}
}

// RegisterHealer 注册修复回调（key 为修复类型，如 "restart_loop" / "rewind" / "llm_config"）。
func (m *Manager) RegisterHealer(key string, fn HealFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.healers[key] = fn
}

// SetLoopStateSource 注册模拟循环状态读取函数（卡死/失败检测用）。
func (m *Manager) SetLoopStateSource(fn func() LoopState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.loopStateFn = fn
}

// Heartbeat 由模拟循环每次 RunDay 后调用，上报运行过程（供日志与卡死检测）。
func (m *Manager) Heartbeat(level, msg string) {
	m.Record(level, msg)
}

// ------ Incident 记录与持久化 ------

func (m *Manager) newIncident(category, severity, detail, diagnosis, action string, fixed bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	status := "resolved"
	if !fixed {
		status = "manual" // 无法自动修复，需用户介入
	}
	inc := Incident{
		ID:        fmt.Sprintf("inc-%d", time.Now().UnixNano()),
		Time:      time.Now().Format("2006-01-02 15:04:05"),
		Category:  category,
		Severity:  severity,
		Detail:    detail,
		Diagnosis: diagnosis,
		Action:    action,
		AutoFixed: fixed,
		Status:    status,
	}
	if fixed {
		m.repairs++
	}
	m.incidents = append(m.incidents, inc)
	// 追加到 jsonl
	if b, err := json.Marshal(inc); err == nil {
		if m.logFile != nil {
			_, _ = m.logFile.WriteString("[INCIDENT] " + string(b) + "\n")
		}
		_ = appendLine(m.incidentPath, b)
	}
	fmt.Printf(" [自愈-检测] [%s|%s] 现象=%s → 诊断=%s → 修复=%s (自动=%v)\n",
		category, severity, detail, diagnosis, action, fixed)
}

// loadIncidents 启动时恢复历史 Incident（供前端面板展示）。
func (m *Manager) loadIncidents() {
	data, err := os.ReadFile(m.incidentPath)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var inc Incident
		if json.Unmarshal([]byte(line), &inc) == nil {
			m.incidents = append(m.incidents, inc)
		}
	}
}

func appendLine(path string, b []byte) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(b, '\n'))
	return err
}

// ------ 对外查询接口 ------

// Status 返回当前整体状态（供 /api/selfheal/status）。
func (m *Manager) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	checks := make([]HealthCheck, 0, len(m.checks))
	for _, c := range m.checks {
		checks = append(checks, c)
	}
	open := 0
	for _, c := range checks {
		if !c.OK {
			open++
		}
	}
	llmOK, llmDetail := false, "未检查"
	if c, ok := m.checks["llm_api"]; ok {
		llmOK = c.OK
		llmDetail = c.Detail
	}
	return Status{
		Enabled:    true,
		UptimeSec:  int64(time.Since(m.startTime).Seconds()),
		LastTick:   m.lastTick.Format("2006-01-02 15:04:05"),
		Checks:     checks,
		OpenIssues: open + m.openIncidents(),
		Repairs:    m.repairs,
		LLMReady:   llmOK,
		LLMDetail:  llmDetail,
	}
}

func (m *Manager) openIncidents() int {
	n := 0
	for _, inc := range m.incidents {
		if inc.Status == "manual" {
			n++
		}
	}
	return n
}

// Incidents 返回全部 Incident（最新在前）。
func (m *Manager) Incidents() []Incident {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Incident, len(m.incidents))
	copy(out, m.incidents)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// setCheck 更新一条监测项状态。
func (m *Manager) setCheck(name, severity, detail string, ok bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checks[name] = HealthCheck{
		Name:        name,
		OK:          ok,
		Severity:    severity,
		Detail:      detail,
		LastChecked: time.Now().Format("2006-01-02 15:04:05"),
	}
}

// heal 触发某个修复回调（若已注册）。
func (m *Manager) heal(key string) (string, bool) {
	m.mu.Lock()
	fn, ok := m.healers[key]
	m.mu.Unlock()
	if !ok {
		return "未注册修复回调", false
	}
	desc, err := fn()
	if err != nil {
		return desc + "（失败: " + err.Error() + "）", false
	}
	return desc, true
}