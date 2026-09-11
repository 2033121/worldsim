[中文](README.md) | English

# WorldSim — Multi-Agent World Simulator → Web-Novel Production Engine

> Let AI simulate "a world that actually runs", then automatically rewrite its

> chronicles into fiction that doesn't read like AI wrote it. Go single binary +
> WebUI, zero external dependencies. Deeply adapted from
> [Nigh/show-me-the-story](https://github.com/Nigh/show-me-the-story).
> A condensed English guide is kept here; the [Chinese README](README.md) is the
> authoritative full reference.
<p align="center"><img src="docs/art/hero-banner.png" width="100%" alt="WorldSim: a simulated world flowing into a novel scroll"/></p>

## What makes it different

- **Readiness-driven, not day-driven** — the simulation ends when there is *enough material to write from* (chapter seeds ≥3 + dramatic beats ≥12 + ≥1 foreshadow paid off + tension ≥0.4), not after "N days". Cultivation settings skip years; apocalypse settings move day by day; the LLM infers the time scale from the worldbook.
- **Playable text-game mode (Play Mode)** — the same world you simulated, now playable: free-form input in three modes (do / say / story). **Fate lives in the dice, not the prompt**: the code rolls d20 + attribute modifier vs a difficulty the referee LLM declares; the code owns the numbers (HP / level / XP / inventory / quest, persisted in `game.json`). The LLM does exactly two things — map your input to intent + difficulty, and narrate the already-decided outcome in second person. A "wait" turn advances the world on its own and heals you. Game stats sync back into engine entities, so a played session can feed the novel pipeline too. Opening a session auto-pauses the background simulation (turn-based, no token burn while idle). UI: `GET /game`.
- **Debate-to-novel pipeline** — a GM agent adjudicates against the worldbook, an event agent generates beats from the B5 event spectrum, the protagonist answers three value/ability/world-line questions, NPCs run Init→Act→React chains, and a foreshadow ledger tracks planting→ripening→payoff.
- **De-AI-flavored prose** — 886+ excerpts from real published web novels (source-titled) plus six writing methodology layers injected as prompts (memory pins, conflict hooks, sensory specificity, POV discipline, banned高频词).
- **Forgiving runtime** — event-sourced state engine with snapshot rewind to any anchor; an embedded self-healing module monitors LLM availability, service liveness, data integrity and loop stalls, and repairs automatically (snapshot rollback, dry-run fallback, loop interruption to stop token burn).
- **World → novel, zero LLM calls** — seed a novel outline and character sheets directly from the simulated world's worldbook / factions / recent events (`world_seed_novel`).
- **Built-in web search** — Tavily (switchable to SearXNG / Bing) for genre research and "search for reference material" while writing.

## 15 themes, one engine

15 theme packs (xianxia / apocalypse / western fantasy / cosmic horror / urban / interstellar / historical …) plus a universal worldbook template. `world_create(theme, desc)` → LLM generates the world book → protagonist, NPCs, locations → background loop → readiness check → novel. Xianxia worlds don't get apocalypse templates: the same engine adapts, not the templates.

## Quick start

```bash
go build -o worldsim .        # single ~10MB binary, zero deps
./worldsim /path/to/data-dir  # unified entry http://localhost:48092
```

Configure `api.json` (in the binary's directory — see Chinese README for the full schema including `model_tiers` fast/normal/premium). Then drive everything by API:

```bash
curl -X POST localhost:48091/api/worlds/create -H 'Content-Type: application/json' \
  -d '{"name":"Qinglan","theme":"经典修仙","desc":"a village boy finds a broken sword blank"}'
curl -X POST localhost:48091/api/world/init
curl -X POST localhost:48091/api/world/loop -H 'Content-Type: application/json' -d '{"action":"start","days":1000}'
curl localhost:48091/api/world/readiness          # ends early when ready
curl -X POST localhost:48091/api/world/novel/generate
curl localhost:48091/api/world/novel/chapter/1

# — or just play it: text-game mode on the same world —
curl -X POST localhost:48091/api/game/start                                   # generates theme-fit panel + scene
curl -X POST localhost:48091/api/game/action -H 'Content-Type: application/json' \
  -d '{"input":"inspect the broken sword blank","mode":"do"}'                  # code rolls, LLM narrates
curl -X POST localhost:48091/api/game/wait                                     # world ticks on (+1 engine day), you heal
curl -X POST localhost:48091/api/game/undo                                     # one-step undo: stats/bag/affection roll back
curl -o save.json localhost:48091/api/game/export                              # save file out…
curl -X POST localhost:48091/api/game/import -H 'Content-Type: application/json' --data-binary @save.json  # …and back in
```

Or open `http://localhost:48092` — the unified front end (browser-style shell:
tabs / address bar / Ctrl+K palette) hosting the novel app, the world console and
the text-game page natively, with a token-usage dashboard; serves the uiteg shell
and proxies `/api/novel/*`→:48090, `/api/*` & `/game`→:48091.

## MCP for AI agents

![](docs/art/mcp-hub.png)

`worldsim-mcp/server.py` is a zero-dependency MCP stdio server (26 tools, incl.
`world_loop_start`, `world_rewind`, `world_readiness`, `world_novel_generate`,
`world_seed_novel`) for Codex CLI / Trae / Claude / Cursor. Setup guides:
`worldsim-mcp/codex.md` and `worldsim-mcp/trae.md`; the repo-root `AGENTS.md`
documents collaboration conventions.

[Play Mode banner](docs/art/play-mode.png) — a jade d20 on a lantern-lit journal; the text-game face of the same engine.

**World-flavored play UI + World Cards (v1.9.0)** — the generated art now lives in the game: a scene banner rotating morning/dusk/night with the world clock, pixel portraits for NPCs present at your location with affection badges (referee declares, code clamps), item-sprite inventory grid, and a **deterministic map view** (sorted locations on a snake grid, tiles mapped by plan kind → place-name keywords → hash; identical state renders cell-for-cell identically). Worldbooks gained `W1 dynamic entries` (`- keywords => intel`): every turn scans the last 6 turns + your input, substring-matches (CJK-safe) with 3-turn sticky and a 1200-char budget, then injects the intel into the referee/narrator — a minimal SillyTavern World Info rebuilt for Chinese. And `GET /api/world/card` zips your worldbook + art + plan into a shareable card; `POST /api/worlds/import` lands it as a fresh playable world. The console gained a 🃏 World Card tab (export/import with optional game progress).

**Play feel (v1.10.0)** — a full usability round (audit → research → fixes, see `docs/可用性审计-v1.10.md`): fixed two fatal bugs in the standalone game page (submit URL ending with a stray dot → 405; input read after clearing → every action became "wait in place"); **↩ Undo last turn** (single-step rollback of stats/bag/affection/location, persisted with game.json, works across restarts); **downed state machine** (HP 0 → red warning banner, wait to recover); **save export/import** with full clamping; d20 roll-in animation, HP bar color by health, input history ↑/↓; waiting advances the engine day so the scene banner rotates while you play; turn handling refactored to short-lock + single-flight (status/log never block for up to 300 s mid-turn); `:48091/` now serves the same unified shell as the gateway (no more duplicate consoles).

**Art Studio (v1.8.0, built-in image generation)** — give every world its own pixel art. Open `/studio`, configure an image provider (OpenAI-images relay or PixelLab), and the planner AI reads your worldbook to produce an **asset plan**: 12 characters / 8 monsters / morning-dusk-night scenes / 16 map tiles / 24 items, each with a Chinese look sketch + an English prompt + a world-specific locked palette (editable in UI). One click batch-generates 5 sheets; the server then crops, flood-fill-removes the background and auto-heals empty cells in-process, delivering transparent sprites straight into the play page (`/art/{file}.png` takes priority, bundled suites are the fallback). Every generation is recorded in `art/history.json` for re-rolls.

Also ships a 20-character transparent pixel-sprite pack (agents / xianxia / apocalypse / western / sci-fi crews) under [`docs/art/pixel-sprites/`](docs/art/pixel-sprites/contact-sheet.png).

## Downloads

Prebuilt binaries for Linux (amd64/arm64), Windows, macOS (amd64/arm64), the
Operit plugin package (26 tools + WebUI + 15 theme packs + style corpus) and the
MCP server package are all on
[Releases](https://github.com/2033121/worldsim/releases). Tagging `v*` triggers
GitHub Actions to cross-compile all 5 platforms and assemble the assets
automatically.

## Security & IP

- `api.json` holds real keys — git-ignored, never commit. Use placeholders in examples.
- `worlds/` `storys/` are personal world data, never committed.
- `material/` (the style corpus, each excerpt credited with source book + chapter, style-reference only) **is** open-sourced in this repo.

## License

[MIT](LICENSE)
