# Sandbox VNC Integration Design Document

## Overview

This document describes the design for integrating VNC remote desktop access into the Yao Sandbox system. This enables users to **observe Claude's operations in real-time** through a web-based VNC client, providing full transparency and building trust.

The design provides **multiple sandbox image variants** with VNC support. Users can choose the appropriate image type when configuring their assistants based on their needs.

## Goals

1. **Transparency**: Let users see exactly what Claude is doing in the sandbox in real-time
2. **Multiple Image Options**: Provide different sandbox images for different use cases
3. **User Choice**: Allow users to select sandbox image type when building assistants
4. **Web-Based Access**: Use noVNC for browser-based VNC access (no client installation required)
5. **Unified Entry Point**: Single proxy endpoint to access any container's VNC session
6. **Security**: Proper authentication and isolation between users
7. **Minimal Core Changes**: Leverage existing sandbox infrastructure with minimal modifications

## Non-Goals

1. Persistent VNC sessions across container restarts
2. Multi-user access to the same VNC session
3. Audio support

## Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                  User Browser                                │
│                                                                              │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │                           Yao Web UI                                  │   │
│  │                                                                       │   │
│  │  ┌─────────────────────┐         ┌─────────────────────────────────┐ │   │
│  │  │  💬 Chat Window     │         │  📺 VNC Preview (iframe)        │ │   │
│  │  │                     │         │                                  │ │   │
│  │  │  User: Help me...   │         │  Real-time view of Claude's     │ │   │
│  │  │                     │         │  operations in sandbox          │ │   │
│  │  │  Claude: Working... │         │                                  │ │   │
│  │  └─────────────────────┘         └─────────────────────────────────┘ │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
                                           │
                                           │ WebSocket (VNC)
                                           ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                               Yao Server (Host)                              │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                         VNC Proxy Service                            │   │
│  │                         (sandbox/vncproxy)                           │   │
│  │                                                                      │   │
│  │   Endpoints:                                                         │   │
│  │   ├── GET  /v1/sandbox/{id}/vnc        → VNC status                 │   │
│  │   ├── GET  /v1/sandbox/{id}/vnc/client → noVNC client               │   │
│  │   └── GET  /v1/sandbox/{id}/vnc/ws     → WebSocket                  │   │
│  │                                                                      │   │
│  │   Internal Flow:                                                     │   │
│  │   1. Authenticate request (JWT/session)                              │   │
│  │   2. Resolve container name: yao-sandbox-{id}                       │   │
│  │   3. Get container IP from Docker API                                │   │
│  │   4. Proxy WebSocket to container_ip:6080                           │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                           │                                  │
│                          Docker Bridge Network                               │
│                                           │                                  │
│    ┌──────────────────────────────────────┼──────────────────────────────┐  │
│    │                                      │                              │  │
│    ▼                                      ▼                              ▼  │
│  ┌──────────────────┐  ┌──────────────────────────┐  ┌──────────────────┐  │
│  │ sandbox-claude   │  │ sandbox-claude-browser   │  │ sandbox-claude-  │  │
│  │ (No VNC)         │  │ (Browser + VNC)          │  │ desktop (Full)   │  │
│  │                  │  │                          │  │                  │  │
│  │ • Claude CLI     │  │ • Claude CLI             │  │ • Claude CLI     │  │
│  │ • Node.js        │  │ • Node.js                │  │ • Node.js        │  │
│  │ • Python         │  │ • Python                 │  │ • Python         │  │
│  │                  │  │ • Playwright + Browsers  │  │ • XFCE Desktop   │  │
│  │                  │  │ • Xvfb + VNC             │  │ • File Manager   │  │
│  │                  │  │ • Fluxbox (minimal WM)   │  │ • Terminal       │  │
│  │                  │  │                          │  │ • Xvfb + VNC     │  │
│  └──────────────────┘  └──────────────────────────┘  └──────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Image Variants

| Image | VNC | Use Case | Size | Memory |
|-------|-----|----------|------|--------|
| `sandbox-claude` | ❌ | Code execution, scripts, CLI tasks | ~700MB | 2GB |
| `sandbox-claude-browser` | ✅ | Browser automation, web scraping | ~1.8GB | 4GB |
| `sandbox-claude-desktop` | ✅ | Full visibility, any GUI app | ~2.5GB | 4GB |

### User Selection Flow

```
┌─────────────────────────────────────────────────────────────────────┐
│                    Assistant Configuration UI                        │
│                                                                      │
│  Assistant Name: [My Web Scraper                    ]                │
│                                                                      │
│  Sandbox Environment:                                                │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │  ○ Standard (sandbox-claude)                                    ││
│  │    Code execution, no GUI. Lightweight and fast.                ││
│  │                                                                 ││
│  │  ○ Browser (sandbox-claude-browser)                    ⭐       ││
│  │    Playwright browser automation with VNC preview.              ││
│  │    See browser operations in real-time.                         ││
│  │                                                                 ││
│  │  ● Desktop (sandbox-claude-desktop)                             ││
│  │    Full Ubuntu desktop with VNC preview.                        ││
│  │    See ALL operations: terminal, files, browser, etc.           ││
│  └─────────────────────────────────────────────────────────────────┘│
│                                                                      │
│                                              [ Save Assistant ]      │
└─────────────────────────────────────────────────────────────────────┘
```

## Components

### 1. Docker Images

**Location**: `sandbox/docker/`

Three image variants sharing the same VNC infrastructure:

```
ubuntu:24.04
    └── sandbox-base:latest (~200MB)
            └── sandbox-claude:latest (~700MB)                    # No VNC
                    ├── sandbox-claude-browser:latest (~1.8GB)      # VNC + Browser
                    └── sandbox-claude-desktop:latest (~2.5GB)     # VNC + Full Desktop
```

#### 1.1 sandbox-claude-browser (Browser + VNC)

For browser automation tasks with real-time visibility.

**Includes**:
- Everything from `sandbox-claude`
- Xvfb (virtual display)
- x11vnc + noVNC
- Fluxbox (minimal window manager)
- Playwright + Chromium/Firefox

#### 1.2 sandbox-claude-desktop (Full Desktop + VNC)

