#!/bin/bash
# VNC services startup script
# Shared by sandbox-claude-playwright and sandbox-claude-desktop
# Starts: Xvfb (virtual display) + Window Manager + x11vnc + websockify (noVNC)

set -e

DISPLAY_NUM="${DISPLAY_NUM:-99}"
RESOLUTION="${RESOLUTION:-1920x1080x24}"
VNC_PORT="${VNC_PORT:-5900}"
NOVNC_PORT="${NOVNC_PORT:-6080}"
VNC_PASSWORD="${VNC_PASSWORD:-}"
DESKTOP="${SANDBOX_DESKTOP:-fluxbox}"

export DISPLAY=:${DISPLAY_NUM}

echo "[VNC] Starting VNC services..."
echo "[VNC] Display: :${DISPLAY_NUM}"
echo "[VNC] Resolution: ${RESOLUTION}"
echo "[VNC] Desktop: ${DESKTOP}"

# Start Xvfb (virtual framebuffer)
echo "[VNC] Starting Xvfb..."
Xvfb :${DISPLAY_NUM} -screen 0 ${RESOLUTION} &
XVFB_PID=$!
sleep 1

if ! kill -0 $XVFB_PID 2>/dev/null; then
    echo "[VNC] ERROR: Xvfb failed to start"
    exit 1
fi
echo "[VNC] Xvfb started (PID: $XVFB_PID)"

# Start window manager / desktop environment
echo "[VNC] Starting ${DESKTOP}..."
case "$DESKTOP" in
    xfce|xfce4)
        # XFCE desktop environment
        startxfce4 &
        ;;
    fluxbox)
        # Minimal window manager for Playwright
        fluxbox &
        ;;
    *)
        # Default to fluxbox
        fluxbox &
        ;;
esac
sleep 2

# Start x11vnc server
echo "[VNC] Starting x11vnc on port ${VNC_PORT}..."
VNC_ARGS="-display :${DISPLAY_NUM} -forever -shared -rfbport ${VNC_PORT} -noxdamage"

if [ -n "$VNC_PASSWORD" ]; then
    mkdir -p ~/.vnc
    x11vnc -storepasswd "$VNC_PASSWORD" ~/.vnc/passwd
    VNC_ARGS="$VNC_ARGS -rfbauth ~/.vnc/passwd"
else
    VNC_ARGS="$VNC_ARGS -nopw"
fi

x11vnc $VNC_ARGS &
X11VNC_PID=$!
sleep 1

if ! kill -0 $X11VNC_PID 2>/dev/null; then
    echo "[VNC] ERROR: x11vnc failed to start"
    exit 1
fi
echo "[VNC] x11vnc started (PID: $X11VNC_PID)"

# Start websockify (noVNC WebSocket proxy)
echo "[VNC] Starting websockify on port ${NOVNC_PORT}..."
websockify --web=/usr/share/novnc/ ${NOVNC_PORT} localhost:${VNC_PORT} &
WEBSOCKIFY_PID=$!
sleep 1

if ! kill -0 $WEBSOCKIFY_PID 2>/dev/null; then
    echo "[VNC] ERROR: websockify failed to start"
    exit 1
fi
echo "[VNC] websockify started (PID: $WEBSOCKIFY_PID)"

echo "[VNC] =================================="
echo "[VNC] VNC services started successfully"
echo "[VNC] Desktop: ${DESKTOP}"
echo "[VNC] VNC port: ${VNC_PORT}"
echo "[VNC] noVNC port: ${NOVNC_PORT}"
echo "[VNC] =================================="

# Note: Don't wait here - let the entrypoint continue
# Background processes will keep running
