# Claude sandbox with browser automation + VNC preview
# Image: sandbox-claude-browser
# Base: sandbox-claude (Ubuntu 24.04 + Node.js + Python + Claude CLI)
# Adds: Xvfb + x11vnc + noVNC + Fluxbox + Playwright/Puppeteer browsers
#
# Lightweight browser environment for web automation tasks
# Supports both amd64 and arm64 architectures

ARG REGISTRY=yaoapp
FROM ${REGISTRY}/sandbox-claude:latest

USER root

# Use MIT mirror (USA) for ARM64
RUN sed -i 's|http://ports.ubuntu.com/ubuntu-ports|http://mirrors.mit.edu/ubuntu-ports|g' /etc/apt/sources.list.d/ubuntu.sources 2>/dev/null || \
    sed -i 's|http://ports.ubuntu.com/ubuntu-ports|http://mirrors.mit.edu/ubuntu-ports|g' /etc/apt/sources.list 2>/dev/null || true

# Install X11, VNC, and minimal window manager
RUN apt-get update && apt-get install -y --no-install-recommends \
    # Sudo for sandbox user
    sudo \
    # Virtual display
    xvfb \
    # VNC server
    x11vnc \
    # noVNC (HTML5 VNC client) and websockify
    novnc \
    python3-websockify \
    # Minimal window manager (lightweight, perfect for Playwright)
    fluxbox \
    # Background/wallpaper utilities
    feh \
    imagemagick \
    # Fonts (required for proper browser rendering)
    fonts-liberation \
    fonts-noto-cjk \
    fonts-noto-color-emoji \
    # X11 utilities
    x11-utils \
    xdotool \
    # Audio (for video playback in browsers, can be disabled)
    pulseaudio \
    && rm -rf /var/lib/apt/lists/*

# Configure passwordless sudo for sandbox user
RUN echo "sandbox ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/sandbox && \
    chmod 0440 /etc/sudoers.d/sandbox

# Install Playwright system dependencies (requires root)
# This installs system libraries needed by Chromium/Firefox
RUN npx playwright install-deps chromium firefox || true

# Install Playwright and browsers as sandbox user
USER sandbox

# Install Playwright for Node.js (global) and Python
RUN npm install -g playwright && \
    pip install --user --break-system-packages playwright && \
    npx playwright install chromium firefox

USER root

# Create directories for branding assets
RUN mkdir -p /usr/local/share/yao

# Copy VNC startup scripts and branding assets
# Note: Build context should be sandbox/docker/, so paths are relative to that
COPY vnc/start-vnc.sh /usr/local/bin/start-vnc.sh
COPY vnc/entrypoint-vnc.sh /usr/local/bin/entrypoint.sh
COPY browser/config/setup-fluxbox.sh /usr/local/bin/setup-fluxbox.sh
COPY browser/config/yao-logo.png /usr/local/share/yao/yao-logo.png
RUN chmod +x /usr/local/bin/start-vnc.sh /usr/local/bin/entrypoint.sh /usr/local/bin/setup-fluxbox.sh

# Environment variables for VNC
ENV DISPLAY=:99
ENV VNC_PORT=5900
ENV NOVNC_PORT=6080
ENV RESOLUTION=1920x1080x24
ENV SANDBOX_VNC_ENABLED=true
ENV SANDBOX_DESKTOP=fluxbox

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
    npx playwright --version && \
    python3 -c "from playwright.sync_api import sync_playwright; print('Python Playwright: OK')" && \
    which fluxbox && \
    which x11vnc && \
    which Xvfb && \
    echo "=== All installations verified ==="

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["sleep", "infinity"]
