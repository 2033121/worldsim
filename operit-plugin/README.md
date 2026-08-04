# WorldSim · Operit 插件包

> **版本**：1.2.0（2026-08-04）· 兼容：Operit（Android / ARM64）
> 多Agent世界模拟器 + 网文生产引擎：创建任意题材世界（15主题包一键生成世界书）→ 驱动世界演化 → 自动改写小说。
> **双控制入口**：AI 对话驱动（worldsim 沙盒包）+ 浏览器可视化控制台（WebUI）。

## 📦 包内文件

```
worldsim/  ← 插件包根目录（/sdcard/Download/Operit/plugins/worldsim/）
├── worldsim         世界模拟服务二进制（ARM64，内置 WebUI）
├── run.sh           一键启动/重启服务（自动处理 sdcard 无执行权限问题）
├── worldsim.js      Operit 沙盒包源码（25 个 AI 控制工具，安装到 packages/）
├── SKILL.md         AI 使用手册（安装到 skills/worldsim/）
├── README.md        本说明
├── TESTING.md       验收清单（装完照做一遍即可确认全部功能）
└── wsdata/          运行数据目录
    ├── worldbooks/  世界书池 + themes/ 15 个主题包
    ├── material/    跨题材写作素材库（886+ 条真实网文示范）
    ├── api.json.example  LLM 配置模板（run.sh 首次启动自动复制）
    └── worlds/      已创建的世界（新装为空）
```

## 🔌 安装（3 步）

1. **放本体**：把 `worldsim/` 整个目录放到 `/sdcard/Download/Operit/plugins/worldsim/`
2. **装 AI 工具**：把 `worldsim.js` 导入 Operit 沙盒包（工具箱 → 导入，或让 AI 执行
   `operit_editor:debug_install_js_package` → source_path=`/sdcard/Download/Operit/plugins/worldsim/worldsim.js`）
3. **装使用手册**：把 `SKILL.md` 放到 `/sdcard/Download/Operit/skills/worldsim/`
   （AI 自动识别触发词："写小说 / 开个新世界 / 跑模拟 / 修仙世界"等）

## 🚀 使用

### 第一步：启动服务（每次重启 Operit 后）
- 方式 A：让 AI 说"启动 WorldSim"（AI 会用终端跑 run.sh）
- 方式 B：自己开终端跑 `sh /sdcard/Download/Operit/plugins/worldsim/run.sh`
- 成功标志：输出 `WorldSim 已启动 → http://127.0.0.1:48091`

### 第二步：配置 LLM（仅首次）
- 浏览器打开 `http://127.0.0.1:48091` → 操作台 → LLM 模式选 **real** → 填 Base URL / 模型 / API Key → 点"应用"
- 之后重启服务自动读 `wsdata/api.json`，无需重复配置
- 换模型：WebUI 里改，或直接编辑 `wsdata/api.json`（`model` / `model_tiers.fast|normal|premium`）后重启

### 第三步：开始创作
- **跟 AI 说**（AI 全自动）："开个克苏鲁世界，设定是海港小镇接连有人失踪"
  → AI 依次调用 `world_create`（选主题包生成世界书）→ `world_init` → `world_loop_start` → `world_readiness` → `world_novel_generate`
- **自己动手**：浏览器打开控制台，点「＋」选主题包建世界 → 初始化 → 开循环 → 等就绪 → 生成小说

## 🎛️ 双控制入口

| 入口 | 谁用 | 能力 |
|---|---|---|
| **WebUI 控制台** `http://127.0.0.1:48091` | 人 | 世界状态 / **决策翻案改选** / 循环开始停止 / 就绪度 / **时间回退** / 编年史 / 记忆 / 伏笔 / 小说阅读 / 建世界（选主题包）/ LLM 配置 / token 统计 |
| **worldsim 沙盒包**（25 工具） | AI | world_list/create/init/run_day/loop_start/loop_stop/loop_status/readiness/decisions/decision_resolve/chronicle/memories/foreshadows/thinking/tokens/novel_list/novel_generate/novel_chapter/themes/webui/snapshots/snapshot/rewind |

## ✅ 快速验收（装完跑一遍，约 10 分钟）

详见 `TESTING.md`。一句话版：启动服务 → 创建末世世界 → 初始化 → 开循环 → 查就绪 → 生成小说。

## ❓ 常见问题

| 问题 | 解决 |
|---|---|
| AI 报"服务未启动" | 先跑 run.sh；或让 AI 帮跑 |
| 创建世界报"主题包不存在" | 先 `world_themes` 看可用主题包名 |
| 创建/初始化"请求超时" | **正常**：LLM 生成本来就要 1-2 分钟，宿主 HTTP 会先超时，但世界会在后台创建完成。稍等后用 `world_list` / `world_state` 确认 |
| 模拟卡住不动 | LLM 中转站抖动（单日失败会 dry-run 兜底）；`world_loop_start` 会自动重试。还不行就 `world_rewind` 回退重来 |
| 小说生成为空 | 先 `world_readiness` 确认就绪（段落≥3/素材≥12/伏笔回收≥1/张力≥0.4） |
| 端口 48091 被占用 | 改 `run.sh` 里的 `PORT` 和 `worldsim.js` 里的 `BASE`（两处一致即可） |
| 缓存命中率>100% | 已修复（1.2.0）；老版本升级即可 |
| 想在电脑上访问控制台 | 手机与电脑同 WiFi，浏览器打开 `http://<手机局域网IP>:48091`（手机 IP 在设置里查） |
| 换设备迁移 | 整个插件目录拷走 → 装好后跑 run.sh → 重配 LLM（数据全在 wsdata/worlds/） |

## 🗑️ 卸载

1. 停服务：`pkill -f worldsim`（或重启 Operit）
2. 删插件目录：`rm -rf /sdcard/Download/Operit/plugins/worldsim/`
3. 删沙盒包：工具箱里关掉/删除 `worldsim` 包
4. 删技能：`rm -rf /sdcard/Download/Operit/skills/worldsim/`

## ⚠️ 说明

- **素材库版权**：`material/` 中的示范片段用于写作风格参考，请勿直接商用原文。
- **模型名**：`api.json.example` 里是演示模型名，请按你的中转站/API 实际模型名修改。
- **依赖**：需要可用的 LLM API（中转站或官方）；`run.sh` 依赖 `curl`（Operit 终端环境自带）。
- **架构**：二进制为 ARM64；x86_64 设备需重新编译（源码见 WorldSim 项目仓库）。

## 📜 更新日志

- **1.2.0**：修复 token 缓存命中率双计 bug；世界书生成/初始化/模拟改为独立 context（客户端断开不再中断）；世界书生成加重试3次；时间回退机制上线；WebUI 控制台大升级
- **1.1.0**：插件包成型（沙盒包 22 工具 + Skill + WebUI + 15 主题包）
