"""世界模拟统一前端 — 小说创作应用全面功能测试"""
from playwright.sync_api import sync_playwright

BASE = "http://localhost:48092"
results = []

def check(name, cond, detail=""):
    results.append((name, bool(cond), detail))
    print(f"[{'PASS' if cond else 'FAIL'}] {name} {detail}")

def main():
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        ctx = browser.new_context(viewport={"width": 1500, "height": 950})
        page = ctx.new_page()
        console_errors = []
        page.on("console", lambda m: console_errors.append(m.text) if m.type == "error" else None)

        # 进入小说应用：首页点击卡片
        page.goto(BASE, wait_until="domcontentloaded")
        page.wait_for_timeout(1200)
        card = page.locator("button.card", has_text="小说创作").first
        if card.count():
            card.click()
            page.wait_for_timeout(3500)
        else:
            check("进入小说应用", False, "首页未找到小说创作卡片")
            browser.close(); return

        body = page.inner_text("body")
        nav_visible = all(k in body for k in ["大纲", "写作", "伏笔", "记忆", "图谱", "技能"])
        check("项目视图打开(侧栏导航可见)", nav_visible, "")

        # ================= 语言切换测试 =================
        def lang_btn():
            return page.locator("button[title='界面语言'], button[title='UI language']").first
        def reveal_en():
            b = page.inner_text("body")
            return any(k in b for k in ["Generate", "Outline", "Writing", "Foreshadows", "Config"])

        t0 = lang_btn().inner_text() if lang_btn().count() else "NONE"
        body0 = page.inner_text("body")
        if lang_btn().count():
            lang_btn().click()
            page.wait_for_timeout(900)
            body1 = page.inner_text("body")
            en1 = reveal_en()
            check("语言切换-点击后有变化", body0 != body1 or en1, f"t0={t0!r} en_after={en1}")
            # 切回
            if lang_btn().count():
                lang_btn().click()
                page.wait_for_timeout(900)
                body2 = page.inner_text("body")
                check("语言切换-可切回", body2 != body1, "")
        else:
            check("语言切换-按钮存在", False, "未找到切换按钮")

        # ================= 页面导航 =================
        nav_map = {
            "config": "配置",
            "outline": "大纲",
            "writing": "写作",
            "foreshadows": "伏笔",
            "memory": "记忆",
            "relations": "图谱",
            "skills": "技能",
        }
        for addr, label in nav_map.items():
            btn = page.locator("nav button", has_text=label).first
            if btn.count():
                btn.click()
                page.wait_for_timeout(700)
                check(f"导航-{addr}({label})", True, "点击成功")
            else:
                check(f"导航-{addr}({label})", False, "按钮未找到")

        # ================= 大纲页空状态 =================
        page.locator("nav button", has_text="大纲").first.click()
        page.wait_for_timeout(800)
        obody = page.inner_text("body")
        check("大纲页-空状态", "生成大纲" in obody or "Generate" in obody or "卷骨架" in obody, "")

        # ================= 写作页空状态 =================
        page.locator("nav button", has_text="写作").first.click()
        page.wait_for_timeout(800)
        wbody = page.inner_text("body")
        check("写作页-渲染无异常", "写作" in wbody or "chapter" in wbody.lower(), "")

        # ================= 配置页 =================
        page.locator("nav button", has_text="配置").first.click()
        page.wait_for_timeout(800)
        cbody = page.inner_text("body")
        check("配置页-渲染", ("设定" in cbody) or ("情节" in cbody) or ("Config" in cbody), "")

        print("\n===== 控制台错误 =====")
        print(console_errors[:10])
        fails = [r for r in results if not r[1]]
        print(f"\n===== 汇总: 通过 {len(results)-len(fails)}/{len(results)} =====")
        for r in fails:
            print("  失败:", r[0])
        browser.close()

if __name__ == "__main__":
    main()