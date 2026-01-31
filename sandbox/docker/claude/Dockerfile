# Claude sandbox image: Claude CLI + Node.js + Python + claude-proxy
# Supports both amd64 and arm64 architectures
# Base: Ubuntu 24.04 LTS
ARG REGISTRY=yaoapp
FROM ${REGISTRY}/sandbox-base:latest

USER root

# Use MIT mirror (USA) for ARM64 - in case base image cache is old
RUN sed -i 's|http://ports.ubuntu.com/ubuntu-ports|http://mirrors.mit.edu/ubuntu-ports|g' /etc/apt/sources.list.d/ubuntu.sources 2>/dev/null || \
    sed -i 's|http://ports.ubuntu.com/ubuntu-ports|http://mirrors.mit.edu/ubuntu-ports|g' /etc/apt/sources.list 2>/dev/null || true

# Python 3.12 (default in Ubuntu 24.04) - install first as Node.js depends on it
RUN apt-get update && apt-get install -y \
    python3 \
    python3-pip \
    python3-venv \
    && rm -rf /var/lib/apt/lists/* \
    && ln -sf /usr/bin/python3 /usr/bin/python

# Node.js 22 LTS (automatically detects architecture)
# Run apt-get update again to refresh package lists after nodesource setup
RUN curl -fsSL https://deb.nodesource.com/setup_22.x | bash - \
    && apt-get update \
    && apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/*

# npm global packages directory for sandbox user
RUN mkdir -p /home/sandbox/.npm-global && \
    chown -R sandbox:sandbox /home/sandbox/.npm-global

USER sandbox

# Configure npm to use user directory
RUN npm config set prefix '/home/sandbox/.npm-global'
ENV PATH="/home/sandbox/.npm-global/bin:${PATH}"

# Install Claude CLI
RUN npm install -g @anthropic-ai/claude-code || \
    echo "Claude CLI installation skipped (may not be available yet)"

USER root

# Install claude-proxy (architecture-specific binary)
ARG TARGETARCH
COPY claude-proxy-${TARGETARCH} /usr/local/bin/claude-proxy
RUN chmod +x /usr/local/bin/claude-proxy

# Create claude-proxy startup script
RUN cat > /usr/local/bin/start-proxy << 'SCRIPT'
#!/bin/bash
# Claude Proxy startup script
# Usage: start-proxy [options]
# Options are passed directly to claude-proxy

LOG_DIR="${WORKSPACE:-/workspace}"
LOG_FILE="${LOG_DIR}/proxy.log"

# Ensure log directory exists
mkdir -p "$LOG_DIR" 2>/dev/null || true

# Default environment variables (can be overridden)
export CLAUDE_PROXY_PORT="${CLAUDE_PROXY_PORT:-3456}"

# Start proxy with logging
exec /usr/local/bin/claude-proxy -v -l "$LOG_FILE" "$@"
SCRIPT
RUN chmod +x /usr/local/bin/start-proxy

# Create claude-run wrapper for easy usage (manual mode)
RUN cat > /usr/local/bin/claude-run << 'SCRIPT'
#!/bin/bash
# Claude CLI wrapper with proxy auto-start
# Usage: claude-run [claude options] "prompt"
#
# Environment variables:
#   CLAUDE_PROXY_BACKEND  - Backend API URL (required)
#   CLAUDE_PROXY_API_KEY  - Backend API Key (required)
#   CLAUDE_PROXY_MODEL    - Backend model name (required)
#   CLAUDE_PROXY_PORT     - Proxy port (default: 3456)
#   WORKSPACE             - Working directory (default: /workspace)

set -e

# Check required environment variables
if [ -z "$CLAUDE_PROXY_BACKEND" ]; then
    echo "Error: CLAUDE_PROXY_BACKEND is not set"
    echo "Example: export CLAUDE_PROXY_BACKEND=https://ark.cn-beijing.volces.com/api/v3/chat/completions"
    exit 1
fi

if [ -z "$CLAUDE_PROXY_API_KEY" ]; then
    echo "Error: CLAUDE_PROXY_API_KEY is not set"
    exit 1
fi

if [ -z "$CLAUDE_PROXY_MODEL" ]; then
    echo "Error: CLAUDE_PROXY_MODEL is not set"
    echo "Example: export CLAUDE_PROXY_MODEL=glm-4-7-251222"
    exit 1
fi

PORT="${CLAUDE_PROXY_PORT:-3456}"
WORKSPACE="${WORKSPACE:-/workspace}"
LOG_FILE="${WORKSPACE}/proxy.log"

# Check if proxy is already running
if curl -s "http://127.0.0.1:${PORT}/health" > /dev/null 2>&1; then
    echo "Proxy already running on port $PORT"
else
    echo "Starting claude-proxy..."
    mkdir -p "$WORKSPACE" 2>/dev/null || true
    nohup /usr/local/bin/claude-proxy -v -l "$LOG_FILE" > /dev/null 2>&1 &
    
    # Wait for proxy to start
    for i in {1..10}; do
        if curl -s "http://127.0.0.1:${PORT}/health" > /dev/null 2>&1; then
            echo "Proxy started successfully"
            break
        fi
        sleep 0.5
    done
    
    if ! curl -s "http://127.0.0.1:${PORT}/health" > /dev/null 2>&1; then
        echo "Error: Failed to start proxy"
        exit 1
    fi
fi

# Set Claude CLI environment
export ANTHROPIC_BASE_URL="http://127.0.0.1:${PORT}"
export ANTHROPIC_API_KEY="dummy"

# Change to workspace directory
cd "$WORKSPACE"

# Run Claude CLI with all arguments
exec claude "$@"
SCRIPT
RUN chmod +x /usr/local/bin/claude-run

# Create start-claude-proxy script for programmatic use (called by Yao)
# This reads proxy config from /workspace/.claude-proxy.json if exists
RUN cat > /usr/local/bin/start-claude-proxy << 'SCRIPT'
#!/bin/bash
# Start claude-proxy from config file or environment variables
# Config file: /workspace/.claude-proxy.json
# Format: {"backend": "...", "api_key": "...", "model": "..."}

CONFIG_FILE="${WORKSPACE:-/workspace}/.claude-proxy.json"
LOG_FILE="${WORKSPACE:-/workspace}/proxy.log"
PORT="${CLAUDE_PROXY_PORT:-3456}"

# Try to read from config file first
if [ -f "$CONFIG_FILE" ]; then
    BACKEND=$(jq -r '.backend // empty' "$CONFIG_FILE" 2>/dev/null)
    API_KEY=$(jq -r '.api_key // empty' "$CONFIG_FILE" 2>/dev/null)
    MODEL=$(jq -r '.model // empty' "$CONFIG_FILE" 2>/dev/null)
    
    if [ -n "$BACKEND" ]; then
        export CLAUDE_PROXY_BACKEND="$BACKEND"
    fi
    if [ -n "$API_KEY" ]; then
        export CLAUDE_PROXY_API_KEY="$API_KEY"
    fi
    if [ -n "$MODEL" ]; then
        export CLAUDE_PROXY_MODEL="$MODEL"
    fi
fi

# Check if we have the required config
if [ -z "$CLAUDE_PROXY_BACKEND" ] || [ -z "$CLAUDE_PROXY_API_KEY" ] || [ -z "$CLAUDE_PROXY_MODEL" ]; then
    echo "Error: Missing proxy configuration"
    echo "Either set environment variables or create $CONFIG_FILE"
    exit 1
fi

# Check if already running
if curl -s "http://127.0.0.1:${PORT}/health" > /dev/null 2>&1; then
    echo "claude-proxy already running"
    exit 0
fi

# Start proxy
mkdir -p "$(dirname "$LOG_FILE")" 2>/dev/null || true
nohup /usr/local/bin/claude-proxy -v -l "$LOG_FILE" > /dev/null 2>&1 &

# Wait for startup
for i in {1..20}; do
    if curl -s "http://127.0.0.1:${PORT}/health" > /dev/null 2>&1; then
        echo "claude-proxy started on port $PORT"
        exit 0
    fi
    sleep 0.5
done

echo "Error: claude-proxy failed to start"
exit 1
SCRIPT
RUN chmod +x /usr/local/bin/start-claude-proxy

# Create entrypoint script
# Note: claude-proxy is started on-demand by Yao (via start-claude-proxy)
# or manually by user (via claude-run)
RUN cat > /usr/local/bin/entrypoint.sh << 'SCRIPT'
#!/bin/bash
# Container entrypoint
# claude-proxy is NOT auto-started here - it's started by:
# 1. Yao's sandbox executor (writes config to .claude-proxy.json, calls start-claude-proxy)
# 2. Manual usage via claude-run command
# 3. Direct invocation of start-claude-proxy

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
SCRIPT
RUN chmod +x /usr/local/bin/entrypoint.sh

# Create a wrapper that sources the env file
RUN cat > /usr/local/bin/claude-env << 'SCRIPT'
#!/bin/bash
# Source claude-proxy environment if available
if [ -f /tmp/claude-proxy-env ]; then
    source /tmp/claude-proxy-env
fi
exec "$@"
SCRIPT
RUN chmod +x /usr/local/bin/claude-env

# Add sourcing to global bashrc so docker exec gets the vars
RUN echo '[ -f /tmp/claude-proxy-env ] && source /tmp/claude-proxy-env' >> /etc/bash.bashrc

USER sandbox

# Verify installations
RUN node --version && npm --version && python3 --version && \
    claude --version || true && \
    claude-proxy --help 2>&1 | head -1 || true

WORKDIR /workspace

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["sleep", "infinity"]
