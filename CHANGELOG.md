# Changelog

本项目所有重要变更都记录在此。格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，版本遵循 [SemVer](https://semver.org/lang/zh-CN/)。

## [Unreleased]

### 🚀 即将到来
- Codex / Trae MCP 接入（`worldsim-mcp/`，已发布）
- 项目治理完善（Issue/PR 模板、CONTRIBUTING、SECURITY、CODE_OF_CONDUCT、Dependabot）

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