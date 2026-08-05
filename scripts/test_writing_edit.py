"""小说创作 — 写作页与编辑功能深度测试"""
from playwright.sync_api import sync_playwright

BASE = "http://localhost:48092"

def main():
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        ctx = browser.new_context(viewport={"width": 1560, "height": 950})
        page = ctx.new_page()
        errors = []
        page.on("console", lambda m: errors.append(m.text) if m.type == "error" else None)
        results = []
        def check(name, cond, detail=""):
            results.append((name, bool(cond)))
            print(f"[{'PASS' if cond else 'FAIL'}] {name} {detail}")

        # 进入小说应用（当前项目已选中）
        page.goto(BASE, wait_until="domcontentloaded"); page.wait_for_timeout(1000)
        card = page.locator("button.card", has_text="小说创作").first
        if card.count():
            card.click(); page.wait_for_timeout(3500)
        else:
            check("进入小说应用", False, "卡片未找到"); browser.close(); return

        # 项目应已打开（有侧栏）
        check("项目打开", page.locator("nav").count() > 0, "")

        # 进入写作页
        page.locator("nav button", has_text="写作").first.click()
        page.wait_for_timeout(1500)
        wbody = page.inner_text("body")
        check("写作页-章节列表显示", "烬余灯" in wbody or "Outline" in wbody, "")
        check("写作页-正文内容显示", len(wbody) > 200, f"body_len={len(wbody)}")

        # 检查正文内容是否显示（数字数）
        content_el = page.locator(".chapter-content, [class*='chapter-content']").first
        if content_el.count():
            txt = content_el.inner_text()
            check("写作页-正文区域有内容", len(txt) > 100, f"content_len={len(txt)}")
        else:
            # 备用：检查是否有块级内容
            check("写作页-正文区域存在", False, "未找到正文容器")

        # 整章编辑按钮
        fulledit_btn = page.locator("button", has_text="整章编辑").first
        check("写作页-整章编辑按钮存在", fulledit_btn.count() > 0, "")
        if fulledit_btn.count():
            fulledit_btn.click()
            page.wait_for_timeout(800)
            # 应出现 textarea
            ta = page.locator("textarea").first
            check("整章编辑-文本域出现", ta.count() > 0, "")
            if ta.count():
                cur = ta.input_value()
                check("整章编辑-载入正文", len(cur) > 100, f"len={len(cur)}")
                # 修改并保存
                new_text = cur + "\n\n【测试追加一行】夜色更深，灯火阑珊。"
                ta.fill(new_text)
                page.locator("button", has_text="保存整章").first.click()
                page.wait_for_timeout(1500)
                # 检查保存后的 toast 或内容
                body_after = page.inner_text("body")
                check("整章编辑-保存成功", "已保存" in body_after or "saved" in body_after.lower(), "")
                # 恢复原内容（避免污染）
                # （保存后重新进入编辑，恢复原稿）
                page.locator("button", has_text="整章编辑").first.click()
                page.wait_for_timeout(600)
                ta2 = page.locator("textarea").first
                if ta2.count():
                    ta2.fill(cur)
                    page.locator("button", has_text="保存整章").first.click()
                    page.wait_for_timeout(1200)

        print("\n===== 控制台错误 =====")
        print(errors[:10])
        print(f"\n===== 汇总 {sum(1 for r in results if r[1])}/{len(results)} =====")
        for r in results:
            if not r[1]: print("  失败:", r[0])
        browser.close()

if __name__ == "__main__":
    main()