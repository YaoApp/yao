"""
LLM + DOM + CDP Browser Automation Demo

Architecture (each layer does what it's best at):
  - Playwright: Opens pages, extracts HTML, keyboard input
  - LLM:        Reads HTML structure, returns CSS selectors for target elements
  - DOM:        Uses LLM's selectors to get precise bounding boxes
  - CDP:        Chrome DevTools Protocol mouse events (isTrusted=true, anti-detection)
  - PyAutoGUI:  OS-level mouse fallback (when CDP fails)

Click strategy (layered, most reliable first):
  1. CDP Input.dispatchMouseEvent — coordinates match bounding_box() exactly,
     generates isTrusted=true events, nearly indistinguishable from real user.
  2. PyAutoGUI OS-level mouse    — true hardware events, but coordinates may
     drift due to window chrome offset.
  3. Playwright .click()         — last resort, may be detected by anti-bot.

No hardcoded selectors — LLM figures out the page structure dynamically.
Works with any cheap text LLM (no vision needed).

Environment variables (required):
  LLM_API_KEY     — API key
  LLM_API_BASE    — OpenAI-compatible endpoint URL
  LLM_MODEL       — Model name/ID

Usage:
  export LLM_API_KEY="your-key"
  export LLM_API_BASE="https://api.openai.com/v1/"
  export LLM_MODEL="gpt-4o-mini"
  DISPLAY=:99 python3 demo-llm-vision.py <search_url> <search_query>

Example:
  DISPLAY=:99 python3 demo-llm-vision.py https://www.google.com "Yao App Engine"
  DISPLAY=:99 python3 demo-llm-vision.py https://www.baidu.com "Yao App Engine"
"""

from playwright.sync_api import sync_playwright
from playwright_stealth import Stealth
import pyautogui
import time
import random
import json
import os
import re
import urllib.request
import urllib.error
import sys
from concurrent.futures import ThreadPoolExecutor, as_completed

# ---------------------------------------------------------------------------
# Config
# ---------------------------------------------------------------------------
pyautogui.FAILSAFE = False
pyautogui.PAUSE = 0.1
SCREENSHOT_DIR = os.environ.get("SCREENSHOT_DIR", "/workspace")

LLM_API_KEY = os.environ.get("LLM_API_KEY", "").strip().strip('"\'')
LLM_API_BASE = os.environ.get("LLM_API_BASE", "").strip().strip('"\'')
LLM_MODEL = os.environ.get("LLM_MODEL", "").strip().strip('"\'')
if LLM_API_BASE and not LLM_API_BASE.endswith("/"):
    LLM_API_BASE += "/"

# Command-line arguments: <search_url> <search_query>
SEARCH_URL = sys.argv[1] if len(sys.argv) > 1 else "https://www.google.com"
SEARCH_QUERY = sys.argv[2] if len(sys.argv) > 2 else "Yao App Engine"


# ---------------------------------------------------------------------------
# LLM API
# ---------------------------------------------------------------------------
def ask_llm(prompt, timeout=60):
    """Send text prompt to LLM, return text response."""
    url = LLM_API_BASE + "chat/completions"
    payload = {
        "model": LLM_MODEL,
        "messages": [{"role": "user", "content": prompt}],
        "max_tokens": 2048,
        "temperature": 0.1,
    }
    headers = {"Content-Type": "application/json", "Authorization": "Bearer " + LLM_API_KEY}
    data = json.dumps(payload).encode("utf-8")
    req = urllib.request.Request(url, data=data, headers=headers, method="POST")

    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            result = json.loads(resp.read().decode("utf-8"))
            return result["choices"][0]["message"]["content"]
    except urllib.error.HTTPError as e:
        body = e.read().decode("utf-8")[:500] if e.fp else ""
        print("   LLM HTTP {}: {}".format(e.code, body), flush=True)
        return None
    except Exception as e:
        print("   LLM error: {}".format(e), flush=True)
        return None


