# Agent Sandbox Implementation Plan

## Overview

This plan covers the implementation of the agent sandbox integration layer (`agent/sandbox/`), which enables coding agents (Claude CLI, Cursor CLI) to run in isolated Docker containers with Yao's LLM pipeline.

## Test Environment

### Environment Configuration

Tests should run with the local development environment:

```bash
# Source environment variables
source /Users/max/Yao/yao/env.local.sh

# Key variables used:
# YAO_TEST_APPLICATION=/Users/max/Yao/yao-dev-app
# YAO_ROOT=$YAO_TEST_APPLICATION
# DEEPSEEK_API_KEY, DEEPSEEK_API_PROXY, DEEPSEEK_MODELS_V3
```

### Test Application

Test assistants at `yao-dev-app/assistants/tests/sandbox/`:

```
yao-dev-app/assistants/tests/
└── sandbox/
    ├── basic/                    # Basic sandbox execution test
    │   ├── package.yao           # uses.search: disabled
    │   └── prompts.yml
    ├── hooks/                    # Hook integration test
    │   ├── package.yao           # uses.search: disabled
    │   ├── prompts.yml
    │   └── src/index.ts
    └── full/                     # Full test with MCPs, Skills, Hooks
        ├── package.yao           # uses.search: disabled, mcp: {servers: [...]}
        ├── prompts.yml
        ├── src/index.ts
        └── skills/echo-test/     # Agent Skills standard
            ├── SKILL.md
            └── scripts/echo.sh
```

### Connector Configuration

Use `deepseek.v3` as the default connector (via Volcengine API).

## Implementation Status

### Phase 1: Core Types and Interfaces ✅ COMPLETED

- [x] Define `Executor` interface with all methods
- [x] Define `Options` struct with JSON tags
- [x] Define `FileInfo` alias to infrastructure sandbox
- [x] Add `DefaultImage()` and `IsValidCommand()` helpers

### Phase 2: Claude Executor Implementation ✅ COMPLETED

- [x] Implement `Executor` struct
- [x] Implement `NewExecutor()` constructor with container reuse
- [x] Implement `Stream()` method with CCR config writing
- [x] Implement `Execute()` method (wrapper)
- [x] Implement `Close()` method (removes container)
- [x] Implement filesystem methods: `ReadFile`, `WriteFile`, `ListDir`
- [x] Implement `Exec()` method
- [x] Implement `GetWorkDir()` method

### Phase 3: CCR Configuration ✅ COMPLETED

- [x] Implement `BuildCCRConfig()` with correct CCR format
- [x] Auto-detect provider type (volcengine, deepseek, openai, claude)
- [x] Add transformer for DeepSeek/Volcengine (maxtoken)
- [x] Generate Router configuration
- [x] Write config to container before execution

### Phase 4: Assistant Integration ✅ COMPLETED

- [x] Implement `GetSandboxManager()` singleton
- [x] Implement `HasSandbox()` method
- [x] Implement `initSandbox()` with cleanup function
- [x] Implement `executeSandboxStream()` method
- [x] Build executor options from assistant config
- [x] Resolve connector settings (host, key, model)
- [x] Add trace logging for sandbox creation
- [x] Send loading message during sandbox init
- [x] Expose executor to hooks via `ctx.SetSandboxExecutor()`
- [x] Handle sandbox lifecycle (create → hooks → execute → cleanup)

### Phase 5: JSAPI Integration ✅ COMPLETED

- [x] Define `SandboxExecutor` interface
- [x] Implement JS bindings for `ReadFile`, `WriteFile`, `ListDir`, `Exec`
- [x] Expose `workdir` property
- [x] Register in context's `NewObject` method

### Phase 6: Concurrency & Resource Management ✅ COMPLETED

- [x] Container creation uses Double-Check Locking (in `manager.GetOrCreate`)
- [x] Same chatID reuses container (by design)
- [x] Container cleanup on request completion (`defer sandboxCleanup()`)
- [x] Unique chatID in tests to avoid conflicts

