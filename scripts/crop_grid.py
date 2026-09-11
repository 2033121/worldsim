#!/usr/bin/env python3
"""crop_grid.py — manifest 驱动的通用像素套件裁剪器（docs/art/pixel/<theme>/ -> sprites/）。

manifest 规则（GRIDS）：
  asset -> (cols, rows, 是否抠底)
切分：先按均匀网格切块 → 每块求非背景紧致 bbox（PAD=8）→ 洪泛抠底转透明 →
按 naming � Enabled 规则输出 {theme}/{asset}-{序号}.png；同时产出各主题 contact-sheet。
"""
import os
from collections import deque
from PIL import Image

SRC = "docs/art/pixel"
PAD = 8
GRIDS = {
    "characters": (4, 3, True, "character"),
    "monsters": (4, 2, True, "monster"),
    "scenes": (3, 1, True, "scene"),
    "maptiles": (4, 4, False, "tile"),
    "items": (6, 4, True, "item"),
}
LINE = "─" * 40

def bbox_of_nonbg(img, bg, thr=60):
    px = img.load()
    w, h = img.size
    minx, miny, maxx, maxy = w, h, -1, -1
    for y in range(h):
        for x in range(w):
            r, g, b = px[x, y][:3]
            if abs(r - bg[0]) + abs(g - bg[1]) + abs(b - bg[2]) > thr:
                if x < minx:
                    minx = x
                if x > maxx:
                    maxx = x
                if y < miny:
                    miny = y
                if y > maxy:
                    maxy = y
    if maxx < 0:
        return None
    return (max(0, minx - PAD), max(0, miny - PAD), min(w, maxx + PAD), min(h, maxy + PAD))

def to_alpha(img, bg, tol=78):
    px = img.load()   # img 必须 RGBA
    w, h = img.size
    seen = [[False] * w for _ in range(h)]
    dq = deque()
    for x in range(w):
        for y in (0, h - 1):
            dq.append((x, y))
    for y in range(h):
        for x in (0, w - 1):
            dq.append((x, y))
    while dq:
        x, y = dq.popleft()
        if seen[y][x]:
            continue
        seen[y][x] = True
        r, g, b = px[x, y][:3]
        if abs(r - bg[0]) + abs(g - bg[1]) + abs(b - bg[2]) >= tol:
            continue
        px[x, y] = (r, g, b, 0)
        for nx, ny in ((x - 1, y), (x + 1, y), (x, y - 1), (x, y + 1)):
            if 0 <= nx < w and 0 <= ny < h and not seen[ny][nx]:
                dq.append((nx, ny))
    return img

def main():
    themes = sorted(d for d in os.listdir(SRC) if os.path.isdir(os.path.join(SRC, d)))
    total = 0
    for theme in themes:
        dir_ = f"{SRC}/{theme}"
        out_dir = f"{dir_}/sprites"
        os.makedirs(out_dir, exist_ok=True)
        count = 0
        for asset, (cols, rows, alpha, label) in GRIDS.items():
            path = f"{dir_}/{asset}.png"
            if not os.path.exists(path):
                print(f"[{theme}] 缺 sheet: {asset}（跳过）")
                continue
            img = Image.open(path).convert("RGB")
            w, h = img.size
            bg = img.getpixel((2, 2))[:3]
            cw, ch = w / cols, h / rows
            produced = 0
            for r in range(rows):
                for c in range(cols):
                    idx = r * cols + c + 1
                    cellimg = img.crop((int(c * cw), int(r * ch), int((c + 1) * cw), int((r + 1) * ch)))
                    if alpha:
                        bb = bbox_of_nonbg(cellimg, bg)
                        if bb is None:
                            print(f"[{theme}/{asset}] cell {idx}: 空块")
                            continue
                        spr = to_alpha(cellimg.crop(bb).convert("RGBA"), bg)
                    else:
                        spr = cellimg.convert("RGBA")  # tile 不需紧致裁剪/抠底：整格即 tile
                    out = f"{out_dir}/{label}-{idx}.png"
                    spr.save(out)
                    produced += 1
                    count += 1
            print(f"[{theme}] {asset}: {produced}/{cols * rows}")
        total += count
        # contact sheet：按序号排序前 24 个
        files = sorted(os.listdir(out_dir))[:24]
        if files:
            cells = [Image.open(os.path.join(out_dir, f)).convert("RGBA") for f in files]
            mw = max(im.size[0] for im in cells) + 12
            mh = max(im.size[1] for im in cells) + 12
            sheet = Image.new("RGBA", (4 * mw, ((len(cells) + 3) // 4) * mh), (238, 238, 238, 255))
            for i, im in enumerate(cells):
                rr, cc = divmod(i, 4)
                sheet.alpha_composite(im, (cc * mw + 6, rr * mh + 6))
            sheet.save(f"{out_dir}/contact-sheet.png")
        print(LINE, f"{theme} 小计 {count}")
    print("TOTAL sprites:", total)

if __name__ == "__main__":
    main()
