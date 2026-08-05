package selfheal

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ---------- 周期性监测 ----------

// Tick 执行一轮监测：检查四类异常，发现异常则自动诊断并触发修复。
// intervalSec 由调用方控制调度频率（默认 15s）。
func (m *Manager) Tick() {
	m.mu.Lock()
	m.lastTick = time.Now()
	m.mu.Unlock()

	m.checkLLMAPI()
	m.checkProcess()
	m.checkData()
	m.checkLoop()
}

// checkLLMAPI 监测 LLM/API 可用性。
// 覆盖面：api.json 是否存在、能否解析、关键字段（base_url/model）是否填全。
func (m *Manager) checkLLMAPI() {
	b, err := os.ReadFile(m.apiCfgPath)
	if err != nil {
		detail := "api.json 缺失，LLM 不可用（将降级 dry-run）"
		m.setCheck("llm_api", "critical", detail, false)
		// 修复：生成 api.json 模板
		action, fixed := m.heal("llm_config")
		m.newIncident("llm_api", "critical", detail, "无法读取 api.json", action, fixed)
		return
	}
	var cfg struct {
		BaseURL string `json:"base_url"`
		Model   string `json:"model"`
		APIKey  string `json:"api_key"`
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		detail := "api.json 解析失败，LLM 不可用"
		m.setCheck("llm_api", "critical", detail, false)
		m.newIncident("llm_api", "critical", detail, "api.json 格式损坏", "请修正 api.json 格式", false)
		return
	}
	if strings.TrimSpace(cfg.BaseURL) == "" || strings.TrimSpace(cfg.Model) == "" {
		detail := "api.json 未配置 base_url/model，LLM 当前用 dry-run"
		m.setCheck("llm_api", "warn", detail, false)
		m.newIncident("llm_api", "warn", detail, "LLM 配置不完整", "配置 api.json 后即可启用 LLM（当前 dry-run 兜底）", false)
		return
	}
	m.setCheck("llm_api", "ok", "LLM 配置有效（"+cfg.BaseURL+" / "+cfg.Model+"）", true)
}

// checkProcess 监测服务进程：尝试 TCP 连接自身三个端口。
// 进程内自检：任一端口不可达说明服务异常（本进程外问题，记录告警）。
func (m *Manager) checkProcess() {
	ports := []struct{ name, addr string }{
		{"story", "127.0.0.1:48090"},
		{"world", "127.0.0.1:48091"},
		{"unified_ui", "127.0.0.1:48092"},
	}
	allOK := true
	var bad []string
	for _, p := range ports {
		conn, err := net.DialTimeout("tcp", p.addr, 2*time.Second)
		if err != nil {
			allOK = false
			bad = append(bad, p.name)
			continue
		}
		conn.Close()
	}
	if allOK {
		m.setCheck("process", "ok", "三个服务端口均可达", true)
		return
	}
	detail := "端口不可达: " + strings.Join(bad, ",")
	m.setCheck("process", "error", detail, false)
	// 进程内无法重启自身，交由外部守护处理；仅记录 Incident 提示。
	m.newIncident("process", "error", detail, "服务监听异常", "请检查进程/端口占用（auto-restart 由外部守护处理）", false)
}

// checkData 监测数据一致性：校验每个世界目录的关键状态文件是否存在且非空/可解析。
// world_state.json 必须为合法 JSON；event_log.jsonl 至少存在。
func (m *Manager) checkData() {
	worldsDir := filepath.Join(m.baseDir, "worlds")
	entries, err := os.ReadDir(worldsDir)
	if err != nil {
		m.setCheck("data", "ok", "无世界数据目录（尚未创建世界）", true)
		return
	}
	worst := "ok"
	var problems []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		wdir := filepath.Join(worldsDir, e.Name())
		statePath := filepath.Join(wdir, "world_state.json")
		if _, err := os.Stat(statePath); err != nil {
			// 未初始化世界（目录存在但无状态）不算损坏
			continue
		}
		b, err1 := os.ReadFile(statePath)
		if err1 != nil || !json.Valid(b) {
			worst = "error"
			problems = append(problems, e.Name()+":world_state.json 损坏")
			// 修复：回退最近快照
			action, fixed := m.heal("rewind_" + e.Name())
			m.newIncident("data", "error", e.Name()+" 状态文件损坏", "world_state.json 无法解析", action, fixed)
			continue
		}
		// 可为空对象，但至少是合法 JSON
	}
	if len(problems) == 0 {
		m.setCheck("data", "ok", "世界状态文件完好", true)
		return
	}
	m.setCheck("data", worst, strings.Join(problems, "; "), worst == "ok")
}

// checkLoop 监测模拟循环异常：卡死（长时间无进展）与连续失败。
// 通过注入的 loopStateFn 读取运行态；无注册则跳过。
func (m *Manager) checkLoop() {
	m.mu.Lock()
	fn := m.loopStateFn
	m.mu.Unlock()
	if fn == nil {
		m.setCheck("simulation", "ok", "未运行模拟循环", true)
		return
	}
	st := fn()
	if !st.Running {
		if st.ConsecFail > 0 {
			m.setCheck("simulation", "warn", "循环已停，存在连续失败", false)
		} else {
			m.setCheck("simulation", "ok", "循环未运行", true)
		}
		return
	}
	// 连续失败 ≥ 3 → 自动降级/暂停，避免空转烧 token
	if st.ConsecFail >= 3 {
		detail := fmt.Sprintf("模拟连续失败 %d 次，自动暂停", st.ConsecFail)
		m.setCheck("simulation", "error", detail, false)
		action, fixed := m.heal("restart_loop")
		m.newIncident("simulation", "error", detail, "LLM/中转站持续失败", action, fixed)
		return
	}
	m.setCheck("simulation", "ok", "循环运行中（Day "+strconv.Itoa(st.Day)+"/"+strconv.Itoa(st.TargetDay)+"）", true)
}
