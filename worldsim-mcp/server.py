#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
WorldSim MCP Server（零依赖，纯标准库）
========================================
把 WorldSim 世界模拟服务（http://127.0.0.1:48091）的 REST API 桥接成标准 MCP 工具，
供 Codex CLI / Trae / Claude / Cursor 等任何支持 MCP 的客户端调用。

用法：
  python3 worldsim-mcp/server.py          # stdio 模式（MCP 客户端拉起）
  python3 worldsim-mcp/server.py --selftest  # 自检

依赖：仅 Python 3.8+ 标准库（urllib/json/sys）。WorldSim 服务需先运行（见 run.sh）。
"""
import json
import sys
import urllib.request

BASE = "http://127.0.0.1:48091"
# 小说服务（48090）：承载 /api/world/seed-novel、/api/search、/api/llm/stats 等。
# process 由 worldsim 单进程同时监听 48090/48091/48092，故默认同机可直达 48090。
NOVEL_BASE = "http://127.0.0.1:48090"
PROTOCOL_VERSION = "2024-11-05"

# ---------------- 工具定义 ----------------
TOOLS = [
    {"name": "world_list", "description": "列出所有世界及当前激活世界、各自 Day", "inputSchema": {"type": "object", "properties": {}}},
    {"name": "world_select", "description": "切换当前操作的世界", "inputSchema": {"type": "object", "properties": {"name": {"type": "string", "description": "世界名称"}}, "required": ["name"]}},
    {"name": "world_create", "description": "创建新世界。传 theme（主题包名）时由 LLM 自动生成世界书；可配 desc 一句话设定；或传 worldbook 指定已有世界书", "inputSchema": {"type": "object", "properties": {
        "name": {"type": "string", "description": "世界名称"},
        "theme": {"type": "string", "description": "主题包名（可选）：经典修仙/都市异能/末世废土/西幻奇幻/恐怖灵异/历史王朝/星际科幻/无限诸天/武侠玄幻/美食种田/原始文明/军旅战争/洪荒神话/克苏鲁异界/系统流"},
        "desc": {"type": "string", "description": "一句话设定（可选）"},
        "worldbook": {"type": "string", "description": "已有世界书文件名（可选）"}
    }, "required": ["name"]}},
    {"name": "world_init", "description": "初始化当前世界：按世界书生成主角/NPC/地点（主角名可不传，LLM 自拟）", "inputSchema": {"type": "object", "properties": {"protagonist": {"type": "string", "description": "主角名（可选）"}}}},
    {"name": "world_state", "description": "查看当前世界状态：Day/天气/张力/主角/NPC/全局事件", "inputSchema": {"type": "object", "properties": {}}},
    {"name": "world_run_day", "description": "手动跑 N 天模拟（上限30）。mode: auto/scene/summary/skip", "inputSchema": {"type": "object", "properties": {
        "days": {"type": "number", "description": "跑几天（1-30，默认1）"},
        "mode": {"type": "string", "description": "auto(默认)/scene/summary/skip"}
    }}},
    {"name": "world_loop_start", "description": "后台持续运行模拟直到目标 Day 或素材就绪（自动停）", "inputSchema": {"type": "object", "properties": {
        "days": {"type": "number", "description": "目标天数（默认1000）"},
        "mode": {"type": "string", "description": "auto(默认)/scene/summary"}
    }}},
    {"name": "world_loop_stop", "description": "停止后台循环", "inputSchema": {"type": "object", "properties": {}}},
    {"name": "world_loop_status", "description": "查询循环状态：运行中/当前Day/目标Day/就绪", "inputSchema": {"type": "object", "properties": {}}},
    {"name": "world_readiness", "description": "模拟就绪度：素材够不够写小说（段落/素材/伏笔/张力四指标）", "inputSchema": {"type": "object", "properties": {}}},
    {"name": "world_decisions", "description": "岔口决策队列（AI 已代决，可翻案）", "inputSchema": {"type": "object", "properties": {}}},
    {"name": "world_decision_resolve", "description": "改选岔口方向（覆盖 AI 代决，写手按用户方向写）", "inputSchema": {"type": "object", "properties": {
        "id": {"type": "string", "description": "决策 ID（如 dec-12-1）"},
        "choice": {"type": "string", "description": "选项 ID：A/B/C"}
    }, "required": ["id", "choice"]}},
    {"name": "world_chronicle", "description": "编年史：世界发生过的所有事（时间线，最近60条）", "inputSchema": {"type": "object", "properties": {}}},
    {"name": "world_memories", "description": "各角色记忆", "inputSchema": {"type": "object", "properties": {}}},
    {"name": "world_foreshadows", "description": "未回收伏笔清单（防忘坑）", "inputSchema": {"type": "object", "properties": {}}},
    {"name": "world_thinking", "description": "主角最近一次三问决策推理", "inputSchema": {"type": "object", "properties": {}}},
    {"name": "world_tokens", "description": "LLM 调用统计（调用次数/缓存命中/tokens）", "inputSchema": {"type": "object", "properties": {}}},
    {"name": "world_novel_list", "description": "小说章节列表", "inputSchema": {"type": "object", "properties": {}}},
    {"name": "world_novel_generate", "description": "生成/续写小说章节（1-3分钟/章，可能超时则后台进行）", "inputSchema": {"type": "object", "properties": {}}},
    {"name": "world_seed_novel", "description": "把当前激活世界直接播种成小说项目（零 LLM 调用）：世界书/角色/势力/近期事件 → 小说大纲与设定。播种后到小说服务 48090 可查看/续写", "inputSchema": {"type": "object", "properties": {"project_name": {"type": "string", "description": "新小说项目名"}, "language": {"type": "string", "description": "小说语言 zh|en（默认 zh）"}}, "required": ["project_name"]}},
    {"name": "world_novel_chapter", "description": "读取某一章正文", "inputSchema": {"type": "object", "properties": {"num": {"type": "number", "description": "章节号"}}, "required": ["num"]}},
    {"name": "world_themes", "description": "列出全部主题包", "inputSchema": {"type": "object", "properties": {}}},
    {"name": "world_snapshots", "description": "时间回退锚点列表（快照：Day/说明/时间）", "inputSchema": {"type": "object", "properties": {}}},
    {"name": "world_snapshot", "description": "手动存档（保存完整快照供时间回退）", "inputSchema": {"type": "object", "properties": {"reason": {"type": "string", "description": "存档说明（可选）"}}}},
    {"name": "world_rewind", "description": "时间回退到 ≤day 的最近快照，重新演化新分支（会先停循环）", "inputSchema": {"type": "object", "properties": {"day": {"type": "number", "description": "回退到的 Day"}}, "required": ["day"]}},
    {"name": "world_webui", "description": "返回 WebUI 控制台地址（浏览器可视化操作）", "inputSchema": {"type": "object", "properties": {}}},
]

# ---------------- HTTP 桥接 ----------------
def api(method, path, body=None, base=BASE):
    req = urllib.request.Request(base + path, method=method)
    req.add_header("Content-Type", "application/json")
    data = json.dumps(body).encode() if body is not None else None
    try:
        with urllib.request.urlopen(req, data=data, timeout=300) as resp:
            return json.loads(resp.read().decode())
    except urllib.error.HTTPError as e:
        try:
            return json.loads(e.read().decode())
        except Exception:
            return {"_error": f"HTTP {e.code}"}
    except Exception as e:
        return {"_error": str(e)}

def ensure_alive():
    r = api("GET", "/api/worlds")
    if "_error" in r:
        raise RuntimeError("WorldSim 服务未启动（端口 48091 无响应）。请先运行 run.sh / run.bat 启动服务。")

def call_tool(name, args):
    """调用 WorldSim API，返回可直接展示的文本结果"""
    ensure_alive()
    if name == "world_list":
        d = api("GET", "/api/worlds")
        return json.dumps(d.get("worlds", []), ensure_ascii=False)
    if name == "world_select":
        return json.dumps(api("POST", "/api/worlds/select", {"world": args["name"]}), ensure_ascii=False)
    if name == "world_create":
        body = {"name": args["name"]}
        for k in ("theme", "desc", "worldbook"):
            if args.get(k):
                body[k] = args[k]
        r = api("POST", "/api/worlds/create", body)
        return json.dumps({**r, "hint": "世界书生成需1-2分钟；若返回超时，稍后 world_list 确认"}, ensure_ascii=False)
    if name == "world_init":
        body = {"protagonist": args["protagonist"]} if args.get("protagonist") else {}
        r = api("POST", "/api/world/init", body)
        return json.dumps({**r, "hint": "初始化生成主角/NPC/地点约1-2分钟；稍后 world_state 确认"}, ensure_ascii=False)
    if name == "world_state":
        d = api("GET", "/api/world/state")
        if "_error" in d:
            return json.dumps(d, ensure_ascii=False)
        t = d.get("world_level", {})
        hero = next((k for k, e in (d.get("entities") or {}).items() if (e.get("extra") or {}).get("role") == "protagonist"), None)
        out = {"day": d.get("day"), "weather": d.get("weather"), "tension": t.get("tension"),
               "hero": hero, "entities": d.get("entities"), "global_events": t.get("global_events", [])}
        return json.dumps(out, ensure_ascii=False)
    if name == "world_run_day":
        body = {"days": min(max(int(args.get("days", 1)), 1), 30), "mode": args.get("mode", "auto")}
        d = api("POST", "/api/world/sim/day", body)
        if "_error" in d:
            return json.dumps(d, ensure_ascii=False)
        last = (d.get("results") or [{}])[-1]
        out = {"day": d.get("day"), "events": last.get("events", []), "dialogue": last.get("dialogue", []),
               "thinking": d.get("thinking") or last.get("thinking", "")}
        return json.dumps(out, ensure_ascii=False)
    if name == "world_loop_start":
        body = {"action": "start", "days": int(args.get("days", 1000)), "mode": args.get("mode", "auto")}
        return json.dumps(api("POST", "/api/world/loop", body), ensure_ascii=False)
    if name == "world_loop_stop":
        return json.dumps(api("POST", "/api/world/loop", {"action": "stop"}), ensure_ascii=False)
    if name == "world_loop_status":
        return json.dumps(api("GET", "/api/world/loop"), ensure_ascii=False)
    if name == "world_readiness":
        d = api("GET", "/api/world/readiness")
        if isinstance(d, dict) and "ready" in d:
            d["hint"] = "素材就绪，可 world_novel_generate 生成小说！" if d.get("ready") else "继续跑模拟积累素材…"
        return json.dumps(d, ensure_ascii=False)
    if name == "world_decisions":
        d = api("GET", "/api/world/decisions")
        return json.dumps(d.get("decisions", []), ensure_ascii=False)
    if name == "world_decision_resolve":
        return json.dumps(api("POST", f"/api/world/decisions/{args['id']}", {"choice": args["choice"]}), ensure_ascii=False)
    if name == "world_chronicle":
        d = api("GET", "/api/world/chronicle")
        return json.dumps((d.get("chronicle") or [])[-60:], ensure_ascii=False)
    if name == "world_memories":
        d = api("GET", "/api/world/memories")
        return json.dumps(d.get("memories", []), ensure_ascii=False)
    if name == "world_foreshadows":
        d = api("GET", "/api/world/foreshadows")
        return json.dumps(d.get("foreshadows", ""), ensure_ascii=False)
    if name == "world_thinking":
        d = api("GET", "/api/world/sim/thinking")
        return json.dumps(d.get("thinking", "（无）"), ensure_ascii=False)
    if name == "world_tokens":
        d = api("GET", "/api/world/token_stats")
        return json.dumps(d, ensure_ascii=False)
    if name == "world_novel_list":
        d = api("GET", "/api/world/novel")
        return json.dumps({"plans": d.get("plans", []), "exports": d.get("exports", [])}, ensure_ascii=False)
    if name == "world_novel_generate":
        r = api("POST", "/api/world/novel/generate", {})
        return json.dumps({**r, "hint": "写作约1-3分钟/章；若超时则后台进行，稍后 world_novel_list 查看"}, ensure_ascii=False)
    if name == "world_seed_novel":
        d = api("GET", "/api/worlds")
        world_id = d.get("current") or ""
        if not world_id and d.get("worlds"):
            world_id = d["worlds"][0].get("name", "")
        if not world_id:
            return json.dumps({"error": "尚无世界，请先 world_create 创建世界"}, ensure_ascii=False)
        body = {"world_id": world_id, "project_name": args["project_name"]}
        if args.get("language"):
            body["language"] = args["language"]
        r = api("POST", "/api/world/seed-novel", body, NOVEL_BASE)
        return json.dumps({**r, "hint": "世界已播种成小说项目，到小说服务 48090 可查看/续写"}, ensure_ascii=False)
    if name == "world_novel_chapter":
        d = api("GET", f"/api/world/novel/chapter/{args['num']}")
        return json.dumps(d, ensure_ascii=False)
    if name == "world_themes":
        d = api("GET", "/api/worldbooks/themes")
        return json.dumps(d.get("themes", []), ensure_ascii=False)
    if name == "world_snapshots":
        d = api("GET", "/api/world/snapshots")
        return json.dumps(d.get("snapshots", []), ensure_ascii=False)
    if name == "world_snapshot":
        return json.dumps(api("POST", "/api/world/snapshot", {"reason": args.get("reason", "手动存档")}), ensure_ascii=False)
    if name == "world_rewind":
        return json.dumps(api("POST", "/api/world/rewind", {"day": int(args["day"])}), ensure_ascii=False)
    if name == "world_webui":
        return json.dumps({"url": "http://127.0.0.1:48091",
                           "hint": "浏览器打开即控制台：决策翻案/循环开关/时间回退/小说阅读"}, ensure_ascii=False)
    raise RuntimeError(f"未知工具: {name}")

# ---------------- MCP stdio 协议 ----------------
def rpc(msg):
    mid = msg.get("id")
    method = msg.get("method")
    if method == "initialize":
        return {"jsonrpc": "2.0", "id": mid, "result": {
            "protocolVersion": PROTOCOL_VERSION,
            "capabilities": {"tools": {}},
            "serverInfo": {"name": "worldsim-mcp", "version": "1.4.0"}}}
    if method == "notifications/initialized" or method == "notifications/cancelled":
        return None
    if method == "ping":
        return {"jsonrpc": "2.0", "id": mid, "result": {}}
    if method == "tools/list":
        return {"jsonrpc": "2.0", "id": mid, "result": {"tools": TOOLS}}
    if method == "tools/call":
        name = (msg.get("params") or {}).get("name", "")
        args = (msg.get("params") or {}).get("arguments", {}) or {}
        try:
            text = call_tool(name, args)
            return {"jsonrpc": "2.0", "id": mid, "result": {"content": [{"type": "text", "text": text}]}}
        except Exception as e:
            return {"jsonrpc": "2.0", "id": mid, "result": {"content": [{"type": "text", "text": f"错误: {e}"}]}}
    return {"jsonrpc": "2.0", "id": mid, "error": {"code": -32601, "message": f"方法不存在: {method}"}}

def main():
    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue
        try:
            msg = json.loads(line)
        except Exception:
            continue
        try:
            resp = rpc(msg)
        except Exception as e:
            resp = {"jsonrpc": "2.0", "id": msg.get("id"), "error": {"code": -32603, "message": str(e)}}
        if resp is not None:
            sys.stdout.write(json.dumps(resp, ensure_ascii=False) + "\n")
            sys.stdout.flush()

if __name__ == "__main__":
    if "--selftest" in sys.argv:
        print("worldsim-mcp selftest: 检查 WorldSim 服务…")
        try:
            ensure_alive()
            d = api("GET", "/api/worlds")
            print("OK: WorldSim 服务在线, 世界:", d.get("worlds", []))
            print("工具数:", len(TOOLS))
        except Exception as e:
            print(f"FAIL: {e}")
        sys.exit(0)
    main()