#!/bin/bash
set -euo pipefail

MAX_WAIT=${MAX_WAIT:-120}
INTERVAL=3
elapsed=0

check_service() {
    local name="$1" url="$2"
    if wget -q --spider --timeout=2 "$url" 2>/dev/null; then
        echo "  ✓ $name"
        return 0
    fi
    return 1
}

echo "Waiting for services (timeout: ${MAX_WAIT}s)..."

while [ $elapsed -lt $MAX_WAIT ]; do
    all_ready=true

    check_service "mock-llm" "http://mock-llm:9999/healthz" || all_ready=false
    check_service "yao-server" "http://yao-server:5099/api/__yao/app/setting" || all_ready=false

    if [ "$all_ready" = true ]; then
        echo "All services ready (${elapsed}s)"
        exit 0
    fi

    sleep $INTERVAL
    elapsed=$((elapsed + INTERVAL))
done

echo "TIMEOUT: services not ready after ${MAX_WAIT}s"
exit 1
