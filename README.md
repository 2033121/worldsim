# WorldSim · 多Agent世界模拟器 → 网文生产引擎

[![CI](https://github.com/2033121/worldsim/actions/workflows/ci.yml/badge.svg)](https://github.com/2033121/worldsim/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/2033121/worldsim/blob/main/LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8.svg)](https://go.dev/)
[![Zero Dependencies](https://img.shields.io/badge/dependencies-0-brightgreen.svg)](#)
[![Release](https://img.shields.io/github/v/release/2033121/worldsim?color=orange)](https://github.com/2033121/worldsim/releases)
[![Stars](https://img.shields.io/github/stars/2033121/worldsim)](https://github.com/2033121/worldsim/stargazers)

> 让 AI 模拟"一个世界真实地运转"，再把世界编年史自动改写成人味十足的小说。
> 基于 [Nigh/show-me-the-story](https://github.com/Nigh/show-me-the-story) 深度改造（Go 单二进制 + WebUI，零外部依赖）。

## ✨ 特性

- **多Agent世界模拟**：总导演(GM)/事件Agent/主角三问决策/感知分发/NPC互动/小说写手，各司其职
- **任意题材通用**：15个主题包（修仙/末世/西幻/克苏鲁/都市/星际/历史…）+ 通用世界书骨架 → 一句话创建新世界
- **时间尺度自适应**：修仙跳年、末世跳日、星际按标准时——LLM 从世界书自行判断，不硬编码
- **就绪度驱动**：模拟不按天数结束，按"素材够不够写小说"（段落/戏剧素材/伏笔回收/张力）自动判定
- **岔口决策队列**：剧情多方向岔口 AI 自动代决（零阻塞），用户可随时翻案，写手按用户方向写
- **时间回退**：快照制存档，剧情跑偏/卡死随时回退到任意锚点重新演化
- **去AI味**：886+ 条真实网文示范素材库 + 六层写作方法论注入（记忆钉/冲突钩子/伏笔/感官五维/视角三不/高频词禁用）
- **双控制入口**：WebUI 可视化控制台（浏览器操作）+ 沙盒包 API（AI 对话驱动）
- **统一前端入口**：浏览器式导航外壳（多标签/地址栏/前进后退/书签/命令面板 Ctrl+K/明暗主题），一键切换小说创作与世界模拟
- **题材研究智能体**：联网搜索热门题材 + 读取附件 → 产出 2-3 个候选对比方案 / 世界书方向 / 题材卡片，再引导建世界
- **联网搜索 & 写作搜素材**：内置 Tavily 搜索后端（可切 SearXNG / Bing / BingHTML），写作页直接搜素材一键复制，搜索结果 + 世界参考资料注入 LLM 上下文
- **世界直接播种小说**：把已模拟世界的世界书/角色/势力/近期编年史事件直接播种成小说大纲与设定，零 LLM 调用
- **LLM 用量看板**：全局 token 统计（实时聚合 + 小时/天时间窗持久化历史 + 费用估算），TokenStats 页面可视化
- **单二进制**：Go embed WebUI，零外部依赖，ARM64/Android 直接跑

## 🏗️ 架构

```
WorldSim
├── 统一前端入口 :48092（浏览器式导航外壳 + API 网关，推荐访问）
│   ├── 小说创作应用（代理到 48090）
│   └── 世界模拟控制台（代理到 48091）
├── 小说创作服务 :48090（show-me-the-story 流水线）
├── 世界模拟服务 :48091
│   ├── State Engine（事件溯源：event_log.jsonl + world_state.json + Replay）
│   ├── Simulator（多Agent调度：事件→感知→主角决策→GM裁决→NPC→记录）
│   │   ├── GM Agent        世界书裁决/段落规划（导演）
│   │   ├── Event Agent     事件生成（B5事件谱：冲突/奇遇/生活切片…）
│   │   ├── Protagonist     三问决策（价值/能力/世界线）→ 记忆沉淀
│   │   ├── NPC 互动        Init→Act→React 对话链
│   │   └── 伏笔账本        埋设/成熟/回收全周期
│   ├── 世界书体系          _template.md 通用骨架 + themes/ 15主题包 + B5事件谱
│   ├── 就绪度             arcs/drama/foreshadows/tension 四指标
│   ├── 时间回退            snapshots/ 快照目录（文件复制制）
│   ├── 题材研究 Agent     联网搜索 → 候选对比 / 世界书方向 / 题材卡片
│   ├── 联网搜索            内置 Tavily（可切 SearXNG/Bing），写作页搜素材
│   ├── 世界→小说播种        WorldData 直接播种大纲/角色/世界观，零 LLM 调用
│   └── LLM 用量统计        实时聚合 + hour/day 时间窗持久化 + 费用估算
└── WebUI              统一前端（Tab 切换小说/世界控制台，含 TokenStats 页）
```

## 🚀 快速开始

### 1. 构建（Go 1.22+）

```bash
go build -o worldsim .
# 产物：单个 ~10MB 二进制，零依赖
```

### 2. 配置 LLM（api.json，放在程序目录）

```json
{
  "base_url": "https://your-api-endpoint/v1",
  "model": "your-model",
  "api_key": "YOUR_API_KEY",
  "http_timeout_seconds": 300,
  "model_tiers": {
    "fast": "your-fast-model",
    "normal": "your-normal-model",
    "premium": "your-premium-model"
  }
}
```

### 3. 启动

```bash
./worldsim /path/to/data-dir
# 统一前端入口: http://localhost:48092（推荐，浏览器式导航）
# 世界模拟服务: http://localhost:48091
# 小说创作服务:  http://localhost:48090
```

浏览器打开 `http://localhost:48092` 即浏览器式导航外壳：首页点击进入小说创作或世界模拟控制台，建世界（选主题包或研究引导）→ 初始化 → 开循环 → 等就绪 → 生成小说。

### 4. 用 API 驱动（一行跑通）

```bash
# 创建世界（主题包+一句话设定 → LLM 自动生成世界书）
curl -X POST localhost:48091/api/worlds/create \
  -H 'Content-Type: application/json' \
  -d '{"name":"青岚界","theme":"经典修仙","desc":"山村少年捡到残破剑胚"}'

# 初始化（按世界书生成主角/NPC/地点）
curl -X POST localhost:48091/api/world/init

# 后台持续运行（到就绪自动停）
curl -X POST localhost:48091/api/world/loop \
  -H 'Content-Type: application/json' -d '{"action":"start","days":1000}'

# 查就绪 → 生成小说 → 读章节
curl localhost:48091/api/world/readiness
curl -X POST localhost:48091/api/world/novel/generate
curl localhost:48091/api/world/novel/chapter/1
```

## 📚 API 一览

| 分组 | 接口 |
|---|---|
| 世界管理 | `GET /api/worlds` `POST /api/worlds/create` `POST /api/worlds/select` `POST /api/world/init` |
| 状态 | `GET /api/world/state` `GET /api/world/chronicle` `GET /api/world/memories` `GET /api/world/foreshadows` |
| 模拟 | `POST /api/world/sim/day` `POST /api/world/loop`(start/stop/status) `GET /api/world/readiness` |
| 决策 | `GET /api/world/decisions` `POST /api/world/decisions/{id}` |
| 时间回退 | `GET /api/world/snapshots` `POST /api/world/snapshot` `POST /api/world/rewind` |
| 小说 | `POST /api/world/novel/generate` `GET /api/world/novel` `GET /api/world/novel/chapter/{num}` |
| 主题包 | `GET /api/worldbooks/themes` |
| 统计 | `GET /api/world/token_stats` `GET /api/world/sim/thinking` |

## 🤖 AI 客户端接入（MCP）

WorldSim 提供标准 MCP Server（`worldsim-mcp/server.py`，零依赖），可接入 Codex CLI / Trae / Claude / Cursor：

- Codex：`worldsim-mcp/codex.md`（config.toml / .mcp.json / codex mcp add）
- Trae：`worldsim-mcp/trae.md`（MCP 面板 stdio 添加）
- 仓库根 `AGENTS.md` 是 Codex 的协作指南

## 📥 下载

多平台二进制 + 插件包 + MCP Server 全部见 [Releases](https://github.com/2033121/worldsim/releases)（最新 **v1.3.1**）：

- Linux amd64 / arm64（tar.gz）
- Windows amd64（zip，含 run.bat 一键启动）
- macOS amd64 / arm64（tar.gz）
- `worldsim_plugin_v1.3.1.zip`（Operit 插件包，直接导入）
- `worldsim-mcp-v1.3.1.zip`（Codex/Trae/Claude MCP Server，零依赖）

> 🔄 **发布全自动**：打 `v*` tag 即触发 GitHub Actions 交叉编译 5 平台 + 自动组装插件包/MCP 包，共 7 个资产，无需手动上传。

## 🔌 Operit 插件包

本仓库是核心源码。打包好的 **Operit 插件包**（沙盒包 25 工具 + Skill + WebUI 控制台 + 15 主题包 + 886 条风格素材 + 验收清单）以 zip 形式随 Release 分发（CI 自动组装），可直接导入 Operit 使用。

## 📂 目录说明

```
worldsim/
├── main.go            服务入口（三端口：48092统一 / 48091世界模拟 / 48090小说）
├── worldapp/          统一前端源码（Svelte+Vite+Tailwind+DaisyUI，浏览器式导航外壳）
├── worldweb/          世界模拟控制台前端源码
├── frontend/          小说创作前端源码
├── uiteg/             统一前端构建产物（embed）
├── wsweb/             世界控制台构建产物（embed）
├── static/            小说前端构建产物（embed）
├── internal/
│   ├── engine/        State Engine（事件溯源/提案/重放/软规则）
│   ├── sim/           多Agent模拟器（事件/决策/NPC/伏笔/记忆/快照/就绪度）
│   ├── worldbook/     世界书解析 + 主题包 + LLM世界书生成
│   ├── research/      题材研究智能体（热门题材/世界书方向/题材卡片）
│   ├── attach/        世界参考资料附件管理
│   ├── search/        SearXNG 联网搜索后端
│   ├── llm/           分层模型调用 + token 追踪 + 前缀缓存统计 + 工具调用
│   ├── novel/         小说写手（素材投喂/章节规划/去AI味铁律）
│   └── config/        配置加载
├── worldbooks/        世界书池（模板+主题包+实例）
└── docs/              设计文档
```

## ⚠️ 安全与版权

- **密钥**：`api.json` 已被 `.gitignore` 排除，**绝不提交**。示例请用占位符。
- **数据**：`worlds/` `storys/` 为个人世界数据，不入库。
- **素材库**：`material/`（886+ 条真实网文风格示范，每条标注来源书名+章节，仅风格参考）已开源入库。

## 📜 License

[MIT](LICENSE)
