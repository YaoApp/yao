# Sandbox V2 base image — self-contained, no dependency on V1 sandbox-base
FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive

# Faster mirror for ARM64
RUN sed -i 's|http://ports.ubuntu.com/ubuntu-ports|http://mirrors.mit.edu/ubuntu-ports|g' /etc/apt/sources.list.d/ubuntu.sources 2>/dev/null || \
    sed -i 's|http://ports.ubuntu.com/ubuntu-ports|http://mirrors.mit.edu/ubuntu-ports|g' /etc/apt/sources.list 2>/dev/null || true

RUN apt-get update && apt-get install -y --no-install-recommends \
    curl wget git ca-certificates gnupg lsb-release jq \
    vim less tree \
    iputils-ping net-tools dnsutils telnet netcat-openbsd \
    zip unzip tar gzip \
    htop procps \
    sed gawk grep \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /workspace

RUN useradd -m -s /bin/bash sandbox && \
    chown -R sandbox:sandbox /workspace

# yao-grpc binary (replaces yao-bridge from V1)
ARG TARGETARCH
COPY yao-grpc-${TARGETARCH} /usr/local/bin/yao-grpc
RUN chmod +x /usr/local/bin/yao-grpc

# claude-proxy binary
COPY claude-proxy-${TARGETARCH} /usr/local/bin/claude-proxy
RUN chmod +x /usr/local/bin/claude-proxy

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

USER sandbox
ENTRYPOINT ["/entrypoint.sh"]
CMD ["sleep", "infinity"]
