# Claude sandbox image: Claude CLI + Node.js + Python
# Supports both amd64 and arm64 architectures
ARG REGISTRY=yaoapp
FROM ${REGISTRY}/sandbox-base:latest

USER root

# Node.js 20 (automatically detects architecture)
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/*

# Python 3.11
RUN apt-get update && apt-get install -y \
    python3.11 \
    python3-pip \
    python3-venv \
    && rm -rf /var/lib/apt/lists/* \
    && ln -sf /usr/bin/python3.11 /usr/bin/python3 \
    && ln -sf /usr/bin/python3 /usr/bin/python

# Claude CLI installation
# Using npm to install @anthropic-ai/claude-code globally
RUN npm install -g @anthropic-ai/claude-code || \
    echo "Claude CLI installation skipped (may not be available yet)"

# npm global packages directory for sandbox user
RUN mkdir -p /home/sandbox/.npm-global && \
    chown -R sandbox:sandbox /home/sandbox/.npm-global

USER sandbox

# Configure npm to use user directory
RUN npm config set prefix '/home/sandbox/.npm-global'
ENV PATH="/home/sandbox/.npm-global/bin:${PATH}"

# Verify installations
RUN node --version && npm --version && python3 --version

WORKDIR /workspace

CMD ["sleep", "infinity"]
