# WorldSim MCP Server

把 WorldSim 世界模拟服务桥接成**标准 MCP 工具**，供 Codex CLI、Trae、Claude、Cursor 等任何 MCP 客户端调用。

**零依赖**：纯 Python 3.8+ 标准库，无需 pip install。

## 快速开始

```bash
# 1. 启动 WorldSim 服务（另开终端）
sh ../run.sh          # Linux / macOS
run.bat               # Windows

# 2. 自检
python3 server.py --selftest

# 3. 接入客户端（二选一）
#    Codex → 看 codex.md
#    Trae  → 看 trae.md
```

## 文件

| 文件 | 说明 |
|---|---|
| `server.py` | MCP stdio server（25 个 `world_*` 工具，零依赖） |
| `mcp.json.example` | 通用 mcpServers 配置模板（Codex .mcp.json / Trae JSON 导入） |
| `codex.md` | Codex CLI/IDE 接入指南（config.toml / .mcp.json / codex mcp add） |
| `trae.md` | Trae 接入指南（MCP 面板 stdio 添加） |

## 工具（25 个）

世界管理：`world_list` `world_select` `world_create` `world_init` `world_state`
模拟：`world_run_day` `world_loop_start` `world_loop_stop` `world_loop_status` `world_readiness`
剧情：`world_decisions` `world_decision_resolve` `world_chronicle` `world_memories` `world_foreshadows` `world_thinking`
时间回退：`world_snapshots` `world_snapshot` `world_rewind`
小说：`world_novel_list` `world_novel_generate` `world_novel_chapter`
其他：`world_themes` `world_tokens` `world_webui`

## 协议

标准 MCP（stdio / JSON-RPC 2.0）：`initialize` → `notifications/initialized` → `tools/list` → `tools/call`