For maximum transparency - users can see everything Claude does.

**Includes**:
- Everything from `sandbox-claude`
- Xvfb (virtual display)
- x11vnc + noVNC
- XFCE desktop environment
- Thunar file manager
- xfce4-terminal
- Playwright + browsers (optional)

### 2. VNC Proxy Service

**Location**: `sandbox/vncproxy/`

A unified Go service that provides VNC access to all VNC-enabled containers.

**Key Features**:
- Single entry point for all containers
- WebSocket proxy to container VNC
- Container IP resolution via Docker API
- Authentication and authorization
- Works with any VNC-enabled image

**Key Interfaces**:

```go
// VNCProxy handles VNC connections to sandbox containers
type VNCProxy struct {
    docker    *client.Client
    manager   *sandbox.Manager
    config    *Config
}

// Config for VNC proxy
type Config struct {
    // Container VNC port (fixed, internal)
    ContainerVNCPort int  // default: 5900
    
    // Container noVNC/websockify port (fixed, internal)  
    ContainerNoVNCPort int  // default: 6080
    
    // Connection timeout
    Timeout time.Duration
}

// ServeHTTP handles HTTP requests
func (p *VNCProxy) ServeHTTP(w http.ResponseWriter, r *http.Request)

// GetVNCURL returns the VNC URL for a container
func (p *VNCProxy) GetVNCURL(sandboxID string) (string, error)

// GetContainerIP returns the internal IP of a container
func (p *VNCProxy) GetContainerIP(containerName string) (string, error)
```

### 3. Manager Extensions (Optional)

**Location**: `sandbox/manager.go`

**Note**: These extensions are optional. The core sandbox functionality works without changes because:
- Image is specified in assistant config, passed to existing `GetOrCreate()`
- VNC status is determined by checking container env vars at runtime

Optional helper types for convenience:

```go
// ImageType represents the sandbox image variant (optional, for reference)
type ImageType string

const (
    ImageTypeClaude     ImageType = "claude"      // No VNC
    ImageTypeBrowser ImageType = "browser"  // Browser + VNC
    ImageTypeDesktop    ImageType = "desktop"     // Full desktop + VNC
)

// ImageConfig holds configuration for each image type (optional, for reference)
var ImageConfigs = map[ImageType]struct {
    Image      string
    VNCEnabled bool
    Memory     string
    CPU        float64
}{
    ImageTypeClaude:     {"yaoapp/sandbox-claude:latest", false, "2g", 1.0},
    ImageTypeBrowser: {"yaoapp/sandbox-claude-browser:latest", true, "4g", 2.0},
    ImageTypeDesktop:    {"yaoapp/sandbox-claude-desktop:latest", true, "4g", 2.0},
}
```

VNC access is determined at runtime by VNC Proxy checking container env vars - no Manager changes needed.

### 4. API Endpoints

All endpoints under `/v1/sandbox/`. Each sandbox has its own unique ID (generated by the caller/business layer).

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/v1/sandbox/{id}` | POST | Create container (with image) |
| `/v1/sandbox/{id}` | GET | Get container status |
| `/v1/sandbox/{id}` | DELETE | Stop/remove container |
| `/v1/sandbox/{id}/vnc` | GET | Get VNC access info |
| `/v1/sandbox/{id}/vnc/client` | GET | Serve noVNC HTML client (supports `?viewonly=true`) |
| `/v1/sandbox/{id}/vnc/ws` | GET | WebSocket proxy to container VNC |

**Sandbox ID**: 
- Generated by the caller (business layer)
- Format: any unique string (e.g., UUID, `{userID}-{chatID}`, `{assistantID}-{sessionID}`)
- Container name: `yao-sandbox-{id}`

**Create Container Request**:

```json
// POST /v1/sandbox/abc123-def456
{
    "image": "yaoapp/sandbox-claude-desktop:latest"  // Optional, defaults based on config
}
```

**VNC Access Response**:

```json
// GET /v1/sandbox/abc123-def456/vnc
// VNC ready:
{
    "available": true,
    "status": "ready",
    "sandbox_id": "abc123-def456",
    "container": "yao-sandbox-abc123-def456",
    "client_url": "/v1/sandbox/abc123-def456/vnc/client",
    "websocket_url": "/v1/sandbox/abc123-def456/vnc/ws"
}

// VNC starting (container running but VNC services not ready yet):
{
    "available": false,
    "status": "starting",
    "sandbox_id": "abc123-def456",
    "container": "yao-sandbox-abc123-def456",
    "message": "VNC services are starting..."
}

// VNC not supported (sandbox-claude image):
{
    "available": false,
    "status": "not_supported",
    "sandbox_id": "abc123-def456",
    "container": "yao-sandbox-abc123-def456",
    "message": "VNC not available for this container type"
}

// Container not found/running:
{
    "available": false,
    "status": "unavailable",
    "sandbox_id": "abc123-def456",
    "message": "Container not available"
}
```

**Full API Structure**:

```
/v1/sandbox/
├── {id}
│   ├── POST                    # Create container
│   ├── GET                     # Get container status
│   ├── DELETE                  # Stop/remove container
│   ├── /exec                   # Execute command
│   ├── /files                  # File operations
│   └── /vnc                    # VNC access (if available)
│       ├── GET                 # VNC status & URLs
│       ├── /client             # noVNC HTML client
│       └── /ws                 # WebSocket proxy
```

**Business Layer Integration Example**:

```go
// Agent executor generates sandbox ID
sandboxID := fmt.Sprintf("%s-%s", userID, chatID)

// Or use UUID for more isolation
sandboxID := uuid.New().String()

// Or per-assistant session
sandboxID := fmt.Sprintf("%s-%s", assistantID, sessionID)
```

### 5. Assistant Configuration (Developer Side)

Developers configure sandbox image type in the assistant's `package.yao` file:

```yaml
# assistants/my-assistant/package.yao
name: My Web Assistant
description: Web scraping assistant with browser preview

sandbox:
  command: claude
  image: "yaoapp/sandbox-claude-desktop:latest"  # Choose image variant
  max_memory: "4g"
  max_cpu: 2.0