### Phase 7: MCP & Skills Integration ✅ COMPLETED

- [x] Build MCP config from assistant's `mcp.servers` configuration
- [x] Write MCP config to container workspace (`.mcp.json`)
- [x] Resolve skills directory from `assistants/{name}/skills/`
- [x] Copy skills to container (`/workspace/.claude/skills/`)
- [x] Skip MCP tool execution in `agent.go` for sandbox mode (Claude CLI handles internally)
- [x] Add unit tests for MCP config building (`TestBuildMCPConfigForSandbox`)
- [x] Add unit tests for skills directory resolution (`TestSandboxMCPAndSkillsOptions`)

### Phase 8: MCP IPC Bridge ✅ COMPLETED

- [x] Modify `BuildMCPConfigForSandbox` to use `yao-bridge` command for IPC
- [x] Create IPC session in `sandbox/manager.createContainer()` (socket created before container)
- [x] Bind mount IPC socket to container at `/tmp/yao.sock`
- [x] Add `SetMCPTools()` method to `ipc.Session` for runtime tool configuration
- [x] Set MCP tools dynamically in `claude.Executor.Stream()` before execution
- [x] IPC session lifecycle managed by `sandbox.Manager` (create on container create, close on remove)
- [x] Load MCP tool definitions from gou/mcp and pass to IPC session
- [x] Add `TestClaudeExecutorIPCSocketMount` to verify socket bind mount
- [x] Verify E2E test shows "Loaded X MCP tools for IPC"

### Phase 9: Workspace Management ⏳ PENDING

- [ ] Implement workspace cleanup configuration
- [ ] Implement stale workspace detection
- [ ] Implement cleanup scheduler

### Phase 9: Cursor Placeholder ⏳ PENDING

- [ ] Create `cursor/README.md` placeholder

## Testing Status

### Unit Tests

| Package | Test File | Status |
|---------|-----------|--------|
| `agent/sandbox` | `types_test.go` | ✅ PASS |
| `agent/sandbox` | `executor_test.go` | ✅ PASS |
| `agent/sandbox/claude` | `command_test.go` | ✅ PASS |
| `agent/sandbox/claude` | `executor_test.go` | ✅ PASS |

### Integration Tests

| Package | Test File | Status |
|---------|-----------|--------|
| `agent/sandbox` | `integration_test.go` | ✅ PASS |

### JSAPI Tests

| Package | Test File | Status |
|---------|-----------|--------|
| `agent/context` | `jsapi_sandbox_test.go` | ✅ PASS |

### Assistant Loading Tests

| Package | Test File | Status |
|---------|-----------|--------|
| `agent/assistant` | `sandbox_test.go` | ✅ PASS |
| `agent/assistant` | `sandbox_integration_test.go` | ✅ PASS |

### E2E Tests

| Package | Test Case | Status |
|---------|-----------|--------|
| `agent/assistant` | `TestSandboxBasicE2E` | ✅ PASS |
| `agent/assistant` | `TestSandboxHooksE2E` | ✅ PASS |
| `agent/assistant` | `TestSandboxFullE2E` | ✅ PASS |
| `agent/assistant` | `TestSandboxContextAccess` | ✅ PASS |
| `agent/assistant` | `TestSandboxLoadConfiguration` | ✅ PASS |
| `agent/assistant` | `TestSandboxMCPToolCall` | ✅ PASS |
| `agent/assistant` | `TestSandboxMCPEchoTool` | ✅ PASS |

### Running Tests

```bash
# Source environment
source /Users/max/Yao/yao/env.local.sh

# Run all sandbox tests
go test -v ./agent/sandbox/...

# Run assistant sandbox tests
go test -v ./agent/assistant -run "Sandbox"

# Run E2E tests (requires Docker)
go test -v ./agent/assistant -run "TestSandbox.*E2E" -timeout 300s
```

## File Structure

