# Package `sandbox`

Container lifecycle management. Provides a unified `Sandbox` interface with three implementations:

| Implementation | Constructor | Backend | Mode |
|----------------|-------------|---------|------|
| **Local** | `NewLocal(addr)` | Direct Docker daemon | Local |
| **Docker** | `NewDocker(addr)` | Docker via Tai proxy | Remote |
| **K8s** | `NewK8s(addr, opts)` | Kubernetes via Tai proxy | Remote |

## Interface

```go
type Sandbox interface {
    Create(ctx context.Context, opts CreateOptions) (id string, err error)
    Start(ctx context.Context, id string) error
    Stop(ctx context.Context, id string, timeout time.Duration) error
    Remove(ctx context.Context, id string, force bool) error
    Exec(ctx context.Context, id string, cmd []string, opts ExecOptions) (*ExecResult, error)
    Inspect(ctx context.Context, id string) (*ContainerInfo, error)
    List(ctx context.Context, opts ListOptions) ([]ContainerInfo, error)
    Close() error
}
```

## Constructors

### NewLocal

```go
func NewLocal(addr string) (Sandbox, error)
```

Connects directly to a Docker daemon. `addr` can be:
- `""` — platform default (Unix socket on Linux/macOS, named pipe on Windows)
- `"unix:///var/run/docker.sock"` — explicit Unix socket
- `"tcp://host:port"` — explicit TCP

Pings the daemon on creation; returns an error if unreachable.

### NewDocker

```go
func NewDocker(addr string) (Sandbox, error)
```

Connects to Docker Engine API through Tai's Docker proxy. `addr` should be `"tcp://tai-host:2375"`.

### NewK8s

```go
func NewK8s(addr string, opts ...K8sOption) (Sandbox, error)
```

Connects to Kubernetes through Tai's TCP proxy. Each sandbox maps to a single-container Pod.

**Parameters:**
- `addr` — `"host:port"` pointing to Tai's K8s proxy endpoint
- `opts.KubeConfig` — path to kubeconfig file (**required**). Relative paths are resolved to absolute.
- `opts.Namespace` — Kubernetes namespace (default `"default"`)

The constructor overrides the kubeconfig's `server` field to point at `addr`, enables insecure TLS (since Tai does TCP passthrough), and verifies connectivity by querying the namespace.

All pods created by K8s sandbox are labeled with `managed-by: yao-tai-sdk`.

## Types

### CreateOptions

```go
type CreateOptions struct {
    Name       string            // container/pod name
    Image      string            // container image
    Cmd        []string          // entrypoint command
    Env        map[string]string // environment variables
    Binds      []string          // volume binds (Docker only)
    WorkingDir string            // working directory
    Memory     int64             // memory limit in bytes, 0 = no limit
    CPUs       float64           // CPU limit, 0 = no limit
    VNC        bool              // enable VNC port mapping (Local only)
    Ports      []PortMapping     // port mappings (Docker only)
}
```

### PortMapping

```go
type PortMapping struct {
    ContainerPort int    // port inside the container
    HostPort      int    // port on the host, 0 = random
    HostIP        string // host bind address, default "127.0.0.1"
    Protocol      string // "tcp" (default) or "udp"
}
```

### ContainerInfo

```go
type ContainerInfo struct {
    ID     string        // container/pod ID
    Name   string        // container/pod name
    Image  string        // image name
    Status string        // "created", "running", "exited", "removing" (Docker)
                         // "Pending", "Running", "Succeeded", "Failed" (K8s)
    IP     string        // container/pod IP address
    Ports  []PortMapping // mapped ports (Docker only)
}
```

### ExecOptions

```go
type ExecOptions struct {
    WorkDir string            // override working directory
    Env     map[string]string // additional environment variables
}
```

### ExecResult

```go
type ExecResult struct {
    ExitCode int
    Stdout   string
    Stderr   string
}
```

### ListOptions

```go
type ListOptions struct {
    All    bool              // include stopped containers
    Labels map[string]string // filter by labels
}
```

### K8sOption

```go
type K8sOption struct {
    Namespace  string // default "default"
    KubeConfig string // path to kubeconfig file (required)
}
```

## Behavioral Differences

| Behavior | Docker (Local/Remote) | K8s |
|----------|----------------------|-----|
| `Create` returns | container ID (hash) | pod name |
| `Start` | starts a stopped container | polls until pod leaves Pending (up to 30s) |
| `Stop` | stops with timeout, container persists | deletes the pod with grace period |
| `Remove(force=true)` | force-removes | deletes with grace period 0 |
| `Exec` | Docker exec API | `kubectl exec` via SPDY |
| `Inspect.Ports` | populated from Docker | always empty |
| `List` | filters by `tai-sdk=true` label | filters by `managed-by=yao-tai-sdk` label |
| `Binds` | supported | not supported |
| `VNC` flag | auto port-maps 6080 on macOS/Windows | not applicable |

## Example

```go
sb, _ := sandbox.NewLocal("")
defer sb.Close()

id, _ := sb.Create(ctx, sandbox.CreateOptions{
    Name:  "worker",
    Image: "alpine:latest",
    Cmd:   []string{"sleep", "300"},
    Env:   map[string]string{"FOO": "bar"},
    Memory: 256 * 1024 * 1024, // 256 MB
})

sb.Start(ctx, id)

result, _ := sb.Exec(ctx, id, []string{"echo", "$FOO"}, sandbox.ExecOptions{})
fmt.Println(result.Stdout)

sb.Stop(ctx, id, 10*time.Second)
sb.Remove(ctx, id, false)
```
