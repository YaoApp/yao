# Sandbox Implementation Plan

## Overview

This document outlines the implementation plan for the Sandbox module, which provides persistent Docker containers for external CLI agents like Claude Code.

**Estimated Code**: ~1200 lines

---

## Phase 1: Core Interfaces & Types ✅ COMPLETED

### Goals

- Define all core types and interfaces
- Set up package structure

### Implemented Files

```
sandbox/
├── manager.go       # Manager implementation
├── types.go         # Container, ExecOptions, ExecResult, FileInfo
├── config.go        # Configuration types
├── errors.go        # Custom errors
├── helpers.go       # Helper functions
└── ipc/
    ├── manager.go   # IPC Manager
    ├── session.go   # IPC Session
    └── types.go     # JSON-RPC types
```

### Completed

- [x] Create package structure
- [x] Define types in `types.go`
  - `Config` struct
  - `Container` struct
  - `ExecOptions` struct
  - `ExecResult` struct
  - `FileInfo` struct
- [x] Define `Manager` in `manager.go`
  - Container lifecycle: `GetOrCreate`, `Stop`, `Start`, `Remove`, `List`, `Cleanup`
  - Command execution: `Stream`, `Exec`
  - Filesystem: `WriteFile`, `ReadFile`, `ListDir`, `Stat`, `MkDir`, `RemoveFile`, `CopyToContainer`, `CopyFromContainer`
- [x] Define IPC types in `ipc/`
  - `Session` struct
  - `Manager` struct
  - `AgentContext` struct
  - `MCPTool` struct
  - JSON-RPC request/response types
- [x] Unit tests for type helpers

---

## Phase 2: Docker Container Management ✅ COMPLETED

### Goals

- Implement container lifecycle management
- Handle container creation, start, stop, remove

### Completed

- [x] Initialize Docker client (`NewManager`)
- [x] Implement `createContainer()`
  - Generate container name: `yao-sandbox-{userID}-{chatID}`
  - Create workspace directory on host
  - Configure mounts (workspace, IPC socket)
  - Set resource limits (memory, CPU)
  - Apply security options (`--cap-drop ALL`, `no-new-privileges`)
- [x] Implement `GetOrCreate()` with double-check locking
- [x] Implement `ensureImage()` - auto-pull missing Docker images
- [x] Implement `ensureRunning()`
- [x] Implement `Stop()` and `Start()`
- [x] Implement `Remove()`
- [x] Implement `List()`
- [x] Concurrency limit (`ErrTooManyContainers`)

---

## Phase 3: Command Execution & Filesystem ✅ COMPLETED

### Goals

- Execute commands inside containers
- Support both streaming and blocking execution
- Full filesystem operations

### Completed

#### Command Execution

- [x] Implement `Stream()` - returns io.ReadCloser
- [x] Implement `Exec()` - blocking execution with result
- [x] Handle timeout via context
- [x] Handle environment variables

#### Filesystem Operations

- [x] `WriteFile()` - tar archive + CopyToContainer
- [x] `ReadFile()` - CopyFromContainer + extract tar
- [x] `ListDir()` - execute `ls -la` and parse
- [x] `Stat()` - execute `stat` and parse
- [x] `MkDir()` - execute `mkdir -p`
- [x] `RemoveFile()` - execute `rm -rf`
- [x] `CopyToContainer()` - tar + Docker API
- [x] `CopyFromContainer()` - Docker API + extract

---

## Phase 4: IPC System ✅ COMPLETED

### Goals

- Implement Unix socket IPC
- Handle MCP JSON-RPC protocol

### Completed

- [x] Implement `ipc.Manager`
  - `NewManager(sockDir string) *Manager`
  - `Create(ctx, sessionID, agentCtx, mcpTools) (*Session, error)`
  - `Close(sessionID) error`
  - `Get(sessionID) (*Session, bool)`
  - `CloseAll()`
- [x] Implement `ipc.Session`
  - Create Unix socket listener
  - Set socket permissions (0660)
  - Handle connection lifecycle
