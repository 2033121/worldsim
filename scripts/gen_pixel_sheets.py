#!/usr/bin/env python3
"""gen_pixel_sheets.py — 生成像素小人 sprite sheet（每张 4 人，共 5 张 → 20 人）。

统一约束：16-bit 像素风、厚描边、正面站姿、全队共同浅灰底色 + 一致比例，
保证 2x2 四宫格裁剪与基线一致。裁剪与其后处理在 crop_pixel_sheets.py。
"""
import json, os, subprocess, sys, time

KEY = os.environ["GPTIMG_KEY"]
BASE = os.environ["GPTIMG_BASE"].rstrip("/")
OUT = "docs/art/pixel-sheets"

RULE = (
    "Pixel-art sprite sheet, 16-bit console style, crisp hard pixels, thick dark outlines, "
    "front-facing full-body standing pose, one character per quadrant of a strict 2x2 grid, "
    "equal character size and identical feet baseline across all four, centered inside each quadrant, "
    "plain solid uniform very-light-grey background (#e8e8e8) covering the whole canvas edge-to-edge, "
    "soft paper-grain-free flat color, consistent 40-color palette, no text, no letters, no numbers, "
    "no logos, no watermarks, no grid lines, no shadow. Original artwork only."
)

SHEETS = [
    ("sheet-agents", "Four persona sprites of the worldsim multi-agent crew, cute chibi proportions: "
        "a director in long robe holding a rolled blank scroll; a quirky scribe holding a huge quill pen; "
        "a watcher holding a glowing brass lantern; a gamer holding a single blank d20 dice orb."),
    ("sheet-xianxia", "Four Chinese xianxia cultivation sprites: a sword cultivator with a flying sword; "
        "a talisman master with paper talismans floating; an alchemist with a tiny incense burner; a spirit herbalist cradling a glowing herb."),
    ("sheet-apocalypse", "Four post-apocalypse survivor sprites: a scavenger with a crowbar and backpack; "
        "a guard in a welding mask holding a crossbow; a medic with a red-cross-free camo medkit; a radio operator with a big antenna radio."),
    ("sheet-western", "Four western-fantasy tavern sprites: a knight with kite shield and sword; a robed mage with a staff orb; "
        "a bard with a mandolin; a gnome engineer with a wrench."),
    ("sheet-scifi", "Four interstellar sci-fi sprites: a starfarer captain with a futuristic visor; a combat android with twin arm cannons; "
        "an alien botanist with a bioluminescent pod; a space engineer with welding goggles and a plasma torch."),
]


if __name__ == "__main__":
    only = sys.argv[1:] or None
    for name, desc in SHEETS:
        if only and name not in only:
            continue
        prompt = "A 2x2 sprite sheet grid, each quadrant containing exactly one standing character.\n\nCharacters: " + desc + "\n\nConstraints: " + RULE
        dst = f"docs/art/pixel-sheets/{name}.png"
        if os.path.exists(dst):
            print("skip:", name)
            continue
        ok = False
        for attempt in range(3):
            try:
                t0 = time.time()
                tmp_body = json.dumps({"model": "gpt-image-2", "prompt": prompt, "size": "1536x1024", "n": 1})
                open(f"/tmp/px_{name}.req", "w").write(tmp_body)
                code = subprocess.run(
                    ["curl", "-s", "--max-time", "280", "-o", f"/tmp/px_{name}.json", "-w", "%{http_code}",
                     "-X", "POST", BASE + "/v1/images/generations",
                     "-H", "Authorization: Bearer " + KEY, "-H", "Content-Type: application/json",
                     "--data-binary", f"@/tmp/px_{name}.req"],
                    capture_output=True, text=True, timeout=300).stdout
                url = json.load(open(f"/tmp/px_{name}.json"))["data"][0].get("url")
                if not url:
                    print(f"no url: {name}", code, open(f"/tmp/px_{name}.json").read()[:120])
                    continue
                subprocess.run(["curl", "-sfL", "--max-time", "150", "-o", dst, url], check=True)
                print(f"OK {name} {os.path.getsize(dst)}B in {time.time()-t0:.0f}s")
                ok = True
                break
            except Exception as e:
                print(f"retry {name} ({attempt+1}/3): {e}")
                time.sleep(5)
        if not ok:
            print(f"FAIL {name}")
    print("sheets done")
