# Tai SDK

Go client library for the [Tai](https://github.com/YaoApp/tai) runtime bridge. Provides unified access to container sandboxes, volume IO, HTTP proxy, and VNC routing — transparently working in both **Local** (direct Docker) and **Remote** (via Tai server) modes.

## Package Layout

| Package | Import Path | Description |
|---------|-------------|-------------|
| `tai` | `github.com/yaoapp/yao/tai` | Top-level client, `New()`, options, `Close()` |
| `sandbox` | `github.com/yaoapp/yao/tai/sandbox` | Container lifecycle (Create/Start/Stop/Exec/Remove) |
| `volume` | `github.com/yaoapp/yao/tai/volume` | File IO and directory sync |
| `workspace` | `github.com/yaoapp/yao/tai/workspace` | `fs.FS`-compatible filesystem over Volume |
| `proxy` | `github.com/yaoapp/yao/tai/proxy` | HTTP reverse proxy URL resolution |
| `vnc` | `github.com/yaoapp/yao/tai/vnc` | VNC WebSocket URL resolution |

## Quick Start

### Local Mode (direct Docker)

```go
c, err := tai.New("")
// or: tai.New("unix:///var/run/docker.sock")
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
    tai.WithPorts(tai.Ports{K8s: 6443}),
)
defer c.Close()
```

## Address Protocols

| Address | Mode | Description |
|---------|------|-------------|
| `""` | Local | Platform default Docker socket |
| `unix:///var/run/docker.sock` | Local | Explicit Unix socket |
| `tcp://host:port` | Local | Explicit TCP Docker daemon |
| `npipe:////./pipe/docker_engine` | Local | Windows named pipe |
| `docker://host:port` | Local | Docker scheme |
| `tai://host` | Remote | Connect via Tai server |

## Options

| Option | Description | Default |
|--------|-------------|---------|
| `WithPorts(Ports{...})` | Override Tai service ports | gRPC=9100, HTTP=8080, VNC=6080 |
| `WithHTTPClient(*http.Client)` | Custom HTTP client for proxy/VNC | `http.DefaultClient` |
| `WithDataDir(path)` | Volume storage root (Local mode) | `/tmp/tai-volumes` |
| `WithKubeConfig(path)` | Kubeconfig file path (K8s mode, **required**) | - |
| `WithNamespace(ns)` | K8s namespace | `"default"` |

## Default Ports

| Service | Port | Description |
|---------|------|-------------|
| gRPC | 9100 | Volume IO + Gateway |
| HTTP | 8080 | HTTP reverse proxy |
| VNC | 6080 | VNC WebSocket router |
| Docker | 2375 | Docker API proxy |
| K8s | 6443 | Kubernetes API proxy |

## Client API

```go
c.Volume()               // volume.Volume
c.Workspace(sessionID)   // workspace.FS
c.Sandbox()              // sandbox.Sandbox
c.Proxy()                // proxy.Proxy
c.VNC()                  // vnc.VNC
c.IsLocal()              // bool
c.Close()                // error
```

## Runtime Constants

```go
tai.Docker  // default — use Docker runtime via Tai
tai.K8s     // use Kubernetes runtime via Tai
```

## Sub-Package Documentation

- [sandbox.md](sandbox.md) — Container lifecycle management
- [volume.md](volume.md) — File IO and sync
- [workspace.md](workspace.md) — fs.FS-compatible filesystem
- [proxy.md](proxy.md) — HTTP reverse proxy
- [vnc.md](vnc.md) — VNC WebSocket routing
