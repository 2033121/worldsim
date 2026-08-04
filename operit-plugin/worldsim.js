/* METADATA
{
    "name": "worldsim",
    "display_name": {
        "zh": "WorldSim 世界模拟器",
        "en": "WorldSim Simulator"
    },
    "description": { "zh": "多Agent世界模拟器 + 网文生产引擎控制入口。可创建任意题材新世界（修仙/末世/西幻/克苏鲁…选主题包一键生成世界书）、驱动世界演化（事件生成/主角决策/时间跳跃）、查看决策队列并改选、检查模拟就绪度、生成与阅读小说章节。用户说'写小说/开个新世界/跑模拟/修仙世界进展'等时使用。", "en": "Multi-agent world simulator & web novel production engine controller. Create any-genre worlds (xianxia/apocalypse/western fantasy/cthulhu...), drive world evolution (events/decisions/time-skip), review & override AI decisions, check readiness, generate & read novel chapters." },
    "enabledByDefault": true,
    "category": "Creation",
    "tools": [
        { "name": "world_list", "description": { "zh": "列出所有世界及当前激活世界、各自 Day。", "en": "List all worlds, active world and each world's Day." }, "parameters": [] },
        { "name": "world_select", "description": { "zh": "切换当前操作的世界（后续工具都作用于它）。", "en": "Switch the active world (subsequent tools operate on it)." }, "parameters": [ { "name": "name", "description": { "zh": "世界名称", "en": "World name" }, "type": "string", "required": true } ] },
        { "name": "world_create", "description": { "zh": "创建新世界。传 theme（主题包名）时由 LLM 自动生成完整世界书；可配 desc 一句话设定；也可传 worldbook 指定已存在的世界书文件名。创建后返回新世界名（已自动切换并自动启用 LLM）。", "en": "Create a new world. With theme, LLM auto-generates a full worldbook; optional desc for one-line setting; or pass worldbook filename for existing one." }, "parameters": [
            { "name": "name", "description": { "zh": "世界名称，如：青岚界·灵剑山", "en": "World name, e.g. '青岚界·灵剑山'" }, "type": "string", "required": true },
            { "name": "theme", "description": { "zh": "主题包名（可选）：经典修仙/都市异能/末世废土/西幻奇幻/恐怖灵异/历史王朝/星际科幻/无限诸天/武侠玄幻/美食种田/原始文明/军旅战争/洪荒神话/克苏鲁异界/系统流。用 world_themes 查看全列表", "en": "Theme pack name (optional). See world_themes for full list." }, "type": "string", "required": false },
            { "name": "desc", "description": { "zh": "一句话设定（可选，theme 模式下 LLM 按此生成世界书）", "en": "One-line world setting (optional, used with theme)" }, "type": "string", "required": false },
            { "name": "worldbook", "description": { "zh": "已存在的世界书文件名（可选，不传则用默认；与 theme 二选一）", "en": "Existing worldbook filename (optional, alternative to theme)" }, "type": "string", "required": false }
        ] },
        { "name": "world_init", "description": { "zh": "初始化当前世界：按世界书生成主角/NPC/地点。主角名可不传（LLM 按世界书自拟）。初始化后世界立即可模拟。", "en": "Initialize current world: generate protagonist/NPCs/locations from worldbook. Protagonist name optional (LLM decides)." }, "parameters": [ { "name": "protagonist", "description": { "zh": "主角名（可选，留空由 LLM 按世界书拟）", "en": "Protagonist name (optional)" }, "type": "string", "required": false } ] },
        { "name": "world_state", "description": { "zh": "查看当前世界状态：Day、天气、张力、主角属性、地点、NPC、全局事件。", "en": "Get current world state: day, weather, tension, hero stats, locations, NPCs, global events." }, "parameters": [] },
        { "name": "world_run_day", "description": { "zh": "手动跑 N 天模拟（上限 30）。mode: auto 自动 / scene 完整展开 / summary 轻量 / skip 快进。结果含当日事件/对话/主角三问决策。", "en": "Run N days of simulation (max 30). mode: auto/scene/summary/skip. Returns events/dialogue/thinking." }, "parameters": [
            { "name": "days", "description": { "zh": "跑几天（1-30，默认1）", "en": "Days to run (1-30, default 1)" }, "type": "number", "required": false },
            { "name": "mode", "description": { "zh": "模式：auto(默认)/scene/summary/skip", "en": "Mode: auto(default)/scene/summary/skip" }, "type": "string", "required": false }
        ] },
        { "name": "world_loop_start", "description": { "zh": "后台持续运行模拟，直到达到目标 Day 或素材就绪（自动停，等待用户看小说）。适合长时间挂机积累剧情。", "en": "Start background simulation loop until target day or readiness (auto-stop)." }, "parameters": [
            { "name": "days", "description": { "zh": "目标天数（默认1000）", "en": "Target days (default 1000)" }, "type": "number", "required": false },
            { "name": "mode", "description": { "zh": "模式：auto(默认)/scene/summary", "en": "Mode: auto(default)/scene/summary" }, "type": "string", "required": false }
        ] },
        { "name": "world_loop_stop", "description": { "zh": "停止后台持续运行循环。", "en": "Stop the background simulation loop." }, "parameters": [] },
        { "name": "world_loop_status", "description": { "zh": "查询后台循环状态：是否运行、当前 Day/目标 Day、是否就绪。", "en": "Query loop status: running, day/target, ready." }, "parameters": [] },
        { "name": "world_readiness", "description": { "zh": "模拟就绪度：素材够不够写小说（完成段落/戏剧素材/伏笔回收/张力 4 项指标，ready=true 即可生成小说）。", "en": "Simulation readiness: enough material to write novel? (arcs/drama/foreshadows/tension). ready=true means ready to write." }, "parameters": [] },
        { "name": "world_decisions", "description": { "zh": "查看岔口决策队列（AI 已代决，用户可改选）。每条含情境/选项/AI 选择与理由/状态。", "en": "List decision queue (AI auto-decided forks, user can override). Each entry has context/options/AI choice & reason/status." }, "parameters": [] },
        { "name": "world_decision_resolve", "description": { "zh": "用户改选某个岔口的方向（覆盖 AI 代决，写手按用户方向写）。choice 是选项 ID（A/B/C）。", "en": "Override a decision's direction (writer follows user's choice). choice is option ID (A/B/C)." }, "parameters": [
            { "name": "id", "description": { "zh": "决策 ID（如 dec-12-1）", "en": "Decision ID (e.g. dec-12-1)" }, "type": "string", "required": true },
            { "name": "choice", "description": { "zh": "选项 ID：A/B/C", "en": "Option ID: A/B/C" }, "type": "string", "required": true }
        ] },
        { "name": "world_chronicle", "description": { "zh": "编年史：世界发生过的所有事（时间线）。", "en": "Chronicle: full timeline of events in the world." }, "parameters": [] },
        { "name": "world_memories", "description": { "zh": "各角色记忆（主角/重要NPC沉淀的认知）。", "en": "Characters' memories (protagonist & key NPCs)." }, "parameters": [] },
        { "name": "world_foreshadows", "description": { "zh": "未回收伏笔清单（防止写手忘坑）。", "en": "Open (unresolved) foreshadows list." }, "parameters": [] },
        { "name": "world_thinking", "description": { "zh": "主角最近一次三问决策推理（内心戏）。", "en": "Protagonist's latest three-question decision reasoning." }, "parameters": [] },
        { "name": "world_tokens", "description": { "zh": "LLM 调用统计（调用次数/缓存命中/tokens 消耗）。", "en": "LLM call stats (calls/cache hit/tokens)." }, "parameters": [] },
        { "name": "world_novel_list", "description": { "zh": "小说章节列表（已生成/待生成）。", "en": "Novel chapter list (generated/pending)." }, "parameters": [] },
        { "name": "world_novel_generate", "description": { "zh": "生成/续写小说章节（写手基于世界编年史创作，1-3分钟/章）。", "en": "Generate/continue novel chapters (writer creates from chronicle, 1-3 min/chapter)." }, "parameters": [] },
        { "name": "world_novel_chapter", "description": { "zh": "读取某一章正文。", "en": "Read a chapter's content." }, "parameters": [ { "name": "num", "description": { "zh": "章节号", "en": "Chapter number" }, "type": "number", "required": true } ] },
        { "name": "world_themes", "description": { "zh": "列出全部主题包（创建世界时选 theme 用）。", "en": "List all theme packs (for world_create theme)." }, "parameters": [] },
        { "name": "world_webui", "description": { "zh": "返回 WorldSim 可视化控制台的访问地址（用户可在浏览器打开：世界状态/决策改选/循环开关/小说阅读）。", "en": "Return the WorldSim web console URL (world state/decision override/loop control/novel reader)." }, "parameters": [] },
        { "name": "world_snapshots", "description": { "zh": "列出全部时间回退锚点（快照：Day/说明/时间）。每30天自动存档，也可手动存。", "en": "List all time-rewind snapshots (day/reason/time). Auto-saved every 30 days, or manual." }, "parameters": [] },
        { "name": "world_snapshot", "description": { "zh": "手动存档：在当前时间点保存完整快照（世界/编年史/记忆/决策/伏笔），供以后时间回退用。", "en": "Manual snapshot: save full state at current point for later rewind." }, "parameters": [ { "name": "reason", "description": { "zh": "存档说明（可选）", "en": "Snapshot note (optional)" }, "type": "string", "required": false } ] },
        { "name": "world_rewind", "description": { "zh": "时间回退：把世界回退到 ≤day 的最近快照（恢复当时的世界/编年史/记忆/决策/伏笔），之后重新演化出新分支。用于剧情跑偏、卡死、想重来。会先停止循环。", "en": "Time rewind: restore world to nearest snapshot ≤ day, then re-evolve a new branch. Use when story went wrong/stuck. Stops loop first." }, "parameters": [ { "name": "day", "description": { "zh": "回退到的 Day（取≤它的最近快照）", "en": "Target day (nearest snapshot ≤ it)" }, "type": "number", "required": true } ] }
    ]
}*/
const WorldSim = (function () {
    const BASE = 'http://127.0.0.1:48091';
    const TIMEOUT_MS = 60000;

    // 基础请求：GET/POST + JSON（走宿主 http_request 工具——沙盒 client 禁止访问本地地址）
    async function api(method, path, body) {
        try {
            const toolParams = { url: BASE + path, method };
            if (body !== undefined) {
                toolParams.body = JSON.stringify(body);
                toolParams.body_type = 'json';
            }
            const result = await toolCall({ name: "http_request", params: toolParams });
            const status = result && result.statusCode || 0;
            const text = typeof result?.content === "string" ? result.content : "";
            let data = null;
            try { data = JSON.parse(text); } catch (e) { data = text; }
            return { ok: status >= 200 && status < 400, status, data };
        } catch (e) {
            return { ok: false, status: 0, data: null, netErr: String(e && e.message || e) };
        }
    }

    function wrap(fn, params, okMsg, errMsg) {
        return fn(params).then(r => ({ success: true, message: okMsg, data: r }))
            .catch(e => ({ success: false, message: `${errMsg}: ${e.message || e}` }));
    }

    // 服务探测：失败时给出启动指引
    async function ensureAlive() {
        const r = await api('GET', '/api/worlds');
        if (!r.ok && r.status === 0) {
            throw new Error('WorldSim 服务未启动（端口 48091 无响应）。请先运行插件包内的 run.sh 启动服务，再重试。');
        }
        return r;
    }

    // ---------- 管理 ----------
    async function world_list() {
        await ensureAlive();
        const r = await api('GET', '/api/worlds');
        if (!r.ok) throw new Error('查询失败: ' + JSON.stringify(r.data));
        const worlds = r.data.worlds || [];
        return worlds.map(w => ({ name: w.name, active: !!w.active, day: w.day || 0 }));
    }

    async function world_select(params) {
        await ensureAlive();
        const r = await api('POST', '/api/worlds/select', { world: params.name });
        if (!r.ok) throw new Error(r.data && r.data.error || '切换失败');
        return { current: r.data.current };
    }

    async function world_create(params) {
        await ensureAlive();
        const body = { name: params.name };
        if (params.theme) body.theme = params.theme;
        if (params.desc) body.desc = params.desc;
        if (params.worldbook) body.worldbook = params.worldbook;
        const r = await api('POST', '/api/worlds/create', body);
        if (r.ok) return { created: r.data.world || params.name, hint: '已自动切换。下一步：world_init 初始化（生成主角/NPC/地点），再 world_loop_start 跑模拟。' };
        if (r.status === 0) {
            // 网络超时：世界书生成较慢（1-2分钟），世界可能在后台创建中（服务端独立 ctx 完成）
            return { created: params.name, pending: true, hint: '请求超时，但世界正在后台创建（世界书由 LLM 生成，约1-2分钟）。稍后 world_list 确认；若已存在直接 world_init。' };
        }
        throw new Error(r.data && r.data.error || '创建失败');
    }

    async function world_init(params) {
        await ensureAlive();
        const body = {};
        if (params.protagonist) body.protagonist = params.protagonist;
        const r = await api('POST', '/api/world/init', body);
        if (r.status === 0) {
            return { ok: true, pending: true, hint: '请求超时，但初始化可能在后台完成（LLM 生成主角/NPC/地点，约1-2分钟）。稍后 world_state 确认。' };
        }
        if (!r.ok) throw new Error(r.data && r.data.error || '初始化失败');
        const d = r.data;
        const hero = d.hero || d.protagonist || '';
        return { ok: true, hero, locations: d.locations || [], npcs: d.npcs || [], hint: '初始化完成，可 world_loop_start 开始模拟。' };
    }

    async function world_state() {
        await ensureAlive();
        const r = await api('GET', '/api/world/state');
        if (!r.ok) throw new Error('查询失败');
        const st = r.data;
        const t = st.world_level || {};
        return {
            day: st.day,
            weather: st.weather,
            tension: t.tension,
            revision: st.revision,
            hero: Object.keys(st.entities || {}).filter(k => (st.entities[k].extra || {}).role === 'protagonist')[0] || null,
            entities: st.entities,
            global_events: t.global_events || [],
        };
    }

    // ---------- 模拟 ----------
    async function world_run_day(params) {
        await ensureAlive();
        const body = { days: Math.min(Math.max(parseInt(params.days) || 1, 1), 30), mode: params.mode || 'auto' };
        const r = await api('POST', '/api/world/sim/day', body);
        if (!r.ok) throw new Error(r.data && r.data.error || '模拟失败（可能是 LLM 超时，稍后重试）');
        const d = r.data;
        const last = d.results && d.results[d.results.length - 1];
        return {
            day: d.day,
            paused: !!d.paused,
            events: last ? last.events : [],
            dialogue: last ? (last.dialogue || []) : [],
            thinking: d.thinking || last && last.thinking || '',
            decisions: d.decisions || [],
        };
    }

    async function world_loop_start(params) {
        await ensureAlive();
        const r = await api('POST', '/api/world/loop', { action: 'start', days: parseInt(params.days) || 1000, mode: params.mode || 'auto' });
        if (!r.ok) throw new Error(r.data && r.data.error || '循环启动失败');
        return { running: true, target_day: r.data.target_day, hint: '后台持续运行中。可 world_loop_status 查进度；素材就绪会自动停。' };
    }

    async function world_loop_stop() {
        await ensureAlive();
        const r = await api('POST', '/api/world/loop', { action: 'stop' });
        if (!r.ok) throw new Error('停止失败');
        return { running: false };
    }

    async function world_loop_status() {
        await ensureAlive();
        const r = await api('GET', '/api/world/loop');
        if (!r.ok) throw new Error('查询失败');
        return r.data;
    }

    async function world_readiness() {
        await ensureAlive();
        const r = await api('GET', '/api/world/readiness');
        if (!r.ok) throw new Error('查询失败');
        const d = r.data;
        return {
            ready: !!d.ready,
            reason: d.reason || '',
            arcs_done: d.arcs_done, drama_entries: d.drama_entries,
            foreshadows_resolved: d.foreshadows_resolved, foreshadows_total: d.foreshadows_total,
            tension: d.tension,
            hint: d.ready ? '素材就绪，可 world_novel_generate 生成小说！' : '继续跑模拟积累素材…',
        };
    }

    // ---------- 内容 ----------
    async function world_decisions() {
        await ensureAlive();
        const r = await api('GET', '/api/world/decisions');
        if (!r.ok) throw new Error('查询失败');
        return (r.data.decisions || []).map(dc => ({
            id: dc.id, day: dc.day, type: dc.type, title: dc.title,
            context: dc.context, options: dc.options,
            ai_choice: dc.ai_choice, ai_reason: dc.ai_reason,
            status: dc.status, user_choice: dc.user_choice,
        }));
    }

    async function world_decision_resolve(params) {
        await ensureAlive();
        const r = await api('POST', '/api/world/decisions/' + params.id, { choice: params.choice });
        if (!r.ok) throw new Error(r.data && r.data.error || '改选失败');
        return { resolved: true, decision: params.id, choice: params.choice, hint: '写手将按此方向推进。' };
    }

    async function world_chronicle() {
        await ensureAlive();
        const r = await api('GET', '/api/world/chronicle');
        if (!r.ok) throw new Error('查询失败');
        return (r.data.chronicle || []).slice(-60).map(c => ({ day: c.day, kind: c.kind, content: c.content }));
    }

    async function world_memories() {
        await ensureAlive();
        const r = await api('GET', '/api/world/memories');
        if (!r.ok) throw new Error('查询失败');
        return r.data.memories || [];
    }

    async function world_foreshadows() {
        await ensureAlive();
        const r = await api('GET', '/api/world/foreshadows');
        if (!r.ok) throw new Error('查询失败');
        return r.data.foreshadows || '';
    }

    async function world_thinking() {
        await ensureAlive();
        const r = await api('GET', '/api/world/sim/thinking');
        if (!r.ok) throw new Error('查询失败');
        return r.data.thinking || '（无）';
    }

    async function world_tokens() {
        await ensureAlive();
        const r = await api('GET', '/api/world/token_stats');
        if (!r.ok) throw new Error('查询失败');
        const d = r.data || {};
        return {
            total_calls: d.total_calls || 0,
            cache_hit_rate: d.cache_hit_rate || '0%',
            total_prompt_tokens: d.total_prompt_tokens || 0,
            total_completion_tokens: d.total_completion_tokens || 0,
        };
    }

    // ---------- 小说 ----------
    async function world_novel_list() {
        await ensureAlive();
        const r = await api('GET', '/api/world/novel');
        if (!r.ok) throw new Error('查询失败');
        return { plans: r.data.plans || [], exports: r.data.exports || [] };
    }

    async function world_novel_generate() {
        await ensureAlive();
        const r = await api('POST', '/api/world/novel/generate', {});
        if (r.status === 0) {
            return { ok: true, pending: true, hint: '请求超时，但写作可能在后台进行中（1-3分钟/章）。稍后 world_novel_list 查看新章节。' };
        }
        if (!r.ok) throw new Error(r.data && r.data.error || '生成失败（约1-3分钟/章，可稍后 world_novel_list 查看）');
        return { ok: true, written: r.data.written || [], hint: '章节已生成/续写，可用 world_novel_list 查看。' };
    }

    async function world_novel_chapter(params) {
        await ensureAlive();
        const r = await api('GET', '/api/world/novel/chapter/' + params.num);
        if (!r.ok) throw new Error(r.data && r.data.error || '章节不存在');
        return { num: params.num, title: r.data.title, content: r.data.content };
    }

    async function world_themes() {
        await ensureAlive();
        const r = await api('GET', '/api/worldbooks/themes');
        if (!r.ok) throw new Error('查询失败');
        return r.data.themes || [];
    }

    async function world_webui() {
        return {
            url: 'http://127.0.0.1:48091',
            hint: '在手机/电脑浏览器打开此地址（需与本机同一网络时用局域网 IP），可查看：世界状态、决策改选、循环开关、就绪度、小说阅读。',
        };
    }

    async function world_snapshots() {
        await ensureAlive();
        const r = await api('GET', '/api/world/snapshots');
        if (!r.ok) throw new Error('查询失败');
        return (r.data.snapshots || []).map(s => ({ day: s.day, revision: s.revision, reason: s.reason, created_at: s.created_at }));
    }

    async function world_snapshot(params) {
        await ensureAlive();
        const r = await api('POST', '/api/world/snapshot', { reason: params.reason || '手动存档' });
        if (!r.ok) throw new Error(r.data && r.data.error || '存档失败');
        const s = r.data.snapshot || {};
        return { saved: true, day: s.day, revision: s.revision, reason: s.reason };
    }

    async function world_rewind(params) {
        await ensureAlive();
        const r = await api('POST', '/api/world/rewind', { day: parseInt(params.day) || 0 });
        if (!r.ok) throw new Error(r.data && r.data.error || '回退失败');
        return {
            rewound_to: r.data.rewound_to, revision: r.data.revision, reason: r.data.reason,
            hint: '已回退，重新 world_loop_start 会走出新分支。',
        };
    }

    // ---------- 工具导出（wrap 统一返回 {success, data}） ----------
    return {
        world_list: (p) => wrap(world_list, p, '世界列表', '查询世界列表失败'),
        world_select: (p) => wrap(world_select, p, '切换世界成功', '切换世界失败'),
        world_create: (p) => wrap(world_create, p, '创建世界成功', '创建世界失败'),
        world_init: (p) => wrap(world_init, p, '世界初始化成功', '世界初始化失败'),
        world_state: (p) => wrap(world_state, p, '世界状态', '查询世界状态失败'),
        world_run_day: (p) => wrap(world_run_day, p, '模拟完成', '模拟失败'),
        world_loop_start: (p) => wrap(world_loop_start, p, '循环已启动', '循环启动失败'),
        world_loop_stop: (p) => wrap(world_loop_stop, p, '循环已停止', '停止失败'),
        world_loop_status: (p) => wrap(world_loop_status, p, '循环状态', '查询失败'),
        world_readiness: (p) => wrap(world_readiness, p, '就绪度', '查询就绪度失败'),
        world_decisions: (p) => wrap(world_decisions, p, '决策队列', '查询决策失败'),
        world_decision_resolve: (p) => wrap(world_decision_resolve, p, '改选成功', '改选失败'),
        world_chronicle: (p) => wrap(world_chronicle, p, '编年史', '查询失败'),
        world_memories: (p) => wrap(world_memories, p, '记忆', '查询失败'),
        world_foreshadows: (p) => wrap(world_foreshadows, p, '伏笔清单', '查询失败'),
        world_thinking: (p) => wrap(world_thinking, p, '主角内心', '查询失败'),
        world_tokens: (p) => wrap(world_tokens, p, '统计', '查询失败'),
        world_novel_list: (p) => wrap(world_novel_list, p, '章节列表', '查询失败'),
        world_novel_generate: (p) => wrap(world_novel_generate, p, '写作完成', '写作失败'),
        world_novel_chapter: (p) => wrap(world_novel_chapter, p, '章节正文', '读取失败'),
        world_themes: (p) => wrap(world_themes, p, '主题包列表', '查询失败'),
        world_webui: (p) => wrap(world_webui, p, '控制台地址', '获取失败'),
        world_snapshots: (p) => wrap(world_snapshots, p, '时间回退锚点', '查询失败'),
        world_snapshot: (p) => wrap(world_snapshot, p, '存档成功', '存档失败'),
        world_rewind: (p) => wrap(world_rewind, p, '回退成功', '回退失败'),
        main: main,
    };
})();

