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

## 2026-08-05 — Phase 3+4 完成：双应用原生迁移 + 视觉统一

### 进展
- **Phase 3 世界模拟原生迁移**（已完成）：`apps/world/` 从 `worldweb` 迁移为原生 Svelte 组件。
  - `WorldApp.svelte` → `world/Console.svelte`（三栏布局：左操作台/中 Tab/右对话），替换 iframe。
  - 组件全集迁入 `apps/world/components/`，API 走统一网关同源 `/api/*`（→48091）。
  - header 精简为 `WorldHeader`（世界选择 + Day + 张力 + 联网搜索徽标 + 刷新）。
- **Phase 4 小说创作原生迁移**（已完成）：`apps/novel/` 从 `frontend` 迁移。
  - 项目列表/大纲/写作/记忆/关系/技能/伏笔/聊天/sse 全套迁入，API 改走 `/api/novel/*`（→48090）。
  - 路由由 hash 改为 store 驱动（`lib/router.js` 的 `currentPage`），适配浏览器外壳。
- **视觉统一**（已完成）：
  - `app.css` 补上缺失的 `@plugin "daisyui"`（原来缺失，导致外壳回退到 daisyUI 默认 deep dark 主题，与宣纸米色冲突）。
  - 硬编码「明亮宣纸水墨」主题变量（复用 worldweb/frontend 的暖色米色），使外壳 + 三个应用共享同一套颜色。
  - 新增 `[data-theme="night"]` 深色「水墨夜」主题，让 ThemeToggle 明暗切换真正生效。
  - 首页快捷卡片由蓝紫鲜艳渐变改为宣纸卡片（米色 + 印章「印/文」），与整体统一。

### 遇到的问题与解决方案
1. **小说服务 API 前缀重复**：对照组件的 url 形如 `/api/projects`（带前导 `/api`），而 `api.js` 的 `PREFIX='/api/novel'`，拼成 `/api/novel/api/projects` 导致 404。修复：`api.js` 拼接时去掉 url 前导 `/api`（`url.slice(4)`）。
2. **多面板 `toFixed` 崩溃白屏**：`ChroniclePanel`/`EventsPanel`/`MemoryPanel` 对可能为 `undefined` 的字段直接 `.toFixed()`，数据不完整时整页崩溃。改为 `(x ?? 0).toFixed(2)`。
3. **`StatusBar` 显示 `[object Object]`**：模板里写 `{statusText}` 而非 `{$statusText}`（未加 `$` 自动订阅），渲染了 store 对象本身。已改为 `{$statusText}`。
4. **浏览器子智能体误报"黑屏"**：子智能体用 URL 直接访问 `/world`、`/novel`，而统一前端是 store 驱动的 SPA 路由，URL 直连会落回首页；且其截图命中旧缓存（旧 JS 哈希）。改用全新 headless Playwright 会话 + 应用内点击导航 + 读取计算样式验证，确认为宣纸米色。

### 验证
- `npm run build` 成功（产物 `uiteg/`）；Linux 交叉编译 + `docker cp` 部署，容器 healthy。
- Playwright 全新会话实测：首页 / 世界模拟 / 小说创作 三页均正常渲染，**无 JS 报错、无 404**。
- 首页卡片背景 `linear-gradient(rgb(254,252,244), rgb(245,242,233))`（宣纸米色），外壳标签栏 `oklab(0.93…)`（浅米色），状态栏无 `[object Object]`。

### 待办（下一步）
- Phase 5 视觉打磨：世界模拟 / 小说创作 内部元素与宣纸风进一步统一（间距、圆角、字体）。
- 世界模拟「测试-薪火之城」等世界状态 API 数据核对。
- Phase 6 生产加固 + 端到端验证 + 文档。

---

## 2026-08-05 — 小说创作功能与界面全面优化

### 需求
1. 小说创作页功能增强：扩充内容显示区、修复失效按钮、优化按钮设计、实现正文编辑。
2. 首页界面优化：重新设计布局与视觉、优化交互体验。
3. 技术完备：稳定可靠、界面友好、可维护可扩展。

### 进展
- **失效按钮修复**：novel 路由已从 hash 迁移为 store 驱动（`lib/router.js`），但 Writing/Outline/Foreshadows 仍用 `window.location.hash`，点击无效。已统一改为 `navigate('outline'|'foreshadows'|'writing')`。
- **扩充内容显示区**：写作页正文容器高度由 `max-h-[calc(100vh-420px)]` 提升到 `max-h-[calc(100vh-360px)]`，章节列表同步调整，长文可完整滚动查看。
- **正文编辑（整章全文编辑）**：新增「📝 整章编辑」按钮，进入大 textarea 编辑态，复用后端 `POST /api/chapter/edit`（`operation=replace_text`，`old_text` 快照 / `new_text` 新稿）实现保存；取消恢复快照。与块级编辑互斥。编辑态顶部显示提示 + 实时字数。
- **按钮视觉统一**：修复整章编辑按钮重复图标（标签内置 📝 又加前缀）；保存按钮加深色图标；统一主操作 `btn-primary`/`btn-success`、次操作 `btn-ghost`、危险操作 `text-error`、分页 `join` 分组，`gap-2 flex-wrap` 保证窄屏换行。
- **i18n 补充**：新增 `writing.fullEdit.hint` 等中英文键。
- **首页重构**：Hero 区（印章「演/创」+ 标题 + 亮点标签）；两大核心入口卡片（世界模拟/小说创作，含印章、tagline、功能勾选列表、hover 上浮）；「全部页面」分组按钮（`btn-ink` 墨色风格）；底部快捷键提示。整体宣纸水墨风格。

### 遇到的问题与解决方案
1. **浏览器子智能体误报"黑屏"**：子智能体在服务器重启瞬间访问，命中 `ERR_EMPTY_RESPONSE`（`/api/novel/events`、`/api/worlds`），且浏览器会话恢复旧标签。待服务稳定后重测，宣纸米色正常渲染。
2. **Playwright `networkidle` 超时**：小说应用有 SSE/长轮询连接，`networkidle` 永不到达。改用 `domcontentloaded` + 固定等待。

### 验证
- `npm run build` 成功（产物 `uiteg/`）；Linux 交叉编译 + `docker cp` 部署，容器 healthy。
- Playwright 全新会话实测：首页（H1、2 卡片、3 全部页面按钮）渲染正确，点击「小说创作」进入应用成功，**无 JS 报错、无 404**。
- 集成浏览器 DOM 快照确认：首页两个卡片含印章「境/文」、功能勾选列表、全部页面按钮组全部渲染。
- 三服务均 200：`48092 /`、`48090 /api/projects`、`48091 /api/worlds`。
- 后端 `POST /api/chapter/edit`（`replace_text`）路由确认存在。

### 交付物
- 计划文档：`docs/NOVEL_UI_OPTIMIZATION_PLAN.md`
- 验证脚本：`scripts/verify_ui.py`
- 改动文件：`writing/Outline/Foreshadows/HomePage` 页面、`zh.js`/`en.js`、构建产物 `uiteg/`

---