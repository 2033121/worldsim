#!/usr/bin/env python3
"""crop_pixel_sheets.py — 把 pixel-sheets/*.png 四宫格裁成 20 个独立像素小人。

处理：按四个象限切分 → 每象限按非背景像素求紧致 bbox（带 8px padding）→
把象限的均匀灰底转透明 → 输出 pixel-sprites/{sheet}-{n}.png。
同时输出一张 4x5 排版的 contact-sheet 预览图。
"""
import os
from PIL import Image

SRC = "docs/art/pixel-sheets"
DST = "docs/art/pixel-sprites"
PAD = 10

def bbox_of_nonbg(img, bg):
    px = img.load()
    w, h = img.size
    minx, miny, maxx, maxy = w, h, -1, -1
    for y in range(h):
        for x in range(w):
            r, g, b = px[x, y][:3]
            dr = abs(r - bg[0]) + abs(g - bg[1]) + abs(b - bg[2])
            if dr > 60:  # 排除均匀灰底与纸粒噪声
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
    return max(0, minx - PAD), max(0, miny - PAD), min(w, maxx + PAD), min(h, maxy + PAD)

def to_alpha(img, bg):
    """从四边出发的连通域洪泛：底色可能带渐变/晕影，逐像素固定阈值抠不净。
    判定：与种子色的总差值 < tol → 视为背景（像素画硬边，tol 需压过渐变但保留描边）。"""
    from collections import deque
    px = img.load()
    w, h = img.size
    tol = 78  # sum of abs channel deltas；经验值：晕影渐变 + 纸感 + JPEG-ish 边缘
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
            continue  # 撞到实体，停止此路
        px[x, y] = (r, g, b, 0)
        for nx, ny in ((x-1,y),(x+1,y),(x,y-1),(x,y+1)):
            if 0 <= nx < w and 0 <= ny < h and not seen[ny][nx]:
                dq.append((nx, ny))
    return img

def main():
    os.makedirs(DST, exist_ok=True)
    names = {}
    total = 0
    for fn in sorted(os.listdir(SRC)):
        path = os.path.join(SRC, fn)
        stem = fn.replace(".png", "").replace("sheet-", "")
        img = Image.open(path).convert("RGB")
        w, h = img.size
        bg = img.getpixel((2, 2))[:3]
        quads = {
            "q1": (0, 0, w // 2, h // 2),
            "q2": (w // 2, 0, w, h // 2),
            "q3": (0, h // 2, w // 2, h),
            "q4": (w // 2, h // 2, w, h),
        }
        out_files = []
        for qi, (qname, box) in enumerate(quads.items(), 1):
            q = img.crop(box)
            bb = bbox_of_nonbg(q, bg)
            if bb is None:
                print(f"  {stem}.{qi}: 空象限，跳过")
                continue
            sprite = q.crop(bb).convert("RGBA")
            sprite = to_alpha(sprite, bg)
            out = os.path.join(DST, f"{stem}-{qi}.png")
            sprite.save(out)
            out_files.append(out)
            total += 1
        names[stem] = out_files
        print(f"{stem}: {len(out_files)} 人")
    print("total sprites:", total)

    # 4x5 contact sheet（4 列 5 行，每格 320px 高）
    cells = []
    maxw = maxh = 0
    for stem, files in names.items():
        for f in files:
            im = Image.open(f).convert("RGBA")
            cells.append(im)
            maxw, maxh = max(maxw, im.size[0]), max(maxh, im.size[1])
    cw, ch = maxw + 16, maxh + 16
    sheet = Image.new("RGBA", (4 * cw, 5 * ch), (238, 238, 238, 255))
    for i, im in enumerate(cells):
        r, c = divmod(i, 4)
        sheet.alpha_composite(im, (c * cw + 8 + (cw - 16 - im.size[0]) // 2, r * ch + 8))
        if i >= 19:
            break
    sheet.save(os.path.join(DST, "contact-sheet.png"))
    print("contact sheet OK ->", os.path.join(DST, "contact-sheet.png"))

if __name__ == "__main__":
    main()
