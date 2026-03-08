# Tai SDK

Go client library for the [Tai](https://github.com/YaoApp/tai) runtime bridge. Provides unified access to container sandboxes, volume IO, HTTP proxy, and VNC routing — transparently working in **Local** (direct Docker), **Remote** (via Tai server), and **Tunnel** (via Yao WebSocket tunnel) modes.

## Package Layout

| Package | Import Path | Description |
|---------|-------------|-------------|
| `tai` | `github.com/yaoapp/yao/tai` | Top-level client, `New()`, options, `Close()` |
| `sandbox` | `github.com/yaoapp/yao/tai/sandbox` | Container lifecycle (Create/Start/Stop/Exec/Remove) |
| `volume` | `github.com/yaoapp/yao/tai/volume` | File IO and directory sync |
| `workspace` | `github.com/yaoapp/yao/tai/workspace` | `fs.FS`-compatible filesystem over Volume |
| `proxy` | `github.com/yaoapp/yao/tai/proxy` | HTTP reverse proxy URL resolution |
| `vnc` | `github.com/yaoapp/yao/tai/vnc` | VNC WebSocket URL resolution |
| `registry` | `github.com/yaoapp/yao/tai/registry` | In-memory Tai node registry (direct + tunnel) |
| `api` | `github.com/yaoapp/yao/tai/api` | HTTP handlers for node registration/heartbeat |
| `tunnel` | `github.com/yaoapp/yao/tai/tunnel` | WebSocket tunnel server (control + data + proxy) |
| `hostexec/pb` | `github.com/yaoapp/yao/tai/hostexec/pb` | HostExec gRPC client (host command execution) |
| `serverinfo/pb` | `github.com/yaoapp/yao/tai/serverinfo/pb` | ServerInfo gRPC client (port/capability discovery) |

## Quick Start

### Local Mode (direct Docker)

```go
c, err := tai.New("local")
// or: tai.New("docker:///var/run/docker.sock")
// or: tai.New("tcp://192.168.1.50:2375")
defer c.Close()

id, _ := c.Sandbox().Create(ctx, sandbox.CreateOptions{
    Name:  "my-sandbox",
    Image: "alpine:latest",
    Cmd:   []string{"sleep", "300"},
})
c.Sandbox().Start(ctx, id)
```

### Remote Mode (via Tai server, Docker runtime)

```go
c, err := tai.New("tai://192.168.1.100")
defer c.Close()

result, _ := c.Sandbox().Exec(ctx, id, []string{"echo", "hello"}, sandbox.ExecOptions{})
fmt.Println(result.Stdout) // "hello\n"
```

### Remote Mode (via Tai server, K8s runtime)

```go
c, err := tai.New("tai://192.168.1.100", tai.K8s,
    tai.WithKubeConfig("/path/to/kubeconfig.yml"),
    tai.WithNamespace("default"),
    tai.WithPorts(tai.Ports{K8s: 16443}),
)
defer c.Close()
```

### Tunnel Mode (via Yao WebSocket tunnel)

```go
// Requires a running Yao server with the Tai node registered via tunnel.
// The taiID is the node's identifier in the registry.
c, err := tai.New("tunnel://tai-abc123")
defer c.Close()
```

## Address Protocols

| Address | Mode | Description |
|---------|------|-------------|
| `"local"` | Local | Platform default Docker socket |
| `"127.0.0.1"` / `"localhost"` / `"::1"` | Local | Auto-detected as local Docker |
| `unix:///var/run/docker.sock` | Local | Explicit Unix socket |
| `tcp://host:port` | Local | Explicit TCP Docker daemon |
| `npipe:////./pipe/docker_engine` | Local | Windows named pipe |
| `docker://host:port` | Local | Docker scheme |
| `tai://host` | Remote | Connect via Tai server (gRPC default 19100) |
| `tai://host:port` | Remote | Connect via Tai server on custom gRPC port |
| `tunnel://tai-id` | Tunnel | Connect via Yao WebSocket tunnel |
| `192.168.x.x` (non-local IP) | Remote | Auto-prepends `tai://` |

## Options

| Option | Description | Default |
|--------|-------------|---------|
| `WithPorts(Ports{...})` | Override Tai service ports (takes precedence over ServerInfo) | gRPC=19100, HTTP=8099, VNC=16080 |
| `WithHTTPClient(*http.Client)` | Custom HTTP client for proxy/VNC | `http.DefaultClient` |
| `WithDataDir(path)` | Volume storage root (Local mode) | `/tmp/tai-volumes` |
| `WithKubeConfig(path)` | Kubeconfig file path (K8s mode, **required**) | - |
| `WithNamespace(ns)` | K8s namespace | `"default"` |
| `WithVolume(vol)` | Inject custom Volume implementation (testing) | - |

## Default Ports

| Service | Port | Description |
|---------|------|-------------|
| gRPC | 19100 | Volume IO + Gateway + ServerInfo + HostExec |
| HTTP | 8099 | HTTP reverse proxy |
| VNC | 16080 | VNC WebSocket router |
| Docker | 12375 | Docker API proxy |
| K8s | 16443 | Kubernetes API proxy |

Ports are auto-discovered via Tai's `ServerInfo.GetInfo` gRPC call. Values set via `WithPorts` take precedence over server-reported values.

## Client API

```go
c.Volume()               // volume.Volume — file IO (never nil)
c.Workspace(sessionID)   // workspace.FS — fs.FS over Volume
c.DataDir()              // string — host-side data directory (Local mode only)
c.Sandbox()              // sandbox.Sandbox — container lifecycle (nil if host-exec-only)
c.Image()                // sandbox.Image — image management (nil if host-exec-only)
c.Proxy()                // proxy.Proxy — HTTP reverse proxy (nil if host-exec-only)
c.VNC()                  // vnc.VNC — VNC WebSocket (nil if host-exec-only)
c.HostExec()             // hepb.HostExecClient — host command execution (nil in local mode)
c.IsLocal()              // bool — true for local mode (docker/unix/tcp/npipe/local)
c.Close()                // error — releases all resources
```

## Runtime Constants

```go
tai.Docker  // default — use Docker runtime via Tai
tai.K8s     // use Kubernetes runtime via Tai
```

## Yao gRPC Compatibility

The `tai` package re-exports Yao gRPC helpers for backward compatibility:

```go
tai.NewTokenManagerFromEnv()           // *TokenManager from env vars
tai.NewTokenManager(access, refresh, sandboxID)
tai.NewYaoClientFromEnv()              // *YaoClient from env vars
tai.DialYao(addr, tm)                  // connect to Yao gRPC
tai.Run(ctx, client, process, args, timeout)   // execute Yao process
tai.Shell(ctx, client, cmd, args, env, timeout) // execute shell command
tai.HeartbeatLoop(ctx, client, sandboxID)       // periodic heartbeat (blocks)
```

New code should use `grpc/client` directly. These wrappers exist for sandbox/container code that imports `tai`.

## Capabilities

When connecting to a remote Tai server, the client calls `ServerInfo.GetInfo` to discover:
- **Ports**: actual listening ports (http, docker, vnc, k8s)
- **Capabilities**: `docker`, `k8s`, `host_exec`

If no usable capabilities are found, `New()` returns an error. Remote mode checks `docker`/`k8s`/`host_exec`; Tunnel mode checks `docker`/`host_exec` (K8s is not supported over tunnel).

## Sub-Package Documentation

- [sandbox.md](sandbox.md) — Container lifecycle & Image management
- [volume.md](volume.md) — File IO and sync
- [workspace.md](workspace.md) — fs.FS-compatible filesystem
- [proxy.md](proxy.md) — HTTP reverse proxy
- [vnc.md](vnc.md) — VNC WebSocket routing
- [registry.md](registry.md) — Tai node registry (direct + tunnel)
- [api.md](api.md) — HTTP registration API
- [tunnel.md](tunnel.md) — WebSocket tunnel handlers
