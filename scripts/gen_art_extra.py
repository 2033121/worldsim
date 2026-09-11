#!/usr/bin/env python3
"""gen_art_extra.py — 二批包装图：主题包封面 ×5 + MCP 接入图 + 时间回退图。

结构同 gen_art.py（场景→主体→布局→风格→光照→显式排除约束，全图零文字）。
密钥走环境变量 GPTIMG_KEY / GPTIMG_BASE，不落数据。
"""
import base64, json, os, sys, time, urllib.request

KEY = os.environ["GPTIMG_KEY"]
BASE = os.environ["GPTIMG_BASE"].rstrip("/")
OUT = "docs/art"

NO_TEXT = "No text, no letters, no numbers, no logos, no watermarks. Original artwork only. No photorealistic faces, no clutter."

JOBS = [
    # ---- 5 张主题包封面（竖版 3:4 图有："横版"改为 "tall vertical cover", 尺寸 1024x1536）----
    ("theme-xianxia", """Tall vertical cover illustration for a Chinese xianxia (cultivation fantasy) world-theme pack.

Scene: a young swordsman's silhouette on a spire above a moonlit sea of clouds; floating stone islands carrying a Daoist temple; distant immortal city gate half-hidden; drifting lotus petals and a falling pagoda lantern.
Layout: tall framing (2:3), subject in lower third, generous quiet sky above for a title overlay; strong vertical rhythm.
Style: refined Chinese ink-wash (shuimo) on rice paper, flowing brushwork, muted ink blues with vermilion and antique gold accents; atmospheric depth, poem-worthy negative space.
Lighting/mood: cold moonlight with a single warm lantern glow, quiet ambition.

Constraints: """ + NO_TEXT, "1024x1536"),
    ("theme-apocalypse", """Tall vertical cover illustration for a post-apocalyptic Chinese wasteland world-theme pack.

Scene: a lone survivor seen from behind at the foot of a collapsed overpass, dawn haze, drifting ash, sun-bleached vending machines and swaying grass reclaiming asphalt; a broken highway sign leans like a monument; small warm light of a survivor camp deep in the fog.
Layout: tall framing (2:3), survivor small in lower third, vast ruin-sky above; breathing negative space.
Style: Chinese ink-wash meets desaturated watercolor, rice-paper grain, ash-grey palette with one ember-orange accent that reads as hope.
Lighting/mood: cold morning light against faint ember warmth, melancholy yet resilient.

Constraints: """ + NO_TEXT, "1024x1536"),
    ("theme-western-fantasy", """Tall vertical cover illustration for a western fantasy world-theme pack under a Chinese ink-wash visual language.

Scene: a knight's silhouette raising a sword at the edge of a floating cathedral ruin above a valley; a dragon's breath curls into ink over a distant castle; banners flutter up the cliff toward a bright rift in the sky.
Layout: tall framing (2:3), diagonal ascent from knight to sky-rift; clean margins for title overlay.
Style: ink-wash + soft watercolor washes, rice paper, slate and gold palette, vermilion sigil accents; confident contour brushwork.
Lighting/mood: golden rift-light spilling down the diagonal, heroic but restrained.

Constraints: """ + NO_TEXT, "1024x1536"),
    ("theme-cosmic-horror", """Tall vertical cover illustration for a cosmic-horror world-theme pack, brushed in Chinese ink-wash.

Scene: a blackwater fishing village at night seen from a sinking jetty; a colossal organic monolith rises beyond the islet shrine, wrapped in fog; three lanterns burn green on the rushing tide; a single paper talisman drifts mid-air.
Layout: tall framing (2:3), monolith looms above the top third cut by mist; village lights small in lower third.
Style: ink-wash with heavy black washes bleeding into wet paper, bone-white untouched paper as the silence between shapes, sickly jade-green lantern accents.
Lighting/mood: dead-calm water, oppressive fog, one forbidden glow.

Constraints: """ + NO_TEXT, "1024x1536"),
    ("theme-interstellar", """Tall vertical cover illustration for an interstellar sci-fi world-theme pack under a Chinese ink-wash visual language.

Scene: a bridge window's massive curvature looking out at a ringed colossus planet and a jade nebula; a distant convoy of migration ships draws a thin luminous line across the dark; inside the bridge, an ink-toned star chart floats mid-air.
Layout: tall framing (2:3), the ringed planet upper-left with a broad diagonal to a small ship silhouette lower-right; deep vignette margins.
Style: ink-wash translated to space: nebulae as wet ink blooms, star lines as dry-brush gold strokes, rice-paper texture underneath teal-black depths.
Lighting/mood: cold starlight with warm ember cabin reflection, vast and quietly hopeful.

Constraints: """ + NO_TEXT, "1024x1536"),
    # ---- MCP / AI 客户端接入（isometric，cookbook 建议的架构图风格）----
    ("mcp-hub", """Isometric 3D concept illustration for an open-source engine's MCP (AI-agent tool hub) integration diagram.

Scene: a dark navy world-model table at center carrying an ink-styled miniature landscape; around it, three floating isometric client panels — one shaped like a chat window, one like a code editor window, one like a mobile phone — each connected to the central hub by glowing cyan conduit ribs; small crates of abstract tool-cards (scroll, dice, book icons as blank shapes) ride the conduits.
Layout: isometric 3/4 view, hub centered with even breathing room; wide framing on a dark background whose bottom fades to deep navy — suitable as a README diagram banner.
Style: clean geometric isometric shapes, minimal strokes, subtle soft shadows, deep navy + electric cyan + one vermilion accent, no gradients heavier than soft rims.
Lighting/mood: neon rims on matte surfaces, confident developer-tool aesthetic.

Constraints: no text, no letters, no logos, no watermarks. Original artwork only. No photorealistic humans.
""", "1536x1024"),
    # ---- 时间回退（快照制）概念图 ----
    ("rewind", """Wide horizontal concept illustration for the snapshot rewind (time travel) feature of a world simulation.

Scene: a single ink-landscape scroll floats above an open journal; at its center one burning vermilion seal-stamp anchor glows; from the anchor the scroll splits into three ghost-translucent forks, each fork's snow-line re-inking different scattered versions of the same mountains and river — one fork flourishing, one fork in drifting snow, one fork returning to blank paper.
Layout: wide framing, anchor slightly right of center as the focal point; forks fan left with clean spacing; generous top margin for title overlay.
Style: Chinese ink-wash on rice paper, translucent layering for the three fork versions, gold thread stitching the branch lines, vermilion red only on the anchor.
Lighting/mood: quiet hour-glass atmosphere, lantern-warm at the anchor fading to cool mist at fork tips.

Constraints: no text, no letters, no numbers, no logos, no watermarks. Original artwork only. No photorealistic faces, no clutter.
""", "1536x1024"),
]