- [x] Implement message loop
  - Accept connection
  - Read NDJSON lines
  - Parse JSON-RPC requests
  - Dispatch to handlers
  - Write JSON-RPC responses
- [x] Implement MCP handlers
  - `initialize` → handshake response
  - `tools/list` → return authorized tools
  - `tools/call` → execute Yao process
  - `resources/list` → list resources
  - `resources/read` → read resource
- [x] JSON-RPC error handling with proper error codes

---

## Phase 5: yao-bridge & Docker Image ✅ COMPLETED

### Goals

- Build yao-bridge binary
- Create Docker images

### Implemented Files

```
sandbox/
├── bridge/
│   └── main.go                     # yao-bridge source
├── docker/
│   ├── base/
│   │   └── Dockerfile.base         # Common base image
│   ├── claude/
│   │   ├── Dockerfile              # Default: Claude + Node + Python
│   │   └── Dockerfile.full         # + Go
│   └── build.sh                    # Build script
```

### Completed

- [x] Implement yao-bridge (`sandbox/bridge/main.go`)
  - stdin/stdout ↔ Unix socket bridge
  - Signal handling for graceful shutdown
- [x] Create Dockerfiles
  - `base/Dockerfile.base` - Ubuntu 22.04, git, curl, yao-bridge
  - `claude/Dockerfile` - + Node.js 20, Python 3.11
  - `claude/Dockerfile.full` - + Go 1.23
- [x] Create build script (`sandbox/docker/build.sh`)
  - Builds yao-bridge as static binary
  - Builds all image variants

---

## Phase 6: ClaudeExecutor Integration 🔲 PENDING

### Goals

- Integrate Sandbox with ClaudeExecutor
- End-to-end execution flow

### Tasks

- [ ] Add Sandbox Manager to ClaudeExecutor

  ```go
  type ClaudeExecutor struct {
      Assistant      *Assistant
      SandboxManager *sandbox.Manager
      IPCManager     *ipc.Manager
  }
  ```

- [ ] Implement `Stream()` method
  1. Get or create container
  2. Create IPC session
  3. Generate .mcp.json
  4. Setup skills
  5. Build Claude CLI args
  6. Execute in container
  7. Parse output

- [ ] Implement `writeMCPConfig()`
  - Generate MCP config with yao-bridge
  - Include external MCP servers
  - Write to workspace

- [ ] Implement `setupSkills()`
  - Symlink skills directory to .claude/skills/

- [ ] Implement output parsing
  - Parse NDJSON stream
  - Extract text content
  - Extract file changes from tool_use
  - Handle result message

- [ ] Handle session mapping
  - Map Yao ChatID to Claude SessionID
  - Support `--resume` for continuation

### Deliverables

- [ ] ClaudeExecutor with Sandbox
- [ ] MCP config generation
- [ ] Skills setup
- [ ] Output parsing
- [ ] Integration tests

---

## Phase 7: Cleanup & Testing ✅ COMPLETED

### Goals

- Implement cleanup strategies
- Comprehensive testing
- Documentation

### Completed

- [x] Implement cleanup loop (every 5 minutes)
- [x] Implement `Cleanup(ctx) error`
- [x] Unit tests (no Docker required)
  - `config_test.go` - Config parsing, validation, env vars, edge cases
  - `helpers_test.go` - parseMemory, mapToSlice, parseLS, parseStat, tar operations
  - `ipc/jsonrpc_test.go` - JSON-RPC parsing, serialization
- [x] Integration tests (Docker required)
  - `manager_test.go` - Container lifecycle, exec, filesystem operations
  - `ipc/manager_test.go` - IPC session management
  - `ipc/session_test.go` - Session message handling, MCP protocol
- [x] README.md with usage examples

### Test Files

