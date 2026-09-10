"""后端 API 功能测试：块编辑 / 配置保存 / 设置"""
import json, urllib.request

BASE = "http://localhost:48090"

def req(method, path, body=None):
    data = json.dumps(body).encode() if body is not None else None
    r = urllib.request.Request(BASE + path, data=data, method=method,
                               headers={"Content-Type": "application/json"})
    try:
        with urllib.request.urlopen(r, timeout=30) as resp:
            return resp.status, resp.read().decode()
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode()
    except Exception as e:
        return -1, str(e)

results = []
def check(name, cond, detail=""):
    results.append((name, bool(cond)))
    print(f"[{'PASS' if cond else 'FAIL'}] {name} {detail}")

# 1. 获取章节块（用第1章；当前选中项目需已生成章节）
num = "1"
st, body = req("GET", f"/api/chapters/{num}")
d = json.loads(body)
blocks = d.get("blocks") or []
check("块编辑-获取章节块", len(blocks) > 0, f"blocks={len(blocks)}")
if blocks:
    bid = blocks[0]["id"]
    orig = blocks[0]["text"]
    # 2. 修改块
    new_content = orig + "\n（块编辑测试 - 追加）"
    st2, body2 = req("PUT", f"/api/chapters/{num}/blocks/{bid}", {"text": new_content})
    check("块编辑-保存修改", st2 in (200, 202), f"HTTP={st2}")
    # 3. 恢复块
    st3, body3 = req("PUT", f"/api/chapters/{num}/blocks/{bid}", {"text": orig})
    check("块编辑-恢复原内容", st3 in (200, 202), f"HTTP={st3}")

# 4. 配置保存
st, body = req("GET", "/api/config")
cfg = json.loads(body)
check("配置-读取", isinstance(cfg, dict), "")
# 保存一个无害字段
saved = dict(cfg)
if "story" not in saved:
    saved["story"] = {}
saved["story"]["title"] = saved.get("story", {}).get("title") or "夜灯烬"
st4, body4 = req("PUT", "/api/config", saved)
check("配置-保存", st4 in (200, 202), f"HTTP={st4}")

# 5. 设置读取/保存
st5, body5 = req("GET", "/api/settings")
check("设置-读取", st5 == 200, f"HTTP={st5}")

# 6. 导出
st6, body6 = req("GET", "/api/export/txt")
check("导出-txt", st6 == 200 and len(body6) > 100, f"HTTP={st6} len={len(body6)}")

print(f"\n===== 汇总 {sum(1 for r in results if r[1])}/{len(results)} =====")
for r in results:
    if not r[1]: print("  失败:", r[0])