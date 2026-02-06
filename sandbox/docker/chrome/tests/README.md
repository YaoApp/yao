# Chrome Browser Automation Demo Tests

Example scripts demonstrating browser automation inside the `sandbox-claude-chrome` Docker image.

## Scripts

| Script | Description |
|--------|-------------|
| `demo-llm-vision.py` | **LLM-driven universal automation** — works with any search engine, no hardcoded selectors. LLM reads page DOM and decides what to click. |
| `demo-baidu.py` | Baidu search demo — hardcoded selectors |
| `demo-duckduckgo.py` | DuckDuckGo search demo — hardcoded selectors |

## demo-llm-vision.py

The main demo. Uses a layered architecture where each component does what it's best at:

```
LLM reads HTML → returns CSS selectors → DOM locates elements → CDP clicks
```

- **Playwright**: Opens pages, extracts DOM, keyboard input
- **LLM**: Reads page structure, returns CSS selectors for target elements (any cheap text model works)
- **DOM**: Uses LLM's selectors to get precise bounding boxes
- **CDP**: Chrome DevTools Protocol mouse events (`isTrusted=true`) for clicking

No hardcoded selectors — LLM figures out the page structure dynamically. Works with Google, Bing, Baidu, DuckDuckGo, Sogou, and any other search engine.

### Key Features

- **Concurrent LLM Race**: DOM is split into chunks, sent to LLM concurrently. First valid response wins — faster than sequential.
- **CDP Click**: Uses `Input.dispatchMouseEvent` via Chrome DevTools Protocol. Coordinates match `bounding_box()` exactly, no offset issues.
- **Ctrl+Click New Tab**: Search results open in new tabs, keeping the results list intact for clicking more links.
- **Fallback Chain**: CDP click → PyAutoGUI OS-level click → Playwright `.click()` → form submit. Always gets through.

### Environment Variables

| Variable | Description |
|----------|-------------|
| `LLM_API_KEY` | API key for the LLM service |
| `LLM_API_BASE` | OpenAI-compatible endpoint URL |
| `LLM_MODEL` | Model name (e.g. `gpt-4o-mini`) |

### Quick Start

```bash
# Start the container
docker run -d --name chrome-test \
  --platform linux/amd64 \
  -p 6080:6080 \
  yaoapp/sandbox-claude-chrome:latest

# Wait for VNC to start
sleep 5

# Copy the script
docker cp tests/demo-llm-vision.py chrome-test:/workspace/

# Run with any search engine
docker exec \
  -e LLM_API_KEY="your-key" \
  -e LLM_API_BASE="https://api.openai.com/v1/" \
  -e LLM_MODEL="gpt-4o-mini" \
  chrome-test bash -c \
  'DISPLAY=:99 python3 /workspace/demo-llm-vision.py "https://www.bing.com" "Yao App Engine"'
```

Open `http://localhost:6080` in your browser to watch the automation in real-time via VNC.

### Tested Search Engines

| Engine | Status | Notes |
|--------|--------|-------|
| Bing | Passed | gpt-4o-mini, ~100s |
| Sogou | Passed | gpt-4o-mini, ~83s |
| DuckDuckGo | Passed | gpt-4o-mini, ~87s |
| Baidu | Passed | glm-4-7, ~160s |
| Google | Passed | May show CAPTCHA on shared IPs |

### Flow

```
Phase 1  Open search engine homepage
Phase 2  [LLM Race] Analyze homepage DOM → get input/button selectors
Phase 3  [CDP] Click search input, type query
Phase 4  [CDP] Click search button (fallback: Enter key → form submit)
Phase 5  [LLM Race] Analyze results page DOM → get link selector
Phase 7+ [CDP Ctrl+Click] Open results in new tabs, screenshot, close
```

## Screenshots

Each demo saves screenshots to `/workspace/` at key steps:

| File | Content |
|------|---------|
| `llm-01-homepage.png` | Search engine homepage |
| `llm-02-typed.png` | Query typed in search box |
| `llm-03-results.png` | Search results page |
| `llm-detail.png` | Result detail page (new tab) |

## Notes

- **Google** may show reCAPTCHA due to IP-based rate limiting. Use a clean IP or proxy.
- **Model choice**: `gpt-4o-mini` recommended for speed. Slower models (e.g. `glm-4-7`) may timeout on large DOMs.
- **Concurrent Race** splits DOM into ~2000-char chunks and sends all chunks + full DOM to LLM simultaneously. First valid JSON response wins.
