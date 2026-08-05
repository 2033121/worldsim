"""worldsim 统一前端验证脚本 — 首页 + 小说创作页（避免 networkidle 超时）"""
from playwright.sync_api import sync_playwright

BASE = "http://localhost:48092"

def main():
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        page = browser.new_page(viewport={"width": 1440, "height": 900})

        console_errors = []
        bad_reqs = []
        page.on("console", lambda m: console_errors.append(m.text) if m.type == "error" else None)
        page.on("response", lambda r: bad_reqs.append(f"{r.status} {r.url}") if r.status >= 400 else None)

        # ---- 1. 首页 ----
        page.goto(BASE, wait_until="domcontentloaded")
        page.wait_for_timeout(1200)
        title = page.title()
        h1 = page.locator("h1").first.inner_text() if page.locator("h1").count() else ""
        cards = page.locator("button.card").count()
        all_pages = page.locator("button.btn-ink").count()
        print("TITLE:", title)
        print("H1:", h1)
        print("APP_CARDS:", cards)
        print("ALL_PAGES_BTN:", all_pages)
        page.screenshot(path="verify_home.png", full_page=True)

        # ---- 2. 点击小说创作卡片 ----
        novel_card = page.locator("button.card", has_text="小说创作").first
        if novel_card.count():
            novel_card.click()
            page.wait_for_timeout(3000)
            body = page.inner_text("body")
            has_novel = ("小说" in body) or ("项目" in body) or ("选择故事项目" in body) or ("大纲" in body) or ("写作" in body)
            print("ENTERED_NOVEL:", has_novel)
            page.screenshot(path="verify_novel.png", full_page=False)
        else:
            print("NOVEL_CARD_NOT_FOUND")

        # ---- 3. 回到首页（domcontentloaded）----
        page.goto(BASE, wait_until="domcontentloaded")
        page.wait_for_timeout(800)

        print("CONSOLE_ERRORS:", console_errors[:8])
        print("BAD_REQS:", bad_reqs[:8])
        print("RESULT:", "PASS" if not console_errors and not bad_reqs else "ISSUES")
        browser.close()

if __name__ == "__main__":
    main()