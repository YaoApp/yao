#!/bin/bash
# Container entrypoint for VNC-enabled sandbox images
# This extends the original sandbox-claude entrypoint with VNC support

# ============================================
# VNC Services Startup
# ============================================
if [ "$SANDBOX_VNC_ENABLED" = "true" ]; then
    echo "[Entrypoint] Starting VNC services..."
    /usr/local/bin/start-vnc.sh &
    # Wait for VNC to initialize
    sleep 3
    echo "[Entrypoint] VNC services started in background"
fi

# ============================================
# Original sandbox-claude entrypoint logic
# (from sandbox-claude Dockerfile)
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
