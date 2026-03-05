#!/bin/bash
# V2 test entrypoint — starts test services + VNC desktop then delegates to base entrypoint

# Start Xvfb (virtual framebuffer)
Xvfb :99 -screen 0 1024x768x24 -ac +extension GLX +render -noreset &
sleep 0.5

# Start fluxbox window manager
fluxbox &

# Start x11vnc (raw RFB on 5900)
x11vnc -display :99 -rfbport 5900 -nopw -shared -forever -xkb -ncache 10 &
sleep 0.3

# Start websockify (WebSocket on 6080 → RFB 5900)
websockify 0.0.0.0:6080 localhost:5900 &

# Test services
python3 /opt/test/ws-echo.py &
python3 /opt/test/sse-server.py &

exec /entrypoint.sh "$@"