def gen(name, prompt, size):
    body = json.dumps({"model": "gpt-image-2", "prompt": prompt, "size": size, "n": 1}).encode()
    req = urllib.request.Request(
        BASE + "/v1/images/generations", data=body, method="POST",
        headers={"Authorization": "Bearer " + KEY, "Content-Type": "application/json", "User-Agent": "curl/8.5.0"})
    d = json.loads(urllib.request.urlopen(req, timeout=280).read())
    item = d["data"][0]
    raw = None
    if item.get("b64_json"):
        raw = base64.b64decode(item["b64_json"])
    elif item.get("url"):
        # Node fetch 不认代理，本机场景扩图床由 curl 兜底（这里直接用 curl 子进程语义照样走 shell 由调用方处理；
        # 简化：urllib 用 UA，不符时抛错由阻塞重试下载）
        img_req = urllib.request.Request(item["url"], headers={"User-Agent": "curl/8.5.0"})
        try:
            raw = urllib.request.urlopen(img_req, timeout=150).read()
        except Exception:
            import subprocess
            raw = subprocess.check_output(["curl", "-sfL", "--max-time", "150", item["url"]], maxBuffer=0) if False else None
            raw = subprocess.check_output(["curl", "-sfL", "--max-time", "150", item["url"]])
    if not raw:
        raise RuntimeError("no image bytes: " + json.dumps(item)[:200])
    path = os.path.join(OUT, name + ".png")
    open(path, "wb").write(raw)
    return path

if __name__ == "__main__":
    only = sys.argv[1:] or None
    for name, prompt, size in JOBS:
        if only and name not in only:
            continue
        dst = os.path.join(OUT, name + ".png")
        if os.path.exists(dst):
            print("skip (exists):", name)
            continue
        for attempt in range(3):
            try:
                t0 = time.time()
                path = gen(name, prompt, size)
                print(f"OK {name} {os.path.getsize(path)}B in {time.time()-t0:.0f}s")
                break
            except Exception as e:
                print(f"retry {name} ({attempt+1}/3): {e}")
                time.sleep(5)
    print("batch done")