def parse_llm_json(text):
    """Extract JSON from LLM response (handles markdown fences, thinking tags)."""
    if not text:
        return None
    cleaned = re.sub(r'<think>[\s\S]*?</think>', '', text).strip()
    cleaned = re.sub(r'^```\w*\n?', '', cleaned)
    cleaned = re.sub(r'\n?```$', '', cleaned)
    cleaned = cleaned.strip()
    try:
        return json.loads(cleaned)
    except json.JSONDecodeError:
        for m in re.finditer(r'(\{[\s\S]*?\}|\[[\s\S]*?\])', cleaned):
            try:
                return json.loads(m.group())
            except json.JSONDecodeError:
                continue
    return None


def ask_llm_race(prompts, timeout=60, validator=None):
    """Send multiple prompts to LLM concurrently, return first valid result.
    Each prompt is sent in a separate thread. As soon as one returns a valid
    JSON result (passing optional validator), return immediately without
    waiting for the remaining threads.

    Args:
        prompts: list of (label, prompt_text) tuples
        timeout: per-request timeout in seconds
        validator: optional fn(parsed_json) -> bool, extra check on result

    Returns:
        (label, parsed_json) of the first valid result, or (None, None)
    """
    if not prompts:
        return None, None

    # Single prompt — no need for concurrency
    if len(prompts) == 1:
        label, prompt_text = prompts[0]
        resp = ask_llm(prompt_text, timeout=timeout)
        parsed = parse_llm_json(resp)
        if parsed and (validator is None or validator(parsed)):
            return label, parsed
        return None, None

    print("   [Race] Sending {} concurrent LLM requests...".format(len(prompts)), flush=True)

    # Don't use `with` — it calls shutdown(wait=True) which blocks until ALL
    # threads finish, even after we found a winner. Instead, manage manually
    # and call shutdown(wait=False) to return immediately.
    pool = ThreadPoolExecutor(max_workers=len(prompts))
    future_map = {}
    for label, prompt_text in prompts:
        f = pool.submit(ask_llm, prompt_text, timeout)
        future_map[f] = label

    try:
        for f in as_completed(future_map):
            label = future_map[f]
            try:
                resp = f.result()
                parsed = parse_llm_json(resp)
                if parsed and (validator is None or validator(parsed)):
                    print("   [Race] Winner: '{}' → {}".format(
                        label, json.dumps(parsed, ensure_ascii=False)[:120]), flush=True)
                    # Cancel pending futures (won't stop running ones, but prevents queued)
                    for other in future_map:
                        if other is not f:
                            other.cancel()
                    return label, parsed
                else:
                    print("   [Race] '{}' returned invalid result, waiting...".format(label), flush=True)
            except Exception as e:
                print("   [Race] '{}' failed: {}".format(label, e), flush=True)
    finally:
        # shutdown(wait=False) — let daemon threads die on their own,
        # don't block the main thread waiting for slow LLM responses.
        pool.shutdown(wait=False, cancel_futures=True)

    print("   [Race] All requests failed", flush=True)
    return None, None


