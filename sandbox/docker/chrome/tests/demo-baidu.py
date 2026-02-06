"""
Baidu Search Demo — PyAutoGUI OS-level Mouse + Smart Keyboard Fallback

Demonstrates anti-detection browser automation inside sandbox-claude-chrome:
  1. Open Baidu homepage
  2. Click search box with PyAutoGUI (real OS mouse event)
  3. Type query with PyAutoGUI keyboard (auto-fallback to Playwright if needed)
  4. Submit search
  5. Click first search result (PyAutoGUI mouse) → view detail page
  6. Go back
  7. Click second search result (PyAutoGUI mouse) → view detail page

All mouse clicks are OS-level X11 events via PyAutoGUI — undetectable by websites.

Prerequisites:
  - Running inside sandbox-claude-chrome container
  - DISPLAY=:99 (Xvfb virtual display)
  - VNC optional for live observation (http://localhost:6080)

Usage:
  DISPLAY=:99 python3 demo-baidu.py
"""

from playwright.sync_api import sync_playwright
from playwright_stealth import Stealth
import pyautogui
import time
import random
import os

# ---------------------------------------------------------------------------
# PyAutoGUI config
# ---------------------------------------------------------------------------
pyautogui.FAILSAFE = False
pyautogui.PAUSE = 0.1

# Screenshot output directory
SCREENSHOT_DIR = os.environ.get("SCREENSHOT_DIR", "/workspace")


# ---------------------------------------------------------------------------
# Human-like helpers
# ---------------------------------------------------------------------------
def human_move(x, y):
    """Move mouse with randomized speed to simulate human behavior."""
    duration = random.uniform(0.4, 0.8)
    pyautogui.moveTo(x, y, duration=duration)
    time.sleep(random.uniform(0.1, 0.3))


def human_click(x, y):
    """Move to (x, y) then click — mimics a real user click."""
    human_move(x, y)
    time.sleep(random.uniform(0.05, 0.15))
    pyautogui.click()
    time.sleep(random.uniform(0.2, 0.5))


def smart_type(element, text):
    """Type text with PyAutoGUI first; fallback to Playwright if it didn't land.

    On native amd64 Linux, PyAutoGUI keyboard works perfectly.
    On ARM Mac (Rosetta 2), X11 keyboard events may not reach Chrome,
    so we detect and automatically fallback to Playwright's type().
    """
    # Attempt PyAutoGUI keyboard (OS-level X11 events)
    for ch in text:
        pyautogui.press(ch)
        time.sleep(random.uniform(0.05, 0.12))
    time.sleep(0.5)

    # Verify input landed
    actual = element.input_value()
    if actual and len(actual) >= len(text) * 0.8:
        print("   Keyboard: PyAutoGUI (OS-level) ✓", flush=True)
        return

    # Fallback: Playwright type()
    print("   PyAutoGUI keyboard didn't land — fallback to Playwright", flush=True)
    element.fill("")
    element.type(text, delay=80)
    actual = element.input_value()
    print("   Keyboard: Playwright fallback — '{}'".format(actual), flush=True)


def find_element(page, selectors, min_width=50, timeout=2000):
    """Try multiple CSS selectors, return (element, bounding_box) or (None, None)."""
    for selector in selectors:
        try:
            el = page.locator(selector).first
            box = el.bounding_box(timeout=timeout)
            if box and box["width"] >= min_width:
                return el, box
        except Exception:
            continue
    return None, None


def find_results(page, selectors, min_count=2, min_width=100):
    """Find clickable search result elements with bounding boxes.

    For Baidu: results are <h3><a href="..." target="_blank">title</a></h3>.
    We need the <a> element — it's the actual clickable link.
    """
    results = []
    for selector in selectors:
        elements = page.locator(selector).all()
        for el in elements[:10]:
            try:
                box = el.bounding_box(timeout=1000)
                title = el.text_content().strip()
                if box and box["width"] >= min_width and title and len(title) > 5:
                    if not any(r["title"] == title for r in results):
                        results.append({"box": box, "title": title, "el": el})
            except Exception:
                pass
        if len(results) >= min_count:
            break
    return results


