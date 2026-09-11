package main

// ws_game.go — 「文字游戏」游玩模式（Play Mode）的 HTTP 层。
//
// 设计：internal/game 持有数值层（game.json）；本文件负责：
//   ① 从 engine 状态拼装"世界上下文包"喂给裁判/叙述者；
//   ② 每回合把游戏面板变化映射回 engine.Changes（health/money/location/stats.game），
//      保证世界→小说播种链路继续消费这些数据；
//   ③ 开局时暂停后台模拟循环（游玩优先，回合制——玩家不动世界不空转）。
//
// 端点（挂在 :48091，统一网关 48092 的 /api/* 自动转发）：
//   GET  /api/game/status — 面板 + 引擎侧主角快照
//   POST /api/game/start  — 进入游玩模式（自动停循环，LLM 生成主题适配面板+开场）
//   POST /api/game/action — 一回合：{input, mode(do|say|story)}
//   POST /api/game/wait   — 等待回合（世界自转+休息回血）
//   POST /api/game/stop   — 退出游玩模式
//   GET  /api/game/log    — 游戏账本（最近 200 回合）

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"worldsim/internal/engine"
	game "worldsim/internal/game"
	"worldsim/internal/worldbook"
)

func (w *worldInstance) lazyGame() *game.Game {
	w.gameOnce.Do(func() {
		w.game = game.Load(w.dir)
	})
	return w.game
}

// gameCtx 拼装世界上下文包（裁判/叙述者共用）：主角引擎快照 + 在场角色 + 最近编年史
func (w *worldInstance) gameCtx(extra ...string) string {
	if w == nil || w.engine == nil {
		return "（引擎未加载）"
	}
	st := w.engine.State()
	var b strings.Builder
	hero := ""
	if w.sim != nil {
		hero = w.sim.HeroName()
	}
	if hero == "" {
		hero = w.heroName
	}
	if e, ok := st.Entities[hero]; ok {
		b.WriteString(fmt.Sprintf("主角引擎快照：位置=%s 健康=%.0f 资产=%v 职业=%s 存活=%v\n",
			e.Location, e.Health, e.Assets, e.Job, e.Alive))
		// 在场角色（同地点 + active）
		present := []string{}
		for name, ent := range st.Entities {
			if name == hero || ent.Status != "active" {
				continue
			}
			if e.Location != "" && ent.Location == e.Location {
				present = append(present, name)
			}
		}
		if len(present) > 0 {
			b.WriteString("同地点角色：" + strings.Join(present, "、") + "\n")
		}
	} else {
		b.WriteString("（主角实体未初始化：世界尚未 init）\n")
	}
	b.WriteString(fmt.Sprintf("世界：%s 第%d天 天气=%s\n", w.name, st.Day, st.Weather))
	if w.wb != nil {
		if rule := w.wb.WorldRule(); rule != "" {
			b.WriteString("世界规则：" + rule + "\n")
		}
	}
	// 最近编年史（对比深度裁剪，防上下文爆炸）
	if w.sim != nil {
		tries := 6
		for i := len(w.sim.Chronicle()) - 1; i >= 0 && tries > 0; i-- {
			e := w.sim.Chronicle()[i]
			content := e.Content
			if len(content) > 120 {
				content = content[:120]
			}
			b.WriteString(fmt.Sprintf("编年史#%d：%s\n", e.Day, content))
			tries--
		}
	}
	// W1 动态条目（关键词触发 lore）：扫描最近回合文本+本回合输入，命中/sticky 注入裁判+叙述者共享上下文
	if w.wb != nil && len(w.wb.WIEntries) > 0 && w.game != nil && w.game.StateCurrent().Enabled {
		buf := wiScanBuffer(w.game) + strings.Join(extra, " ")
		if w.wiSticky == nil {
			w.wiSticky = map[string]int{}
		}
		if hits := worldbook.ActivateWI(w.wb.WIEntries, buf, w.wiSticky, 1200, 3); len(hits) > 0 {
			keys := []string{}
			for _, e := range hits {
				keys = append(keys, strings.Join(e.Keys, ","))
			}
			fmt.Printf(" [游戏] WI 动态情报注入 %d 条：%s\n", len(hits), strings.Join(keys, " / "))
			w.wiLast.Store(append([]string(nil), keys...)) // 供 /api/game/status 透明化
			b.WriteString("动态情报（按最近剧情关键词触发）：\n")
			for _, e := range hits {
				b.WriteString("- " + e.Content + "\n")
			}
		}
	}
	return b.String()
}

