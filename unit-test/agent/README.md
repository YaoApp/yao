# Agent Test Framework

## Architecture Overview

The agent test framework uses a **single unified environment** managed by the Go `testprepare` package. There are no external Yao or Tai servers to start — everything runs in-process within the Go test binary.

```
External (manually started)         In-process (testprepare)
┌──────────────────────┐            ┌──────────────────────────────────┐
│  Mock LLM Server     │            │  loadAgentTestEnv()              │
│  (:6920, standalone) │◄──conn────►│    Parse agent-test.env          │
└──────────────────────┘            │    Generate app/.env             │
                                    │    os.Setenv (unconditional)     │
┌──────────────────────┐            │                                  │
│  PostgreSQL          │            │  config.Init()                   │
│  (optional, CI)      │◄──────────►│    godotenv.Overload(app/.env)   │
└──────────────────────┘            │                                  │
                                    │  loadApp()                       │
                                    │    DB + models + scripts + V8    │
                                    │    Agent stack + connectors      │
                                    │    setup.Initialize              │
                                    │    SetupTestUsers                │
                                    │    gRPC server (random port)     │
                                    │                                  │
                                    │  InitStack()                     │
                                    │    tai-docker (child, random)    │
                                    │    tai-hostexec (child, random)  │
                                    └──────────────────────────────────┘
```

### External Dependencies

| Service        | Required? | Notes                                           |
| -------------- | --------- | ----------------------------------------------- |
| Mock LLM       | Yes       | Provides mock OpenAI/Anthropic APIs on `:6920`  |
| PostgreSQL     | Optional  | sqlite3 by default; set in `.env.local` for PG  |

### Network Configuration (Required)

Docker containers reach the host via `host.tai.internal` (injected by Tai at container creation via `ExtraHosts: host.tai.internal:host-gateway`). HostExec sub-processes run on the host OS directly. To unify addressing, `MOCK_LLM_HOST` uses `host.tai.internal` and the host machine must also resolve this name.

**Add to your hosts file (one-time setup):**

Linux / macOS (`/etc/hosts`):
```bash
sudo sh -c 'echo "127.0.0.1 host.tai.internal" >> /etc/hosts'
```

Windows (`%SystemRoot%\System32\drivers\etc\hosts`):
```powershell
Add-Content -Path "$env:SystemRoot\System32\drivers\etc\hosts" -Value "`n127.0.0.1`thost.tai.internal" -Force
```

CI workflows add this entry automatically. If missing, `PrepareE2E` will fail with a clear message showing the command to run.

### Test Levels

| Function                     | What it does                                           |
| ---------------------------- | ------------------------------------------------------ |
| `testprepare.PrepareUnit(t)` | Load env + validate paths only                         |
| `testprepare.PrepareSandbox(t)` | Full Yao Runtime + Tai nodes (in-process)           |
| `testprepare.PrepareE2E(t)`  | PrepareSandbox + verify LLM availability               |

## Configuration

### Single Source of Truth

`unit-test/agent/env/agent-test.env` is the authoritative configuration file.

**Flow:**
1. `testprepare.loadAgentTestEnv()` reads `agent-test.env`
2. Overlays `env/.env.local` (if exists) for local API key overrides
3. Writes all key=val pairs to `app/.env` (except `TEST_*`/`SANDBOX_TEST_*`)
4. Sets all variables via `os.Setenv` (unconditional — env file always wins)
5. `config.Init()` loads `app/.env` via `godotenv.Overload` — this is the authoritative source for Yao runtime
6. Connectors resolve `$ENV.MOCK_LLM_HOST`, `$ENV.OPENAI_API_KEY`, etc. from `app/.env`

### Local Overrides

Create `unit-test/agent/env/.env.local` (gitignored) to override values without modifying `agent-test.env`:

```properties
# Use PostgreSQL locally
YAO_DB_DRIVER=postgres
YAO_DB_PRIMARY=postgres://yao:yao@127.0.0.1:5432/agent_test?sslmode=disable

# Real API keys for E2E tests
OPENAI_API_KEY=sk-...
ANTHROPIC_API_KEY=sk-ant-...
```

### Connector LLM Matrix