def click_result(ctx, page, result, label):
    """Click a search result using PyAutoGUI and handle new tab navigation.

    Baidu results have target=_blank, so clicking opens a new tab.
    We scroll the element into view first, then use PyAutoGUI for the OS-level click.
    """
    el = result["el"]

    # Scroll element to a safe click zone.
    # Baidu results page has a tall fixed search bar at the top (~150px).
    # We need the element at y > 300 to avoid clicking the search input.
    #
    # Strategy: use PyAutoGUI mouse wheel scroll (real OS event) to position
    # the element in the middle of the viewport, then re-read coordinates.
    el.scroll_into_view_if_needed(timeout=3000)
    time.sleep(0.3)
    box = el.bounding_box(timeout=2000)

    if box and box["y"] < 300:
        # Element is too close to top — behind the fixed search bar.
        # Use Playwright mouse.wheel to scroll page UP (negative deltaY)
        # so the element moves DOWN in the viewport to a safe y > 350.
        delta = int(box["y"]) - 400  # negative value scrolls page up
        print("   Scrolling page (delta={}) to clear fixed header (y={})".format(
            delta, int(box["y"])), flush=True)
        page.mouse.wheel(0, delta)
        time.sleep(0.8)

    # Re-read bounding box after scroll
    box = el.bounding_box(timeout=2000)
    if not box:
        print("   ⚠ Lost element after scroll", flush=True)
        return False

    # Click the left portion of the link text (more reliable than center)
    rx = int(box["x"] + min(box["width"] * 0.3, 150))
    ry = int(box["y"] + box["height"] / 2)
    print("   [PyAutoGUI] Clicking at ({},{})".format(rx, ry), flush=True)

    pages_before = len(ctx.pages)
    human_click(rx, ry)

    # Wait for new tab — Baidu links have target=_blank, so clicking should
    # open a new tab. Give it enough time for the Baidu redirect.
    for _ in range(10):
        page.wait_for_timeout(500)
        if len(ctx.pages) > pages_before:
            break

    if len(ctx.pages) > pages_before:
        target = ctx.pages[-1]
        target.wait_for_timeout(6000)
        title = target.title()
        url = target.url
        print("   [New Tab] {} | {}".format(title[:50], url[:80]), flush=True)
        screenshot(target, label)
        target.close()
        page.wait_for_timeout(1000)
        return True
    else:
        # Check if URL changed (same-tab navigation via Baidu redirect)
        current_url = page.url
        if "baidu.com/s?" not in current_url:
            page.wait_for_timeout(5000)
            print("   Landed: {} | {}".format(page.title()[:50], page.url[:80]), flush=True)
            screenshot(page, label)
            print("   [PyAutoGUI] Going back...", flush=True)
            pyautogui.hotkey("alt", "Left")
            page.wait_for_timeout(3000)
            return True
        else:
            print("   ⚠ Click didn't navigate — still on search page", flush=True)
            print("   Trying Playwright click as fallback...", flush=True)
            el.click(timeout=5000)
            page.wait_for_timeout(3000)
            if len(ctx.pages) > pages_before:
                target = ctx.pages[-1]
                target.wait_for_timeout(6000)
                print("   [New Tab via Playwright] {} | {}".format(
                    target.title()[:50], target.url[:80]), flush=True)
                screenshot(target, label)
                target.close()
                page.wait_for_timeout(1000)
                return True
            elif "baidu.com/s?" not in page.url:
                page.wait_for_timeout(5000)
                print("   [Playwright] Landed: {}".format(page.title()[:50]), flush=True)
                screenshot(page, label)
                pyautogui.hotkey("alt", "Left")
                page.wait_for_timeout(3000)
                return True
            return False


