# Claude sandbox image: Claude CLI + Node.js + Python + CCR
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

# Install Claude Code Router (CCR) for third-party LLM support
# Supports: DeepSeek, GLM, Volcengine, OpenRouter, etc.
RUN npm install -g @musistudio/claude-code-router

# Create CCR config directory
RUN mkdir -p /home/sandbox/.claude-code-router

# Create entrypoint script for CCR daemon mode
USER root
RUN cat > /usr/local/bin/ccr-run << 'SCRIPT'
#!/bin/bash
# CCR wrapper: starts CCR daemon and runs ccr code
# Usage: ccr-run "prompt" or ccr-run -c /path/to/config.json "prompt"

CONFIG=""
while [[ $# -gt 0 ]]; do
    case $1 in
        -c|--config)
            CONFIG="$2"
            shift 2
            ;;
        *)
            break
            ;;
    esac
done

# Apply config if provided
if [ -n "$CONFIG" ] && [ -f "$CONFIG" ]; then
    cp "$CONFIG" ~/.claude-code-router/config.json
fi

# Start CCR daemon (nohup because ccr start -d doesn't work in containers)
nohup ccr start >/dev/null 2>&1 &
sleep 2

# Run ccr code
exec ccr code -p "$*"
SCRIPT
RUN chmod +x /usr/local/bin/ccr-run

USER sandbox

# Verify installations
RUN node --version && npm --version && python3 --version && \
    claude --version || true && \
    ccr --version || true

WORKDIR /workspace

CMD ["sleep", "infinity"]
