# Claude sandbox with real Google Chrome + anti-detection stealth
# Image: sandbox-claude-chrome
# Base: sandbox-claude (Ubuntu 24.04 + Node.js + Python + Claude CLI)
# Adds: Xvfb + x11vnc + noVNC + Fluxbox + Real Chrome + Patchright + PyAutoGUI
#
# Anti-bot detection browser environment for web research tasks
# amd64 architecture only (Google Chrome has no official arm64 Linux build)

ARG REGISTRY=yaoapp
FROM ${REGISTRY}/sandbox-claude:latest

USER root

# ============================================
# 1. VNC + Window Manager (same as browser image)
# ============================================
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
    # Minimal window manager
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
    # Audio (for video playback)
    pulseaudio \
    # PyAutoGUI X11 dependencies
    python3-tk \
    python3-dev \
    scrot \
    # Misc
    xterm \
    && rm -rf /var/lib/apt/lists/*

# Configure passwordless sudo for sandbox user
RUN echo "sandbox ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/sandbox && \
    chmod 0440 /etc/sudoers.d/sandbox

# ============================================
# 2. Real Google Chrome (amd64 only)
# ============================================
RUN curl -fsSL https://dl.google.com/linux/linux_signing_key.pub \
      | gpg --dearmor -o /usr/share/keyrings/google-chrome.gpg && \
    echo "deb [arch=amd64 signed-by=/usr/share/keyrings/google-chrome.gpg] \
      http://dl.google.com/linux/chrome/deb/ stable main" \
      > /etc/apt/sources.list.d/google-chrome.list && \
    apt-get update && apt-get install -y google-chrome-stable && \
    rm -rf /var/lib/apt/lists/*

# ============================================
# 3. Playwright system deps (for Patchright compatibility)
# ============================================
RUN npx playwright install-deps chromium || true

# ============================================
# 4. Python anti-detection libraries
# ============================================
USER sandbox

# Install Patchright (stealth Playwright fork) + PyAutoGUI + stealth libs
RUN pip install --user --break-system-packages \
    patchright \
    pyautogui \
    playwright-stealth \
    playwright && \
    # Install Patchright browser deps (uses system Chrome, no Chromium download)
    python3 -m patchright install chromium || true

# Install Node.js Playwright + stealth plugins
RUN npm install -g playwright playwright-extra puppeteer-extra-plugin-stealth

USER root

# ============================================
# 5. Copy config files
# ============================================
RUN mkdir -p /usr/local/share/yao

# VNC startup scripts (shared with browser/desktop)
COPY vnc/start-vnc.sh /usr/local/bin/start-vnc.sh
COPY vnc/entrypoint-vnc.sh /usr/local/bin/entrypoint.sh

# Chrome-specific config files
COPY chrome/config/setup-fluxbox.sh /usr/local/bin/setup-fluxbox.sh
COPY chrome/config/chrome-stealth.sh /usr/local/bin/chrome-stealth
COPY chrome/config/stealth-init.js /usr/local/share/yao/stealth-init.js
COPY chrome/config/chrome-preferences.json /usr/local/share/yao/chrome-preferences.json

# Reuse yao-logo from browser image
COPY browser/config/yao-logo.png /usr/local/share/yao/yao-logo.png

RUN chmod +x /usr/local/bin/start-vnc.sh \
             /usr/local/bin/entrypoint.sh \
             /usr/local/bin/setup-fluxbox.sh \
             /usr/local/bin/chrome-stealth

# ============================================
# 6. Default Chrome profile + X11 auth
# ============================================
RUN mkdir -p /home/sandbox/.config/google-chrome/Default && \
    cp /usr/local/share/yao/chrome-preferences.json \
       /home/sandbox/.config/google-chrome/Default/Preferences && \
    # Mark first run as done
    touch /home/sandbox/.config/google-chrome/First\ Run && \
    # Create .Xauthority for PyAutoGUI (Xvfb runs without auth)
    touch /home/sandbox/.Xauthority && \
    chown -R sandbox:sandbox /home/sandbox/.config/google-chrome /home/sandbox/.Xauthority

# ============================================
# 7. Environment variables
# ============================================
ENV DISPLAY=:99
ENV VNC_PORT=5900
ENV NOVNC_PORT=6080
ENV RESOLUTION=1920x1080x24
ENV SANDBOX_VNC_ENABLED=true
ENV SANDBOX_DESKTOP=fluxbox

# Node.js environment
ENV NODE_PATH=/home/sandbox/.npm-global/lib/node_modules

# Timezone
ENV TZ=America/New_York

# Expose VNC ports (internal use only, accessed via proxy)
EXPOSE 5900 6080

USER sandbox
WORKDIR /workspace

# ============================================
# 8. Verify installations
# ============================================
RUN echo "=== Verifying installations ===" && \
    google-chrome-stable --version && \
    node --version && \
    npm --version && \
    python3 --version && \
    python3 -c "from patchright.sync_api import sync_playwright; print('Patchright: OK')" && \
    python3 -c "from playwright.sync_api import sync_playwright; print('Playwright: OK')" && \
    python3 -c "from playwright_stealth import Stealth; print('Playwright-Stealth: OK')" && \
    pip3 show pyautogui | head -2 && echo "PyAutoGUI: OK" && \
    which fluxbox && \
    which x11vnc && \
    which Xvfb && \
    which chrome-stealth && \
    test -f /usr/local/share/yao/stealth-init.js && \
    test -f /home/sandbox/.config/google-chrome/Default/Preferences && \
    echo "=== All installations verified ==="

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["sleep", "infinity"]
