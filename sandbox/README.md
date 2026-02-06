# Yao Sandbox

Sandbox provides persistent Docker containers as isolated execution environments for external CLI agents like Claude Code.

## Overview

The sandbox module enables Yao to safely run external AI coding agents (like Claude CLI) in isolated Docker containers. Each user+chat session gets its own container with:

- Persistent workspace for code and dependencies
- IPC communication via Unix sockets
- Resource limits (CPU, memory)
- Security isolation
- **VNC remote desktop** for visual transparency (optional)

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
│  ┌────────────────────────┴────────────────────────────────┐  │
│  │                    VNC Proxy Service                     │  │
│  │                                                          │  │
│  │   - GET /v1/sandbox/{id}/vnc        → VNC status        │  │
│  │   - GET /v1/sandbox/{id}/vnc/client → noVNC page        │  │
│  │   - GET /v1/sandbox/{id}/vnc/ws     → WebSocket proxy   │  │
│  └──────────────────────────────────────────────────────────┘  │
│                           │                                   │
│           ┌───────────────┼───────────────┐                   │
│           ▼               ▼               ▼                   │
│    ┌────────────┐  ┌────────────┐  ┌────────────┐             │
│    │ sandbox-   │  │ sandbox-   │  │ sandbox-   │             │
│    │ claude     │  │ playwright │  │ desktop    │             │
│    │ (No VNC)   │  │ (VNC)      │  │ (VNC)      │             │
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

# Build base image
./build.sh claude

# Build VNC-enabled images
./build.sh browser      # Browser (Playwright) + Fluxbox + VNC
./build.sh desktop      # XFCE Desktop + VNC

# Build all images
./build.sh all
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

| Variable                       | Default                             | Description                                    |
| ------------------------------ | ----------------------------------- | ---------------------------------------------- |
| `YAO_SANDBOX_IMAGE`            | `yao/sandbox-claude:latest`         | Docker image                                   |
| `YAO_SANDBOX_WORKSPACE`        | `{YAO_DATA_ROOT}/sandbox/workspace` | Workspace directory                            |
| `YAO_SANDBOX_IPC`              | `{YAO_DATA_ROOT}/sandbox/ipc`       | IPC socket directory                           |
| `YAO_SANDBOX_MAX`              | `100`                               | Max concurrent containers                      |
| `YAO_SANDBOX_IDLE_TIMEOUT`     | `30m`                               | Idle timeout                                   |
| `YAO_SANDBOX_MEMORY`           | `2g`                                | Memory limit                                   |
| `YAO_SANDBOX_CPU`              | `1.0`                               | CPU limit                                      |
| `YAO_SANDBOX_VNC_PORT_MAPPING` | `false`                             | Enable VNC port mapping (for Docker Desktop)   |

### Docker Desktop (macOS/Windows)

Docker Desktop runs containers in a LinuxKit VM, so container IPs are not directly accessible from the host. Enable VNC port mapping for local development:

```bash
export YAO_SANDBOX_VNC_PORT_MAPPING=true
export YAO_SANDBOX_IMAGE="yaoapp/sandbox-claude-browser:latest"
```

When enabled, VNC ports (6080, 5900) are automatically mapped to random available host ports on `127.0.0.1`.

## Docker Images

| Image                                      | VNC | Description                           |
| ------------------------------------------ | --- | ------------------------------------- |
| `yaoapp/sandbox-base:latest`               | ❌  | Base image with git, curl, yao-bridge |
| `yaoapp/sandbox-claude:latest`             | ❌  | + Claude CLI, Node.js 20, Python 3.11 |
| `yaoapp/sandbox-claude:full`               | ❌  | + Go 1.23                             |
| `yaoapp/sandbox-claude-browser:latest`     | ✅  | + Playwright, Fluxbox, VNC (~3.4GB)   |
| `yaoapp/sandbox-claude-desktop:latest`     | ✅  | + XFCE Desktop, VNC (~3.1GB)          |

## IPC Communication

Sandbox containers communicate with Yao via Unix sockets using the MCP (Model Context Protocol) JSON-RPC format. The `yao-bridge` binary inside containers bridges stdio ↔ socket.

Supported methods:

- `initialize` - Handshake
- `tools/list` - List available tools
- `tools/call` - Execute a tool

## VNC Remote Desktop

VNC-enabled images (playwright, desktop) provide real-time visibility into Claude's operations.

### API Endpoints

| Endpoint                        | Description                        |
| ------------------------------- | ---------------------------------- |
| `GET /v1/sandbox/{id}/vnc`      | VNC status (ready/starting/unavailable) |
| `GET /v1/sandbox/{id}/vnc/client` | noVNC HTML client page            |
| `GET /v1/sandbox/{id}/vnc/ws`   | WebSocket proxy to container VNC   |

### View Modes

- **Interactive** (default): User can use keyboard and mouse
- **View-only** (`?viewonly=true`): User can only watch

For detailed design, see [DESIGN-PLAYWRIGHT-VNC.md](./DESIGN-PLAYWRIGHT-VNC.md).

## Directory Structure

```
sandbox/
├── bridge/          # yao-bridge source
├── docker/          # Dockerfiles and build script
│   ├── base/
│   ├── claude/
│   ├── browser/     # Browser (Playwright) + VNC image
│   ├── desktop/     # XFCE Desktop + VNC image
│   ├── vnc/         # Shared VNC scripts
│   └── build.sh
├── ipc/             # IPC system
│   ├── manager.go
│   ├── session.go
│   └── types.go
├── vncproxy/        # VNC proxy service
│   ├── proxy.go
│   ├── config.go
│   └── proxy_test.go
├── config.go        # Configuration
├── errors.go        # Error types
├── helpers.go       # Helper functions
├── manager.go       # Main manager
└── types.go         # Type definitions
```

## Testing

```bash
# Load environment variables first
source env.local.sh

# Unit tests (no Docker required)
go test -v ./sandbox/... -run "^Test.*Validation|^Test.*Generation|^Test.*Parsing"

# All tests (requires Docker)
go test -v ./sandbox/...

# VNC proxy tests only
go test -v ./sandbox/vncproxy/...
```

## Security

- Containers run as non-root user
- `--cap-drop ALL` removes all capabilities
- `no-new-privileges` prevents privilege escalation
- Only workspace directory is mounted
- Per-session IPC sockets with authorized tools only
