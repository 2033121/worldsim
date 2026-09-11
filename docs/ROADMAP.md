# WorldSim ROADMAP（2026-09-11 建立导航）

> 本文件整合散落在 `UNIFIED_FRONTEND_PROGRESS.md` / `NOVEL_UI_OPTIMIZATION_PLAN.md` /
> `NOVEL_UI_OPTIMIZATION_PLAN` / CHANGELOG 中的开口项，作为后续额度投入的统一导航。
> 勾选状态以代码实测为准；勾选依据在括号内注明。

## 状态

- [x] Phase 3+4 双应用原生迁移 + 视觉统一（已完成，验证记录见 UNIFIED_FRONTEND_PROGRESS.md）
- [x] 小说创作功能与界面优化八项（NOVEL_UI_OPTIMIZATION_PLAN.md 清单 2026-09-11 全部按代码实况勾选）
- [x] v1.5.0 self-healing / Tavily 搜索 / 世界播种 / token 统计（见 CHANGELOG）
- [x] CI：Go build/vet/test + MCP 协议自检（2026-09-11 新增 **worldapp 前端构建步骤**，与 operit 主仓同款修复）

## 开口项（未开始）

1. **Phase 5 视觉打磨**：世界模拟 / 小说创作应用内部元素与宣纸风进一步统一（间距/圆角/字体）。需要真实浏览器会话对照三主题（宣纸/水墨夜/daisyUI 默认）逐项微调，适合留给带浏览器的会话处理。
2. **Phase 6 生产加固**：端到端 Playwright 套件入库（`worldapp/scripts/` 可复用），多标签并发、SSE 断线重连、超长编年史、Docker 镜像自动发布（已有 softprops/action-gh-release 动作，一致引用 CHANGELOG 语义版本）。
3. **世界状态 API 数据核对**：「测试-薪火之城」等世界自愈快照回放后的字段完整性核对（UNIFIED_FRONTEND_PROGRESS 待办遗留）。
4. **selfheal 增强 idea**：incidents.jsonl 提供 `/api/selfheal/incidents?limit=&level=` 过滤 + 前端面板分层展示——数据已有，纯增量功能。
5. **主题包扩展流水线**：`internal/worldbook/themes/` 新主题包接入一次验收 checklist（主题包数量、_template 覆盖度、就绪度四指标在全题材下回归）。

## 验证基线（2026-09-11 复核）

- `go build ./...` / `go vet` / CI 绿
- `worldapp` 本地 `npm run build` 成功且与 `uiteg/` 零 diff（产物已同步）
- 本 ROADMAP 之前的"跨智能体适配"工作发生在 operit / quant-oral-to-code / mhsc，非本仓库范围
