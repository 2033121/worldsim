# WorldSim HTTP API 文档

基础地址：`http://127.0.0.1:48091`（默认）
所有接口返回 JSON。

## 世界管理

### GET /api/worlds
列出所有世界及当前激活世界。
```json
{"worlds": [{"name": "灰潮纪元·孤岛净化厂", "day": 6, "active": true, "has_data": true}], "current": "灰潮纪元·孤岛净化厂"}
```

### POST /api/worlds/create
创建新世界。`theme`（主题包名）时由 LLM 自动生成世界书；可配 `desc` 一句话设定；或传 `worldbook` 指定已有世界书。
```json
{"name": "青岚界·灵剑山", "theme": "经典修仙", "desc": "山村少年测出灵根，拜入剑宗"}
```
> 世界书生成 1-2 分钟；宿主超时则后台继续（pending 机制），稍后 GET /api/worlds 确认。

### POST /api/worlds/select
```json
{"world": "青岚界·灵剑山"}
```

### POST /api/world/init
初始化当前世界（生成主角/NPC/地点）。`protagonist` 可不传，LLM 自拟。
```json
{"protagonist": "沈燃"}
```

## 状态查询

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | /api/world/state | 世界状态：Day/天气/张力/主角/NPC/全局事件 |
| GET | /api/world/chronicle | 编年史（时间线，最近 60 条） |
| GET | /api/world/memories | 各角色记忆 |
| GET | /api/world/foreshadows | 未回收伏笔 |
| GET | /api/world/sim/thinking | 主角最近一次三问决策推理 |
| GET | /api/world/token_stats | LLM 调用统计（次数/缓存命中/tokens） |
| GET | /api/world/worldbook | 当前世界世界书全文 |
| GET | /api/worldbooks/themes | 全部主题包列表 |
| GET | /api/world/readiness | 模拟就绪度（段落/素材/伏笔/张力 4 指标） |

## 模拟控制

### POST /api/world/sim/day
手动跑 N 天（上限 30）。`mode`: auto(默认)/scene/summary/skip
```json
{"days": 5, "mode": "auto"}
```

### POST /api/world/loop
后台持续运行直到目标 Day 或素材就绪（自动停）。
```json
{"action": "start", "days": 1000, "mode": "auto"}
{"action": "stop"}
```

### GET /api/world/loop
查询循环状态：running/current_day/target_day/ready。

## 决策翻案

### GET /api/world/decisions
岔口决策队列（AI 已代决，可改选）。每条含 id/情境/选项/AI 选择与理由/状态。

### POST /api/world/decisions/{id}
用户改选方向（覆盖 AI 代决）。
```json
{"choice": "B"}
```

## 时间回退（快照制）

### GET /api/world/snapshots
全部回退锚点（Day/说明/时间）。

### POST /api/world/snapshot
手动存档。
```json
{"reason": "第100天里程碑"}
```

### POST /api/world/rewind
回退到 ≤day 的最近快照，重新演化新分支（先停循环）。
```json
{"day": 100}
```

## 小说

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | /api/world/novel | 章节计划 + 已导出章节 |
| POST | /api/world/novel/generate | 生成/续写章节（1-3 分钟/章，超时后台进行） |
| GET | /api/world/novel/chapter/{num} | 读取第 N 章正文 |

## 小说服务（48090）

- GET / → 小说阅读前端
- GET /api/novel/... → 章节/卷/元数据
- POST /api/search → 联网搜索素材（写作页「🔍 搜素材」）。Tavily 内置后端。
  ```json
  {"query": "唐代长安城布局", "max": 5, "language": "zh"}
  ```
  ```json
  {"results": [{"title": "...", "url": "...", "content": "...", "engine": "tavily"}]}
  ```
- POST /api/world/seed-novel → 世界→小说直接播种（零 LLM 调用）。把已模拟世界的世界书/角色/势力/近期事件播种成小说大纲与设定。
  ```json
  {"world_id": "灰潮纪元·孤岛净化厂", "project_name": "孤岛净化厂外传", "language": "zh"}
  ```
  ```json
  {"project_name": "孤岛净化厂外传", "world_name": "...", "character_count": 5, "worldview_count": 12, "outline_chapter_count": 8, "day": 120}
  ```

## LLM 用量统计

- GET /api/llm/stats → 全局实时用量总览（进程启动时间 + 各环节聚合 + 缓存 + 费用估算）
- GET /api/llm/stats/history?window=hour|day → 按时间窗的 token 消耗历史（跨重启持久化）

## 错误格式

```json
{"error": "错误描述"}
```

## 超时与容错

- 长操作（世界书生成/初始化/小说生成）用独立 context，宿主断开不取消
- 世界书生成失败自动重试 3 次，仍失败回退模板
- LLM 调用超时 dry-run 兜底（循环自愈重试）