#!/usr/bin/env python3
"""gen_art.py — 批量生成 worldsim 包装图（gpt-image-2，OpenAI 兼容 /v1/images/generations）

用法： GPTIMG_KEY=... GPTIMG_BASE=... python3 gen_art.py
产物： docs/art/*.png（不再写入 API key 到任何仓库文件）
"""
import base64, json, os, sys, time, urllib.request

KEY = os.environ["GPTIMG_KEY"]
BASE = os.environ["GPTIMG_BASE"].rstrip("/")
OUT = os.environ.get("GPTIMG_OUT", "docs/art")

JOBS = [
    # (输出名, 提示词, 尺寸)  —— 提示词结构对齐 openai-cookbook：场景→主体→布局→风格→光照→约束
    ("hero-banner", """Wide hero banner illustration for the README of an open-source engine that simulates a whole world with AI agents and rewrites it into Chinese web novels.

Scene: on the left-of-center, a miniature Eastern fantasy world floats above an unfurling ancient scroll — a pagoda, layered mountains, sea of clouds, a tiny walled town with pinhead lights; the scroll's ink strokes flow rightward and transform into elegant narrative text-lines that condense into a glowing manuscript page with a bamboo brush resting on it.
Layout: wide 16:9-style framing at eye level; generous negative space in the upper area and along the bottom edge for a markdown title overlay; single calm focal axis from scroll to manuscript.
Style: Chinese ink-wash (shuimo) painting on warm rice-paper texture, flowing brushwork, muted black-ink palette with vermilion and antique-gold accents; subtle paper grain.
Lighting/mood: soft diffuse dawn light, faint gold bloom around the manuscript, serene and confident.

Constraints:
- No text, no letters, no numbers, no logos, no watermarks
- No photorealistic humans, no clutter, one focal composition only
- Original artwork only
""", "1536x1024"),
    ("play-mode", """Wide banner illustration for the playable text-adventure mode of a world-simulator project.

Scene: a night study desk lit by one warm brass lantern; center-left an open journal glowing like a soft terminal screen with abstract ink paragraphs; beside it stands a carved jade twenty-sided die as the visual hero; above the journal hovers a single blank oracle-card pouring thin ink trails; small paper-cutout character silhouettes line the journal's edge like a waiting queue.
Layout: wide framing, dice as singular focal point slightly right of center; generous empty space at top and right for title overlay; eye-level view.
Style: Chinese ink-wash blended with modern flat illustration, teal-night background, rice-paper grain overlay, vermilion and gold accents surviving the dark palette.
Lighting/mood: lantern warmth against cool night, gentle glow on the die facets, mysterious but inviting.

Constraints:
- No text, no letters, no numbers, no logos, no watermarks
- No photorealistic humans, no clutter, single focal subject
- Original artwork only
""", "1536x1024"),
    ("themes-triptych", """Wide triptych illustration for one simulation engine that runs any genre world.

Scene: three vertical panels joined by flowing ink washes. Left panel: a xianxia swordsman silhouette on a mountain spire above cloud-sea with floating islands and a distant immortal city. Middle panel: a wasteland survivor beside a ruined stone arch, drifting ash, cold haze, broken road. Right panel: a starfarer silhouette at a bridge window facing a colossal ringed planet and emerald nebula.
Layout: even thirds, one shared horizon energy left to right; wide framing; margins kept clean above each panel.
Style: unified Chinese ink-wash on rice paper, gold accent lines threading all three panels, each panel distinct mood yet one visual language.
Lighting/mood: warm dawn in left panel, cold daylight in middle, deep starlight in right.

Constraints:
- No text, no letters, no numbers, no logos, no watermarks
- No photorealistic faces, no clutter
- Original artwork only
""", "1536x1024"),
    ("agent-mandala", """Square emblem illustration for a multi-agent world simulation engine (social preview / site icon use).

Scene: a circular ink constellation on warm rice paper. Center: a serene ink-brushstroke globe with a tiny pagoda silhouette. Six orbiting ink-dot figures connected by delicate gold constellation lines — a director holding a scroll, a quill-writer, a lantern-watcher, a branching tree of fate, a gamer holding a single blank die, and a small closed book.
Layout: perfectly centered symmetric mandala, generous padding to all four edges, icon-safe.
Style: refined ink-wash minimalism, vermilion seal-red accent dots, antique gold lines, subtle paper grain.
Lighting/mood: soft diffuse daylight, balanced and dignified.

Constraints:
- No text, no letters, no numbers, no logos, no watermarks
- No photorealistic humans, no clutter
- Original artwork only
""", "1024x1024"),
]

def gen(name, prompt, size):
    body = json.dumps({"model": "gpt-image-2", "prompt": prompt, "size": size, "n": 1}).encode()
    req = urllib.request.Request(
        BASE + "/v1/images/generations", data=body, method="POST",
        headers={"Authorization": "Bearer " + KEY, "Content-Type": "application/json", "User-Agent": "curl/8.5.0"})
    d = json.loads(urllib.request.urlopen(req, timeout=240).read())
    item = d["data"][0]
    if item.get("b64_json"):
        raw = base64.b64decode(item["b64_json"])
        path = os.path.join(OUT, name + ".png")
        open(path, "wb").write(raw)
        return path
    url = item.get("url")
    if not url:
        raise RuntimeError("no image in response: " + json.dumps(item)[:200])
    raw = urllib.request.urlopen(url, timeout=120).read()
    path = os.path.join(OUT, name + ".png")
    open(path, "wb").write(raw)
    return path

if __name__ == "__main__":
    os.makedirs(OUT, exist_ok=True)
    only = sys.argv[1:] or None
    ok = 0
    for name, prompt, size in JOBS:
        if only and name not in only:
            continue
        dst = os.path.join(OUT, name + ".png")
        if os.path.exists(dst):
            print("skip (exists):", name)
            ok += 1
            continue
        for attempt in range(3):
            try:
                t0 = time.time()
                path = gen(name, prompt, size)
                print(f"OK {name} {os.path.getsize(path)}B in {time.time()-t0:.0f}s")
                ok += 1
                break
            except Exception as e:
                print(f"retry {name} ({attempt + 1}/3): {e}")
                time.sleep(5)
    print("done", ok, "/", len(JOBS))