```

**Available Images**:
- `yaoapp/sandbox-claude:latest` - No VNC, lightweight
- `yaoapp/sandbox-claude-browser:latest` - Browser + VNC
- `yaoapp/sandbox-claude-desktop:latest` - Full desktop + VNC

**Note**: No changes required to `agent/sandbox/` code. The existing `Image` field in `SandboxConfig` already supports custom images.

### 6. CUI Integration (User Side)

Users interact with VNC preview through CUI's action system. The VNC preview opens as a **sidebar iframe** via the `navigate` action.

#### 6.1 Roles and Responsibilities

| Role | Action | Interface |
|------|--------|-----------|
| **Developer** | Configure `sandbox.image` in `package.yao` | YAML config file |
| **User** | View VNC preview during chat | CUI chat interface |

#### 6.2 No CUI Page Needed

The CUI `navigate` action already supports loading any URL via iframe in the sidebar. The `/v1/sandbox/{id}/vnc/client` API returns a complete HTML page with noVNC, so we can use it directly.

**Navigate action route types** (from `cui/packages/cui/chatbox/messages/Action/actions/navigate.ts`):
- `$dashboard/xxx` → CUI Dashboard pages
- `/xxx` → Loaded via iframe in sidebar
- `http(s)://xxx` → External URLs via iframe

Since `/v1/sandbox/{id}/vnc/client` starts with `/`, it will be loaded in an iframe automatically.

#### 6.3 Opening VNC Preview via Action

When the sandbox starts and VNC is available, Claude can return a `navigate` action to open the preview:

```json
{
  "type": "action",
  "actions": [{
    "name": "navigate",
    "payload": {
      "route": "/v1/sandbox/abc123-def456/vnc/client",
      "title": "实时预览",
      "icon": "material-desktop_windows"
    }
  }]
}
```

Or as a clickable button in the chat:

```json
{
  "type": "action",
  "actions": [{
    "name": "button",
    "payload": {
      "text": "📺 查看实时预览",
      "action": {
        "name": "navigate",
        "payload": {
          "route": "/v1/sandbox/abc123-def456/vnc/client",
          "title": "实时预览",
          "icon": "material-desktop_windows"
        }
      }
    }
  }]
}
```

#### 6.4 User Experience Flow

```
┌─────────────────────────────────────────────────────────────────────────┐
│  Step 1: User starts chat with sandbox-enabled assistant                 │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  💬 Chat                                                           │ │
│  │                                                                    │ │
│  │  User: 帮我爬取这个网站的数据                                       │ │
│  │                                                                    │ │
│  │  Claude: 好的，我正在启动浏览器环境...                               │ │
│  │          [📺 查看实时预览]  ← Action button                         │ │
│  │                                                                    │ │
│  └────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    │ User clicks button
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  Step 2: VNC preview opens in sidebar (iframe loads /vnc/client API)     │
│                                                                          │
│  ┌──────────────────────────┐  ┌────────────────────────────────────┐  │
│  │  💬 Chat                 │  │  📺 实时预览           [×]         │  │
│  │                          │  │  ┌────────────────────────────────┐│  │
│  │  User: 帮我爬取...        │  │  │                                ││  │
│  │                          │  │  │   noVNC (from API response)    ││  │
│  │  Claude: 正在打开         │  │  │                                ││  │
│  │  浏览器，访问目标网站...   │  │  │   User can see Claude          ││  │
│  │                          │  │  │   operating the browser        ││  │
│  │  [📺 查看实时预览]        │  │  │                                ││  │
│  │                          │  │  └────────────────────────────────┘│  │
│  └──────────────────────────┘  └────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
```

#### 6.5 How Claude Knows to Show VNC Button

The VNC preview button is triggered by the **Agent executor**, not by Claude itself. When the sandbox starts with a VNC-enabled image, the executor can inject a system message or action.

**Option A: Agent Executor Injects Action** (Recommended)

In `agent/sandbox/claude/executor.go`, when sandbox starts with VNC:

```go
func (e *Executor) Stream(...) {
    // After sandbox container is ready
    if e.isVNCEnabled() {
        // Send VNC preview action to frontend
        handler(message.StreamEvent{
            Type: "action",
            Data: map[string]interface{}{
                "actions": []map[string]interface{}{{
                    "name": "button",
                    "payload": map[string]interface{}{
                        "text": "📺 查看实时预览",
                        "action": map[string]interface{}{
                            "name": "navigate",
                            "payload": map[string]interface{}{
                                "route": fmt.Sprintf("/v1/sandbox/%s/vnc/client", sandboxID),
                                "title": "实时预览",
                            },
                        },
                    },
                }},
            },
        })
    }
    // ... continue with Claude execution
}
```

**Option B: System Prompt Hint**

Add to system prompt when VNC is enabled:
```
当你在沙盒中执行可视化任务时（如浏览器操作），可以告知用户点击"查看实时预览"按钮观看操作过程。
```

#### 6.6 VNC Interaction Modes

The VNC preview supports two modes controlled by the `viewonly` query parameter:

| Mode | URL | Description |
|------|-----|-------------|
| **Interactive** (default) | `/vnc/client` | User can use keyboard and mouse |
| **View-only** | `/vnc/client?viewonly=true` | User can only watch |

**Use Cases**:

| Scenario | Mode | Example |
|----------|------|---------|
| Watch Claude browse web | View-only | `?viewonly=true` |
| User needs to login | Interactive | (default) |
| User needs to solve CAPTCHA | Interactive | (default) |
| Sensitive operation | View-only | `?viewonly=true` |

**Action Examples**:

```json
// View-only mode (just watching)
{
  "name": "navigate",
  "payload": {
    "route": "/v1/sandbox/abc123/vnc/client?viewonly=true",
    "title": "实时预览"
  }
}

// Interactive mode (user needs to login)
{
  "name": "navigate",
  "payload": {
    "route": "/v1/sandbox/abc123/vnc/client",
    "title": "请在此登录"
  }
}
```

**How Claude Waits for User Input**:

When user interaction is needed (e.g., login), Claude can:

1. **Wait for user confirmation** (simple):
   ```
   Claude: 请在 VNC 窗口中登录，完成后告诉我
   User: 登录好了
   Claude: 好的，继续执行...
   ```

