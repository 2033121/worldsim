package main

// ws_worldfx.go — 游玩页「世界观感」扩展（v1.9.0）：
//   /api/game/status 增补 scene（场景横幅）、portraits（在场角色头像）、
//   item_icons（背包物品图标）、map（确定性网格地图）——素材来自美术工坊 plan
//   或打包套件；没有美术时全部优雅降级（字段缺省，前端隐藏）。
//
// 设计纪律（借鉴 talemate world_state + SimTracker 生态，落地在我们代码层数值纪律上）：
//   - 布局/图标映射全部**确定性**（排序+snake 网格+关键词映射），LLM 不参与，
//     同一 state 两次渲染逐格一致（有单测锁定）；
//   - 匹配不到素材的名字用稳定 hash 取索引，绝不报错阻塞。

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	art "worldsim/internal/art"
	game "worldsim/internal/game"
)

// ---------- plan.json TTL 缓存（一次 status 最多触发 5 次 LoadPlan，重复读盘收口） ----------

var planCacheMu sync.Mutex
var planCache = map[string]planCacheEntry{}

type planCacheEntry struct {
	at   time.Time
	plan *art.Plan
}

const planCacheTTL = 10 * time.Second

// loadPlanCached 带 TTL 的 plan 读取（工坊改规划后最多延迟 10s 生效——渲染层可接受）
func loadPlanCached(inst *worldInstance) *art.Plan {
	if inst == nil {
		return nil
	}
	planCacheMu.Lock()
	defer planCacheMu.Unlock()
	if e, ok := planCache[inst.dir]; ok && time.Since(e.at) < planCacheTTL {
		return e.plan
	}
	plan, _ := art.LoadPlan(inst.dir)
	planCache[inst.dir] = planCacheEntry{at: time.Now(), plan: plan}
	return plan
}

// ---------- 素材名册（plan 优先，打包套件兜底） ----------

// assetRoster 返回某类 sprite 的 (素材名列表, URL 列表)
// fileLabel: character|monster|scene|tile|item
func assetRoster(inst *worldInstance, fileLabel string) ([]string, []string) {
	if inst == nil {
		return nil, nil
	}
	count := map[string]int{"character": 12, "monster": 8, "scene": 3, "tile": 16, "item": 24}[fileLabel]
	if count == 0 {
		return nil, nil
	}
	// plan 优先：本世界专属
	if plan := loadPlanCached(inst); plan != nil {
		var names []PlanEntry
		switch fileLabel {
		case "character":
			names = plan.Characters
		case "monster":
			names = plan.Monsters
		case "scene":
			names = plan.Scenes
		case "tile":
			names = plan.Tiles
		case "item":
			names = plan.Items
		}
		planNames := make([]string, len(names))
		urls := make([]string, len(names))
		for i, e := range names {
			planNames[i] = e.Name
			urls[i] = fmt.Sprintf("/art/%s-%d.png", fileLabel, i+1)
		}
		if len(names) > 0 {
			return planNames, urls
		}
	}
	// 打包套件兜底
	if theme := detectPixelTheme(inst); theme != "" {
		urls := make([]string, count)
		for i := range urls {
			urls[i] = fmt.Sprintf("/pixel-art/%s/%s-%d.png", theme, fileLabel, i+1)
		}
		return nil, urls
	}
	return nil, nil
}

// PlanEntry 是 plan 条目的最小接口（避免引入整个 art.Plan 类型签名）
type PlanEntry = art.PlanAsset

// matchAssetIndex 在素材名里精确/子串匹配；失败用稳定 hash 兜底（1..total）
func matchAssetIndex(names []string, target string, total int) int {
	for i, n := range names {
		if n == target {
			return i + 1
		}
	}
	for i, n := range names {
		if n != "" && (strings.Contains(target, n) || strings.Contains(n, target)) {
			return i + 1
		}
	}
	return hashIndex(target, total) + 1
}

