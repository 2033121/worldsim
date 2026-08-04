# Changelog

本项目所有重要变更都记录在此。格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，版本遵循 [SemVer](https://semver.org/lang/zh-CN/)。

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