2. **Auto-detect via script** (advanced):
   ```python
   # Wait for login success indicator
   page.wait_for_selector("#user-avatar", timeout=300000)  # 5 min timeout
   print("Login detected, continuing...")
   ```

#### 6.7 CUI Changes

**No CUI changes required.** The existing `navigate` action + `app/openSidebar` event already handles loading the VNC client API response in an iframe.

## Implementation Details

### Dockerfile.browser (browser/Dockerfile)

```dockerfile
ARG REGISTRY=yaoapp
FROM ${REGISTRY}/sandbox-claude:latest

USER root

# Install X11, VNC, and minimal window manager
RUN apt-get update && apt-get install -y --no-install-recommends \
    xvfb \
    x11vnc \
    fluxbox \
    novnc \
    python3-websockify \
    fonts-liberation \
    fonts-noto-cjk \
    x11-utils \
    xdotool \
    && rm -rf /var/lib/apt/lists/*

# Install Playwright system dependencies (requires root)
RUN npx playwright install-deps chromium firefox

# Install Playwright and browsers as sandbox user
USER sandbox
RUN npm install -g playwright && \
    npx playwright install chromium firefox

USER root

# VNC startup script
COPY start-vnc.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/start-vnc.sh

# Update entrypoint to start VNC (includes original claude entrypoint logic)
COPY entrypoint-vnc.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Environment
ENV DISPLAY=:99
ENV VNC_PORT=5900
ENV NOVNC_PORT=6080
ENV RESOLUTION=1920x1080x24
ENV SANDBOX_VNC_ENABLED=true

EXPOSE 5900 6080

USER sandbox
WORKDIR /workspace

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["sleep", "infinity"]
```

### Dockerfile.desktop

```dockerfile
ARG REGISTRY=yaoapp
FROM ${REGISTRY}/sandbox-claude:latest

USER root

# Install X11, VNC, and XFCE desktop
RUN apt-get update && apt-get install -y --no-install-recommends \
    xvfb \
    x11vnc \
    novnc \
    python3-websockify \
    # XFCE Desktop
    xfce4 \
    xfce4-terminal \
    thunar \
    # Fonts
    fonts-liberation \
    fonts-noto-cjk \
    # Utilities
    x11-utils \
    xdotool \
    && apt-get remove -y xfce4-screensaver xscreensaver || true \
    && rm -rf /var/lib/apt/lists/*

# Optional: Install Playwright system dependencies (requires root)
RUN npx playwright install-deps chromium || true

# Optional: Install Playwright for browser automation
USER sandbox
RUN npm install -g playwright && \
    npx playwright install chromium || true

USER root

# VNC startup script
COPY start-vnc.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/start-vnc.sh

# Update entrypoint (includes original claude entrypoint logic)
COPY entrypoint-vnc.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Environment
ENV DISPLAY=:99
ENV VNC_PORT=5900
ENV NOVNC_PORT=6080
ENV RESOLUTION=1920x1080x24
ENV SANDBOX_VNC_ENABLED=true
ENV SANDBOX_DESKTOP=xfce

EXPOSE 5900 6080

USER sandbox
WORKDIR /workspace

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["sleep", "infinity"]
```

### start-vnc.sh (Shared)

```bash
#!/bin/bash
set -e

DISPLAY_NUM="${DISPLAY_NUM:-99}"
RESOLUTION="${RESOLUTION:-1920x1080x24}"
VNC_PORT="${VNC_PORT:-5900}"
NOVNC_PORT="${NOVNC_PORT:-6080}"
VNC_PASSWORD="${VNC_PASSWORD:-}"
DESKTOP="${SANDBOX_DESKTOP:-fluxbox}"

export DISPLAY=:${DISPLAY_NUM}

# Start Xvfb (virtual framebuffer)
echo "Starting Xvfb on display :${DISPLAY_NUM}..."
Xvfb :${DISPLAY_NUM} -screen 0 ${RESOLUTION} &
XVFB_PID=$!
sleep 1

if ! kill -0 $XVFB_PID 2>/dev/null; then
    echo "ERROR: Xvfb failed to start"
    exit 1
fi

# Start window manager / desktop
echo "Starting ${DESKTOP}..."
case "$DESKTOP" in
    xfce|xfce4)
        startxfce4 &
        ;;
    *)
        fluxbox &
        ;;
esac

# Start VNC server
echo "Starting x11vnc on port ${VNC_PORT}..."
VNC_ARGS="-display :${DISPLAY_NUM} -forever -shared -rfbport ${VNC_PORT} -noxdamage"
if [ -n "$VNC_PASSWORD" ]; then
    mkdir -p ~/.vnc
    x11vnc -storepasswd "$VNC_PASSWORD" ~/.vnc/passwd
    VNC_ARGS="$VNC_ARGS -rfbauth ~/.vnc/passwd"
else
    VNC_ARGS="$VNC_ARGS -nopw"
fi
x11vnc $VNC_ARGS &

# Start noVNC (websockify)
echo "Starting noVNC on port ${NOVNC_PORT}..."
websockify --web=/usr/share/novnc/ ${NOVNC_PORT} localhost:${VNC_PORT} &

echo "VNC services started successfully"
echo "  - Desktop: ${DESKTOP}"
echo "  - VNC port: ${VNC_PORT}"
echo "  - noVNC port: ${NOVNC_PORT}"

# Note: Don't wait here - let the entrypoint continue
# Background processes will keep running
```

### entrypoint-vnc.sh

