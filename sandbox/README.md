# Yao Sandbox

Sandbox provides persistent Docker containers as isolated execution environments for external CLI agents like Claude Code.

## Overview

The sandbox module enables Yao to safely run external AI coding agents (like Claude CLI) in isolated Docker containers. Each user+chat session gets its own container with:

- Persistent workspace for code and dependencies
- IPC communication via Unix sockets
- Resource limits (CPU, memory)
- Security isolation

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        Yao Server                            │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐  │
│  │                   Sandbox Manager                       │  │
│  │                                                         │  │
│  │   - GetOrCreate(userID, chatID) → container             │  │
│  │   - Exec/Stream commands in container                   │  │
│  │   - Filesystem operations (read, write, copy)           │  │
│  │                                                         │  │
│  └────────────────────────┬────────────────────────────────┘  │
│                           │                                   │
│           ┌───────────────┼───────────────┐                   │
│           ▼               ▼               ▼                   │
│    ┌────────────┐  ┌────────────┐  ┌────────────┐             │
│    │ Container  │  │ Container  │  │ Container  │             │
│    │ (user1)    │  │ (user2)    │  │ (user3)    │             │
│    └─────┬──────┘  └─────┬──────┘  └─────┬──────┘             │
│          │               │               │                    │
│    ──────┴───────────────┴───────────────┴────                │
│                   Unix Socket IPC                             │
│             (one socket per container)                        │
└───────────────────────────────────────────────────────────────┘
```

## Quick Start

### Build Docker Images

```bash
cd sandbox/docker
./build.sh claude
```

### Usage

```go
import "github.com/yaoapp/yao/sandbox"

// Create manager
config := sandbox.DefaultConfig()
config.Init("/path/to/yao/data")

manager, err := sandbox.NewManager(config)
if err != nil {
    log.Fatal(err)
}
defer manager.Close()

// Get or create container
container, err := manager.GetOrCreate(ctx, "user123", "chat456")
if err != nil {
    log.Fatal(err)
}

// Execute command
result, err := manager.Exec(ctx, container.Name, []string{"echo", "hello"}, nil)
fmt.Println(result.Stdout) // "hello\n"

// Write file
err = manager.WriteFile(ctx, container.Name, "/workspace/test.txt", []byte("content"))

// Read file
data, err := manager.ReadFile(ctx, container.Name, "/workspace/test.txt")
```

## Configuration

### Environment Variables

| Variable                   | Default                             | Description               |
| -------------------------- | ----------------------------------- | ------------------------- |
| `YAO_SANDBOX_IMAGE`        | `yao/sandbox-claude:latest`         | Docker image              |
| `YAO_SANDBOX_WORKSPACE`    | `{YAO_DATA_ROOT}/sandbox/workspace` | Workspace directory       |
| `YAO_SANDBOX_IPC`          | `{YAO_DATA_ROOT}/sandbox/ipc`       | IPC socket directory      |
| `YAO_SANDBOX_MAX`          | `100`                               | Max concurrent containers |
| `YAO_SANDBOX_IDLE_TIMEOUT` | `30m`                               | Idle timeout              |
| `YAO_SANDBOX_MEMORY`       | `2g`                                | Memory limit              |
| `YAO_SANDBOX_CPU`          | `1.0`                               | CPU limit                 |

## Docker Images

| Image                       | Description                           |
| --------------------------- | ------------------------------------- |
| `yao/sandbox-base:latest`   | Base image with git, curl, yao-bridge |
| `yao/sandbox-claude:latest` | + Claude CLI, Node.js 20, Python 3.11 |
| `yao/sandbox-claude:full`   | + Go 1.23                             |

## IPC Communication

Sandbox containers communicate with Yao via Unix sockets using the MCP (Model Context Protocol) JSON-RPC format. The `yao-bridge` binary inside containers bridges stdio ↔ socket.

Supported methods:

- `initialize` - Handshake
- `tools/list` - List available tools
- `tools/call` - Execute a tool

## Directory Structure

```
sandbox/
├── bridge/          # yao-bridge source
├── docker/          # Dockerfiles and build script
│   ├── base/
│   ├── claude/
│   └── build.sh
├── ipc/             # IPC system
│   ├── manager.go
│   ├── session.go
│   └── types.go
├── config.go        # Configuration
├── errors.go        # Error types
├── helpers.go       # Helper functions
├── manager.go       # Main manager
└── types.go         # Type definitions
```

## Testing

```bash
# Unit tests (no Docker required)
go test -v ./sandbox/... -run "^Test.*Validation|^Test.*Generation|^Test.*Parsing"

# All tests (requires Docker)
go test -v ./sandbox/...
```

## Security

- Containers run as non-root user
- `--cap-drop ALL` removes all capabilities
- `no-new-privileges` prevents privilege escalation
- Only workspace directory is mounted
- Per-session IPC sockets with authorized tools only
