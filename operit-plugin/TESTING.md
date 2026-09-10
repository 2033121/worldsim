# WorldSim 插件包 · 验收清单

> 装完插件包后，照此清单跑一遍（约 10-20 分钟，含 LLM 生成等待），全部通过即安装成功。
> 全程只使用 **worldsim 沙盒包工具** 和 **WebUI**，不需要碰命令行（除第 1 步启动服务）。

## 前置

- [ ] Operit 已启用 `worldsim` 沙盒包（工具箱里可见，26 个工具）
- [ ] `skills/worldsim/SKILL.md` 已就位
- [ ] 已配置 LLM（WebUI 操作台填好 Base URL/模型/Key 并点"应用"）

## 第 1 步：服务自检（2 分钟）

1. 启动服务：让 AI 执行 `sh /sdcard/Download/Operit/plugins/worldsim/run.sh`
   （或自己终端跑），看到 `WorldSim 已启动` 即成功
2. AI 调用 `worldsim:world_webui` → 应返回 `http://127.0.0.1:48091`
3. AI 调用 `worldsim:world_list` → 应返回空列表（新装）或已有世界
4. AI 调用 `worldsim:world_themes` → 应返回 15 个主题包

**预期**：✅ 全部通过

## 第 2 步：创建世界（3-5 分钟，含 LLM 生成）

1. AI 调用 `worldsim:world_create`：
   - name: `验收·末世孤岛`
   - theme: `末世废土`
   - desc: `丧尸爆发十年后，一座孤岛避难所，淡水净化厂是生命线，最近取水口有人失踪`
2. 若返回"请求超时"提示 → **属正常**（LLM 生世界书约 1-2 分钟）
3. 等 1-2 分钟后调用 `worldsim:world_list` → 应看到该世界 day=0

**预期**：✅ 世界出现，且 `wsdata/worldbooks/验收·末世孤岛.md` 是末世主题（非默认模板）

## 第 3 步：初始化（2-3 分钟，含 LLM 生成）

1. AI 调用 `worldsim:world_init`（不传主角名）
2. 等 1-2 分钟后调用 `worldsim:world_state`

**预期**：✅ 主角职业贴合末世（如"净化塔维修工/幸存者"），有 2-4 个 NPC，有全局事件

## 第 4 步：跑模拟（后台）

1. AI 调用 `worldsim:world_loop_start`（days=100）
2. 等 1 分钟后调用 `worldsim:world_loop_status` → running=true 且 day 在增长
3. 调用 `worldsim:world_readiness` → 返回 4 项指标

**预期**：✅ 循环运行、day 增长、就绪度指标可见（ready 可能 false，正常）

## 第 5 步：时间回退（可选，1 分钟）

1. 调用 `worldsim:world_snapshot`（手动存档）
2. 调用 `worldsim:world_snapshots` → 应看到刚才的存档
3. 调用 `worldsim:world_rewind`（day=1）→ 世界回退到 Day1
4. 重新 `worldsim:world_loop_start` 继续跑

**预期**：✅ 回退后 day=1，重新演化

## 第 6 步：决策翻案（等出现岔口后）

1. `worldsim:world_decisions` → 有队列后
2. `worldsim:world_decision_resolve`（id + choice=A/B/C）→ 返回"改选成功"

**预期**：✅ 决策可改选

## 第 7 步：生成小说（就绪后，5-10 分钟）

1. `worldsim:world_readiness` → ready=true（段落≥3/素材≥12/伏笔回收≥1/张力≥0.4）
2. `worldsim:world_novel_generate` → 若超时提示"后台写作中"，等 2-3 分钟
3. `worldsim:world_novel_list` → 应有章节
4. `worldsim:world_novel_chapter`（num=1）→ 返回正文

**预期**：✅ 小说章节内容与末世世界设定一致（有生活质感，非 AI 腔）

## 第 8 步：WebUI 人工检查（2 分钟）

浏览器打开 `http://127.0.0.1:48091`：
- [ ] 世界列表/状态/编年史显示正常
- [ ] 🎯 决策 tab 可翻案
- [ ] 🔁 循环开关可用
- [ ] ⏪ 时间回退面板可见
- [ ] 📖 小说 tab 可阅读
- [ ] 📚 世界书 tab 显示世界书全文

## 全部通过 → 🎉 插件包安装成功！

## 常见失败点

| 现象 | 原因/处理 |
|---|---|
| 所有工具报"服务未启动" | 没跑 run.sh，或服务崩了（看 wsdata/run.log） |
| 创建世界超时后 world_list 没有 | LLM 中转站挂了 → 等网络恢复重试；或看 run.log 是否"世界书生成失败" |
| 初始化后 world_state 无实体 | 等 1-2 分钟重查（后台生成中） |
| 循环不动 | 中转站慢/超时，单日 150s 兜底后会继续；或 world_rewind 回退 |
| 就绪度一直不到 | 修仙/长周期世界要多跑；看 4 项指标哪项缺 |