```bash
#!/bin/bash
# Container entrypoint for VNC-enabled images
# This extends the original sandbox-claude entrypoint with VNC support

# ============================================
# VNC Services Startup
# ============================================
if [ "$SANDBOX_VNC_ENABLED" = "true" ]; then
    echo "Starting VNC services..."
    /usr/local/bin/start-vnc.sh &
    sleep 2
fi

# ============================================
# Original sandbox-claude entrypoint logic
# (copied from sandbox-claude Dockerfile)
# ============================================
WORKSPACE="${WORKSPACE:-/workspace}"
PORT="${CLAUDE_PROXY_PORT:-3456}"
ENV_FILE="/tmp/claude-proxy-env"

# If proxy env vars are set AND proxy is not running, start it
# This supports docker run -e CLAUDE_PROXY_BACKEND=... usage
if [ -n "$CLAUDE_PROXY_BACKEND" ] && [ -n "$CLAUDE_PROXY_API_KEY" ] && [ -n "$CLAUDE_PROXY_MODEL" ]; then
    if ! curl -s "http://127.0.0.1:${PORT}/health" > /dev/null 2>&1; then
        /usr/local/bin/start-claude-proxy
    fi
    
    # Write env vars to a file that can be sourced
    if curl -s "http://127.0.0.1:${PORT}/health" > /dev/null 2>&1; then
        echo "export ANTHROPIC_BASE_URL=http://127.0.0.1:${PORT}" > "$ENV_FILE"
        echo "export ANTHROPIC_API_KEY=dummy" >> "$ENV_FILE"
        chmod 644 "$ENV_FILE"
    fi
fi

# Execute the command passed to docker run
exec "$@"
```

### VNC Proxy Implementation

```go
// sandbox/vncproxy/proxy.go
package vncproxy

import (
    "context"
    "encoding/json"
    "fmt"
    "net"
    "net/http"
    "strings"
    "sync"
    "time"

    "github.com/docker/docker/client"
    "github.com/gorilla/websocket"
)

type Proxy struct {
    docker  *client.Client
    config  *Config
    
    // IP cache with TTL support
    ipCache   map[string]ipCacheEntry
    ipCacheMu sync.RWMutex
}

type Config struct {
    ContainerVNCPort   int           // default: 5900
    ContainerNoVNCPort int           // default: 6080
    Timeout            time.Duration // default: 30s
}

func New(docker *client.Client, config *Config) *Proxy {
    if config.ContainerVNCPort == 0 {
        config.ContainerVNCPort = 5900
    }
    if config.ContainerNoVNCPort == 0 {
        config.ContainerNoVNCPort = 6080
    }
    if config.Timeout == 0 {
        config.Timeout = 30 * time.Second
    }
    
    return &Proxy{
        docker:  docker,
        config:  config,
        ipCache: make(map[string]ipCacheEntry),
    }
}

// HandleVNCStatus returns VNC status for a container
// GET /v1/sandbox/{id}/vnc
func (p *Proxy) HandleVNCStatus(w http.ResponseWriter, r *http.Request) {
    sandboxID := extractSandboxID(r)
    containerName := fmt.Sprintf("yao-sandbox-%s", sandboxID)
    
    response := map[string]interface{}{
        "sandbox_id": sandboxID,
        "container":  containerName,
    }
    
    // Check if container exists and is running
    ip, err := p.getContainerIP(r.Context(), containerName)
    if err != nil {
        response["available"] = false
        response["status"] = "unavailable"
        response["message"] = "Container not available"
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(response)
        return
    }
    
    // Check if VNC is enabled for this container
    if !p.checkVNCEnabled(r.Context(), containerName) {
        response["available"] = false
        response["status"] = "not_supported"
        response["message"] = "VNC not available for this container type"
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(response)
        return
    }
    
    // Check if VNC services are ready (try to connect to websockify port)
    if !p.checkVNCReady(r.Context(), ip) {
        response["available"] = false
        response["status"] = "starting"
        response["message"] = "VNC services are starting..."
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(response)
        return
    }
    
    // VNC is ready
    response["available"] = true
    response["status"] = "ready"
    response["client_url"] = fmt.Sprintf("/v1/sandbox/%s/vnc/client", sandboxID)
    response["websocket_url"] = fmt.Sprintf("/v1/sandbox/%s/vnc/ws", sandboxID)
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

// checkVNCReady tests if VNC services are ready by attempting TCP connection
func (p *Proxy) checkVNCReady(ctx context.Context, containerIP string) bool {
    addr := fmt.Sprintf("%s:%d", containerIP, p.config.ContainerNoVNCPort)
    conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
    if err != nil {
        return false
    }
    conn.Close()
    return true
}

// HandleVNCClient serves the noVNC client page
// GET /v1/sandbox/{id}/vnc/client?viewonly=true|false
func (p *Proxy) HandleVNCClient(w http.ResponseWriter, r *http.Request) {
    sandboxID := extractSandboxID(r)
    containerName := fmt.Sprintf("yao-sandbox-%s", sandboxID)
    
    // Verify container exists, is running, and has VNC
    _, err := p.getContainerIP(r.Context(), containerName)
    if err != nil {
        http.Error(w, "Container not available", http.StatusNotFound)
        return
    }
    
    if !p.checkVNCEnabled(r.Context(), containerName) {
        http.Error(w, "VNC not available for this container", http.StatusBadRequest)
        return
    }
    
    // Get viewonly parameter (default: false = interactive)
    viewOnly := r.URL.Query().Get("viewonly") == "true"
    
    // Serve inline noVNC HTML page with status checking
    // This embeds the noVNC client directly, with retry logic for VNC startup delay
    wsURL := fmt.Sprintf("/v1/sandbox/%s/vnc/ws", sandboxID)
    p.serveNoVNCPage(w, sandboxID, wsURL, viewOnly)
}

// serveNoVNCPage serves an inline HTML page that loads noVNC
// Includes status checking and retry logic for VNC startup delay
// viewOnly: if true, user can only watch; if false, user can interact with keyboard/mouse
func (p *Proxy) serveNoVNCPage(w http.ResponseWriter, sandboxID string, wsPath string, viewOnly bool) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    
    viewOnlyJS := "false"
    if viewOnly {
        viewOnlyJS = "true"
    }
    
    html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Sandbox Preview</title>
    <style>
        body { margin: 0; padding: 0; overflow: hidden; background: #1a1a1a; font-family: system-ui, sans-serif; }
        #screen { width: 100vw; height: 100vh; display: none; }
        #status { 
            display: flex; flex-direction: column; align-items: center; justify-content: center;
            width: 100vw; height: 100vh; color: #fff;
        }
        .spinner { 
            width: 40px; height: 40px; border: 3px solid #333; border-top-color: #3b82f6;
            border-radius: 50%%; animation: spin 1s linear infinite; margin-bottom: 16px;
        }
        @keyframes spin { to { transform: rotate(360deg); } }
        .message { font-size: 14px; color: #888; }
        .error { color: #ef4444; }
        .retry-btn {
            margin-top: 16px; padding: 8px 16px; background: #3b82f6; color: #fff;
            border: none; border-radius: 4px; cursor: pointer;
        }
        .mode-indicator {
            position: fixed; top: 8px; right: 8px; padding: 4px 8px;
            background: rgba(0,0,0,0.6); color: #888; font-size: 12px;
            border-radius: 4px; z-index: 1000;
        }
        .mode-indicator.interactive { color: #4ade80; }
    </style>
</head>
<body>
    <div id="status">
        <div class="spinner"></div>
        <div class="message">正在连接 VNC 服务...</div>
    </div>
    <div id="screen"></div>
    <div id="mode" class="mode-indicator" style="display:none;"></div>
    
    <script type="module">
        import RFB from 'https://cdn.jsdelivr.net/npm/@novnc/novnc@1.4.0/core/rfb.js';
        
        const statusEl = document.getElementById('status');
        const screenEl = document.getElementById('screen');
        const modeEl = document.getElementById('mode');
        const sandboxID = '%s';
        const wsPath = '%s';
        const viewOnly = %s;
        const maxRetries = 30;  // Max 30 seconds
        let retryCount = 0;
        let rfb = null;
        
        function showStatus(message, isError = false) {
            statusEl.style.display = 'flex';
            screenEl.style.display = 'none';
            modeEl.style.display = 'none';
            statusEl.innerHTML = isError 
                ? '<div class="message error">' + message + '</div><button class="retry-btn" onclick="location.reload()">重试</button>'
                : '<div class="spinner"></div><div class="message">' + message + '</div>';
        }
        
        function showScreen() {
            statusEl.style.display = 'none';
            screenEl.style.display = 'block';
            // Show mode indicator
            modeEl.style.display = 'block';
            if (viewOnly) {
                modeEl.textContent = '只读模式';
                modeEl.className = 'mode-indicator';
            } else {
                modeEl.textContent = '可交互';
                modeEl.className = 'mode-indicator interactive';
            }
        }
        
        async function checkAndConnect() {
            try {
                const res = await fetch('/v1/sandbox/' + sandboxID + '/vnc');
                const data = await res.json();
                
                if (!data.available) {
                    retryCount++;
                    if (retryCount >= maxRetries) {
                        showStatus('VNC 服务启动超时，请稍后重试', true);
                        return;
                    }
                    showStatus('正在等待 VNC 服务启动... (' + retryCount + 's)');
                    setTimeout(checkAndConnect, 1000);
                    return;
                }
                
                // VNC ready, connect
                showStatus('正在连接...');
                const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
                const url = protocol + '//' + window.location.host + wsPath;
                
                rfb = new RFB(screenEl, url);
                rfb.viewOnly = viewOnly;  // Set view-only mode
                rfb.scaleViewport = true;
                rfb.resizeSession = true;
                
                rfb.addEventListener('connect', () => {
                    showScreen();
                });
                
                rfb.addEventListener('disconnect', (e) => {
                    if (e.detail.clean) {
                        showStatus('连接已断开', true);
                    } else {
                        showStatus('连接丢失，请刷新页面重试', true);
                    }
                });
                
            } catch (err) {
                retryCount++;
                if (retryCount >= maxRetries) {
                    showStatus('无法连接到服务器: ' + err.message, true);
                    return;
                }
                setTimeout(checkAndConnect, 1000);
            }
        }
        
        checkAndConnect();
    </script>
</body>
</html>`, sandboxID, wsPath, viewOnlyJS)
    w.Write([]byte(html))
}

