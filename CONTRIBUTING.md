# 贡献指南

感谢你愿意为 WorldSim 贡献！不管是提 bug、加功能、写文档还是修代码，都欢迎。

## 项目结构速览

```
main.go                 # 服务入口：REST API + WebUI（embed）
internal/engine/        # 事件溯源 State Engine（世界状态/事件日志/回放）
internal/sim/           # 多Agent模拟器：事件/决策/NPC/伏笔/记忆/快照/就绪度
internal/worldbook/     # 世界书解析 + 15 主题包 + LLM 生成世界书
internal/novel/         # 小说写手：素材投喂 / 去AI味 / 章节生成
internal/llm/           # LLM 调用 + token 统计 + 缓存命中
wsweb/ static/          # 前端（编译进二进制）
worldsim-mcp/           # 标准 MCP Server（Codex/Trae/Claude 接入）
docs/                   # 设计文档
```

## 开发环境

- Go 1.22+（本项目**零第三方依赖**，纯标准库，好维护！）
- Python 3.8+（仅 MCP server 用，也是零依赖）

## 贡献流程

1. **先开 Issue 讨论**（尤其新功能，避免白做）
2. Fork 仓库 → 新建分支：`feat/xxx` 或 `fix/xxx`
3. 写代码 + 测试（`internal/**/*_test.go`）
4. 本地验证：
   ```bash
   go vet ./...
   go test ./... -count=1
   python3 worldsim-mcp/server.py --selftest   # 改了 MCP 时
   ```
5. 提交 PR（用 `.github/PULL_REQUEST_TEMPLATE.md` 模板）

## 编码约定

- 保持**零第三方依赖**——新功能优先用标准库实现，这是本项目最大卖点
- 新增 API 记得：① 注册路由 ② 前端操作台加按钮 ③ `worldsim-mcp/server.py` 加工具 ④ 文档更新（四件套！）
- 错误处理：长耗时操作（LLM 生成）必须用独立 context，**不要**挂在 `r.Context()` 上（宿主超时会取消）
- 中文注释为主，命名英文

## 提交信息规范

```
feat: 新功能
fix: 修 bug
docs: 文档
refactor: 重构
ci: CI 相关
test: 测试
```

## 安全

- **绝对不要**提交 `api.json`、任何 `.key`/`.pem`/token
- 发现漏洞看 `SECURITY.md`，别在 Issue 里贴密钥

## 测试思路

- 纯逻辑（事件解析/伏笔匹配/就绪度计算）→ 单元测试
- API 层 → `internal/httpapi/project_compat_test.go` 参考
- 全链路 → 按 `TESTING.md` 验收清单手动跑（在 Operit 插件包目录）