#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/_common.sh"

load_env

log_info "=== Starting Mock LLM Server ==="

kill_by_port "$MOCK_LLM_PORT"
kill_by_pidfile "$DATA_DIR/pids/mock-llm.pid"

mkdir -p "$BUILD_DIR" "$DATA_DIR/pids"

EXE_EXT=""
[ "$OS" = "windows" ] && EXE_EXT=".exe"

log_info "Building mock-llm..."
(cd "$YAO_SRC/unit-test/agent/mock-llm" && CGO_ENABLED=0 go build -o "$BUILD_DIR/mock-llm${EXE_EXT}" .)

FIXTURES_DIR="$YAO_SRC/unit-test/agent/mock-llm/fixtures"
log_info "Starting mock-llm on port $MOCK_LLM_PORT..."
"$BUILD_DIR/mock-llm${EXE_EXT}" -port "$MOCK_LLM_PORT" -fixtures "$FIXTURES_DIR" \
  > "$DATA_DIR/mock-llm.log" 2>&1 &
echo $! > "$DATA_DIR/pids/mock-llm.pid"

wait_for_url "http://127.0.0.1:${MOCK_LLM_PORT}/healthz" "mock-llm" 15

log_info "Mock LLM server started (PID $(cat "$DATA_DIR/pids/mock-llm.pid"))"
