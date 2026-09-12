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
- **文字游戏模式（Play Mode）**：同一世界可"直接玩"——HTTP API（`/api/game/start|action|wait|status`，`GET /game` 终端风游戏页）、统一外壳「文字游戏」页（48092），或 MCP 工具（`world_game_start/play/wait/status`）。**数值与掷骰在代码层**（`game.json`：d20+属性修正 vs 裁判申报的 DC），LLM 只做裁判与第二人称叙事，不许在叙述里改数值；默认属性表=世界书 `## 游玩属性` 段（数据层）→ LLM 现场生成 → 通用兜底，属性名不硬编码；**像素套件自动换装**：`docs/art/pixel/<theme>/`（修仙/末世/西幻/克苏鲁/星际各 63 资产）由 `GET /pixel-art/{theme}/{file}.png` 直出，`/api/game/status` 返回 theme 与 sprite 列表（detectPixelTheme 三层猜题材）。**美术工坊（v1.8.0）**：`internal/art` 世界书驱动生成专属像素套件——规划 Agent（世界书→`art/plan.json`：12人物/8怪物/3场景/16地图块/24物品+英文提示词+调色板）→ 可插拔生成器（中转站/PixelLab）→ Go 原生裁剪抠底 → `/art/{file}.png` 直出并在 `/api/game/status` 优先于打包套件；HTTP `GET /studio` 工坊页与 `/api/art/*`（config 掩码/plan/generate 异步/jobs/assets/history）。**游玩页世界观感（v1.9.0）**：`W1 动态条目`（世界书 `- keys => 情报`，中文子串匹配+sticky 3 回合+1200 字预算，`worldbook.ActivateWI`）；`game.json` 好感度 `relations`（裁判申报、代码钳 -10..10）；`/api/game/status` 扩展 `scene/portraits/item_icons/map`（确定性 snake 网格地图，plan kind→地名关键词→hash 三级 tile 兜底）；世界卡 `GET /api/world/card` 导出 / `POST /api/worlds/import` 导入（不含编年史——隐私）。**游玩手感（v1.10.0）**：`POST /api/game/undo` 单步撤销（`game.json.prev` 快照随存档落盘）；`GET /api/game/export` / `POST /api/game/import` 存档带出/回灌（越界自动收数）；`downed` 倒下状态机（HP 0 触发、等待回血恢复，LLM 不许宣布死亡）；`wi_hits` 情报命中透明化；wait 推进引擎日（场景横幅随游玩轮转）；`internal/game` 短锁+单飞（回合期间 status/log 不阻塞、并发回合明确报错）；`:48091/` 直出 uiteg 统一外壳（`GET /{path...}`），wsweb 仅存 game.html/studio.html 专页——审计全文见 `docs/可用性审计-v1.10.md`
- **就绪度不是天数**：看 `world_readiness`，别等"跑满N天"
- **LLM 超时容忍**：中转站慢时单日失败会 dry-run 兜底，循环自愈重试；长时间不动用 `world_rewind` 回退
- **推理模型空正文陷阱（v1.10.1）**：带思考通道的模型（如 `deepseek-v4.x`）先花 `max_tokens` 推理，长 JSON 任务易 `finish=length` 且正文为空——`api.json` 提高 `max_tokens`（≥32000）或用 `"extra_body": {"enable_thinking": false}` 关思考通道；`internal/llm` 会直接把这条提示作为错误抛出（不再只有难懂的 JSON 解析错误）
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