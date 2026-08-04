# WorldSim 统一前端 — 开发进度与问题记录

> 长期项目进度日志。每次推进记录进展、遇到的问题与解决方案。
> 对应计划：`.trae/specs/unified-frontend/PLAN.md`

---

## 2026-08-05 — Phase 1+2 完成：统一入口 + 浏览器外壳骨架

### 进展
- **Phase 1 后端统一入口**（已完成）：`main.go` 新增 `:48092` 统一前端入口，含：
  - serve 统一 SPA（`uiteg/` 构建产物，embed 进二进制）。
  - API 网关：`/api/novel/*` → 48090（去前缀），`/api/world*`、`/api/worlds`、`/api/research`、`/api/system`、`/api/worldbooks`、其余 `/api/*` → 48091。
  - 统一前端只与 :48092 同源通信，无 CORS/跨域。
  - curl 实测三个路由均 200：`/api/worlds`、`/api/world/state`、`/api/novel/version`。
- **Phase 2 浏览器外壳骨架**（已完成，`worldapp/`）：
  - 状态模型 `lib/browser.js`：多标签、每标签独立历史栈、书签、命令候选，localStorage 持久化。
  - 组件：`TabBar`（多标签新建/关闭/切换/右键关闭）、`AddressBar`（点击编辑回车跳转）、`BookmarkBar`、`CommandPalette`（Ctrl+K）、`StatusBar`、`ThemeToggle`（明暗切换）。
  - 页面：`HomePage`（首页 + 快捷入口卡片）、`WorldApp`/`NovelApp`（iframe 嵌 48091/48090 过渡）。
  - 主应用 `App.svelte`：外壳 + 页面渲染 + 键盘快捷键 + toast。
- **右键关闭标签**、**地址栏可编辑**、**命令面板 Ctrl+Enter 新标签** 等交互已实现。

### 遇到的问题与解决方案
1. **embed 目录拼写错误**：`main.go` 中 `//go:embed uiteg` 但运行时读取 `uitg`（笔误），会导致 `fs.Sub` 启动失败。已统一改为 `uiteg`。
2. **`get` 从错误模块导入**：`App.svelte` 从 `svelte` 导入 `get` 报错，改为从 `svelte/store` 导入。
3. **`setText` 方法不存在**：`BrowserChrome.svelte` 中误用 `statusText.setText()`，应为 store 的 `.set()`，导致首页崩溃白屏。已修复并重新部署。
4. **npm install 网络慢**：首次安装依赖耗时约 11 分钟（国内网络），后缘构建恢复正常。

### 验证
- `npm run build` 成功，产物输出到 `uiteg/`。
- `go build` 通过；Linux 交叉编译 + `docker cp` 部署，容器 healthy。
- 浏览器实测（browser 子智能体）：首页、标签打开/切换、世界模拟 iframe、小说创作 iframe、命令面板、书签+持久化 全部 **PASS**。
  - 前进/后退按钮在"新建标签"场景下显示禁用属**正常**（新标签历史栈仅 1 条，无可后退），需在单标签内多次导航才能验证。

### 待办（下一步）
- Phase 3 原生集成世界模拟（worldweb 迁移，替换 iframe）。
- Phase 4 原生集成小说创作（frontend 迁移）。
- Phase 5 视觉统一 + 页面过渡动画。
- Phase 6 生产加固 + 端到端验证 + 文档。

---