# ---------------------------------------------------------------------------
# DOM extraction — compact page summary for LLM
# ---------------------------------------------------------------------------
def extract_page_dom(page):
    """Extract a compact text summary of all interactive elements on the page.
    Includes inputs, buttons, and visible links — generic, no hardcoded selectors.
    LLM reads this to decide what to interact with."""
    return page.evaluate("""() => {
        const lines = [];
        lines.push('URL: ' + location.href);
        lines.push('Title: ' + document.title);
        lines.push('');

        // Helper: describe element visibility
        function vis(el) {
            const r = el.getBoundingClientRect();
            return (r.width > 5 && r.height > 5)
                ? '[VISIBLE ' + Math.round(r.width) + 'x' + Math.round(r.height) + ']'
                : '[HIDDEN]';
        }

        // Helper: build a minimal CSS selector for an element
        function sel(el) {
            const tag = el.tagName.toLowerCase();
            if (el.id) return tag + '#' + el.id;
            if (el.className) {
                const cls = el.className.toString().trim().split(/\\s+/).slice(0, 2).join('.');
                if (cls) return tag + '.' + cls;
            }
            return tag;
        }

        // 1. Inputs, textareas, buttons
        lines.push('=== Inputs & Buttons ===');
        document.querySelectorAll('input, textarea, button, [role="textbox"], [contenteditable="true"]').forEach(el => {
            const tag = el.tagName.toLowerCase();
            const parts = [sel(el)];
            if (el.type && el.type !== 'text') parts.push('type="' + el.type + '"');
            if (el.name) parts.push('name="' + el.name + '"');
            if (el.placeholder) parts.push('placeholder="' + el.placeholder.substring(0, 40) + '"');
            if (el.value) parts.push('value="' + el.value.substring(0, 30) + '"');
            const text = (el.innerText || '').trim().substring(0, 30);
            if (text && tag === 'button') parts.push('text="' + text + '"');
            if (el.getAttribute('aria-label')) parts.push('aria-label="' + el.getAttribute('aria-label').substring(0, 30) + '"');
            parts.push(vis(el));
            lines.push('  ' + parts.join(' '));
        });

        // 2. Visible links
        lines.push('');
        lines.push('=== Links ===');
        const seen = new Set();
        let count = 0;
        document.querySelectorAll('a[href]').forEach(a => {
            if (count >= 30) return;
            const href = a.href || '';
            if (!href || href.startsWith('javascript:')) return;

            const rect = a.getBoundingClientRect();
            if (rect.width === 0 || rect.height === 0) return;
            if (rect.top < 30) return;

            const text = (a.innerText || '').trim().replace(/\\s+/g, ' ').substring(0, 80);
            if (!text || text.length < 2 || seen.has(text)) return;
            seen.add(text);

            const parent = a.parentElement;
            let ctx = parent ? sel(parent) + ' > ' : '';
            const heading = a.querySelector('h1,h2,h3,h4');
            const htag = heading ? ' [has <' + heading.tagName.toLowerCase() + '>]' : '';

            lines.push('  ' + ctx + sel(a) + htag + ' ' + vis(a) + ' → "' + text + '"');
            count++;
        });

        return lines.join('\\n');
    }""")



# ---------------------------------------------------------------------------
# CDP click — Chrome DevTools Protocol mouse events (isTrusted=true)
# ---------------------------------------------------------------------------
_cdp_session = None
_cdp_page_id = None


def get_cdp_session(page):
    """Get or create a CDP session for the given page.
    Recreates session if page changed (e.g. after tab close/navigation)."""
    global _cdp_session, _cdp_page_id
    page_id = id(page)
    if _cdp_session is None or _cdp_page_id != page_id:
        try:
            if _cdp_session:
                _cdp_session.detach()
        except Exception:
            pass
        _cdp_session = page.context.new_cdp_session(page)
        _cdp_page_id = page_id
    return _cdp_session


def cdp_click(page, x, y, ctrl=False):
    """Click at (x, y) via CDP Input.dispatchMouseEvent.
    Coordinates are in viewport space (same as bounding_box()).
    Generates isTrusted=true events — nearly indistinguishable from real user.
    When ctrl=True, holds Ctrl modifier to force new tab (like Ctrl+Click)."""
    cdp = get_cdp_session(page)

    # Simulate human-like: small random offset (±2px)
    x += random.uniform(-2, 2)
    y += random.uniform(-2, 2)

    modifiers = 2 if ctrl else 0  # 2 = Ctrl modifier in CDP

    # mouseMoved — simulate cursor arriving
    cdp.send("Input.dispatchMouseEvent", {
        "type": "mouseMoved",
        "x": x, "y": y,
        "button": "none",
        "modifiers": modifiers,
        "pointerType": "mouse",
    })
    time.sleep(random.uniform(0.05, 0.15))

    # mousePressed
    cdp.send("Input.dispatchMouseEvent", {
        "type": "mousePressed",
        "x": x, "y": y,
        "button": "left",
        "clickCount": 1,
        "modifiers": modifiers,
        "pointerType": "mouse",
    })
    time.sleep(random.uniform(0.03, 0.08))

    # mouseReleased
    cdp.send("Input.dispatchMouseEvent", {
        "type": "mouseReleased",
        "x": x, "y": y,
        "button": "left",
        "clickCount": 1,
        "modifiers": modifiers,
        "pointerType": "mouse",
    })
    time.sleep(random.uniform(0.1, 0.3))