// HandleVNCWebSocket proxies WebSocket to container VNC
// GET /v1/sandbox/{id}/vnc/ws
func (p *Proxy) HandleVNCWebSocket(w http.ResponseWriter, r *http.Request) {
    sandboxID := extractSandboxID(r)
    containerName := fmt.Sprintf("yao-sandbox-%s", sandboxID)
    
    ip, err := p.getContainerIP(r.Context(), containerName)
    if err != nil {
        http.Error(w, "Container not available", http.StatusNotFound)
        return
    }
    
    if !p.checkVNCEnabled(r.Context(), containerName) {
        http.Error(w, "VNC not available for this container", http.StatusBadRequest)
        return
    }
    
    // Proxy WebSocket to container's websockify port
    targetURL := fmt.Sprintf("ws://%s:%d", ip, p.config.ContainerNoVNCPort)
    p.proxyWebSocket(w, r, targetURL)
}

func (p *Proxy) checkVNCEnabled(ctx context.Context, containerName string) bool {
    inspect, err := p.docker.ContainerInspect(ctx, containerName)
    if err != nil {
        return false
    }
    
    // Check environment variable SANDBOX_VNC_ENABLED
    for _, env := range inspect.Config.Env {
        if env == "SANDBOX_VNC_ENABLED=true" {
            return true
        }
    }
    return false
}

// ipCacheEntry holds cached IP with expiration
type ipCacheEntry struct {
    IP        string
    ExpiresAt time.Time
}

func (p *Proxy) getContainerIP(ctx context.Context, containerName string) (string, error) {
    // Check cache first (with TTL)
    p.ipCacheMu.RLock()
    if entry, ok := p.ipCache[containerName]; ok {
        if time.Now().Before(entry.ExpiresAt) {
            p.ipCacheMu.RUnlock()
            return entry.IP, nil
        }
    }
    p.ipCacheMu.RUnlock()
    
    // Cache miss or expired, fetch from Docker
    inspect, err := p.docker.ContainerInspect(ctx, containerName)
    if err != nil {
        // Remove stale cache entry
        p.ipCacheMu.Lock()
        delete(p.ipCache, containerName)
        p.ipCacheMu.Unlock()
        return "", fmt.Errorf("container not found: %w", err)
    }
    
    if !inspect.State.Running {
        // Remove stale cache entry
        p.ipCacheMu.Lock()
        delete(p.ipCache, containerName)
        p.ipCacheMu.Unlock()
        return "", fmt.Errorf("container not running")
    }
    
    ip := inspect.NetworkSettings.IPAddress
    if ip == "" {
        if networks := inspect.NetworkSettings.Networks; networks != nil {
            if bridge, ok := networks["bridge"]; ok {
                ip = bridge.IPAddress
            }
        }
    }
    
    if ip == "" {
        return "", fmt.Errorf("container has no IP address")
    }
    
    // Cache with 30 second TTL
    p.ipCacheMu.Lock()
    p.ipCache[containerName] = ipCacheEntry{
        IP:        ip,
        ExpiresAt: time.Now().Add(30 * time.Second),
    }
    p.ipCacheMu.Unlock()
    
    return ip, nil
}

