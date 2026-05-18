#!/bin/bash
set -euo pipefail

echo "=== Agent test setup ==="

# Build ci-token tool
echo "→ Building ci-token..."
cd /workspace/yao
go build -tags ci -v -o /tmp/ci-token ./cmd/ci-token 2>&1

# Generate tunnel tokens for each Tai instance
SHARED=/shared
mkdir -p "$SHARED"

for instance in tai-docker tai-k8s tai-hostexec; do
    echo "→ Generating token for $instance..."
    /tmp/ci-token \
        -subject "${YAO_CI_OAUTH_SUBJECT:-ci-test-user}" \
        -user    "${YAO_CI_OAUTH_USER_ID:-ci-test-user}" \
        -team    "${YAO_CI_OAUTH_TEAM_ID:-ci-test-team}" \
        -scope   "${YAO_CI_OAUTH_SCOPE:-tai:tunnel}" \
        -ttl     "${YAO_CI_OAUTH_TTL:-24h}" \
        > "$SHARED/${instance}.token" 2>&1 || true
done

echo "=== Setup complete ==="
ls -la "$SHARED/"
