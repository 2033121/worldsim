# Changelog

本项目所有重要变更都记录在此。格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，版本遵循 [SemVer](https://semver.org/lang/zh-CN/)。

## [1.10.1] - 2026-09-12

### 🩺 推理模型空正文陷阱（真实排障产物）
配置世界模拟的真实 LLM 中转站时踩到：`deepseek/deepseek-v4.1-flash` 这类**带思考通道**的模型会先消耗 `max_tokens` 做推理，
`max_tokens=8192` 时素材规划实测 `finish_reason=length` 且 `content` **完全为空**（推理吃掉全部预算），
上层只看到 `规划 JSON 解析失败: unexpected end of JSON input` 这种难懂报错。

- **可操作报错**：`internal/llm` 在同步/流式两条路径上区分「正文为空 + finish=length」与「真的空响应」，
  直接报出 `模型把 max_tokens 用尽在思考上……请提高 max_tokens（≥32000）或加 extra_body {"enable_thinking": false}`；
  同时保留原有的「正文空但思考存在且正常结束 → 用思考内容兜底」行为（仅当不是被截断时）
- **`extra_body` 支持**：`api.json` 新增 `extra_body`（任意键值原样并入每次 `chat/completions` 请求体，
  如 `{"enable_thinking": false}` 关闭思考通道）——推理模型长 JSON 任务从此可显著提速并避免空正文
- **场景补名**：素材规划里模型常只填 `time`（morning/dusk/night）而把场景 `name` 留空，
  `clampScenes` 现按时间位确定性补 `晨间/黄昏/夜半`（工坊表格与素材匹配不再出现空白行）
- 文档：中英 README 的 `api.json` 段补 `max_tokens`/`extra_body` 说明与推理模型提示

### ✅ 验证（真实链路）
- 直连中转站对照实验：同提示同模型 `max_tokens=8192`（不关思考）→ `finish=length, content=0`；
  `enable_thinking=false` → `finish=stop, content=6756 字符合法 JSON`；`max_tokens=32000` → 7740 字符合法 JSON
- 通过 worldsim 自身链路重跑素材规划（`extra_body` 生效）：66s 产出 plan（12 人物/8 怪物/3 场景/16 地图块/24 物品），
  场景名自动补为 `晨间/黄昏/夜半`；单张场景图生成落盘 1.78MB 并 `GET /art/scene-1.png` 200 直出，
  `/api/game/status` 的 `scene` 字段随之指向该素材
- 单测新增：`mergeExtraBody`（并入/覆盖/非法输入原样返回）、提示文案可操作性、`clampScenes` 补名三态

## [1.10.0] - 2026-09-12

### 🛠 全面可用性轮（审计 → 调研 → 分类改进）
> 审计与调研记录见 `docs/可用性审计-v1.10.md`（含 A1/A2 两个致命 bug 的实测取证与参照项目：AI Dungeon 系 undo / chasm_client 角色状态 / SillyTavern 分享卡文化）

### 🐛 致命修复（v1.9.0 遗留，页面 JS 层从未被真实浏览器验证过）
- **`/game` 页提交必然失败**：`post()` 把路径拼成 `/api/game/action.`（尾点多一个点）→ 落到 `GET /` 兜底返回 405——实测证实，页面所有回合提交 100% 失败；改为显式传参的正确路径
- **所有行动变等待**：`send()` 先清空输入框再让 `post()` 重读该值 → 提交的永远是空字符串，服务端按「原地等待」处理；改为闭包传值
- **CSS 破损规则** `'.chip b`（多余引号）导致面板数值高亮失效；**场景横幅竖图缩成窄条**（body 为 flex column 时 `margin:0 auto` 的 flex item 收缩为内容宽——布局级 bug，居中行容器统一补 `width:100%`，并用固定 `height+object-fit:cover` 全宽铺满）

### ✨ 新增（可玩性/功能补全）
- **撤销上一步**：回合开始前深拷贝状态进 `prev`（随 game.json 落盘，重启可撤），`POST /api/game/undo` 单步回滚数值/背包/好感/位置/倒下位并去掉该回合日志；借鉴 AI Dungeon 系 Retry/Rewind
- **倒下状态机**：HP 触底 → `downed=true`（UI 红色横幅警示）；等待回合回血自动恢复意识；裁判 prompt 感知低血量危机纪律（HP=0 由代码判定，LLM 不许宣布死亡）
- **存档导出/导入**：`GET /api/game/export`（下载 game.json）+ `POST /api/game/import`（校验+clamp 收数：属性/好感/等级/倒下位全部收进合法域）；两个前端各配 💾/📥 按钮
- **世界卡前端入口**：统一外壳控制台新增「🃏 世界卡」标签（CardPanel：导出可选附游玩进度 + 导入 zip 自定义命名）——v1.9.0 只有 curl 的功能终于有 UI
- **WI 动态情报透明化**：`/api/game/status` 增 `wi_hits`（最近回合命中的 W1 关键词组），游玩页显示「🔎 情报」chips——lore 触发从服务端黑盒变成玩家可感知
- **等待回合推进世界时钟**：wait 后引擎 Day +1 并落盘（模拟器同款模式）——游玩期间场景横幅 day%3 终于会轮转
- **回合单飞 + 长锁重构**：`internal/game` 改为「短锁快照 → 无锁 LLM → 短锁提交」，回合期间 `/api/game/status|log` 不再被挂起（此前最长阻塞 300s）；并发回合立即得到明确错误
- **`/api/game/status` 增 `world` 世界名**；Roll 属性校验（裁判点名的属性不在面板时按无修正，不再暗中 -5）；plan.json 渲染层 10s TTL 缓存（一次 status 最多 5 次读盘收口）

### 🎨 游戏感 / UX / 前端
- d20 检定行滚动入场动画 + 失败抖动；HP 条按血量变色（绿/金/红）+ XP 进度条；**删除检定失败把主角换装成怪物的误用逻辑**；输入历史 ↑/↓（两个前端同步）
- 已开局时「开始/重开」先确认（防误触清进度）；开局成功后自动刷新场景/地图/crew；`worldChip` 显示「世界：名 · Day n」；favicon；studio 规划生成失败立即报错、任务轮询单例化防叠加、死代码清理
- **消除双控制台漂移**：`:48091/` 改为直出 uiteg 统一外壳（`GET /{path...}` + 指纹资产长缓存 + 未知路径回壳），删除 wsweb 里的旧版构建（此前 48091 与 48092 是两个不同前端）；Svelte GamePage 对齐全部新能力（关系徽章/NPC 地图计数/倒下横幅/撤销/存档）

### ✅ 验证
- 单测新增：单步撤销（回滚/用完即弃/持久化）、倒下状态机（触发/等待恢复）、回合单飞（并发窗口实测）、存档导入校验（非法 JSON 拒收/越界收数）、Roll 未知属性无惩罚——18 包全绿 + gofmt/vet 干净
- **真实 E2E**（新二进制 + 真实 LLM）：wait 回合 turn=6/day=2 且引擎日同步；undo 后 turn=5 日志回退；存档导出 6.5KB → 改 gold+777 导回生效=811；非法存档拒收；downed 存档导入即出红色横幅（浏览器截图取证）；`:48091/` 返回统一外壳 + 未知路径回壳；世界卡 3.7MB 仍通

## [1.9.0] - 2026-09-12

### ✨ 新增（游玩页世界观感 + 世界卡：同类项目调研落地）
> 调研结论见 `docs/同类项目调研-v1.9.md`（SillyTavern World Info / talemate / ST 状态追踪扩展生态）

- **W1 动态条目引擎**（借鉴 SillyTavern World Info 最小实用集）：世界书新增 `## W1 动态条目` 段（`- keys => 情报`，中英文逗号分隔），游玩模式每回合把**最近 6 条回合文本+本回合输入**拼成扫描缓冲，命中（**中文子串匹配**——ST 文档明示 whole-word 毁 CJK）或 sticky（命中后保持 3 回合，重复不刷新计时）即注入裁判+叙述者共享上下文；总预算 1200 字符；解析进 `worldbook.WIEntries`，激活器 `worldbook.ActivateWI` 纯函数可测
- **好感度追踪**（借鉴 BetterSimTracker 生态）：`game.json` 增 `relations:{NPC:-10..10}`；裁判申报 JSON 增可选 `relations:[{name,delta}]`，代码收数钳制；游玩面板角色头像带好感度徽标
- **游玩页视觉升级**（`/game` 终端页 + GamePage.svelte 同步）：①场景横幅按世界时钟 day%3 轮转晨/暮/夜三联 ②在场角色头像条（characters sprite + 好感度徽标）③背包物品图标网格（plan/套件素材按名字精确/子串/hash 匹配）④**确定性地图视图**：引擎实体地点集合 → 排序+snake 网格布局，tile 序号按 plan kind 语义 → 地名关键词 → hash 三级兜底（同 state 两次渲染逐格一致，单测锁定），主角格高亮
- **世界卡导出/导入**（模板生态闭环）：`GET /api/world/card[?with_game=1]` → zip（worldbook.md + art/plan.json + sheets + sprites，编年史/记忆不导出——隐私边界）；`POST /api/worlds/import` multipart 上传 → 重名自动 -2 后缀落盘新世界并选中；stdlib archive/zip 零依赖
- **status 扩展字段**：`scene` / `portraits[{name,img,relation}]` / `item_icons[{name,img}]` / `map{side,cells,hero,npcs,base}`——素材缺失时字段缺省优雅降级
- `worldbooks/_template.md` 增 W1 段用法说明；`docs/同类项目调研-v1.9.md` 全量调研记录

### ✅ 验证
- 单测全绿：`ActivateWI`（子串命中/sticky 保持与衰减/预算裁剪）、好感度（申报/越界钳制/持久化）、地图纯函数（关键词映射/hash 稳定性/素材匹配）
- **真实 E2E**（真回合×4）：W1 注入日志 `[游戏] WI 动态情报注入 3 条`（2 条 sticky 保持+1 条本回合命中）；裁判自动申报好感度 `童恒+1` 落盘；`/api/game/status` 返回 scene/map(3×3,7 地点,hero 定位)/portraits；世界卡导出 3.7MB（27 文件）→ 导入生成 `凡尘仙途测试-2`（worldbook+24 sprite+plan 落盘，可选中可玩）

## [1.8.0] - 2026-09-12

### ✨ 新增（美术工坊：世界书驱动的内建图片生成）
- **新包 `internal/art`**（零第三方依赖）：
  - **可插拔生成客户端** `imagegen.go`：OpenAI images 协议（中转站 gpt-image-2，UA 伪装头实测必需）+ PixelLab Pixflux 预留适配（noBackground 直出透明底）；退避重试 1s/3s/9s；配置存程序数据目录 `img.json`（key 只写不读、API 永远掩码回显，仓库零密钥）
  - **素材规划 Agent** `plan.go`：LLM 读世界书（A1-A4/A6/A7/B5 摘要 + 可选「美术设定」段 + 引擎实体名单）→ `art/plan.json`——12 人物 / 8 怪物 / 3 场景（晨暮夜）/ 16 地图块 / 24 物品，每条含名称/角色/中文外观速写/英文提示词；风格契约（16-bit 像素/厚描边/统一 #e8e8e8 底/禁文字）与调色板（含 palette_hex 色值）由代码层注入每条 prompt
  - **Go 原生裁剪管线** `crop.go`：sheet 网格切分 → 非背景紧致 bbox → 边缘 BFS 洪泛抠底（与 scripts/crop_grid.py 同构，stdlib image/png 实现）；tile 整格不抠底；空格占比校验
  - **任务编排** `jobs.go`：批量生成（每类 1 张 sheet，断点续跑）+ **单品重生成**（改某条 prompt 只出 1 图 1024x1024）+ 空格自动单品补齐 + `art/history.json` 生成留痕（追溯/重掷）
- **HTTP API**（48091，网关 48092 自动转发）：`GET/POST /api/art/config`（key 掩码）+ `POST /api/art/config/test` 连通试生成、`POST/GET/PUT /api/art/plan`、`POST /api/art/generate`（批量/单品，异步）、`GET /api/art/jobs[/{id}]`、`GET /api/art/assets`、`GET /api/art/history`、`GET /art/{file}`（本世界 sprites → 打包套件 fallback）、`GET /studio` 美术工坊页；网关新增 `/art` `/studio` 反代
- **游玩页接线**：`/api/game/status` 的 pixel 载荷优先用本世界 `plan.json` 素材（hero/monster/三类列表 + palette_hex），无规划回落 detectPixelTheme 打包套件——game.html 与 GamePage.svelte 零改动自动生效
- **前端**：`wsweb/studio.html`（自包含美术工坊页：服务配置+连通测试 / 规划表编辑 / 批量+单品生成 / 素材画廊，embed 进二进制）；统一外壳新增 `StudioPanel.svelte`（侧栏概览+生成入口）与「美术工坊」Tab（iframe 内嵌 /studio）
- **世界书模板**：`_template.md` 增可选「美术设定」段（视觉基调/主角外观/调色板/标志性场景——规划 Agent 最高优先）；LLM 生成世界书时自动附带 3~5 行美术段（worldbook_gen.go）

### ✅ 验证
- `go vet ./...` 通过；`internal/art` 单测全绿（合成 sheet 裁剪/空格检出/tile 整格/config 掩码与落盘/规划 JSON 容错与数量钳制/prompt 契约注入/httptest 伪中转站全流程含 UA/鉴权/b64/重试）
- **真实 E2E**（中转站实图）：规划 Agent 从世界书《九州·凡尘仙途》产出完整规划（主角林砚带中文速写+英文 prompt）→ 实生成 items sheet → 自动裁剪出 24 个透明 sprite（角 alpha=0/主体 alpha=255）→ `/art/item-1.png` HTTP 200 → `/api/game/status` 返回 custom pixel 载荷 → `art/history.json` 留痕；PUT 编辑回读一致；网关 /studio /art 200

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

## [1.7.0] - 2026-08-05

### ✨ 新增（像素套件 + 游玩页接线）
- **五大主题像素套件**（25 张 sheet → 315 个透明 sprite 小图）：每主题 characters(4x3=12) / monsters(4x2=8, 题材自拟) / scenes(晨暮夜三联=3) / maptiles(4x4=16 整格 tile) / items(6x4=24)——凑齐人物/怪物/背景/地图/物品全套；目录 `docs/art/pixel/<主题>/`
- **manifest 驱动通用裁剪器** `scripts/crop_grid.py`：按资产类型声明网格规格（cols×rows+是否抠底），自动象限切分→紧致 bbox→边洪泛抠底→逐图导出+contact-sheet；全量量化（palette 255 色）后总积 108MB→22MB，观感无损
- **游玩页自动换装**：`detectPixelTheme`（世界书标题/文件名/头部原文三层题材推断）→ `/api/game/status` 返回 theme + hero/monster/characters/monsters/scenes URL 组；`GET /pixel-art/{theme}/{file}.png` 磁盘直出（wsdata/art 先、docs/art 兜底）
- **前端接线**：`wsweb/game.html`（内置终端页，主角 sprite 常显 + 检定失败时主角换怪物像强调代价）与 worldapp `GamePage.svelte`（hero 图 + 主题标签，失败切换怪物 sprite）
- 生成/裁剪脚本链：`gen_pixel_suite.py`（5×5 sheet，中转站 UA+curl 兜底）→ `crop_grid.py`；端到端冒烟（主题识别 xianxia/chars 12/资产 200）通过

## [1.6.4] - 2026-08-05

### ✨ 新增
- **像素角色包（20 人，透明 PNG）**：5 张 2x2 sprite sheet（多智能体 crew / 修仙师 / 末世幸存者 / 西幻酒馆 / 星际远航）→ 按象限紧致裁剪 + 从边洪泛抠底转透明 → `docs/art/pixel-sprites/` 20 张独立 sprite + contact-sheet 合集；README 中英接入
  - 生成 `scripts/gen_pixel_sheets.py`（统一 16-bit 风/正面站姿/共同底色保证可裁剪）；裁剪+抠底 `scripts/crop_pixel_sheets.py`（坑修复：PIL 对 RGB 图赋 alpha 四元组静默无效，必须先转 RGBA）

## [1.6.3] - 2026-08-05

### ✨ 新增
- **包装图二批（7 张，gpt-image-2）**：5 张主题包竖版封面（修仙/末世/西幻/克苏鲁/星际，2:3 竖版+顶部标题留白）、`mcp-hub`（isometric 深蓝风：聊天/代码/移动三客户端到中央水墨世界再到工具卡输送带）、`rewind`（朱砂锚点分叉三重世界的快照回退概念图）
  - 生成脚本 `scripts/gen_art_extra.py`（延续 cookbook 结构化提示词与全图零文字纪律）
  - README 中文「包装图」区扩展；英文 README MCP 段加 `mcp-hub` 横幅

## [1.6.2] - 2026-08-05

### ✨ 新增
- **README 包装图组（`docs/art/`，gpt-image-2 生成）**：水墨卷轴 hero 顶图（世界→小说稿的项目隐喻）、游玩模式横幅（玉雕 d20 + 手账 + 纸片人排队出场）、修仙/末世/星际三联（一个引擎任意题材）、方形多智能体徽章（社交预览图）
  - 提示词对齐 openai-cookbook 图像指南（场景→主体→布局→风格→光照→显式排除约束无字/无水印），生产脚本 `scripts/gen_art.py`（可复跑/可扩展，不落密钥）
  - README（中/英）接入顶图与图集；图组同时可作 Release 页素材

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