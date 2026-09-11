#!/usr/bin/env python3
"""gen_pixel_suite.py — 像素套件批量生成器（5 主题 × 人物/怪物/场景/地图/物品 …）。

产物：docs/art/pixel/<theme>/*.png（sheet 级），裁剪由 crop_grid.py 按 manifest 执行。
风格契约对象齐书：16-bit 像素风、厚描边、正面站姿（人物/怪物）、统一底色、零文字。
"""
import ast, json, os, subprocess, sys, time

KEY = os.environ["GPTIMG_KEY"]
BASE = os.environ["GPTIMG_BASE"].rstrip("/")
OUT = "docs/art/pixel"

THIN = "16-bit console pixel art, crisp hard pixels, thick dark outlines"

STYLE_ROLE = (  # 人物/怪物 共用
    "Rear background: plain solid uniform very-light-grey (#e8e8e8) covering the whole canvas edge-to-edge. "
    "Strict strict grid as specified, each cell exactly one full-figure centered with identical feet baseline. "
    + THIN + ", consistent 40-color theme palette, no text/letters/numbers/logos/watermarks/grid lines/shadows. Original artwork only."
)

PALETTES = {
    "xianxia": "ink-blue + jade + antique gold + vermilion accents on rice-cream",
    "apocalypse": "ash-grey + moss + ember-orange + rust accents",
    "western": "steel + royal blue + parchment + gold trim (knight-crest blue and gold)",
    "cosmic": "blackwater teal + bone-white + sickly jade + deep-ink purple accents",
    "interstellar": "deep teal + emerald nebula + warm brass cabin + thin gold star lines",
}

# 每主题的 5 张：(asset, grid(label), 尺寸, 内容描述)
ASSETS = [
    ("characters", "12 characters in a 4x3 (4 columns x 3 rows) grid", "1536x1024",
     "A diverse controllable-character cast of the theme: protagonist, core companions, mentors, rivals, walk-on villagers — 12 distinct individuals."),
    ("monsters", "8 monsters in a 4x2 (4 columns x 2 rows) grid", "1536x1024",
     "Threat & creature roster native to the theme, from common minions to minibosses — let the theme define shapes, no anime clichés unless natural."),
    ("scenes", "Three scene panels side-by-side (morning / dusk / night) divided by clean vertical washes", "1792x1024",
     "One wide banner showing the SAME iconic location of the theme at three times of day, with different weather/mood per panel."),
    ("maptiles", "16 map tiles in a 4x4 (4 columns x 4 rows) grid, viewed top-down (bird's eye), fully tileable", "1024x1024",
     "Tiles the theme's overworld would be built from: safe ground, road, dangerous ground, water, water edge, forest/trees, rock, wall, ruin floor, campsite, bridge, gate, pit, treasure spot, transport point, special landmark."),
    ("items", "24 item icons in a 6x4 (6 columns x 4 rows) grid, each icon a full clean sprite item with 1-pixel rim", "1536x1024",
     "Usables & assets the theme's economy would carry: weapons, tools, consumables, currency pouch, valuables, quest tokens, curios, transportation tokens."),
]

THEME_DESC = {
    "xianxia": "Chinese xianxia (cultivation) world: ink mountains, sects, spirit beasts, talisman craft, immortal cities.",
    "apocalypse": "post-apocalyptic wasteland: collapsed infrastructure, scavenger camps, mutation and ash weather.",
    "western": "western fantasy kingdom: knights, mage orders, taverns, dragons on mountain keeps.",
    "cosmic": "cosmic-horror coastal New England fishing villages: blackwater, forbidden monoliths, unknowable sea.",
    "interstellar": "interstellar migration era: ringed planets, convoy fleets, orbital stations, star map charts.",
}

def build_prompt(theme, grid_desc, content):
    return (
        f"PROMPT GOAL: a pixel-art asset sheet for the \"{theme}\" theme of a world-simulation game.\n\n"
        f"Theme context: {THEME_DESC[theme]}\n"
        f"Sheet layout: {grid_desc}.\n"
        f"Contents: {content}\n\n"
        f"Palette: {PALETTES[theme]}.\n\n"
        "Constraints: " + STYLE_ROLE
    )

def one(theme, asset, grid_desc, size, content):
    """按 curl→下载→落盘 的路径出一张图；返回 True/False"""
    dst = f"{OUT}/{theme}/{asset}.png"
    if os.path.exists(dst):
        print("skip:", dst)
        return True
    body = json.dumps({"model": "gpt-image-2", "prompt": build_prompt(theme, grid_desc, content), "size": size, "n": 1})
    req_path = f"/tmp/pxs_{theme}_{asset}.req"
    open(req_path, "w").write(body)
    for attempt in range(3):
        try:
            t0 = time.time()
            code = subprocess.run(
                ["curl", "-s", "--max-time", "300", "-o", f"/tmp/pxs_{theme}_{asset}.json", "-w", "%{http_code}",
                 "-X", "POST", BASE + "/v1/images/generations",
                 "-H", "Authorization: Bearer " + KEY, "-H", "Content-Type: application/json",
                 "--data-binary", f"@{req_path}"],
                capture_output=True, text=True, timeout=310).stdout
            data = json.load(open(f"/tmp/pxs_{theme}_{asset}.json"))
            url = (data.get("data") or [{}])[0].get("url")
            if not url:
                print(f"no url {theme}/{asset} HTTP {code}: {json.dumps(data)[:160]}")
                time.sleep(4)
                continue
            os.makedirs(os.path.dirname(dst), exist_ok=True)
            subprocess.run(["curl", "-sfL", "--max-time", "180", "-o", dst, url], check=True)
            print(f"OK {theme}/{asset} {os.path.getsize(dst)}B in {time.time()-t0:.0f}s")
            return True
        except Exception as e:
            print(f"retry {theme}/{asset} ({attempt+1}/3): {e}")
            time.sleep(5)
    print(f"FAIL {theme}/{asset}")
    return False

if __name__ == "__main__":
    os.makedirs(OUT, exist_ok=True)
    only_theme = sys.argv[1:] or None
    fails = []
    for theme in ["xianxia", "apocalypse", "western", "cosmic", "interstellar"]:
        if only_theme and theme not in only_theme:
            continue
        print("="*8, theme)
        for asset, grid_desc, size, content in ASSETS:
            dst = f"{OUT}/{theme}/{asset}.png"
            if only_theme == ["maptiles"] or asset == "maptiles":
                pass
            if os.path.exists(dst):
                print("skip:", dst)
                continue
            one(theme, asset, grid_desc, size, content)
    print("suite batch finished")
