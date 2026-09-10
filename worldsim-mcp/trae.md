# WorldSim MCP · Trae 配置

把 WorldSim 世界模拟器接入 Trae（字节跳动 AI IDE），在 IDE 里直接开世界、跑模拟、写小说。

## 添加 MCP Server

1. 打开 Trae → 顶部 **MCP** 图标（或 设置 → MCP）
2. 点 **添加 MCP Server** → 选择 **stdio** 模式
3. 填写：

| 字段 | 值 |
|---|---|
| 名称 | worldsim |
| 命令 | `python3` |
| 参数 | `/absolute/path/to/worldsim-mcp/server.py`（Args 列表加这一项） |
| 环境变量 | 留空 |

4. 保存后，MCP 面板里 `worldsim` 显示已连接，展开能看到 26 个 `world_*` 工具

> 也支持直接导入 JSON：`worldsim-mcp/mcp.json.example` 的内容粘贴到"JSON 导入"。

## 项目级配置（可选）

在项目根目录放 `.mcp.json`（内容见 `mcp.json.example`），Trae 打开该项目时自动加载。

## 前置条件

1. WorldSim 服务已启动：`sh run.sh`（Linux）或 `run.bat`（Windows），确认 `http://127.0.0.1:48091` 可访问
2. 已配置 LLM（浏览器打开 `http://127.0.0.1:48091` 操作台填 API Key）

## 验证

在 Trae 的 MCP 面板点 `worldsim` 的测试按钮，或直接对 AI 说：

- "调用 world_list 看有哪些世界"
- "用 world_create 开一个克苏鲁世界"
- "world_loop_start 跑模拟，就绪了 world_novel_generate 写小说"

## 工具清单（26 个）

`world_list` `world_select` `world_create` `world_init` `world_state` `world_run_day`
`world_loop_start` `world_loop_stop` `world_loop_status` `world_readiness` `world_decisions`
`world_decision_resolve` `world_chronicle` `world_memories` `world_foreshadows` `world_thinking`
`world_tokens` `world_novel_list` `world_novel_generate` `world_seed_novel` `world_novel_chapter` `world_themes`
`world_snapshots` `world_snapshot` `world_rewind` `world_webui`

> `world_seed_novel` 走小说服务 48090，把当前世界直接播种成小说项目（零 LLM 调用）。