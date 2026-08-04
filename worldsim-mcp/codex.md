# WorldSim MCP · Codex 配置

把 WorldSim 世界模拟器接入 Codex CLI / Codex IDE，让 Codex 能直接开世界、跑模拟、写小说。

## 方式一：项目级 `.mcp.json`（推荐，随项目走）

在项目根目录创建 `.mcp.json`：

```json
{
  "mcpServers": {
    "worldsim": {
      "command": "python3",
      "args": ["/absolute/path/to/worldsim-mcp/server.py"]
    }
  }
}
```

Codex 启动时自动加载，直接可用 `world_*` 工具。

## 方式二：全局 `~/.codex/config.toml`

```toml
# ~/.codex/config.toml
[mcp_servers.worldsim]
command = "python3"
args = ["/absolute/path/to/worldsim-mcp/server.py"]
```

或使用 CLI 命令添加：

```bash
codex mcp add worldsim -- python3 /absolute/path/to/worldsim-mcp/server.py
codex mcp list
```

## 方式三：会话内指定

```bash
codex --mcp-config /path/to/mcp.json
```

## 前置条件

1. WorldSim 服务已启动：`sh run.sh`（Linux）或 `run.bat`（Windows），确认 `http://127.0.0.1:48091` 可访问
2. 已配置 LLM（WebUI 操作台填好 API Key）

## 验证

```bash
python3 worldsim-mcp/server.py --selftest
# 输出：OK: WorldSim 服务在线, 世界: [...], 工具数: 25
```

## 使用示例（对 Codex 说的话）

- "用 worldsim 开一个修仙世界，主角是山村少年"
- "world_init 初始化，然后 world_loop_start 跑 500 天"
- "看看世界就绪了吗（world_readiness），够了就 world_novel_generate 写第一章"
- "有岔口决策吗（world_decisions）？把 dec-12-1 改成 B"

## 完整工具清单（25 个）

`world_list` `world_select` `world_create` `world_init` `world_state` `world_run_day`
`world_loop_start` `world_loop_stop` `world_loop_status` `world_readiness` `world_decisions`
`world_decision_resolve` `world_chronicle` `world_memories` `world_foreshadows` `world_thinking`
`world_tokens` `world_novel_list` `world_novel_generate` `world_novel_chapter` `world_themes`
`world_snapshots` `world_snapshot` `world_rewind` `world_webui`