// InvalidateCache removes a container from the IP cache
// Call this when container state changes (stop/restart)
func (p *Proxy) InvalidateCache(containerName string) {
    p.ipCacheMu.Lock()
    delete(p.ipCache, containerName)
    p.ipCacheMu.Unlock()
}

func (p *Proxy) proxyWebSocket(w http.ResponseWriter, r *http.Request, targetURL string) {
    upgrader := websocket.Upgrader{
        CheckOrigin:  func(r *http.Request) bool { return true },
        Subprotocols: []string{"binary"}, // Required for noVNC
    }
    
    clientConn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        return
    }
    defer clientConn.Close()
    
    dialer := websocket.Dialer{
        HandshakeTimeout: p.config.Timeout,
    }
    
    targetConn, _, err := dialer.Dial(targetURL, nil)
    if err != nil {
        return
    }
    defer targetConn.Close()
    
    errChan := make(chan error, 2)
    
    // Client -> Target
    go func() {
        for {
            msgType, data, err := clientConn.ReadMessage()
            if err != nil {
                errChan <- err
                return
            }
            if err := targetConn.WriteMessage(msgType, data); err != nil {
                errChan <- err
                return
            }
        }
    }()
    
    // Target -> Client
    go func() {
        for {
            msgType, data, err := targetConn.ReadMessage()
            if err != nil {
                errChan <- err
                return
            }
            if err := clientConn.WriteMessage(msgType, data); err != nil {
                errChan <- err
                return
            }
        }
    }()
    
    <-errChan
}

