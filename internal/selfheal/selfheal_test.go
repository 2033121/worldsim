package selfheal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// 验证 Manager 初始化：创建 selfheal 目录与日志文件
func TestManagerInit(t *testing.T) {
	dir := t.TempDir()
	m, err := New(dir, filepath.Join(dir, "api.json"))
	if err != nil {
		t.Fatalf("New 失败: %v", err)
	}
	defer m.Close()
	if _, err := os.Stat(filepath.Join(dir, "selfheal", "runtime.log")); err != nil {
		t.Errorf("runtime.log 未创建: %v", err)
	}
}

// 验证 LLM/API 可用性检测：缺失 api.json → critical，并触发 llm_config 修复回调
func TestCheckLLM_ConfigMissing(t *testing.T) {
	dir := t.TempDir()
	apiPath := filepath.Join(dir, "api.json") // 故意不创建
	m, _ := New(dir, apiPath)
	defer m.Close()

	fixed := false
	m.RegisterHealer("llm_config", func() (string, error) {
		fixed = true
		return "已生成模板", nil
	})

	m.checkLLMAPI()
	c := m.checks["llm_api"]
	if c.OK {
		t.Errorf("缺失 api.json 时 check 应为不健康")
	}
	if c.Severity != "critical" {
		t.Errorf("severity=%s，期望 critical", c.Severity)
	}
	if !fixed {
		t.Errorf("应触发 llm_config 修复回调")
	}
	// Incident 已记录
	if len(m.incidents) == 0 {
		t.Errorf("应记录 Incident")
	}
}

// 验证 LLM/API 可用性检测：配置有效 → ok
func TestCheckLLM_ConfigValid(t *testing.T) {
	dir := t.TempDir()
	apiPath := filepath.Join(dir, "api.json")
	cfg := map[string]any{"base_url": "https://api.example.com/v1", "model": "gpt-x", "api_key": "k"}
	b, _ := json.Marshal(cfg)
	_ = os.WriteFile(apiPath, b, 0644)

	m, _ := New(dir, apiPath)
	defer m.Close()
	m.checkLLMAPI()
	if c := m.checks["llm_api"]; !c.OK {
		t.Errorf("有效配置应 OK，实际: %s", c.Detail)
	}
}

// 验证数据一致性检测：损坏的 world_state.json → 触发 rewind 修复回调
func TestCheckData_CorruptState(t *testing.T) {
	dir := t.TempDir()
	apiPath := filepath.Join(dir, "api.json")
	m, _ := New(dir, apiPath)
	defer m.Close()

	// 造一个世界目录 + 损坏状态文件
	worldDir := filepath.Join(dir, "worlds", "测试世界")
	_ = os.MkdirAll(worldDir, 0755)
	_ = os.WriteFile(filepath.Join(worldDir, "world_state.json"), []byte("{not-json"), 0644)

	rewound := false
	m.RegisterHealer("rewind_测试世界", func() (string, error) {
		rewound = true
		return "已回退快照", nil
	})

	m.checkData()
	c := m.checks["data"]
	if c.OK {
		t.Errorf("损坏状态文件时 data check 应为不健康")
	}
	if !rewound {
		t.Errorf("应触发 rewind 修复回调")
	}
}

// 验证模拟循环监测：连续失败 ≥3 → 触发 restart_loop 修复
func TestCheckLoop_ConsecFail(t *testing.T) {
	dir := t.TempDir()
	m, _ := New(dir, filepath.Join(dir, "api.json"))
	defer m.Close()

	m.SetLoopStateSource(func() LoopState {
		return LoopState{Running: true, ConsecFail: 5, Day: 10, TargetDay: 100}
	})
	stopped := false
	m.RegisterHealer("restart_loop", func() (string, error) {
		stopped = true
		return "已中断循环", nil
	})

	m.checkLoop()
	if !stopped {
		t.Errorf("连续失败应触发 restart_loop")
	}
}

// 验证 Incident 持久化与 Status 统计
func TestIncidentPersistence(t *testing.T) {
	dir := t.TempDir()
	m, _ := New(dir, filepath.Join(dir, "api.json"))
	m.newIncident("llm_api", "warn", "测试现象", "测试诊断", "测试修复", true)
	m.Close() // 释放日志句柄，避免占住临时目录影响清理

	// 重新加载，验证持久化
	m2, _ := New(dir, filepath.Join(dir, "api.json"))
	defer m2.Close()
	if len(m2.incidents) != 1 {
		t.Errorf("重启后应恢复 1 条 Incident，实际 %d", len(m2.incidents))
	}
	if m2.incidents[0].Detail != "测试现象" {
		t.Errorf("Incident 内容不符: %s", m2.incidents[0].Detail)
	}
}