```
yao/agent/sandbox/                    # Executor layer
├── DESIGN.md                         # ✅ Design document
├── PLAN.md                           # ✅ This file
├── types.go                          # ✅ Common types and interfaces
├── types_test.go                     # ✅ Types tests
├── executor.go                       # ✅ Factory function
├── executor_test.go                  # ✅ Factory tests
├── integration_test.go               # ✅ Integration tests
├── claude/
│   ├── types.go                      # ✅ Claude-specific types
│   ├── executor.go                   # ✅ Executor implementation
│   ├── executor_test.go              # ✅ Executor tests
│   ├── command.go                    # ✅ Command builder + CCR config
│   └── command_test.go               # ✅ Command tests
└── cursor/
    └── README.md                     # ⏳ Placeholder (pending)

yao/agent/assistant/                  # Integration layer
├── sandbox.go                        # ✅ Sandbox handler
├── sandbox_test.go                   # ✅ Loading tests
├── sandbox_integration_test.go       # ✅ Integration tests
├── sandbox_e2e_test.go               # ✅ E2E tests
├── sandbox_debug_test.go             # ✅ Debug tests
└── agent.go                          # ✅ Modified: sandbox detection in Stream()

yao/agent/context/                    # Context layer
├── jsapi_sandbox.go                  # ✅ Sandbox JSAPI bindings
└── jsapi_sandbox_test.go             # ✅ Sandbox JSAPI tests

yao-dev-app/assistants/tests/sandbox/ # Test assistants
├── basic/                            # ✅ Basic sandbox test
├── hooks/                            # ✅ Hooks test
└── full/                             # ✅ Full test with MCPs and Skills
```

## Key Design Decisions

### 1. Container Reuse

Same `userID + chatID` reuses the same container:
- Workspace directory persists across requests
- CCR config is written on each request (same content, safe to overwrite)
- Container is removed when request completes

### 2. Concurrency

- Container creation: Protected by mutex + double-check locking
- Container execution: Multiple requests can run concurrently in same container
- Claude CLI: Supports concurrent execution

### 3. CCR Configuration

CCR requires specific JSON format:
```json
{
  "Providers": [{"name": "volcengine", "api_base_url": "...", ...}],
  "Router": {"default": "volcengine,model", ...}
}
```

Auto-detection of provider type based on host URL.

### 4. Resource Cleanup

- `executor.Close()` removes the container and closes IPC session
- `defer sandboxCleanup()` in `agent.go` ensures cleanup
- Tests use unique chatID (timestamp) to avoid conflicts

### 5. MCP IPC Architecture

```
Host (Yao)                              Container (Claude CLI)
┌────────────────────────┐              ┌────────────────────────┐
│ IPC Manager            │              │ yao-bridge             │
│   └─ Session           │◄─────────────│   (stdio ↔ socket)     │
│       └─ MCPTools      │  Unix Socket │                        │
│           └─ Process   │   (/tmp/     │ Claude CLI reads       │
│              executor  │    yao.sock) │ .mcp.json and calls    │
└────────────────────────┘              │ yao-bridge for tools   │
                                        └────────────────────────┘
```

- `.mcp.json` points to single "yao" server using `yao-bridge /tmp/yao.sock`
- IPC session created with authorized MCP tools from assistant config
- Tools executed via `process.New()` in IPC session handler

## Known Issues

### macOS Docker Desktop Socket Permissions

On macOS with Docker Desktop (gRPC-FUSE), Unix socket permissions are not properly preserved when bind mounting from the host. The IPC socket created on the host with `0666` permissions appears as `0660` inside the container.

**Solution**: After container start, we execute `chmod 666 /tmp/yao.sock` as root inside the container to fix permissions. This is handled automatically by `sandbox.Manager.fixIPCSocketPermissions()`.

## Notes

- All tests validate return values (use `require`/`assert`)
- Docker must be available for integration and E2E tests
- Tests automatically clean up containers after completion
- Use `uses.search: disabled` in test assistants to avoid auto-search LLM calls