// hashIndex 稳定字符串 hash（FNV-1a）
func hashIndex(s string, mod int) int {
	if mod <= 0 {
		return 0
	}
	var h uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return int(h) % mod
}

// ---------- 场景横幅 ----------

// sceneURLFor 昼夜轮换：day%3 → scenes[0..2]（晨/暮/夜按世界时钟轮转）
func sceneURLFor(inst *worldInstance, day int) string {
	_, urls := assetRoster(inst, "scene")
	if len(urls) == 0 {
		return ""
	}
	return urls[day%len(urls)]
}

// ---------- 在场角色头像 ----------

// presentPortraits 在场（同地点 active、非主角）角色 → {name,img,relation}
func presentPortraits(inst *worldInstance) []map[string]any {
	if inst == nil || inst.engine == nil {
		return nil
	}
	st := inst.engine.State()
	hero := inst.heroName
	if inst.sim != nil && inst.sim.HeroName() != "" {
		hero = inst.sim.HeroName()
	}
	heroEnt, ok := st.Entities[hero]
	if !ok || heroEnt.Location == "" {
		return nil
	}
	g := inst.lazyGame().StateCurrent()
	charNames, charURLs := assetRoster(inst, "character")
	names := make([]string, 0, len(st.Entities))
	for name, e := range st.Entities {
		if name == hero || e.Status != "active" {
			continue
		}
		if e.Location != "" && e.Location == heroEnt.Location {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	out := make([]map[string]any, 0, len(names))
	for _, name := range names {
		m := map[string]any{"name": name}
		if len(charURLs) > 0 {
			m["img"] = charURLs[(matchAssetIndex(charNames, name, len(charURLs))-1)%len(charURLs)]
		}
		if g.Relations != nil {
			if r, has := g.Relations[name]; has {
				m["relation"] = r
			}
		}
		out = append(out, m)
	}
	return out
}

// ---------- 背包物品图标 ----------

// inventoryIcons 背包条目 → {name,img}
func inventoryIcons(inst *worldInstance, g *game.GameState) []map[string]any {
	if g == nil || len(g.Inventory) == 0 {
		return nil
	}
	itemNames, itemURLs := assetRoster(inst, "item")
	out := make([]map[string]any, 0, len(g.Inventory))
	for _, it := range g.Inventory {
		m := map[string]any{"name": it}
		if len(itemURLs) > 0 {
			m["img"] = itemURLs[(matchAssetIndex(itemNames, it, len(itemURLs))-1)%len(itemURLs)]
		}
		out = append(out, m)
	}
	return out
}

// ---------- 确定性地图 ----------

// stdTileKeyword 地名关键词 → 标准 tile 语义位（1..16）
// 标准套件谱系：1安全地 2道路 3危险地 4水 5水缘 6林 7岩 8墙 9遗迹 10营地 11桥 12门 13坑 14宝点 15传送点 16地标
func stdTileKeyword(name string) int {
	lower := strings.ToLower(name)
	has := func(keys ...string) bool {
		for _, k := range keys {
			if strings.Contains(lower, k) {
				return true
			}
		}
		return false
	}
	switch {
	case has("水", "湖", "河", "海", "water", "lake", "river"):
		return 4
	case has("林", "树", "森", "forest"):
		return 6
	case has("山", "岩", "崖", "rock", "mount"):
		return 7
	case has("墙", "wall"):
		return 8
	case has("遗迹", "废墟", "ruin"):
		return 9
	case has("营", "寨", "camp"):
		return 10
	case has("桥", "bridge"):
		return 11
	case has("门", "关", "gate"):
		return 12
	case has("坑", "pit"):
		return 13
	case has("宝", "treasure"):
		return 14
	case has("传送", "港", "transport"):
		return 15
	case has("城", "镇", "村", "街", "市", "city", "town"):
		return 2
	case has("险", "danger"):
		return 3
	case has("safe", "出发"):
		return 1
	default:
		return 0 // 无语义 → 用安全地或 hash 兜底
	}
}

// mapLayout 从引擎实体 location 集合构建确定性网格地图。
// 布局：地名排序后按 snake（奇数行反向）填入 ceil(sqrt(n)) 网格；
// tile 序号：plan 有 kind 语义位时优先映射，否则关键词语义位，最后 hash；
// 返回 {side, cells:[{name,tile,x,y}], hero:{x,y}, npcs:[在场NPC名]}。
func mapLayout(inst *worldInstance, g *game.GameState) map[string]any {
	if inst == nil || inst.engine == nil {
		return nil
	}
	st := inst.engine.State()
	hero := inst.heroName
	if inst.sim != nil && inst.sim.HeroName() != "" {
		hero = inst.sim.HeroName()
	}
	// 1) 收集地点
	seen := map[string]bool{}
	names := []string{}
	if g != nil && g.Location != "" {
		seen[g.Location] = true
		names = append(names, g.Location)
	}
	for _, e := range st.Entities {
		if e.Location != "" && !seen[e.Location] {
			seen[e.Location] = true
			names = append(names, e.Location)
		}
	}
	if len(names) == 0 {
		return nil
	}
	sort.Strings(names)
	side := 2
	for side*side < len(names) {
		side++
	}
	// 2) plan kind 语义位 → tile 序号
	kindTile := map[string]int{}
	if plan := loadPlanCached(inst); plan != nil {
		for i, t := range plan.Tiles {
			if t.Kind != "" {
				if _, dup := kindTile[t.Kind]; !dup {
					kindTile[t.Kind] = i + 1
				}
			}
		}
	}
	stdKind := map[int]string{1: "safe_ground", 2: "road", 3: "dangerous", 4: "water", 5: "water_edge", 6: "forest", 7: "rock", 8: "wall", 9: "ruin", 10: "camp", 11: "bridge", 12: "gate", 13: "pit", 14: "treasure", 15: "transport", 16: "landmark"}
	tileFor := func(name string) int {
		idx := stdTileKeyword(name)
		if idx > 0 {
			if kind, has := stdKind[idx]; has {
				if t, ok := kindTile[kind]; ok {
					return t
				}
			}
			return idx // 套件谱系位即 tile 序号
		}
		return hashIndex(name, 16) + 1
	}
	// 3) snake 布局
	heroEnt, heroOK := st.Entities[hero]
	cells := make([]map[string]any, 0, len(names))
	posOf := map[string]map[string]int{}
	for i, name := range names {
		row := i / side
		col := i % side
		if row%2 == 1 {
			col = side - 1 - col
		}
		cells = append(cells, map[string]any{"name": name, "tile": tileFor(name), "x": col, "y": row})
		posOf[name] = map[string]int{"x": col, "y": row}
	}
	// 4) 主角定位 + 在场 NPC
	heroPos := map[string]int{"x": 0, "y": 0}
	npcNames := []string{}
	if heroOK && heroEnt.Location != "" && posOf[heroEnt.Location] != nil {
		heroPos = posOf[heroEnt.Location]
	} else if g != nil && posOf[g.Location] != nil {
		heroPos = posOf[g.Location]
	}
	if heroEnt, ok2 := st.Entities[hero]; ok2 && heroEnt.Location != "" {
		for name, e := range st.Entities {
			if name == hero || e.Status != "active" || e.Location != heroEnt.Location {
				continue
			}
			npcNames = append(npcNames, name)
		}
	}
	sort.Strings(npcNames)
	base := "/art/"
	if theme := detectPixelTheme(inst); theme != "" {
		if _, err := os.Stat(filepath.Join(inst.dir, "art", "plan.json")); err != nil {
			base = "/pixel-art/" + theme + "/"
		}
	}
	return map[string]any{"side": side, "cells": cells, "hero": heroPos, "npcs": npcNames, "base": base}
}
