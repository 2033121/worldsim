# Changelog

本项目所有重要变更都记录在此。格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，版本遵循 [SemVer](https://semver.org/lang/zh-CN/)。

## [1.6.0] - 2026-08-05

### ✨ 新增
- **文字游戏游玩模式（Play Mode）**：把「世界模拟器 → 小说引擎」升级为「也能直接玩」——同一世界两种用法（模拟出小说 / 亲自下场玩）
  - **新包 `internal/game`**：数值层由代码持有（HP/等级/经验/背包/任务/属性表，落盘 `worlds/<世界>/game.json`，重启可续），LLM 不靠 prompt 记数字
  - **回合管线**：玩家自由输入（do行动/say说话/story叙事三模式）→ 裁判 LLM 输出意图+难度+申报变更 → **代码掷骰**（d20+属性修正 vs DC，nat20 必成/nat1 必败）→ 叙述者把既定结果讲成第二人称游戏叙事 → 落账 `game.json`
  - **等级系统**：经验每 100 点升级，升级上限+10 且满血；属性增减由裁判申报、代码统一收数（1~10）
  - **等待回合 `POST /api/game/wait`**：世界自转（NPC/环境动态由裁判给出）+ 休息回血（每次上限 10%）
  - **开局 `POST /api/game/start`**：自动暂停后台模拟循环（回合制不空转烧 token），LLM 生成主题适配的属性表（修仙=炼气/体魄/心性…不硬编码）
  - **数值层→引擎同步**：每回合把 HP 比例→`health`、金币→`money`、地点→`location`、面板→`stats.game` 提交进事件溯源引擎并落盘——游戏产物反向喂小说播种链路
  - **终端风游戏页**：`GET /game`（自包含单文件 HTML，embed 进二进制，零外部资源）
  - **HTTP API**：`GET /api/game/status` `POST /api/game/start|action|wait|stop` `GET /api/game/log`
  - **单元测试**：`internal/game/game_test.go`（掷骰边界/收数与升级/持久化/等待回血/停服守卫/JSON 打捞）

### ✅ 验证
- `go build ./...` 与 `go vet ./...` 通过
- `go test ./internal/game/...` 全绿（6/6）
- 端到端冒烟：建世界 → init → start → action → wait → `world_state.json` 中 `stats.game`/`health`/`money` 同步落盘全数通过（断言式冒烟脚本）

## [1.6.1] - 2026-08-05

### ✨ 新增（游玩模式收尾三件）
- **统一前端入口 :48092 真实落地**：此前 README 宣告 48092 网关但仓库里没有监听实现（selfheal 探活项一直 unhealthy）；新增 `ws_gateway.go`——uiteg 外壳（SPA fallback）+ `/api/novel/*`→48090、其余 `/api/*` 与 `/game` 代理到 48091；worldapp 首页新增第三张入口卡「文字游戏」+ 完整 `game` 路由页（`.svelte` 组件直接玩：_do/say/story 三模式 + 掷骰展示 + 面板_）
- **MCP 新增 4 个游戏工具**：`world_game_start` / `world_game_play`（input+mode）/ `world_game_wait` / `world_game_status`——AI 客户端（Codex/Trae/Claude）可直接陪玩或代打；工具总数 26→30
- **游玩属性数据层**：世界书新增可选段 `## 游玩属性`（`- 属性名: 1~10`），15 个主题包各自预置主题适配属性表（修仙=炼气/体魄/道心/灵识，末世=体能/搜刮/冷静/装备…），通用模板补说明；`Worldbook.GameAttrs()` 解析（全中文冒号/半角/带引号行跳过均有测试）；开局降级顺序 = LLM 现场生成 → 世界书数据段 → 通用三属性，任意一层都不硬编码属性名

### ✅ 验证
- `go test ./...` 全绿（worldbook 3 新测试 + game 7/7）
- 端到端冒烟：:48092 `/`（uiteg 外壳）、`/game`（代理）、`/api/*`（代理）全通；建世界→init→开局后 `attrs == {剑术:5, 体魄:4}`（世界书段直出，LLM 未接入也成立）→ action 回合成功
- `worldapp npm run build` 重建 uiteg（含 GamePage），二进制 embed 校验一致

## [1.5.0] - 2026-08-05

### ✨ 新增
- **内嵌持续监测与自动修复模块（self-healing）**：`internal/selfheal` 内嵌于世界模拟服务，跟踪运行过程与错误日志，异常自动诊断根源并自主修复
  - **四类异常监测**（15s 一轮）：LLM/API 可用性（api.json 缺失/损坏/未配置）、服务进程（48090/48091/48092 端口探活）、数据一致性（world_state.json 合法性）、模拟循环（连续 RunDay 失败/卡死）
  - **自动修复动作**：LLM 配置缺失 → 生成 api.json 模板；模拟连续失败 → 中断异常循环防空转烧 token；数据损坏 → 回退最近快照并重建 Simulator
  - **运行日志落盘**：`wsdata/selfheal/runtime.log`；检测/诊断/修复记录持久化到 `wsdata/selfheal/incidents.jsonl`（跨重启保留）
  - **前端监测面板**：世界控制台新增「🛠️ 监测」Tab（`SelfHealPanel.svelte`），展示运行时长/自动修复次数/LLM 就绪状态/各项健康检查/历史 Incident 记录
  - **监测接口**：`GET /api/selfheal/status` 与 `GET /api/selfheal/incidents`
  - **单元测试**：`internal/selfheal/selfheal_test.go` 覆盖初始化/LLM 检测修复/数据回退/循环失败/ Incident 持久化

### ✅ 验证
- `go build ./...` 通过
- `go test ./internal/selfheal/...` 全绿（5/5）

## [1.4.0] - 2026-08-05

### ✨ 新增
- **联网搜索（内置 Tavily）**：新增 `internal/search/tavily.go`，写作页「🔍 搜素材」面板直接搜素材一键复制/打开原文；`POST /api/search` 接口 + provider 注入；docker-compose 移除自托管 SearXNG（改用内置 Tavily，无需额外容器）
- **世界直接播种小说**：`internal/bridge` + `POST /api/world/seed-novel`，把世界书/角色/势力/近期编年史事件直接播种成小说大纲与设定，零 LLM 调用；前端「🌱 从世界播种」面板 + 世界页「📖 据此生成小说」
- **LLM 用量统计**：全局 token 实时聚合 + hour/day 时间窗持久化（`stats_store`）+ 费用估算；`GET /api/llm/stats` / `/history`；前端 TokenStats 页 + 任务 token 徽章
- **技能注入写作/大纲**：大纲/分卷/章节生成注入已启用技能 SOP（`FormatSkillsContent`）
- **技能卡重命名**：5 个技能卡规范化为中文名（开篇写作SOP/章节写作SOP/修改润色SOP/数据诊断与调整SOP/武器系统描写方法论）+ `skills_test.go`

### ♻️ 重构
- `internal/bridge` 手写插入排序 `sortStrings` 改用标准库 `sort.Strings`（消除重复实现）
- 移除 `tavily.go` 中 `language` 参数被误映射为 `topic=general` 的冗余逻辑（Tavily basic 深度无语言过滤，保留参数以符合 Provider 接口）

### ✅ 验证
- `go vet ./...` 与 `go test ./internal/...` 全绿（bridge/httpapi/llm/research/search 均 ok）
- Docker 重建后 `/api/novel/search` 实测返回优质中文结果（engine: tavily）

## [1.3.2] - 2026-08-05

### 🔧 维护与更新
- **依赖全面升级**：三个前端（`worldapp`/`worldweb`/`frontend`）执行 `npm install` + `npm update`，升级至最新兼容版本（Vite 5.4.21 / daisyUI 5.7.16 / Svelte 4.2.x），并重建所有 embed 产物
- **embed 产物同步**：`uiteg/`（统一前端 48092）、`wsweb/`（世界控制台 48091）、`static/`（小说服务 48090）全部重建并同步最新构建，随二进制分发
- **仓库治理**：`.gitignore` 新增本地编译二进制（`worldsim_bin`/`worldsim-linux`/`worldsim_linux`）与 `worldweb/dist/` 忽略项；`worldsim_bin`（15MB 二进制）解除 git 跟踪，二进制不再入库（Docker 从源码构建）
- **技能卡重命名**：`kc-skill-240/241` 文件名规范化（`·` → `_`，漫剧适配·战斗场景视觉化 / 场景空间设计），内容微调，embed 目录整包加载不受影响
- **新增源文件入库**：`worldweb/index.html`、`.mcp.json`（IDE MCP 接入配置）、`scripts/ocr.ps1`（Windows 系统 OCR 工具）
- **测试脚本健壮化**：`test_backend_api.py` 章节号参数化（12→动态第1章）；`test_writing_edit.py` 章节列表断言改为项目无关（兼容多个测试项目标题）

### ✅ 验证
- Go 编译 `go build ./...` 通过，`go test ./...` 全绿
- 三前端 `npm run build` 全部成功
- Docker 镜像重建 + 容器重启健康，48090/48091/48092 三端口均返回 200 且 serve 最新前端资源
- 后端 API 测试 7/7、前端导航/语言 13/13、写作编辑 8/8、语言持久化 PASS

## [1.3.1] - 2026-08-04

### ✨ 新增
- 发布流水线全自动化：`operit-plugin/` 资源入库，release.yml 自动编译 ARM64 + 组装 Operit 插件包与 MCP zip
- 打 `v*` tag 即自动出 7 个资产（5 平台二进制 + 插件包 + MCP），无需手动上传

### 🐛 修复
- release.yml Windows 架构提取 bug（`.exe` 后缀导致 arch=exe）
- release.yml 打包目录与二进制同名冲突（改用 `pkg/` 子目录隔离）

## [1.3.0] - 2026-08-04

### ✨ 新增
- **Codex / Trae MCP 接入**（`worldsim-mcp/`）：零依赖 Python MCP server（25 工具），Codex config.toml / .mcp.json、Trae MCP 面板双配置指南
- **仓库根 `AGENTS.md`**：Codex 协作指南（概念/操作/规则/代码结构）
- **素材库开源入库**：886 条真实网文风格示范（8 大类 127 子类，来源已标注，仅风格参考）
- **仓库治理全套**：Issue/PR 模板、CONTRIBUTING、SECURITY、CODE_OF_CONDUCT、CHANGELOG、docs/API.md、Dependabot
- **README 徽章**：CI/License/Go/Release/Stars

### 🔧 工程化
- CI 增强：gofmt 格式检查 + MCP 语法/协议自检 job + setup-go 缓存
- gofmt 全量格式化（49 个文件）

### 🐛 修复
- CI workflow YAML 解析失败：plain scalar 中 `MCP: syntax OK` 冒号+空格被当嵌套 mapping → 改为 `MCP syntax OK`

## [Unreleased]

### 🚀 即将到来
- （规划中）

## [1.2.0] - 2026-08-04

### ✨ 新增
- **时间回退机制**：快照制（8 文件完整状态复制），每 30 天自动存档 + 手动存档，滚动保留 20 个；3 个 API（`world_snapshots`/`world_snapshot`/`world_rewind`）；WebUI ⏪ 回退面板
- **前端控制台升级**（674 行单文件）：决策翻案 tab、后台循环开关 + 进度条、就绪度 4 指标面板、新建世界选主题包 + 一句话设定、世界书全文 tab、token 统计 tab、未回收伏笔面板
- **后台循环**：`POST /api/world/loop`，就绪度驱动自动停
- **后端接口**：`/api/worldbooks/themes`、`/api/world/worldbook`、`/api/world/foreshadows`、`/api/world/loop`
- **LLM 自动启用**：启动/新建世界/init 后自动从 api.json 启用，不再手动配置
- **多平台分发**：5 平台二进制（linux amd64/arm64、windows amd64、darwin amd64/arm64）+ Windows run.bat
- **Operit 插件包**：沙盒包 25 工具 + SKILL + WebUI + 15 主题包 + 886 条素材

### 🐛 修复
- **缓存命中率破 100%**（194.2%）：`span_tracker.go` 两个字段重复累加 → 取较大者 + 封顶 100%
- **宿主 http_request 超时断连**导致世界书生成/初始化中断 → 长操作全部独立 context + 重试 3 次
- **engine.Submit 幂等**导致 LLM 初始化方案不落库 → CommandID 加时间戳后缀
- **slice 生活切片被平淡日误杀**（severity 0.2-0.4 < 0.4 阈值）→ 平淡日也落编年史
- **sdcard 无执行权限位** → 二进制部署到 /tmp 执行，数据留在插件包
- 时间回退后重跑验证（LLM 输出不重现，形成新分支）

## [1.1.0] - 2026-08-03

### ✨ 新增
- 插件包成型（沙盒包 22 工具 + SKILL + WebUI + 15 主题包）
- 多世界支持（世界实例池 + 切换）
- 小说生成流水线（写手 Agent + 去AI味）

### 🐛 修复
- 多世界循环并发冲突
- 小说生成防重入

## [1.0.0] - 2026-08-02

### ✨ 新增
- 多Agent世界模拟器核心：GM / 事件 / 主角三问 / NPC / 写手
- 15 主题包 + 世界书通用骨架
- 事件溯源架构（event_log.jsonl + Replay）
- WebUI 基础版（461 行单文件）

<!-- 版本对比链接（有 tag 后启用） -->
<!-- [1.2.0]: https://github.com/2033121/worldsim/compare/v1.1.0...v1.2.0 -->
<!-- [1.1.0]: https://github.com/2033121/worldsim/compare/v1.0.0...v1.1.0 -->