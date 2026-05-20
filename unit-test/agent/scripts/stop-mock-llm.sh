#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/_common.sh"

load_env

log_info "=== Stopping Mock LLM Server ==="

kill_by_pidfile "$DATA_DIR/pids/mock-llm.pid"
kill_by_port "$MOCK_LLM_PORT"

log_info "Mock LLM server stopped."