def screenshot(page, name):
    """Save screenshot to SCREENSHOT_DIR."""
    path = os.path.join(SCREENSHOT_DIR, name)
    page.screenshot(path=path)
    print("   Screenshot: " + path, flush=True)


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
def main():
    print("=" * 60, flush=True)
    print("  Baidu Search — PyAutoGUI OS-Level Demo", flush=True)
    print("=" * 60, flush=True)

    with Stealth().use_sync(sync_playwright()) as p:
        browser = p.chromium.launch(
            channel="chrome",
            headless=False,
            args=[
                "--no-sandbox",
                "--disable-blink-features=AutomationControlled",
                "--disable-dev-shm-usage",
                "--window-size=1920,1080",
                "--window-position=0,0",
            ],
        )
        ctx = browser.new_context(
            viewport={"width": 1920, "height": 1080},
            locale="zh-CN",
            timezone_id="Asia/Shanghai",
        )
        page = ctx.new_page()

        # Inject stealth script
        stealth_path = "/usr/local/share/yao/stealth-init.js"
        if os.path.exists(stealth_path):
            page.add_init_script(open(stealth_path).read())

        # ---- Step 1: Open Baidu ----
        print("\n[1/7] Opening Baidu...", flush=True)
        page.goto("https://www.baidu.com", timeout=30000)
        page.wait_for_timeout(3000)
        print("   Title: " + page.title(), flush=True)
        screenshot(page, "baidu-01-homepage.png")

        # ---- Step 2: Find & click search box ----
        print("\n[2/7] Finding search box...", flush=True)
        search_selectors = [
            "#kw", "input[name=wd]", "input[name=word]",
            "input.s_ipt", "input[type=text]", "input[type=search]",
        ]
        search_el, box = find_element(page, search_selectors)

        if not box:
            # Fallback: Baidu search box typical position
            print("   Using fallback coordinates", flush=True)
            box = {"x": 600, "y": 350, "width": 600, "height": 40}

        cx = int(box["x"] + box["width"] / 2)
        cy = int(box["y"] + box["height"] / 2)
        print("   [PyAutoGUI] Clicking search box at ({}, {})".format(cx, cy), flush=True)
        human_move(100, 100)
        time.sleep(0.3)
        human_click(cx, cy)

        # ---- Step 3: Type search query ----
        print("\n[3/7] Typing search query...", flush=True)
        query = "yao app engine"
        if search_el:
            smart_type(search_el, query)
        else:
            # No element reference — type blindly with PyAutoGUI
            for ch in query:
                pyautogui.press(ch)
                time.sleep(random.uniform(0.06, 0.12))
        time.sleep(1)
        screenshot(page, "baidu-02-typed.png")

        # ---- Step 4: Submit search ----
        print("\n[4/7] Submitting search...", flush=True)
        pyautogui.press("enter")
        page.wait_for_timeout(5000)
        print("   URL: " + page.url[:100], flush=True)
        print("   Title: " + page.title(), flush=True)
        screenshot(page, "baidu-03-results.png")

        # ---- Step 5: Find results ----
        print("\n[5/7] Finding search results...", flush=True)
        result_selectors = [".c-container h3 a", "h3 a", "a:has(h3)"]
        results = find_results(page, result_selectors)
        print("   Found {} results".format(len(results)), flush=True)
        for i, r in enumerate(results[:5]):
            print("   [{}] {}".format(i + 1, r["title"][:60]), flush=True)

        if len(results) < 3:
            print("\n   ⚠ Not enough results. Page may show CAPTCHA.", flush=True)
            screenshot(page, "baidu-04-no-results.png")
        else:
            # Baidu results page has a fixed search bar at the top (~150px).
            # The first result (index 0) is often right under it, making
            # PyAutoGUI click hit the search input instead of the link.
            # So we click results starting from index 1 (second result).

            # ---- Step 6: Click result #2 ----
            print("\n[6/7] Clicking result #2: " + results[1]["title"][:60], flush=True)
            click_result(ctx, page, results[1], "baidu-04-page1.png")

            # ---- Step 7: Click result #3 ----
            # Re-find results (page may have scrolled, coordinates changed)
            page.evaluate("window.scrollTo(0, 0)")
            page.wait_for_timeout(1000)
            results2 = find_results(page, result_selectors)
            print("\n   Re-found {} results".format(len(results2)), flush=True)

            if len(results2) >= 3:
                print("\n[7/7] Clicking result #3: " + results2[2]["title"][:60], flush=True)
                click_result(ctx, page, results2[2], "baidu-05-page2.png")
            else:
                print("\n[7/7] ⚠ Could not re-find results for second click", flush=True)

        # Keep browser open for VNC observation
        print("\n" + "=" * 60, flush=True)
        print("  Demo complete! Browser stays open 30s for observation.", flush=True)
        print("  Connect via VNC: http://localhost:6080", flush=True)
        print("=" * 60, flush=True)
        page.wait_for_timeout(30000)
        browser.close()

    print("Done.", flush=True)


if __name__ == "__main__":
    main()
