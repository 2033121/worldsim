package selfheal

// Incident 记录一次检测到的异常 + 自动诊断结果 + 修复动作。
// 持久化到 wsdata/selfheal/incidents.jsonl（最新在前展示）。
type Incident struct {
	ID        string `json:"id"`
	Time      string `json:"time"`
	Category  string `json:"category"`            // llm_api | simulation | process | data
	Severity  string `json:"severity"`            // warn | error | critical
	Detail    string `json:"detail"`              // 检测到的现象
	Diagnosis string `json:"diagnosis"`           // 定位到的根源
	Action    string `json:"action"`              // 采取的修复动作
	AutoFixed bool   `json:"auto_fixed"`          // 是否已自动修复
	Status    string `json:"status"`              // resolved | pending | manual
}

// HealthCheck 一条监测项的当前健康状态（供前端面板展示）
type HealthCheck struct {
	Name        string `json:"name"`
	OK          bool   `json:"ok"`
	Severity    string `json:"severity"` // ok | warn | error | critical
	Detail      string `json:"detail"`
	LastChecked string `json:"last_checked"`
}

// Status 自愈模块整体状态（/api/selfheal/status）
type Status struct {
	Enabled    bool           `json:"enabled"`
	UptimeSec  int64          `json:"uptime_sec"`
	LastTick   string         `json:"last_tick"`
	Checks     []HealthCheck  `json:"checks"`
	OpenIssues int            `json:"open_issues"`
	Repairs    int            `json:"repairs"`
	LLMReady   bool           `json:"llm_ready"`
	LLMDetail  string         `json:"llm_detail"`
}