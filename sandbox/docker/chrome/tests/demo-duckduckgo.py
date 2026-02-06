"""
DuckDuckGo Search Demo — PyAutoGUI OS-level Mouse + Smart Keyboard Fallback

Demonstrates anti-detection browser automation inside sandbox-claude-chrome:
  1. Open DuckDuckGo homepage
  2. Click search box with PyAutoGUI (real OS mouse event)
  3. Type query with PyAutoGUI keyboard (auto-fallback to Playwright if needed)
  4. Submit search
  5. Click first search result (PyAutoGUI mouse)
  6. Go back
  7. Click second search result (PyAutoGUI mouse)

DuckDuckGo has no IP-based rate limiting or reCAPTCHA, making it ideal for
demonstrating pure anti-fingerprint capabilities without IP interference.

Prerequisites:
  - Running inside sandbox-claude-chrome container
  - DISPLAY=:99 (Xvfb virtual display)
  - VNC optional for live observation (http://localhost:6080)

Usage:
  DISPLAY=:99 python3 demo-duckduckgo.py
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


def find_results(page, selectors, min_count=2, min_width=80):
    """Find search result elements with bounding boxes."""
    results = []
    for selector in selectors:
        elements = page.locator(selector).all()
        for el in elements[:10]:
            try:
                box = el.bounding_box(timeout=1000)
                title = el.text_content().strip()
                if (box and box["width"] >= min_width and box["y"] > 0
                        and title and len(title) > 5):
                    if not any(r["title"] == title for r in results):
                        results.append({"box": box, "title": title})
            except Exception:
                pass
        if len(results) >= min_count:
            break
    return results


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
    print("  DuckDuckGo Search — PyAutoGUI OS-Level Demo", flush=True)
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
            locale="en-US",
            timezone_id="America/New_York",
        )
        page = ctx.new_page()

        # Inject stealth script
        stealth_path = "/usr/local/share/yao/stealth-init.js"
        if os.path.exists(stealth_path):
            page.add_init_script(open(stealth_path).read())

        # ---- Step 1: Open DuckDuckGo ----
        print("\n[1/7] Opening DuckDuckGo...", flush=True)
        page.goto("https://duckduckgo.com", timeout=30000)
        page.wait_for_timeout(3000)
        print("   Title: " + page.title(), flush=True)
        screenshot(page, "ddg-01-homepage.png")

        # ---- Step 2: Find & click search box ----
        print("\n[2/7] Finding search box...", flush=True)
        search_selectors = [
            "input[name=q]", "#searchbox_input",
            "input[type=text]", "input[placeholder*='Search']",
        ]
        search_el, box = find_element(page, search_selectors, min_width=100)

        if not box:
            print("   ⚠ Could not find search box!", flush=True)
            screenshot(page, "ddg-02-no-searchbox.png")
            browser.close()
            return

        cx = int(box["x"] + box["width"] / 2)
        cy = int(box["y"] + box["height"] / 2)
        print("   [PyAutoGUI] Clicking search box at ({}, {})".format(cx, cy), flush=True)
        human_move(200, 200)
        time.sleep(0.3)
        human_click(cx, cy)

        # ---- Step 3: Type search query ----
        print("\n[3/7] Typing search query...", flush=True)
        query = "yao app engine github"
        smart_type(search_el, query)
        time.sleep(0.5)
        screenshot(page, "ddg-02-typed.png")

        # ---- Step 4: Submit search ----
        print("\n[4/7] Submitting search...", flush=True)
        # Use Playwright Enter on the element (reliable cross-platform)
        search_el.press("Enter")
        page.wait_for_timeout(6000)
        print("   URL: " + page.url[:120], flush=True)
        print("   Title: " + page.title()[:80], flush=True)
        screenshot(page, "ddg-03-results.png")

        # Check if we actually reached results page
        on_results = (
            "q=" in page.url
            or "/search" in page.url
            or page.title() != "DuckDuckGo - Protection. Privacy. Peace of mind."
        )
        if not on_results:
            print("   ⚠ Still on homepage — search may not have submitted", flush=True)
            screenshot(page, "ddg-03-still-homepage.png")
            browser.close()
            return

        # ---- Step 5: Find results ----
        print("\n[5/7] Finding search results...", flush=True)
        result_selectors = [
            "article h2 a", "a[data-testid='result-title-a']",
            "h2 a[href]", "ol li h2 a", "h2 a",
        ]
        results = find_results(page, result_selectors, min_count=3)
        print("   Found {} results".format(len(results)), flush=True)
        for i, r in enumerate(results[:5]):
            print("   [{}] {}".format(i + 1, r["title"][:70]), flush=True)

        if len(results) < 2:
            print("\n   ⚠ Not enough results to click.", flush=True)
            screenshot(page, "ddg-04-no-results.png")
        else:
            # ---- Step 6: Click first result ----
            r1 = results[0]
            rx = int(r1["box"]["x"] + r1["box"]["width"] / 2)
            ry = int(r1["box"]["y"] + r1["box"]["height"] / 2)
            print("\n[6/7] [PyAutoGUI] Clicking result #1 at ({},{})".format(rx, ry), flush=True)
            print("   " + r1["title"][:70], flush=True)
            pyautogui.scroll(-1)
            time.sleep(0.3)
            human_click(rx, ry)
            page.wait_for_timeout(6000)
            print("   Landed: {} | {}".format(page.title()[:50], page.url[:80]), flush=True)
            screenshot(page, "ddg-04-page1.png")

            # Go back
            print("   [PyAutoGUI] Going back (Alt+Left)...", flush=True)
            pyautogui.hotkey("alt", "Left")
            page.wait_for_timeout(4000)

            # ---- Step 7: Click second result ----
            results2 = find_results(page, result_selectors, min_count=3)
            if len(results2) >= 2:
                r2 = results2[1]
                rx2 = int(r2["box"]["x"] + r2["box"]["width"] / 2)
                ry2 = int(r2["box"]["y"] + r2["box"]["height"] / 2)
                print("\n[7/7] [PyAutoGUI] Clicking result #2 at ({},{})".format(rx2, ry2), flush=True)
                print("   " + r2["title"][:70], flush=True)
                human_click(rx2, ry2)
                page.wait_for_timeout(6000)
                print("   Landed: {} | {}".format(page.title()[:50], page.url[:80]), flush=True)
                screenshot(page, "ddg-05-page2.png")
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