The test application (`unit-test/agent/app/`) includes connectors that reference environment variables:

| Connector Pattern          | Variable Referenced         |
| -------------------------- | --------------------------- |
| `mock-*.conn.yao`          | `$ENV.MOCK_LLM_HOST`       |
| `openai-*.conn.yao`        | `$ENV.OPENAI_API_KEY`      |
| `anthropic-*.conn.yao`     | `$ENV.ANTHROPIC_API_KEY`   |
| `deepseek-*.conn.yao`      | `$ENV.DEEPSEEK_V4_API_KEY` |

All these variables are written to `app/.env` so Yao can resolve them at runtime.

## Quick Start

### 1. Start Mock LLM

```bash
bin/test-agent start
```

### 2. Run Tests

```bash
# Run all tests
bin/test-agent run

# Run by tier
bin/test-agent run --tier unit      # Pure unit tests (fast, no deps)
bin/test-agent run --tier sandbox   # Sandbox tests (needs Docker)
bin/test-agent run --tier e2e       # E2E tests (needs mock-llm + Docker)

# Run specific package/function
bin/test-agent run --pkg ./agent/llm/... --run TestStream

# Verbose with custom timeout
bin/test-agent run --tier sandbox --verbose --timeout 300s
```

### 3. Stop

```bash
bin/test-agent stop
```

## Test Tiers

Tests are organized into three tiers based on their dependencies:

### Unit (no external deps)

Pure logic tests. No Yao Runtime, no Docker, no LLM.

| Package | Tests |
|---------|-------|
| `sandbox/v2` | ExecOption, AttachOption, LifecyclePolicy, Watcher, Box/Host/Manager unit tests, BuildGRPCEnv, Errors |
| `agent/sandbox/v2` | ParseMemory, BuildCreateOptions, ShellWrap, Runners, BuildIdentifier, CacheKey, Token, Stream |
| `agent/sandbox/v2/opencode` | command, platform, parse, roles, config tests |
| `agent/sandbox/v2/claude` | command, platform, parse tests |
| `agent/sandbox/v2/shared` | InjectSystemSkills, linereader tests |
| `agent/sandbox/v2/types` | RoleConnector, SandboxConfig tests |

### Sandbox (needs Docker daemon)

Full Yao Runtime + Tai nodes running in-process. Requires Docker.

| Package | Tests |
|---------|-------|
| `sandbox/v2` | Sandbox_Init, Image_*, Workspace_*, Host_Exec/Stream, Box_Exec/Stream/Info, Manager_Create/Get/List/Remove |
| `sandbox/v2/jsapi` | JSAPI_Create/Get/Delete/List/Exec/Stream/ComputerInfo/Nodes |
| `agent/sandbox/v2` | GetComputer_*, LifecycleAction_*, RunPrepareSteps_* |

**Image requirement:** Sandbox tests use `yaoapp/tai-sandbox-base:latest` (configured via `SANDBOX_TEST_IMAGE`). This image contains the `sandbox` user (UID 1000) required by Tai.

### E2E (needs Mock LLM + Docker)

Full Runtime + Tai + LLM. The Mock LLM server must be running.

| Package | Tests |
|---------|-------|
| `agent/sandbox/v2/yaocode` | TestSandboxV2_Yao_JSAPI |
| `agent/sandbox/v2/opencode` | TestOpenCode_Oneshot, TestOpenCode_Session |
| `agent/sandbox/v2/claude` | TestSandboxV2_Claude_E2E, Claude_Attachments, Claude_ToolCallE2E |

**Networking:** E2E tests run CLI tools (opencode, claude) inside Docker containers. `MOCK_LLM_HOST` is set to `http://host.tai.internal:6920` so the same URL works inside containers (via Tai `ExtraHosts`) and on the host (via hosts file entry). See "Network Configuration" above.

## Verification Checklist

### V1: Environment Isolation

Verify that external shell environment variables do NOT pollute the test environment.

```bash
# Set a conflicting variable
export YAO_DB_DRIVER=mysql

# Start mock-llm and run tests
bin/test-agent start
bin/test-agent run --tier unit --verbose

# Tests should use sqlite3 (from agent-test.env), NOT mysql
# Check generated app/.env:
cat unit-test/agent/app/.env | grep YAO_DB_DRIVER
# Expected: YAO_DB_DRIVER=sqlite3

unset YAO_DB_DRIVER
```

