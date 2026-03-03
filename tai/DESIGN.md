# Tai Go SDK

Go client library for [Tai](https://github.com/yaoapp/tai) — the universal runtime bridge for Yao Sandbox.

## Overview

Provides a unified API for container lifecycle, filesystem operations, HTTP proxy, and VNC access.
Supports two modes via a single entry point:

- **Local** (`docker://` or `""`) — direct Docker daemon connection
- **Remote** (`tai://host`) — via Tai Server proxy (Docker, K8s)

All sub-packages follow the same pattern: **interface + Remote/Local implementations**.

## Package Layout

```
yao/tai/
├── tai.go                      # Client, New(), Option, Close()
├── volume/                     # Volume IO + Sync
├── workspace/                  # Go fs.FS wrapper over volume.Volume
├── sandbox/                    # Container lifecycle (Create/Start/Stop/Exec/Remove)
│   ├── sandbox.go              # Interface + shared types
│   ├── local.go                # Direct Docker socket
│   ├── docker.go               # Docker via Tai proxy
│   ├── docker_core.go          # Shared Docker SDK logic
│   └── k8s.go                  # Kubernetes via Tai TCP proxy
├── proxy/                      # HTTP reverse proxy URL resolution
└── vnc/                        # VNC WebSocket URL resolution
```

## Quick Start

```go
import "github.com/yaoapp/yao/tai"

// Local — default Docker socket
c, _ := tai.New("")

// Local — explicit address
c, _ := tai.New("docker:///var/run/docker.sock")
c, _ := tai.New("docker://192.168.1.50:2375")

// Remote — via Tai Server (Docker runtime, default)
c, _ := tai.New("tai://192.168.1.100")

// Remote — via Tai Server (K8s runtime)
c, _ := tai.New("tai://10.0.0.5", tai.K8s,
    tai.WithKubeConfig("/path/to/kubeconfig.yml"),
    tai.WithNamespace("sandbox"),
)

defer c.Close()

// Container lifecycle
id, _ := c.Sandbox().Create(ctx, sandbox.CreateOptions{
    Image: "node:20",
    Cmd:   []string{"sleep", "infinity"},
})
c.Sandbox().Start(ctx, id)

// Filesystem
ws := c.Workspace("session-1")
ws.WriteFile("app.js", []byte("console.log('hi')"), 0644)
data, _ := ws.ReadFile("app.js")

// HTTP proxy URL
url, _ := c.Proxy().URL(ctx, id, 3000, "/api/health")

// VNC URL
vncURL, _ := c.VNC().URL(ctx, id)
```

## Address Protocol

| Prefix | Mode | Description |
|--------|------|-------------|
| `""` | Local | Platform default Docker socket |
| `docker://...` | Local | Direct Docker daemon (socket or TCP) |
| `tai://host` | Remote | Via Tai Server, all services proxied |

## Sub-Package Interfaces

### volume.Volume

File IO and directory sync between Yao and the container workspace.

- `ReadFile`, `WriteFile`, `Stat`, `ListDir`, `Remove`, `Rename`, `MkdirAll`
- `SyncPush` (Yao -> Tai), `SyncPull` (Tai -> Yao)
- **Remote**: gRPC to Tai `:9100`
- **Local**: direct disk IO under `dataDir/{sessionID}/`

### workspace.FS

Go `fs.FS`-compatible interface wrapping `volume.Volume`, adding write operations.

### sandbox.Sandbox

Container lifecycle: `Create`, `Start`, `Stop`, `Remove`, `Exec`, `Inspect`, `List`.

- **Local**: direct Docker socket, handles VNC port mapping and capabilities
- **Docker**: via Tai `:2375` (Docker Engine API proxy)
- **K8s**: via Tai `:6443` (kube-apiserver TCP proxy, single-container Pod per sandbox)

### proxy.Proxy

HTTP service URL resolution: `URL(ctx, containerID, port, path)`.

- **Remote**: `http://tai-host:8080/{id}:{port}/{path}`
- **Local**: `http://127.0.0.1:{hostPort}/{path}` via `sandbox.Inspect`

### vnc.VNC

VNC WebSocket URL resolution: `URL(ctx, containerID)`.

- **Remote**: `ws://tai-host:6080/vnc/{id}/ws`
- **Local**: `ws://127.0.0.1:{vncHostPort}/ws` via `sandbox.Inspect`

## Options

```go
tai.Docker                     // Docker runtime (default, can omit)
tai.K8s                        // Kubernetes runtime
tai.WithPorts(Ports{})         // custom port mapping
tai.WithHTTPClient(hc)         // custom HTTP client
tai.WithDataDir(dir)           // workspace root (Local mode)
tai.WithKubeConfig(path)       // kubeconfig file path (K8s runtime)
tai.WithNamespace(ns)          // namespace for K8s (default "default")
```

## Default Ports

| Service | Default Port |
|---------|-------------|
| gRPC (Volume + Gateway) | 9100 |
| HTTP Proxy | 8080 |
| VNC Router | 6080 |
| Docker API Proxy | 2375 |
| K8s API Proxy | 6443 |

## Dependencies

- `github.com/yaoapp/tai/volume/pb` — gRPC proto types
- `google.golang.org/grpc`
- `github.com/pierrec/lz4/v4` — sync compression
- `github.com/docker/docker` — Docker SDK
- `k8s.io/client-go` + `k8s.io/api` + `k8s.io/apimachinery` — Kubernetes SDK