| File                  | Tests | Description                          |
| --------------------- | ----- | ------------------------------------ |
| `config_test.go`      | 10    | Config parsing, env vars, edge cases |
| `helpers_test.go`     | 6     | Utility functions                    |
| `ipc/jsonrpc_test.go` | 8     | JSON-RPC types                       |
| `manager_test.go`     | 18    | Container lifecycle (Docker)         |
| `ipc/manager_test.go` | 10    | IPC sessions                         |
| `ipc/session_test.go` | 11    | Session handlers (Docker optional)   |

### Unit Tests (No Docker)

```
✅ TestDefaultConfig
✅ TestConfigInit
✅ TestConfigInitWithEnv
✅ TestConfigInitWithWorkspaceEnv
✅ TestConfigInitWithPresetValues
✅ TestConfigInitInvalidEnvValues
✅ TestConfigInitNegativeValues
✅ TestConfigInitZeroMax
✅ TestContainerName
✅ TestConfigEnvPriority
✅ TestParseMemory
✅ TestMapToSlice
✅ TestParseLS
✅ TestParseStat
✅ TestParseLSMode
✅ TestCreateAndExtractTar
✅ TestJSONRPCRequestParsing
✅ TestJSONRPCResponseSerialization
✅ TestJSONRPCErrorResponse
✅ TestToolCallParams
✅ TestToolResult
✅ TestToolsListResult
✅ TestInitializeResult
```

### Integration Tests (Docker Required)

```
✅ TestNewManager
✅ TestNewManagerWithNilConfig
✅ TestGetOrCreate
✅ TestContainerStartStopRemove
✅ TestExec
✅ TestExecWithEnv
✅ TestExecWithTimeout
✅ TestFileOperations
✅ TestCopyOperations
✅ TestListContainers
✅ TestConcurrencyLimit
✅ TestConcurrentAccess
✅ TestContainerNotFound
✅ TestCleanup
✅ TestGetAccessors
✅ TestEnsureImageAutoPull
✅ TestManagerWithYaoApp (requires YAO_TEST_APPLICATION)
```

### IPC Tests

```
✅ TestNewManager
✅ TestCreateSession
✅ TestGetSession
✅ TestCloseSession
✅ TestCloseNonExistentSession
✅ TestCloseAllSessions
✅ TestSessionReplace
✅ TestConcurrentSessionAccess
✅ TestSessionConnection
✅ TestToolsList
✅ TestMethodNotFound
✅ TestParseError
✅ TestInitializedNotification
✅ TestSessionHandleInitialize
✅ TestSessionHandleResourcesList
✅ TestSessionHandleResourcesRead
✅ TestSessionHandleToolsCallInvalidParams
✅ TestSessionHandleToolsCallUnauthorized
✅ TestSessionToolsCallWithYaoApp (requires YAO_TEST_APPLICATION)
✅ TestSessionMultipleRequests
✅ TestSessionClose
✅ TestSessionEmptyLines
```

### Running Tests

```bash
# Unit tests only (no Docker needed)
go test -v ./sandbox/... -run "Test(Default|Config|Parse|Map|LS|Stat|Tar|JSONRPC|Tool)"

# Integration tests (Docker required)
source env.local.sh
go test -v ./sandbox/...

# With Yao application (full integration)
export YAO_TEST_APPLICATION=/path/to/yao-dev-app
source env.local.sh
go test -v ./sandbox/...

# Using Makefile (pulls test images automatically)
make unit-test-sandbox
```

---

## Phase Summary

| Phase | Description                    | Status       |
| ----- | ------------------------------ | ------------ |
| 1     | Core Interfaces & Types        | ✅ COMPLETED |
| 2     | Docker Container Management    | ✅ COMPLETED |
| 3     | Command Execution & Filesystem | ✅ COMPLETED |
| 4     | IPC System                     | ✅ COMPLETED |
| 5     | yao-bridge & Docker Image      | ✅ COMPLETED |
| 6     | ClaudeExecutor Integration     | 🔲 PENDING   |
| 7     | Cleanup & Testing              | ✅ COMPLETED |

---

## Implementation Summary