### V2: app/.env Correctness

After running any test, inspect `unit-test/agent/app/.env`:

```bash
# Should contain ALL of these keys (values may be empty for API keys):
grep -E '^(YAO_DB_DRIVER|MOCK_LLM_HOST|OPENAI_API_KEY|ANTHROPIC_API_KEY|DEEPSEEK_V4_API_KEY|SERPAPI_API_KEY)=' \
  unit-test/agent/app/.env
```

Expected output:
```
YAO_DB_DRIVER=sqlite3
MOCK_LLM_HOST=http://host.tai.internal:6920
OPENAI_API_KEY=
ANTHROPIC_API_KEY=
DEEPSEEK_V4_API_KEY=
SERPAPI_API_KEY=
```

### V3: sqlite3 Mode (Default)

```bash
bin/test-agent start
bin/test-agent run --verbose
# Tests should pass with sqlite3 database
bin/test-agent stop
```

### V4: PostgreSQL Mode

```bash
# Create .env.local with PG config
cat > unit-test/agent/env/.env.local <<'EOF'
YAO_DB_DRIVER=postgres
YAO_DB_PRIMARY=postgres://yao:yao@127.0.0.1:5432/agent_test?sslmode=disable
EOF

bin/test-agent start
bin/test-agent run --verbose

# Verify PG was used:
cat unit-test/agent/app/.env | grep YAO_DB_DRIVER
# Expected: YAO_DB_DRIVER=postgres

# Clean up
rm unit-test/agent/env/.env.local
bin/test-agent stop
```

### V5: Mock LLM Integration

```bash
bin/test-agent start

# Verify mock-llm is reachable (via localhost or host.tai.internal)
curl -s http://host.tai.internal:6920/healthz
# Expected: {"status":"ok"} or 200 response

# Run E2E tests (they will use mock-llm)
bin/test-agent run --tier e2e --verbose

bin/test-agent stop
```

### V6: No External Yao/Tai Needed

Verify that there are no leftover processes on ports 6099 (Yao HTTP), 6199 (Yao gRPC), 6100 (Tai Docker), 6110 (Tai HostExec) after starting tests. The gRPC and Tai services run in-process on random ports.

```bash
# After running tests, check that no services are left on fixed ports
lsof -i :6099 -i :6199 -i :6100 -i :6110
# Expected: no output (or only unrelated processes)
```

## File Structure

```
unit-test/agent/
├── env/
│   ├── agent-test.env           # Configuration source of truth
│   ├── agent-test.env.template  # Template for new setups
│   └── .env.local               # Local overrides (gitignored)
├── app/                         # Test Yao application
│   ├── app.yao                  # Application manifest
│   ├── .env                     # GENERATED by testprepare (do not edit)
│   ├── connectors/              # LLM connector definitions
│   ├── models/                  # Data models
│   ├── scripts/                 # JS scripts (setup.ts, etc.)
│   └── assistants/              # Agent/assistant definitions
├── mock-llm/                    # Mock LLM server source
├── scripts/
│   ├── _common.sh               # Shared utilities
│   ├── start-mock-llm.sh        # Start mock LLM server
│   └── stop-mock-llm.sh         # Stop mock LLM server
├── testprepare/                 # Go test preparation package
│   ├── prepare.go               # Entry points: PrepareUnit/Sandbox/E2E
│   ├── apploader.go             # In-process Yao Runtime loader
│   └── sandboxtest/             # Tai sandbox test utilities
└── README.md                    # This file
```

## Removed Components

The following files were removed as part of the framework redesign. Their functionality is now handled by `testprepare` in-process:

- `scripts/start-yao-server.sh` — Yao server now runs in-process via `loadApp()`
- `scripts/start-tai.sh` — Tai nodes now launched as child processes via `sandboxtest.InitStack()`
- `scripts/check-env.sh` — Environment validation done in `loadAgentTestEnv()`
- `scripts/stop-all.sh` — Replaced by `stop-mock-llm.sh` (only external process)
- `agent/env_smoke_test.go` — External service health checks no longer needed