def cdp_move(page, x, y, steps=10):
    """Simulate human-like mouse movement via CDP (curved path)."""
    cdp = get_cdp_session(page)
    # Start from a random nearby position
    sx = x + random.uniform(-200, 200)
    sy = y + random.uniform(-100, 100)
    for i in range(steps + 1):
        t = i / steps
        # Ease-in-out curve
        t = t * t * (3 - 2 * t)
        mx = sx + (x - sx) * t + random.uniform(-1, 1)
        my = sy + (y - sy) * t + random.uniform(-1, 1)
        cdp.send("Input.dispatchMouseEvent", {
            "type": "mouseMoved",
            "x": mx, "y": my,
            "button": "none",
            "pointerType": "mouse",
        })
        time.sleep(random.uniform(0.01, 0.03))


def smart_click(page, x, y, label=""):
    """Click using CDP (primary) with PyAutoGUI fallback.
    Returns the method used: 'cdp', 'pyautogui', or None on failure."""
    tag = "[CDP]" if label else "[CDP]"
    try:
        cdp_move(page, x, y)
        cdp_click(page, x, y)
        print("   {} Click at ({},{}){}".format(
            tag, int(x), int(y),
            " '{}'".format(label[:40]) if label else ""), flush=True)
        return "cdp"
    except Exception as e:
        print("   {} Failed: {} → PyAutoGUI fallback".format(tag, e), flush=True)
        try:
            pyautogui.moveTo(x, y, duration=random.uniform(0.4, 0.8))
            time.sleep(random.uniform(0.05, 0.15))
            pyautogui.click()
            time.sleep(random.uniform(0.2, 0.5))
            print("   [PyAutoGUI] Click at ({},{})".format(int(x), int(y)), flush=True)
            return "pyautogui"
        except Exception as e2:
            print("   [PyAutoGUI] Also failed: {}".format(e2), flush=True)
            return None


# ---------------------------------------------------------------------------
# Interaction helpers
# ---------------------------------------------------------------------------
def box_center(box):
    return int(box["x"] + box["width"] / 2), int(box["y"] + box["height"] / 2)


def take_screenshot(page, name):
    path = os.path.join(SCREENSHOT_DIR, name)
    page.screenshot(path=path)
    print("   Screenshot: " + path, flush=True)


def locate_element(page, selector):
    """Use a CSS selector to find element, return (element, bounding_box) or (None, None)."""
    try:
        loc = page.locator(selector)
        count = loc.count()
        print("   [locate] '{}' matched {} elements".format(selector, count), flush=True)
        if count == 0:
            return None, None
        el = loc.first
        # Try to make it visible first
        try:
            el.scroll_into_view_if_needed(timeout=2000)
        except Exception:
            pass
        box = el.bounding_box(timeout=5000)
        if box:
            print("   [locate] box: x={} y={} w={} h={}".format(
                int(box["x"]), int(box["y"]), int(box["width"]), int(box["height"])), flush=True)
            if box["width"] > 5:
                return el, box
        else:
            print("   [locate] bounding_box returned None (element hidden?)", flush=True)
    except Exception as e:
        print("   [locate] error: {}".format(e), flush=True)
    return None, None


def locate_elements(page, selector, min_y=0):
    """Find all visible elements matching selector, return list of (element, box, text)."""
    results = []
    try:
        els = page.locator(selector).all()
        for el in els:
            try:
                box = el.bounding_box(timeout=500)
                if box and box["width"] > 30 and box["y"] > min_y:
                    text = el.inner_text(timeout=500)[:80]
                    results.append((el, box, text))
            except Exception:
                continue
    except Exception:
        pass
    return results


