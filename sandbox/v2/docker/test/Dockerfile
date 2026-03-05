# Sandbox V2 test image — adds test services + VNC desktop on top of v2-base
FROM yaoapp/sandbox-v2-base:latest

USER root

RUN apt-get update && apt-get install -y --no-install-recommends \
    nginx \
    python3 \
    python3-pip \
    xvfb \
    x11vnc \
    fluxbox \
    xterm \
    && pip3 install --break-system-packages websockets websockify \
    && rm -rf /var/lib/apt/lists/*

# Test service scripts
COPY ws-echo.py /opt/test/ws-echo.py
COPY sse-server.py /opt/test/sse-server.py
COPY entrypoint.sh /test-entrypoint.sh
RUN chmod +x /test-entrypoint.sh

ENV DISPLAY=:99

USER sandbox
EXPOSE 5900 6080
ENTRYPOINT ["/test-entrypoint.sh"]
CMD ["sleep", "infinity"]
