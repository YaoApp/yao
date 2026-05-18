#!/usr/bin/env bash
# Shared utilities for agent test scripts.
#
# This file provides:
#   - Path resolution (SCRIPT_DIR, AGENT_DIR, YAO_SRC, etc.)
#   - Logging helpers
#   - Process management (kill_by_pidfile, kill_by_port, wait_for_url)
#   - load_env()          -- parse agent-test.env unconditionally (env file wins)
#   - generate_yao_env()  -- write app/.env from agent-test.env for Yao runtime

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AGENT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
YAO_SRC="$(cd "$AGENT_DIR/../.." && pwd)"

detect_os() {
  case "$(uname -s)" in
    Darwin)  echo "darwin" ;;
    Linux)   echo "linux" ;;
    MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
    *)       echo "unknown" ;;
  esac
}

OS="$(detect_os)"

log_info()  { echo "[INFO]  $(date '+%H:%M:%S') $*"; }
log_warn()  { echo "[WARN]  $(date '+%H:%M:%S') $*" >&2; }
log_error() { echo "[ERROR] $(date '+%H:%M:%S') $*" >&2; }
log_fatal() { log_error "$@"; exit 1; }

kill_by_pidfile() {
  local pidfile="$1"
  [ -f "$pidfile" ] || return 0
  local pid
  pid=$(cat "$pidfile" 2>/dev/null) || return 0
  [ -z "$pid" ] && return 0
  if kill -0 "$pid" 2>/dev/null; then
    log_info "Killing process $pid (from $pidfile)"
    kill "$pid" 2>/dev/null || true
    sleep 1
    kill -0 "$pid" 2>/dev/null && kill -9 "$pid" 2>/dev/null || true
  fi
  rm -f "$pidfile"
}

kill_by_port() {
  local port="$1"
  case "$OS" in
    darwin|linux)
      local pids
      pids=$(lsof -ti :"$port" 2>/dev/null) || true
      if [ -n "$pids" ]; then
        log_info "Killing processes on port $port: $pids"
        echo "$pids" | xargs kill -9 2>/dev/null || true
        sleep 1
      fi
      ;;
    windows)
      local pids
      pids=$(netstat -ano 2>/dev/null | grep ":${port} " | awk '{print $NF}' | sort -u) || true
      for pid in $pids; do
        [ "$pid" = "0" ] && continue
        log_info "Killing process $pid on port $port (Windows)"
        taskkill //F //PID "$pid" 2>/dev/null || true
      done
      ;;
  esac
}

wait_for_url() {
  local url="$1" name="$2" timeout="${3:-30}"
  log_info "Waiting for $name ($url)..."
  for i in $(seq 1 "$timeout"); do
    if curl -sf "$url" > /dev/null 2>&1; then
      log_info "$name is ready (${i}s)"
      return 0
    fi
    sleep 1
  done
  log_fatal "$name failed to start within ${timeout}s"
}

# ---------------------------------------------------------------------------
# load_env -- parse agent-test.env, then .env.local overlay
#
# All variables are exported UNCONDITIONALLY (env file always wins over
# any inherited shell environment). .env.local is loaded AFTER so it can
# override keys like API keys for local development.
# ---------------------------------------------------------------------------
load_env() {
  local env_file="$AGENT_DIR/env/agent-test.env"

  if [ ! -f "$env_file" ]; then
    log_fatal "Config file not found: $env_file"
  fi

  while IFS= read -r line || [ -n "$line" ]; do
    line="${line%%#*}"
    line="$(echo "$line" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')"
    [ -z "$line" ] && continue
    [[ "$line" == *=* ]] || continue
    local key="${line%%=*}"
    local val="${line#*=}"
    export "$key=$val"
  done < "$env_file"

  # .env.local overlay (API keys, local DB overrides, etc.)
  if [ -f "$AGENT_DIR/env/.env.local" ]; then
    set -a
    source "$AGENT_DIR/env/.env.local"
    set +a
  fi

  # Derived paths (always computed, never from env)
  APP_DIR="$AGENT_DIR/app"
  BUILD_DIR="$YAO_SRC/.build/test"
  DATA_DIR="$BUILD_DIR/data"

  export APP_DIR BUILD_DIR DATA_DIR
  export YAO_SRC AGENT_DIR
}

# ---------------------------------------------------------------------------
# generate_yao_env -- write app/.env from agent-test.env
#
# Writes ALL key=val lines from agent-test.env into app/.env so that
# Yao's config.Init() -> godotenv.Overload() picks them up. This includes
# YAO_* variables, MOCK_LLM_HOST, and LLM API keys that connectors
# reference via $ENV.*.
#
# .env.local values are overlaid on top (already in the shell env after
# load_env), so the written file reflects the final merged state.
# ---------------------------------------------------------------------------
generate_yao_env() {
  local env_file="$AGENT_DIR/env/agent-test.env"
  local target="$APP_DIR/.env"

  log_info "Generating Yao .env at $target"
  : > "$target"

  while IFS= read -r line || [ -n "$line" ]; do
    line="${line%%#*}"
    line="$(echo "$line" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')"
    [ -z "$line" ] && continue
    [[ "$line" == *=* ]] || continue
    local key="${line%%=*}"
    # Skip test-orchestration-only variables
    [[ "$key" == TEST_* ]] && continue
    [[ "$key" == SANDBOX_TEST_* ]] && continue
    # Use the current env value (includes .env.local overlay from load_env)
    local val="${!key:-${line#*=}}"
    echo "${key}=${val}" >> "$target"
  done < "$env_file"
}
