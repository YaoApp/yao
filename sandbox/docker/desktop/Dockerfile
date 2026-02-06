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
    # Sudo for sandbox user
    sudo \
    # Virtual display
    xvfb \
    # VNC server
    x11vnc \
    # noVNC (HTML5 VNC client) and websockify
    novnc \
    python3-websockify \
    # D-Bus (required for XFCE)
    dbus-x11 \
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

# Configure passwordless sudo for sandbox user
RUN echo "sandbox ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers.d/sandbox && \
    chmod 0440 /etc/sudoers.d/sandbox

# Create chromium wrapper script (uses Playwright's Chromium, starts maximized)
RUN echo '#!/bin/bash\nexec /home/sandbox/.cache/ms-playwright/chromium-1208/chrome-linux/chrome --no-sandbox --start-maximized "$@"' > /usr/local/bin/chromium && \
    chmod +x /usr/local/bin/chromium

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

# Copy Yao branding assets
RUN mkdir -p /usr/share/yao
COPY desktop/config/yao-logo-48.png /usr/share/yao/yao-logo-48.png
COPY desktop/config/yao-logo-128.png /usr/share/yao/yao-logo-128.png
COPY desktop/config/yao-logo-256.png /usr/share/yao/yao-logo-256.png
COPY desktop/config/panel-launcher-chromium.desktop /usr/share/yao/panel-launcher-chromium.desktop
COPY desktop/config/workspace.desktop /usr/share/yao/workspace.desktop
COPY desktop/config/setup-xfce.sh /usr/local/bin/setup-xfce.sh
RUN chmod +x /usr/local/bin/setup-xfce.sh

# Environment variables for VNC
ENV DISPLAY=:99
ENV VNC_PORT=5900
ENV NOVNC_PORT=6080
ENV RESOLUTION=1920x1080x24
ENV SANDBOX_VNC_ENABLED=true
ENV SANDBOX_DESKTOP=xfce
# Set hostname for XFCE panel display
ENV HOSTNAME="Yao Sandbox"

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
