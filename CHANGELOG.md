# Changelog

本项目所有重要变更都记录在此。格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，版本遵循 [SemVer](https://semver.org/lang/zh-CN/)。

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