### Files Created

| File                                    | Lines | Description                    |
| --------------------------------------- | ----- | ------------------------------ |
| `sandbox/errors.go`                     | 22    | Error types                    |
| `sandbox/types.go`                      | 52    | Core type definitions          |
| `sandbox/config.go`                     | 90    | Configuration with env vars    |
| `sandbox/helpers.go`                    | 305   | Helper functions               |
| `sandbox/manager.go`                    | 541   | Main manager implementation    |
| `sandbox/ipc/types.go`                  | 139   | IPC type definitions           |
| `sandbox/ipc/manager.go`                | 101   | IPC session manager            |
| `sandbox/ipc/session.go`                | 252   | Session handling               |
| `sandbox/bridge/main.go`                | 59    | yao-bridge binary              |
| `sandbox/docker/base/Dockerfile.base`   | 34    | Base Docker image (multi-arch) |
| `sandbox/docker/claude/Dockerfile`      | 43    | Claude image                   |
| `sandbox/docker/claude/Dockerfile.full` | 34    | Full Claude image (multi-arch) |
| `sandbox/docker/build.sh`               | 145   | Build script (multi-arch)      |
| `sandbox/config_test.go`                | 175   | Config tests                   |
| `sandbox/helpers_test.go`               | 190   | Helper tests                   |
| `sandbox/manager_test.go`               | 520   | Manager integration tests      |
| `sandbox/ipc/jsonrpc_test.go`           | 236   | JSON-RPC tests                 |
| `sandbox/ipc/manager_test.go`           | 330   | IPC manager tests              |
| `sandbox/ipc/session_test.go`           | 420   | IPC session tests              |
| `sandbox/README.md`                     | 152   | Documentation                  |

**Total**: ~3800 lines

---

## Dependencies

### External

- Docker Engine (or Docker Desktop)
- Claude CLI (placeholder in Dockerfile)

### Go Packages

- `github.com/docker/docker/client` - Docker SDK
- `github.com/docker/docker/api/types/container` - Docker types

### Internal

- `github.com/yaoapp/gou/process` - Yao process execution

---

## Next Steps

1. **Phase 6: ClaudeExecutor Integration**
   - Implement in `yao/agent/assistant/executor/claude/`
   - Wire up sandbox with assistant execution flow

2. **CI/CD for Docker Images** ✅ COMPLETED
   - Images already built and pushed to Docker Hub:
     - `yaoapp/sandbox-base:latest` (amd64, arm64)
     - `yaoapp/sandbox-claude:latest` (amd64, arm64)
     - `yaoapp/sandbox-claude-full:latest` (amd64, arm64)
   - Set up automated builds on version tags

3. **CI/CD for Tests** ✅ COMPLETED
   - Sandbox tests run separately from core tests
   - Makefile: `make unit-test-sandbox`
   - GitHub Actions workflows updated:
     - `unit-test.yml`: Added `sandbox-test` job
     - `pr-test.yml`: Added `SandboxTest` job
   - Test images pre-pulled before tests:
     - `alpine:latest`
     - `yaoapp/sandbox-base:latest`
     - `yaoapp/sandbox-claude:latest`

---

## Success Criteria

### Functional ✅ (Sandbox Core)

- [x] Can create/start/stop/remove containers
- [x] Can execute commands in containers
- [x] IPC communication works bidirectionally
- [ ] Claude CLI can call Yao MCP tools (requires Phase 6)
- [x] Data persists across container restarts

### Performance (To be validated)

- [ ] Container creation < 5 seconds
- [ ] Command execution latency < 100ms overhead
- [ ] IPC round-trip < 10ms

### Reliability

- [x] Handles connection drops gracefully
- [x] Cleans up resources on errors
- [x] No resource leaks (cleanup loop)

### Security

- [x] User isolation enforced (one container per user+chat)
- [x] Resource limits enforced (memory, CPU)
- [x] No privilege escalation (`--cap-drop ALL`, `no-new-privileges`)