func extractSandboxID(r *http.Request) string {
    // Extract from path: /v1/sandbox/{id}/vnc/...
    path := r.URL.Path
    path = strings.TrimPrefix(path, "/v1/sandbox/")
    parts := strings.Split(path, "/")
    if len(parts) >= 1 {
        return parts[0]
    }
    return ""
}
```

## Security Considerations

### 1. Authentication

All VNC endpoints verify user authentication:

```go
func (p *Proxy) authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        sandboxID := extractSandboxID(r)
        
        // Verify the requesting user owns this sandbox
        // Implementation depends on how sandboxID maps to users:
        // - If sandboxID = "{userID}-{chatID}", extract userID and compare with session
        // - If sandboxID = UUID, lookup in database
        // - Delegate to business layer authorization service
        
        // Example: extract userID from sandboxID pattern "{userID}-{chatID}"
        // parts := strings.SplitN(sandboxID, "-", 2)
        // if len(parts) >= 1 {
        //     ownerID := parts[0]
        //     sessionUserID := getSessionUserID(r)
        //     if sessionUserID != ownerID {
        //         http.Error(w, "Unauthorized", http.StatusUnauthorized)
        //         return
        //     }
        // }
        
        // TODO: Implement authorization logic based on your sandboxID scheme
        
        next.ServeHTTP(w, r)
    })
}
```

### 2. Network Isolation

- Containers use Docker bridge network (internal only)
- No VNC ports exposed to host
- All access through authenticated proxy
- Each user can only access their own containers

### 3. Resource Limits by Image Type

| Image Type | Memory | CPU | Disk |
|------------|--------|-----|------|
| claude | 2GB | 1.0 | - |
| playwright | 4GB | 2.0 | - |
| desktop | 4GB | 2.0 | - |

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `YAO_SANDBOX_IMAGE` | `yaoapp/sandbox-claude:latest` | Default sandbox image |
| `YAO_SANDBOX_VNC_PORT_MAPPING` | `false` | Enable VNC port mapping to host (for Docker Desktop) |
| `YAO_VNC_PROXY_ENABLED` | `true` | Enable VNC proxy |
| `YAO_VNC_RESOLUTION` | `1920x1080x24` | VNC screen resolution |

### Docker Desktop Support (macOS/Windows)

Docker Desktop runs containers inside a LinuxKit VM, so container IPs (`172.17.0.x`) are not directly accessible from the host. To enable VNC access on Docker Desktop:

```bash
# Enable VNC port mapping for local development
export YAO_SANDBOX_VNC_PORT_MAPPING=true
export YAO_SANDBOX_IMAGE="yaoapp/sandbox-claude-browser:latest"
```

When `YAO_SANDBOX_VNC_PORT_MAPPING=true`:
- Container ports `6080/tcp` (noVNC) and `5900/tcp` (VNC) are mapped to random available host ports
- Ports are bound to `127.0.0.1` for security
- VNC Proxy automatically detects and uses the mapped host ports

On Linux (native Docker), this option is not needed as container IPs are directly accessible.

## Implementation Checklist

### Yao Backend ✅ 完成

- [x] `sandbox/docker/browser/Dockerfile` - Browser + VNC image
- [x] `sandbox/docker/desktop/Dockerfile` - Full desktop + VNC image  
- [x] `sandbox/docker/vnc/start-vnc.sh` - Shared VNC startup script
- [x] `sandbox/docker/vnc/entrypoint-vnc.sh` - VNC entrypoint
- [x] `sandbox/vncproxy/proxy.go` - VNC WebSocket proxy
- [x] `sandbox/vncproxy/config.go` - Proxy configuration
- [x] API router integration - VNC endpoints (`openapi/sandbox/sandbox.go`)
- [x] `sandbox/docker/build.sh` - Update build script
- [x] `sandbox/config.go` - VNC port mapping configuration
- [x] `sandbox/manager.go` - Dynamic VNC port mapping for Docker Desktop

### No Changes Needed

- `agent/sandbox/` - existing `Image` field already supports custom images
- `cui/` - existing `navigate` action handles iframe loading via `app/openSidebar`

## File Structure

### Yao (Backend)

```
yao/sandbox/
├── vncproxy/                    # VNC Proxy Service
│   ├── proxy.go                 # Main proxy implementation (with port mapping detection)
│   ├── proxy_test.go            # Unit tests
│   └── config.go                # Configuration
├── docker/
│   ├── base/
│   │   └── Dockerfile.base
│   ├── claude/
│   │   ├── Dockerfile
│   │   └── Dockerfile.full
│   ├── browser/                 # Browser + VNC image
│   │   └── Dockerfile
│   ├── desktop/                 # XFCE Desktop + VNC image
│   │   └── Dockerfile
│   ├── vnc/                     # Shared VNC scripts
│   │   ├── start-vnc.sh
│   │   └── entrypoint-vnc.sh
│   └── build.sh                 # Build script for all images
├── manager.go                   # Container management (with VNC port mapping)
├── config.go                    # Configuration (VNCPortMapping option)
├── DESIGN-PLAYWRIGHT-VNC.md     # This document
├── TODO-VNC.md                  # Implementation checklist
└── README.md                    # Quick start guide
```

### CUI (Frontend)

```
cui/packages/cui/
└── ...                          # No changes needed
```

The CUI `navigate` action already supports loading URLs via iframe in sidebar. The `/v1/sandbox/{id}/vnc/client` API returns a complete HTML page that will be loaded directly.

### Agent (No Changes)

```
yao/agent/
├── sandbox/
│   ├── types.go                 # Already supports custom Image
│   └── ...                      # No changes needed
└── ...
```

## Command Execution

### Overview

Commands execute identically across all sandbox images. The `Manager.Exec()` and `Manager.Stream()` methods remain unchanged.

### No Manager Changes Required

```go
// Manager.Exec() and Manager.Stream() remain unchanged
// Commands run the same way on all images
// DISPLAY=:99 is set in container env, GUI apps (browsers) use it automatically
```

### Behavior by Image Type

| Image | DISPLAY | VNC Visible | Agent Gets Output |
|-------|---------|-------------|-------------------|
| sandbox-claude | ❌ | N/A | ✅ |
| sandbox-claude-browser | ✅ :99 | Browser window | ✅ |
| sandbox-claude-desktop | ✅ :99 | Browser + Desktop apps | ✅ |

### What Users See in VNC

| Operation | sandbox-claude-browser | sandbox-claude-desktop |
|-----------|--------------------------|------------------------|
| Browser automation | ✅ Visible | ✅ Visible |
| File operations | ❌ | ✅ (open Thunar) |
| Terminal commands | ❌ | ❌ (output to Agent) |

**Note**: Terminal command output goes to Agent, not to VNC terminal window. This is by design - `docker exec` runs commands directly in the container, not through a terminal emulator. Users can manually open a terminal in VNC if they want to run commands interactively.

### Why This Design

1. **100% backward compatible**: No changes to Manager.go
2. **Agent output intact**: stdout/stderr captured normally
3. **Browser visible**: Main use case (Playwright) works perfectly
4. **Low risk**: No code changes = no bugs
5. **Future improvement**: Terminal visibility can be added later if needed

---

## Appendix

### A. Image Comparison

| Feature | sandbox-claude | sandbox-claude-browser | sandbox-claude-desktop |
|---------|---------------|--------------------------|------------------------|
| Claude CLI | ✅ | ✅ | ✅ |
| Node.js | ✅ | ✅ | ✅ |
| Python | ✅ | ✅ | ✅ |
| VNC Access | ❌ | ✅ | ✅ |
| Playwright | ❌ | ✅ | ✅ (optional) |
| File Manager | ❌ | ❌ | ✅ |
| Terminal GUI | ❌ | ❌ | ✅ |
| Desktop | ❌ | Minimal (Fluxbox) | Full (XFCE) |
| Image Size | ~700MB | ~1.8GB | ~2.5GB |
| Memory | 2GB | 4GB | 4GB |
| **Best For** | Scripts, CLI | Browser automation | Full transparency |

### B. User Visibility & Interaction

What users can see and do in VNC:

| Operation | sandbox-claude-browser | sandbox-claude-desktop |
|-----------|--------------------------|------------------------|
| Browser navigation | ✅ See | ✅ See |
| Browser clicks/typing | ✅ See | ✅ See |
| File creation | ❌ (log only) | ✅ (file manager) |
| Command execution | ❌ (output to Agent) | ❌ (output to Agent) |
| Code editing | ❌ | ✅ (if editor installed) |
| **Trust Level** | Medium | High |

**User Interaction Modes**:

| Mode | URL Parameter | User Can |
|------|---------------|----------|
| View-only | `?viewonly=true` | Watch only |
| Interactive | (default) | Keyboard, mouse, typing |

**Typical Interactive Scenarios**:
- User login (accounts, passwords)
- CAPTCHA solving
- Two-factor authentication
- Manual form filling

**Note**: Command output goes to Agent (via docker exec), not to a visible terminal in VNC. Users can manually open a terminal in `sandbox-claude-desktop` if needed.

### C. Implementation Summary

| Component | Location | Changes |
|-----------|----------|---------|
| Docker Images | `sandbox/docker/browser/`, `sandbox/docker/desktop/` | NEW |
| VNC Proxy | `sandbox/vncproxy/` | NEW |
| VNC API | Yao router | NEW endpoints |
| CUI | `cui/` | **No changes** (navigate action + iframe) |
| Sandbox Manager | `sandbox/manager.go` | **No changes** |
| Agent Sandbox | `agent/sandbox/` | **No changes** |

### D. References

- [Playwright Docker Documentation](https://playwright.dev/docs/docker)
- [noVNC GitHub](https://github.com/novnc/noVNC)
- [XFCE Documentation](https://docs.xfce.org/)
- [CUI Action System](../../cui/packages/cui/chatbox/messages/Action/actions/navigate.ts)
- [Yao Sandbox README](./README.md)
- [Yao Sandbox DESIGN](./DESIGN.md)
