# AGENTS.md — Codex 协作指南

WorldSim 是"多Agent世界模拟器 → 网文生产引擎"：模拟一个世界真实运转，再把编年史自动改写成小说。

## 核心概念

- **世界**：每个世界一个目录（`wsdata/worlds/<名>/`），含编年史/记忆/决策/伏笔/小说
- **世界书**：决定世界质感（`wsdata/worldbooks/`，含 15 个主题包）
- **就绪度**：素材够不够写小说（段落≥3 + 戏剧素材≥12 + 伏笔回收≥1 + 张力≥0.4）
- **时间回退**：快照制，可回退到任意锚点重新演化

## 如何操作 WorldSim（MCP）

本仓库提供 `worldsim-mcp/server.py`（零依赖 Python MCP server）。配置后可用 `world_*` 工具：

1. 启动服务：`sh run.sh`（或 Windows `run.bat`）
2. 建世界：`world_create`（theme 主题包 + desc 一句话设定 → LLM 自动生成世界书）
3. 初始化：`world_init`（生成主角/NPC/地点）
4. 跑模拟：`world_loop_start`（后台持续，就绪自动停）
5. 查就绪：`world_readiness`
6. 写小说：`world_novel_generate` → `world_novel_list` → `world_novel_chapter`

配置方法见 `worldsim-mcp/codex.md`。

## 关键规则

- **题材自适应**：任何世界同一套引擎。修仙=测灵根/丹炉/宗门，末世=搜寻/尸潮/营地，别拿别的题材模板套
- **决策翻案**：用户明确说"选B"就 `world_decision_resolve`，后续按用户方向写
- **就绪度不是天数**：看 `world_readiness`，别等"跑满N天"
- **LLM 超时容忍**：中转站慢时单日失败会 dry-run 兜底，循环自愈重试；长时间不动用 `world_rewind` 回退
- **安全**：`api.json` 含真实密钥，**永不提交**（.gitignore 已排除）；`worlds/` `material/` 不入库

## 常用命令

```bash
go build -o worldsim .        # 构建（Go 1.22+）
./worldsim /path/to/data-dir  # 启动（48091 WebUI / 48090 小说服务）
python3 worldsim-mcp/server.py --selftest  # MCP 自检
```

## 代码结构

- `main.go` — 服务入口（REST API + WebUI embed）
- `internal/engine/` — 事件溯源 State Engine
- `internal/sim/` — 多Agent模拟器（事件/决策/NPC/伏笔/记忆/快照/就绪度）
- `internal/worldbook/` — 世界书解析 + 主题包 + LLM 生成
- `internal/novel/` — 小说写手（素材投喂/去AI味）
- `wsweb/` `static/` — 前端（embed 进二进制）