def click_new_tab(ctx, page, el, box, label):
    """Ctrl+Click an element via CDP to force open in new tab.
    Keeps the search results page intact. Waits for new tab, screenshots, closes it.
    Fallback chain: CDP Ctrl+Click → Playwright Ctrl+Click → JS window.open"""
    cx, cy = box_center(box)

    # Scroll into view if needed
    if cy < 100 or cy > 1000:
        try:
            el.scroll_into_view_if_needed(timeout=2000)
            time.sleep(0.5)
            box = el.bounding_box(timeout=1000)
            if box:
                cx, cy = box_center(box)
        except Exception:
            pass

    pages_before = len(ctx.pages)

    # --- Attempt 1: CDP Ctrl+Click (isTrusted=true, new tab) ---
    try:
        cdp_move(page, cx, cy)
        cdp_click(page, cx, cy, ctrl=True)
        print("   [CDP Ctrl+Click] '{}' at ({},{})".format(label[:40], int(cx), int(cy)), flush=True)
    except Exception as e:
        print("   [CDP Ctrl+Click] Failed: {}".format(e), flush=True)

    # Wait for new tab
    for _ in range(12):
        page.wait_for_timeout(500)
        if len(ctx.pages) > pages_before:
            break

    if len(ctx.pages) > pages_before:
        target = ctx.pages[-1]
        target.wait_for_timeout(6000)
        print("   ✓ [New Tab] {} | {}".format(target.title()[:50], target.url[:80]), flush=True)
        take_screenshot(target, "llm-detail.png")
        target.close()
        page.wait_for_timeout(500)
        # Bring focus back to search results page
        page.bring_to_front()
        return True

    # --- Attempt 2: Playwright modifier click ---
    print("   CDP Ctrl+Click no new tab → Playwright modifier click", flush=True)
    try:
        el.click(modifiers=["Control"], timeout=3000)
        page.wait_for_timeout(3000)
        if len(ctx.pages) > pages_before:
            target = ctx.pages[-1]
            target.wait_for_timeout(6000)
            print("   ✓ [New Tab] {} | {}".format(target.title()[:50], target.url[:80]), flush=True)
            take_screenshot(target, "llm-detail.png")
            target.close()
            page.wait_for_timeout(500)
            page.bring_to_front()
            return True
    except Exception:
        pass

    # --- Attempt 3: JS window.open with href ---
    print("   Modifier click failed → JS window.open fallback", flush=True)
    try:
        href = el.get_attribute("href", timeout=2000)
        if href:
            new_page = ctx.new_page()
            new_page.goto(href, timeout=15000)
            new_page.wait_for_timeout(5000)
            print("   ✓ [JS Tab] {} | {}".format(new_page.title()[:50], new_page.url[:80]), flush=True)
            take_screenshot(new_page, "llm-detail.png")
            new_page.close()
            page.wait_for_timeout(500)
            page.bring_to_front()
            return True
    except Exception as e:
        print("   [JS Tab] Failed: {}".format(e), flush=True)

    print("   ✗ All methods failed to open new tab", flush=True)
    return False


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
def main():
    print("=" * 60, flush=True)
    print("  LLM + DOM + CDP Browser Automation", flush=True)
    print("=" * 60, flush=True)
    print("  URL:      " + SEARCH_URL, flush=True)
    print("  Query:    " + SEARCH_QUERY, flush=True)
    print("  Model:    " + LLM_MODEL, flush=True)
    print("  Endpoint: " + LLM_API_BASE[:60], flush=True)
    print("  Key:      " + (LLM_API_KEY[:8] + "..." if LLM_API_KEY else "NOT SET"), flush=True)
    print("", flush=True)
    print("  Flow: LLM reads HTML → returns CSS selectors →", flush=True)
    print("        DOM locates elements → CDP clicks (isTrusted)", flush=True)

    if not all([LLM_API_KEY, LLM_API_BASE, LLM_MODEL]):
        print("\n⚠ Missing LLM config! Set: LLM_API_KEY, LLM_API_BASE, LLM_MODEL", flush=True)
        return

    with Stealth().use_sync(sync_playwright()) as p:
        browser = p.chromium.launch(
            channel="chrome", headless=False,
            args=["--no-sandbox", "--disable-blink-features=AutomationControlled",
                  "--disable-dev-shm-usage", "--window-size=1920,1080", "--window-position=0,0"])
        ctx = browser.new_context(
            viewport={"width": 1920, "height": 1080},
            locale="zh-CN", timezone_id="Asia/Shanghai")
        page = ctx.new_page()

        stealth_path = "/usr/local/share/yao/stealth-init.js"
        if os.path.exists(stealth_path):
            page.add_init_script(open(stealth_path).read())

        # ============================================================
        # Phase 1: Open search engine
        # ============================================================
        print("\n[Phase 1] Opening {}...".format(SEARCH_URL), flush=True)
        page.goto(SEARCH_URL, timeout=30000)
        page.wait_for_timeout(3000)
        print("   Title: " + page.title(), flush=True)
        take_screenshot(page, "llm-01-homepage.png")

        # ============================================================
        # Phase 2: LLM analyzes homepage HTML → gives selectors
        # ============================================================
        print("\n[Phase 2] [LLM] Analyzing homepage structure...", flush=True)
        elements = extract_page_dom(page)
        print(elements[:500], flush=True)
        if len(elements) > 500:
            print("   ... ({} chars total)".format(len(elements)), flush=True)

        hp_prompt_tpl = """Below is the DOM structure of a search engine homepage.
Each element is marked [VISIBLE WxH] or [HIDDEN].

I want to:
1. Type a search query into the search input box
2. Click the search submit button

IMPORTANT: Only pick elements marked [VISIBLE]. Ignore [HIDDEN] elements.
Give me CSS selectors for both elements.

{}

Reply ONLY with JSON (no other text):
{{"input_selector": "<CSS selector for a VISIBLE input>", "button_selector": "<CSS selector for a VISIBLE button>"}}"""

        # Split homepage DOM into chunks for concurrent LLM calls
        hp_lines = elements.split('\n')
        hp_header = []
        hp_body = []
        for line in hp_lines:
            if line.startswith("URL:") or line.startswith("Title:") or line == "":
                hp_header.append(line)
            else:
                hp_body.append(line)
        hp_hdr = '\n'.join(hp_header[:3])

        hp_chunks = []
        cur_chunk = []
        cur_len = 0
        for line in hp_body:
            cur_chunk.append(line)
            cur_len += len(line) + 1
            if cur_len >= 2000:
                hp_chunks.append('\n'.join(cur_chunk))
                cur_chunk = []
                cur_len = 0
        if cur_chunk:
            hp_chunks.append('\n'.join(cur_chunk))

        hp_prompts = []
        if len(hp_chunks) > 1:
            for i, chunk in enumerate(hp_chunks):
                hp_prompts.append(("chunk-{}".format(i + 1), hp_prompt_tpl.format(hp_hdr + '\n' + chunk)))
        hp_prompts.append(("full", hp_prompt_tpl.format(elements)))

        print("   [Phase 2] {} concurrent LLM requests".format(len(hp_prompts)), flush=True)

        def _valid_homepage(parsed):
            return (isinstance(parsed, dict)
                    and bool(parsed.get("input_selector", "").strip())
                    and bool(parsed.get("button_selector", "").strip()))

        label, selectors = ask_llm_race(hp_prompts, timeout=180, validator=_valid_homepage)

        if not selectors or not isinstance(selectors, dict):
            print("   ✗ LLM failed to return selectors", flush=True)
            browser.close()
            return

        input_sel = selectors.get("input_selector", "")
        button_sel = selectors.get("button_selector", "")
        print("   → Input:  {}".format(input_sel), flush=True)
        print("   → Button: {}".format(button_sel), flush=True)

        # ============================================================
        # Phase 3: DOM locates elements → CDP clicks + Playwright types
        # ============================================================
        print("\n[Phase 3] [DOM] Locating input: '{}'".format(input_sel), flush=True)
        input_el, input_box = locate_element(page, input_sel)
        if not input_el:
            print("   ✗ Selector '{}' didn't match! Aborting.".format(input_sel), flush=True)
            browser.close()
            return

        ix, iy = box_center(input_box)
        print("   ✓ Input at ({},{}) size={}x{}".format(
            ix, iy, int(input_box["width"]), int(input_box["height"])), flush=True)

        smart_click(page, ix, iy, "search input")
        time.sleep(0.3)

        print("   [Playwright] Type '{}'".format(SEARCH_QUERY), flush=True)
        input_el.type(SEARCH_QUERY, delay=80)
        time.sleep(0.5)
        take_screenshot(page, "llm-02-typed.png")

        # ============================================================
        # Phase 4: DOM locates button → CDP clicks
        # ============================================================
        print("\n[Phase 4] [DOM] Locating button: '{}'".format(button_sel), flush=True)
        btn_el, btn_box = locate_element(page, button_sel)
        if btn_el and btn_box:
            bx, by = box_center(btn_box)
            print("   ✓ Button at ({},{})".format(bx, by), flush=True)
            smart_click(page, bx, by, "search button")
        else:
            print("   Button not found → Enter key via CDP", flush=True)
            smart_click(page, ix, iy, "input focus")
            time.sleep(0.2)
            try:
                cdp = get_cdp_session(page)
                cdp.send("Input.dispatchKeyEvent", {
                    "type": "keyDown", "key": "Enter", "code": "Enter",
                    "windowsVirtualKeyCode": 13, "nativeVirtualKeyCode": 13,
                })
                cdp.send("Input.dispatchKeyEvent", {
                    "type": "keyUp", "key": "Enter", "code": "Enter",
                    "windowsVirtualKeyCode": 13, "nativeVirtualKeyCode": 13,
                })
            except Exception:
                pyautogui.press("enter")

        page.wait_for_timeout(5000)
        results_url = page.url
        print("   URL: " + results_url[:100], flush=True)
        print("   Title: " + page.title()[:60], flush=True)
        take_screenshot(page, "llm-03-results.png")

        # If URL unchanged, fallback: Enter key → Playwright click → form submit
        homepage = SEARCH_URL.rstrip("/")
        if results_url.rstrip("/") == homepage:
            print("   URL unchanged — trying CDP Enter key fallback...", flush=True)
            smart_click(page, ix, iy, "input refocus")
            time.sleep(0.2)
            try:
                cdp = get_cdp_session(page)
                cdp.send("Input.dispatchKeyEvent", {
                    "type": "keyDown", "key": "Enter", "code": "Enter",
                    "windowsVirtualKeyCode": 13, "nativeVirtualKeyCode": 13,
                })
                cdp.send("Input.dispatchKeyEvent", {
                    "type": "keyUp", "key": "Enter", "code": "Enter",
                    "windowsVirtualKeyCode": 13, "nativeVirtualKeyCode": 13,
                })
            except Exception:
                pyautogui.press("enter")
            page.wait_for_timeout(5000)
            results_url = page.url

        if results_url.rstrip("/") == homepage:
            print("   Still unchanged — trying Playwright click fallback...", flush=True)
            try:
                if btn_el:
                    btn_el.click(timeout=3000)
                else:
                    input_el.press("Enter")
                page.wait_for_timeout(5000)
                results_url = page.url
            except Exception:
                pass

        if results_url.rstrip("/") == homepage:
            print("   Still unchanged — trying form submit fallback...", flush=True)
            try:
                page.evaluate("document.querySelector('form')?.submit()")
                page.wait_for_timeout(5000)
                results_url = page.url
            except Exception:
                pass

        print("   Final URL: " + results_url[:100], flush=True)
        take_screenshot(page, "llm-03-results.png")

        if results_url.rstrip("/") == homepage:
            print("   ✗ All submit methods failed", flush=True)
            browser.close()
            return

        # ============================================================
        # Phase 5: LLM analyzes results page → gives link selector
        #          Split DOM into chunks, race concurrent LLM calls
        # ============================================================
        print("\n[Phase 5] [LLM] Analyzing search results page...", flush=True)

        results_dom = extract_page_dom(page)
        print(results_dom[:500], flush=True)
        if len(results_dom) > 500:
            print("   ... ({} chars total)".format(len(results_dom)), flush=True)

        # Split DOM into chunks for concurrent LLM calls
        dom_lines = results_dom.split('\n')
        link_prompt_tpl = """Below is part of the DOM from a search results page.
Each line shows: parent > link_selector [has <h3> if any] → "link text"

I need a CSS selector that matches the organic search result title links.
NOT ads, NOT navigation, NOT pagination — only the main result links.

{}

Reply ONLY JSON: {{"link_selector": "<CSS selector>"}}"""

        # Build chunks: split at ~2000 char boundaries, always include URL/Title header
        header_lines = []
        body_lines = []
        for line in dom_lines:
            if line.startswith("URL:") or line.startswith("Title:") or line == "":
                header_lines.append(line)
            else:
                body_lines.append(line)
        header = '\n'.join(header_lines[:3])  # URL + Title + blank

        chunks = []
        current_chunk = []
        current_len = 0
        chunk_limit = 2000
        for line in body_lines:
            current_chunk.append(line)
            current_len += len(line) + 1
            if current_len >= chunk_limit:
                chunks.append('\n'.join(current_chunk))
                current_chunk = []
                current_len = 0
        if current_chunk:
            chunks.append('\n'.join(current_chunk))

        # Also send the full DOM as one prompt (in case chunks miss context)
        prompts = []
        if len(chunks) > 1:
            for i, chunk in enumerate(chunks):
                chunk_dom = header + '\n' + chunk
                prompts.append(("chunk-{}".format(i + 1), link_prompt_tpl.format(chunk_dom)))
        # Always include the full DOM as the last prompt
        prompts.append(("full", link_prompt_tpl.format(results_dom)))

        print("   [Phase 5] {} concurrent LLM requests ({} chunks + full)".format(
            len(prompts), len(chunks) if len(chunks) > 1 else 0), flush=True)

        def _valid_link_selector(parsed):
            return isinstance(parsed, dict) and bool(parsed.get("link_selector", "").strip())

        label, link_info = ask_llm_race(prompts, timeout=180, validator=_valid_link_selector)

        link_sel = ""
        link_results = []
        if link_info and isinstance(link_info, dict):
            link_sel = link_info.get("link_selector", "")
            print("   → Selector: '{}' (from {})".format(link_sel, label), flush=True)
            if link_sel:
                link_results = locate_elements(page, link_sel, min_y=100)

        print("   Found {} clickable links".format(len(link_results)), flush=True)
        for i, (el, box, text) in enumerate(link_results[:5]):
            cx, cy = box_center(box)
            print("   [{}] ({},{}) '{}'".format(i, cx, cy, text[:50]), flush=True)

        if len(link_results) < 1:
            print("   ✗ No links found", flush=True)
            take_screenshot(page, "llm-04-no-links.png")
            browser.close()
            return

        # ============================================================
        # Phase 7+: Ctrl+Click results (open in new tab, keep list intact)
        # ============================================================
        max_clicks = min(len(link_results), 3)
        for idx in range(max_clicks):
            el_r, box_r, text_r = link_results[idx]
            print("\n[Phase {}] Ctrl+Click result #{}: '{}'".format(
                7 + idx, idx + 1, text_r[:50]), flush=True)
            click_new_tab(ctx, page, el_r, box_r, text_r)
            page.wait_for_timeout(1000)

        # Done
        print("\n" + "=" * 60, flush=True)
        print("  ✓ Demo complete!", flush=True)
        print("  VNC: http://localhost:6080", flush=True)
        print("=" * 60, flush=True)
        page.wait_for_timeout(30000)
        browser.close()

    print("Done.", flush=True)


if __name__ == "__main__":
    if len(sys.argv) > 1 and sys.argv[1] in ("-h", "--help"):
        print("Usage: python3 demo-llm-vision.py <search_url> <search_query>")
        print("")
        print("Arguments:")
        print("  search_url    Search engine URL (default: https://www.google.com)")
        print("  search_query  What to search for (default: Yao App Engine)")
        print("")
        print("Environment variables (required):")
        print("  LLM_API_KEY   API key for the LLM service")
        print("  LLM_API_BASE  OpenAI-compatible endpoint URL")
        print("  LLM_MODEL     Model name/ID")
        print("")
        print("Examples:")
        print('  python3 demo-llm-vision.py https://www.google.com "Yao App Engine"')
        print('  python3 demo-llm-vision.py https://www.bing.com "Yao App Engine"')
        print('  python3 demo-llm-vision.py https://duckduckgo.com "Yao App Engine"')
        print('  python3 demo-llm-vision.py https://www.baidu.com "Yao App Engine"')
        print('  python3 demo-llm-vision.py https://www.sogou.com "Yao App Engine"')
        sys.exit(0)
    main()
