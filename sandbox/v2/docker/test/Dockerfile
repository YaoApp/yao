# Sandbox V2 test image — adds test services on top of v2-base
FROM yaoapp/sandbox-v2-base:latest

USER root

RUN apt-get update && apt-get install -y --no-install-recommends \
    nginx \
    python3 \
    python3-pip \
    && pip3 install --break-system-packages websockets \
    && rm -rf /var/lib/apt/lists/*

# Test service scripts
COPY ws-echo.py /opt/test/ws-echo.py
COPY sse-server.py /opt/test/sse-server.py
COPY entrypoint.sh /test-entrypoint.sh
RUN chmod +x /test-entrypoint.sh

USER sandbox
ENTRYPOINT ["/test-entrypoint.sh"]
CMD ["sleep", "infinity"]
