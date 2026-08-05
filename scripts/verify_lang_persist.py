"""验证语言切换持久性修复：切换后刷新应保持所选语言"""
from playwright.sync_api import sync_playwright

BASE = "http://localhost:48092"

def reveal_lang(page):
    b = page.inner_text("body")
    # 侧栏导航词可唯一区分小说应用 UI 语言（外层外壳固定中文，不含这些词）
    if "Outline" in b and "Writing" in b:
        return "EN"
    if "大纲" in b and "写作" in b and "Outline" not in b:
        return "ZH"
    return "REFRESH-MODE"

def main():
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        ctx = browser.new_context(viewport={"width": 1500, "height": 950})
        page = ctx.new_page()
        errors = []
        page.on("console", lambda m: errors.append(m.text) if m.type == "error" else None)

        # 进入小说应用
        page.goto(BASE, wait_until="domcontentloaded"); page.wait_for_timeout(1000)
        page.locator("button.card", has_text="小说创作").first.click()
        page.wait_for_timeout(3000)
        print("初始语言:", reveal_lang(page))

        # 点击切换按钮到英文
        toggle = page.locator("button[title='界面语言'], button[title='UI language']").first
        print("TOGGLE_COUNT:", page.locator("button[title='界面语言'], button[title='UI language']").count())
        if toggle.count():
            print("TOGGLE_TEXT_BEFORE:", toggle.inner_text())
            toggle.click()
            page.wait_for_timeout(1200)
            print("切换后语言:", reveal_lang(page))
            # 打印侧栏导航文本片段
            nav_text = page.locator("nav").first.inner_text() if page.locator("nav").count() else "NO-NAV"
            print("NAV_TEXT:", nav_text.replace("\n"," | ")[:120])
        else:
            print("切换按钮未找到")
            browser.close(); return

        # 刷新页面，验证语言保持
        page.reload(wait_until="domcontentloaded")
        page.wait_for_timeout(3000)
        print("刷新后语言:", reveal_lang(page))
        if reveal_lang(page) == "EN":
            print("RESULT: PASS - 语言切换持久化")
        else:
            print("RESULT: FAIL - 刷新后回退")

        print("CONSOLE_ERRORS:", errors[:6])
        browser.close()

if __name__ == "__main__":
    main()