// wiScanBuffer 扫描缓冲：最近 6 条回合的输入+叙述（控上下文，等价 ST 的 scan depth）
func wiScanBuffer(g *game.Game) string {
	log := g.StateCurrent().Log
	start := 0
	if len(log) > 6 {
		start = len(log) - 6
	}
	var parts []string
	for _, e := range log[start:] {
		parts = append(parts, e.Input, e.Narration)
	}
	return strings.Join(parts, " ")
}

// POST /api/game/start — 进入游玩模式
func (ws *worldServer) handleGameStart(w http.ResponseWriter, r *http.Request) {
	inst := ws.inst()
	if inst == nil {
		ws.writeJSON(w, 400, map[string]any{"ok": false, "error": "没有可用世界，请先创建"})
		return
	}
	// 游玩优先：暂停后台模拟循环（回合制，玩家不动世界不空转）
	ws.loopMu.Lock()
	if ws.loopCancel != nil {
		ws.loopCancel()
		ws.loopRunning = false
	}
	ws.loopMu.Unlock()

	g := inst.lazyGame()
	hero := inst.heroName
	if inst.sim != nil && inst.sim.HeroName() != "" {
		hero = inst.sim.HeroName()
	}
	brief := ""
	if e, ok := inst.engine.State().Entities[hero]; ok {
		if j, err := json.Marshal(e); err == nil {
			brief = string(j)
		}
	}
	worldDesc := ""
	var wbAttrs map[string]int
	if inst.wb != nil {
		worldDesc = inst.wb.WorldRule()
		wbAttrs = inst.wb.GameAttrs() // Play Mode 数据层默认属性表（游玩属性段，不硬编码）
	}
	scene, err := g.Start(r.Context(), callerFrom(inst), inst.name, worldDesc, hero, brief, wbAttrs)
	if err != nil {
		ws.writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	ws.writeJSON(w, 200, map[string]any{"ok": true, "scene": scene, "state": g.StateCurrent()})
}

// callerFrom 把 inst.llm 转成游戏包的 Caller（含世界参考资料注入与 Mock 模式）
func callerFrom(inst *worldInstance) game.Caller {
	if inst == nil || inst.llm == nil {
		return nil
	}
	cli := inst.llm
	return func(ctx context.Context, tier, system, user string) (string, error) {
		return cli.CompleteTier(ctx, tier, system, user)
	}
}

// POST /api/game/action — 一回合
func (ws *worldServer) handleGameAction(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	inst := ws.inst()
	if inst == nil {
		ws.writeJSON(w, 400, map[string]any{"ok": false, "error": "没有可用世界"})
		return
	}
	var req struct {
		Input string `json:"input"`
		Mode  string `json:"mode"` // do | say | story
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ws.writeJSON(w, 400, map[string]any{"ok": false, "error": "参数错误"})
		return
	}
	g := inst.lazyGame()
	st := g.StateCurrent()
	if !st.Enabled {
		ws.writeJSON(w, 400, map[string]any{"ok": false, "error": "游戏未开启，先 POST /api/game/start"})
		return
	}
	// 回合期间禁止后台模拟并发写状态
	ws.loopMu.Lock()
	if ws.loopCancel != nil {
		ws.loopCancel()
		ws.loopRunning = false
	}
	ws.loopMu.Unlock()

	day := inst.engine.State().Day
	// 独立 context：客户端断连不影响回合完成
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()
	narr, err := g.Turn(ctx, callerFrom(inst), day, req.Input, req.Mode, inst.gameCtx(req.Input))
	if err != nil {
		ws.writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	// 数值层 → 引擎同步（世界→小说播种链路继续可用）
	syncGameToEngine(inst, g)
	ws.writeJSON(w, 200, map[string]any{
		"ok": true, "narration": narr, "state": g.StateCurrent(),
		"elapsed_ms": time.Since(start).Milliseconds(),
	})
}

// POST /api/game/wait — 等待回合
func (ws *worldServer) handleGameWait(w http.ResponseWriter, r *http.Request) {
	inst := ws.inst()
	if inst == nil {
		ws.writeJSON(w, 400, map[string]any{"ok": false, "error": "没有可用世界"})
		return
	}
	g := inst.lazyGame()
	if !g.StateCurrent().Enabled {
		ws.writeJSON(w, 400, map[string]any{"ok": false, "error": "游戏未开启，先 POST /api/game/start"})
		return
	}
	ws.loopMu.Lock()
	if ws.loopCancel != nil {
		ws.loopCancel()
		ws.loopRunning = false
	}
	ws.loopMu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()
	narr, err := g.Wait(ctx, callerFrom(inst), inst.engine.State().Day, inst.gameCtx())
	if err != nil {
		ws.writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	syncGameToEngine(inst, g)
	// 时钟同步（v1.10.0）：等待=世界自转一天，引擎 Day 推进+落盘——
	// 游玩期间后台循环是暂停的，不同步则场景横幅 day%3 永不轮转（模拟器直接改 State 的同款模式）
	if inst.engine != nil {
		inst.engine.State().Day++
		if err := inst.engine.Save(filepath.Join(inst.dir, "world_state.json")); err != nil {
			fmt.Printf(" [游戏] 引擎日推进落盘失败：%v\n", err)
		}
	}
	ws.writeJSON(w, 200, map[string]any{"ok": true, "narration": narr, "state": g.StateCurrent()})
}

// POST /api/game/undo — 撤销最近一步（回滚到上一回合落账前；单步）
func (ws *worldServer) handleGameUndo(w http.ResponseWriter, r *http.Request) {
	inst := ws.inst()
	if inst == nil {
		ws.writeJSON(w, 400, map[string]any{"ok": false, "error": "没有可用世界"})
		return
	}
	st, err := inst.lazyGame().Undo()
	if err != nil {
		ws.writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	syncGameToEngine(inst, inst.lazyGame())
	ws.writeJSON(w, 200, map[string]any{"ok": true, "state": st})
}

// GET /api/game/export — 下载存档（game.json）
func (ws *worldServer) handleGameExport(w http.ResponseWriter, r *http.Request) {
	inst := ws.inst()
	if inst == nil {
		ws.writeJSON(w, 400, map[string]any{"ok": false, "error": "没有可用世界"})
		return
	}
	data, err := os.ReadFile(filepath.Join(inst.dir, game.GameStateFile))
	if err != nil {
		ws.writeJSON(w, 404, map[string]any{"ok": false, "error": "还没有存档（先开局）"})
		return
	}
	name := urlEscape(inst.name)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s-game.json", name))
	_, _ = w.Write(data)
}

// POST /api/game/import — 导入存档（body=game.json 原文；enabled 以存档为准）
func (ws *worldServer) handleGameImport(w http.ResponseWriter, r *http.Request) {
	inst := ws.inst()
	if inst == nil {
		ws.writeJSON(w, 400, map[string]any{"ok": false, "error": "没有可用世界"})
		return
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, 4<<20))
	if err != nil || len(data) == 0 {
		ws.writeJSON(w, 400, map[string]any{"ok": false, "error": "读取请求体失败"})
		return
	}
	if err := inst.lazyGame().ImportFrom(data); err != nil {
		ws.writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	syncGameToEngine(inst, inst.lazyGame())
	ws.writeJSON(w, 200, map[string]any{"ok": true, "state": inst.lazyGame().StateCurrent()})
}

// POST /api/game/stop — 退出游玩模式
func (ws *worldServer) handleGameStop(w http.ResponseWriter, r *http.Request) {
	inst := ws.inst()
	if inst == nil {
		ws.writeJSON(w, 400, map[string]any{"ok": false, "error": "没有可用世界"})
		return
	}
	inst.lazyGame().Stop()
	ws.writeJSON(w, 200, map[string]any{"ok": true})
}

// GET /api/game/status — 面板 + 引擎侧主角快照
func (ws *worldServer) handleGameStatus(w http.ResponseWriter, r *http.Request) {
	inst := ws.inst()
	if inst == nil {
		ws.writeJSON(w, 400, map[string]any{"ok": false, "error": "没有可用世界"})
		return
	}
	g := inst.lazyGame().StateCurrent()
	resp := map[string]any{"ok": true, "game": g, "world": inst.name}
	// W1 动态情报透明化：最近回合命中的关键词组（玩家可感知 lore 触发）
	if v := inst.wiLast.Load(); v != nil {
		if hits, ok := v.([]string); ok && len(hits) > 0 {
			resp["wi_hits"] = hits
		}
	}
	// 美术工坊：本世界 plan.json 优先；无规划回落 detectPixelTheme 打包套件
	if payload := artPixelPayload(inst); payload != nil {
		resp["theme"] = payload["theme"]
		resp["pixel"] = payload
	} else if theme := detectPixelTheme(inst); theme != "" {
		resp["theme"] = theme
		sp := map[string]string{"hero": "/pixel-art/" + theme + "/character-1.png",
			"monster": "/pixel-art/" + theme + "/monster-1.png"}
		resp["pixel"] = map[string]any{"theme": theme, "hero": sp["hero"],
			"monster":    sp["monster"],
			"characters": spriteListPath(theme, "character", 12),
			"monsters":   spriteListPath(theme, "monster", 8),
			"scenes":     spriteListPath(theme, "scene", 3),
		}
	}
	if inst.engine != nil {
		hero := inst.heroName
		if inst.sim != nil && inst.sim.HeroName() != "" {
			hero = inst.sim.HeroName()
		}
		if e, ok := inst.engine.State().Entities[hero]; ok {
			resp["entity"] = e
		}
		resp["day"] = inst.engine.State().Day
	}
	// 世界观感扩展（v1.9.0）：场景横幅/在场头像/物品图标/确定性地图——素材缺失时字段缺省
	if g.Relations != nil {
		resp["relations"] = g.Relations
	}
	day := 0
	if inst.engine != nil {
		day = inst.engine.State().Day
	}
	if scene := sceneURLFor(inst, day); scene != "" {
		resp["scene"] = scene
	}
	if portraits := presentPortraits(inst); len(portraits) > 0 {
		resp["portraits"] = portraits
	}
	if icons := inventoryIcons(inst, &g); len(icons) > 0 {
		resp["item_icons"] = icons
	}
	if inst.engine != nil {
		if m := mapLayout(inst, &g); m != nil {
			resp["map"] = m
		}
	}
	ws.writeJSON(w, 200, resp)
}

// GET /api/game/log — 游戏账本
func (ws *worldServer) handleGameLog(w http.ResponseWriter, r *http.Request) {
	inst := ws.inst()
	if inst == nil {
		ws.writeJSON(w, 400, map[string]any{"ok": false, "error": "没有可用世界"})
		return
	}
	ws.writeJSON(w, 200, map[string]any{"ok": true, "log": inst.lazyGame().StateCurrent().Log})
}

// syncGameToEngine 把游戏面板映射回引擎实体（health/money/location/stats.game）——供养小说播种等下游
func syncGameToEngine(inst *worldInstance, g *game.Game) {
	if inst == nil || inst.engine == nil {
		return
	}
	hero := inst.heroName
	if inst.sim != nil && inst.sim.HeroName() != "" {
		hero = inst.sim.HeroName()
	}
	st := inst.engine.State()
	e, ok := st.Entities[hero]
	if !ok {
		return
	}
	gs := g.StateCurrent()
	base := st.Revision
	changes := []engine.Change{}
	// health：engine 侧按 0~100 刻度；game 侧 HP/MaxHP 比例换算，避免双写漂移（seed/小说播种可读）
	newHealth := float64(0)
	if gs.MaxHP > 0 {
		newHealth = float64(gs.HP) / float64(gs.MaxHP) * 100
	}
	if (newHealth-e.Health) > 0.5 || (e.Health-newHealth) > 0.5 {
		changes = append(changes, engine.Change{Path: fmt.Sprintf("entities.%s.health", hero), Op: "set", Value: newHealth})
	}
	if uintGS(gs.Gold) != e.Money {
		changes = append(changes, engine.Change{Path: fmt.Sprintf("entities.%s.money", hero), Op: "set", Value: float64(gs.Gold)})
	}
	if gs.Location != "" && gs.Location != e.Location {
		changes = append(changes, engine.Change{Path: fmt.Sprintf("entities.%s.location", hero), Op: "set", Value: gs.Location})
	}
	// stats.game：面板核心（供播种/复盘）
	panel, _ := json.Marshal(map[string]any{
		"turn": gs.Turn, "level": gs.Level, "hp": gs.HP, "max_hp": gs.MaxHP,
		"gold": gs.Gold, "xp": gs.XP, "inventory": gs.Inventory, "quest": gs.Quest,
	})
	if existing, ok2 := e.Stats["game"]; !ok2 || fmt.Sprint(existing) != string(panel) {
		changes = append(changes, engine.Change{Path: fmt.Sprintf("entities.%s.stats.game", hero), Op: "set", Value: json.RawMessage(panel)})
	}
	if len(changes) == 0 {
		return
	}
	prop := engine.Proposal{
		CommandID:    fmt.Sprintf("game-%04d", gs.Turn),
		ActorID:      "player",
		BaseRevision: base,
		Type:         "state_change",
		Changes:      changes,
		Reason:       "游玩模式回合落账（数值层→引擎同步）",
	}
	if err := inst.engine.Submit(context.Background(), &prop); err != nil {
		fmt.Printf(" [游戏] 引擎同步失败（下次回合重试）：%v\n", err)
		return
	}
	// 落盘：同步到底不发就等于没发生（断电/重启后播种读到旧值）
	if err := inst.engine.Save(filepath.Join(inst.dir, "world_state.json")); err != nil {
		fmt.Printf(" [游戏] 引擎落盘失败：%v\n", err)
	}
}

func uintGS(i int) float64 { return float64(i) }

// GET /game — 自包含游戏页（终端风 UI，零外部资源，embed 进二进制）
func (ws *worldServer) handleGamePage(w http.ResponseWriter, r *http.Request) {
	data, err := wsWeb.ReadFile("wsweb/game.html")
	if err != nil {
		http.Error(w, "游戏页未安装", 404)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

// detectPixelTheme 从世界书标题/文件名推断像素套件主题（修仙/末世/西幻/克苏鲁/星际）
func detectPixelTheme(inst *worldInstance) string {
	if inst == nil || inst.wb == nil {
		return ""
	}
	text := inst.wb.Title + " " + inst.name
	if raw := strings.ToLower(strings.TrimSpace(inst.wb.Raw[:min(len(inst.wb.Raw), 600)])); raw != "" {
		text += " " + raw // 主题包世界书没有 A1 标题段，从头部原文猜题材
	}
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "修仙") || strings.Contains(lower, "仙") || strings.Contains(lower, "xianxia") || strings.Contains(lower, "cultivation"):
		return "xianxia"
	case strings.Contains(lower, "末世") || strings.Contains(lower, "废土") || strings.Contains(lower, "apocalypse") || strings.Contains(lower, "wasteland"):
		return "apocalypse"
	case strings.Contains(lower, "西幻") || strings.Contains(lower, "骑士") || strings.Contains(lower, "奇幻") || strings.Contains(lower, "fantasy") || strings.Contains(lower, "knight"):
		return "western"
	case strings.Contains(lower, "克苏鲁") || strings.Contains(lower, "异界") || strings.Contains(lower, "cosmic") || strings.Contains(lower, "cthulhu"):
		return "cosmic"
	case strings.Contains(lower, "星际") || strings.Contains(lower, "科幻") || strings.Contains(lower, "interstellar") || strings.Contains(lower, "star"):
		return "interstellar"
	}
	return ""
}

// spriteListPath 列出套件里某类 sprite 的 URL（存在与否由前端按 404 处理也行，这里给出理论列表）
func spriteListPath(theme, label string, n int) []string {
	urls := []string{}
	for i := 1; i <= n; i++ {
		urls = append(urls, fmt.Sprintf("/pixel-art/%s/%s-%d.png", theme, label, i))
	}
	return urls
}

// pixelArtPath 解析磁盘上的套件文件（顺序尝试 wsdata/art 先、docs/art 兜底）
func pixelArtPath(progDirs []string, theme, name string) string {
	for _, base := range progDirs {
		for _, sub := range []string{"art/pixel", "docs/art/pixel"} {
			p := filepath.Join(base, sub, theme, name)
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}
	return ""
}

// GET /pixel-art/{theme}/{file} —— 像素套件资产直出（磁盘文件，按需 fallback 到 docs/）
func (ws *worldServer) handlePixelArt(w http.ResponseWriter, r *http.Request) {
	theme := r.PathValue("theme")
	file := r.PathValue("file")
	if theme == "" || file == "" || !strings.HasSuffix(file, ".png") || strings.Contains(file, "/") {
		http.NotFound(w, r)
		return
	}
	// 资产解析顺序：程序目录(wsdata)/art/pixel → 程序目录 docs/art/pixel（开发态=仓库根）
	progRoot := filepath.Dir(ws.baseDir)
	candidates := []string{}
	for _, base := range []string{progRoot, filepath.Join(progRoot, "docs")} {
		for _, sub := range []string{"art/pixel", ""} {
			candidates = append(candidates,
				filepath.Join(base, sub, theme, file),
				filepath.Join(base, sub, theme, "sprites", file),
			)
		}
	}
	for _, p := range candidates {
		if data, err := os.ReadFile(p); err == nil {
			w.Header().Set("Content-Type", "image/png")
			w.Header().Set("Cache-Control", "public, max-age=3600")
			_, _ = w.Write(data)
			return
		}
	}
	http.NotFound(w, r)
}