async function main(params, complete) {
    const results = [];
    const run = async (name, fn) => {
        try {
            const r = await fn({});
            results.push(`[OK] ${name}: ${JSON.stringify(r).slice(0, 200)}`);
        } catch (e) {
            results.push(`[ERR] ${name}: ${e.message}`);
        }
    };
    await run('world_list', world_list);
    await run('world_themes', world_themes);
    await run('world_webui', world_webui);
    complete({ success: true, message: 'WorldSim 插件自检完成', data: results.join('\n') });
}

exports.world_list = WorldSim.world_list;
exports.world_select = WorldSim.world_select;
exports.world_create = WorldSim.world_create;
exports.world_init = WorldSim.world_init;
exports.world_state = WorldSim.world_state;
exports.world_run_day = WorldSim.world_run_day;
exports.world_loop_start = WorldSim.world_loop_start;
exports.world_loop_stop = WorldSim.world_loop_stop;
exports.world_loop_status = WorldSim.world_loop_status;
exports.world_readiness = WorldSim.world_readiness;
exports.world_decisions = WorldSim.world_decisions;
exports.world_decision_resolve = WorldSim.world_decision_resolve;
exports.world_chronicle = WorldSim.world_chronicle;
exports.world_memories = WorldSim.world_memories;
exports.world_foreshadows = WorldSim.world_foreshadows;
exports.world_thinking = WorldSim.world_thinking;
exports.world_tokens = WorldSim.world_tokens;
exports.world_novel_list = WorldSim.world_novel_list;
exports.world_novel_generate = WorldSim.world_novel_generate;
exports.world_novel_chapter = WorldSim.world_novel_chapter;
exports.world_themes = WorldSim.world_themes;
exports.world_webui = WorldSim.world_webui;
exports.world_snapshots = WorldSim.world_snapshots;
exports.world_snapshot = WorldSim.world_snapshot;
exports.world_rewind = WorldSim.world_rewind;
exports.main = WorldSim.main;