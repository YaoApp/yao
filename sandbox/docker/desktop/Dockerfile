# Claude sandbox with full XFCE desktop + VNC preview
# Image: sandbox-claude-desktop
# Base: sandbox-claude (Ubuntu 24.04 + Node.js + Python + Claude CLI)
# Adds: Xvfb + x11vnc + noVNC + XFCE desktop + File Manager + Terminal
#
# Supports both amd64 and arm64 architectures

ARG REGISTRY=yaoapp
FROM ${REGISTRY}/sandbox-claude:latest

USER root

# Use MIT mirror (USA) for ARM64
RUN sed -i 's|http://ports.ubuntu.com/ubuntu-ports|http://mirrors.mit.edu/ubuntu-ports|g' /etc/apt/sources.list.d/ubuntu.sources 2>/dev/null || \
    sed -i 's|http://ports.ubuntu.com/ubuntu-ports|http://mirrors.mit.edu/ubuntu-ports|g' /etc/apt/sources.list 2>/dev/null || true

# Install X11, VNC, and XFCE desktop environment
RUN apt-get update && apt-get install -y --no-install-recommends \
    # Virtual display
    xvfb \
    # VNC server
    x11vnc \
    # noVNC (HTML5 VNC client) and websockify
    novnc \
    python3-websockify \
    # XFCE Desktop (full-featured but lightweight)
    xfce4 \
    xfce4-terminal \
    thunar \
    # Fonts (required for proper rendering)
    fonts-liberation \
    fonts-noto-cjk \
    fonts-noto-color-emoji \
    # X11 utilities
    x11-utils \
    xdotool \
    # Audio
    pulseaudio \
    # Remove screensaver (causes issues in container)
    && apt-get remove -y xfce4-screensaver xscreensaver || true \
    && rm -rf /var/lib/apt/lists/*

# Optional: Install Playwright system dependencies (requires root)
# Users can run browser automation in desktop mode too
RUN npx playwright install-deps chromium || true

# Optional: Install Playwright for browser automation
USER sandbox
RUN npm install -g playwright && \
    pip install --user --break-system-packages playwright && \
    npx playwright install chromium || true

USER root

# Copy VNC startup scripts
# Note: Build context should be sandbox/docker/, so paths are relative to that
COPY vnc/start-vnc.sh /usr/local/bin/start-vnc.sh
COPY vnc/entrypoint-vnc.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/start-vnc.sh /usr/local/bin/entrypoint.sh

# Environment variables for VNC
ENV DISPLAY=:99
ENV VNC_PORT=5900
ENV NOVNC_PORT=6080
ENV RESOLUTION=1920x1080x24
ENV SANDBOX_VNC_ENABLED=true
ENV SANDBOX_DESKTOP=xfce

# Node.js environment - ensure global modules are accessible
ENV NODE_PATH=/home/sandbox/.npm-global/lib/node_modules

# Expose VNC ports (internal use only, accessed via proxy)
EXPOSE 5900 6080

USER sandbox
WORKDIR /workspace

# Verify installations
RUN echo "=== Verifying installations ===" && \
    node --version && \
    npm --version && \
    python3 --version && \
    which startxfce4 && \
    which thunar && \
    which xfce4-terminal && \
    which x11vnc && \
    which Xvfb && \
    echo "=== All installations verified ==="

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["sleep", "infinity"]
