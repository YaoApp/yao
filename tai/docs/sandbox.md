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
    Create(ctx context.Context, opts CreateOptions) (string, error)
    Start(ctx context.Context, id string) error
    Stop(ctx context.Context, id string, timeout time.Duration) error
    Remove(ctx context.Context, id string, force bool) error
    Exec(ctx context.Context, id string, cmd []string, opts ExecOptions) (*ExecResult, error)
    ExecStream(ctx context.Context, id string, cmd []string, opts ExecOptions) (*StreamHandle, error)
    Inspect(ctx context.Context, id string) (*ContainerInfo, error)
    List(ctx context.Context, opts ListOptions) ([]ContainerInfo, error)
    Close() error
}
```

### StreamHandle

```go
type StreamHandle struct {
    Stdin  io.WriteCloser
    Stdout io.Reader
    Stderr io.Reader
    Wait   func() (int, error) // blocks until exec finishes, returns exit code
    Cancel func()              // aborts the exec process
}
```

`ExecStream` provides real-time I/O access to a running exec process. Unlike `Exec` which collects all output, `ExecStream` returns immediately with readers/writers for interactive use.

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

Connects to Docker Engine API through Tai's Docker proxy. `addr` should be `"tcp://tai-host:12375"`.

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
    VNC        bool              // enable VNC port mapping (Local and Docker modes)
    Ports      []PortMapping     // port mappings (Docker only)
    Labels     map[string]string // container/pod labels for discovery and management
    User       string            // container user, e.g. "1000:1000" or "sandbox"
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
    ID     string            // container/pod ID
    Name   string            // container/pod name
    Image  string            // image name
    Status string            // "created", "running", "exited", "removing" (Docker)
                             // "Pending", "Running", "Succeeded", "Failed" (K8s)
    IP     string            // container/pod IP address
    Ports  []PortMapping     // mapped ports (Docker only)
    Labels map[string]string // container/pod labels
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
| `Start` | starts a stopped container | polls until pod leaves Pending (up to 60s) |
| `Stop` | stops with timeout, container persists | deletes the pod with grace period |
| `Remove(force=true)` | force-removes | deletes with grace period 0 |
| `Exec` | Docker exec API | `kubectl exec` via SPDY |
| `Inspect.Ports` | populated from Docker | always empty |
| `List` | filters only by `opts.Labels` (no auto label) | auto-merges `managed-by=yao-tai-sdk` + `opts.Labels` |
| `Binds` | supported | not supported |
| `VNC` flag | auto port-maps 6080 and 5900 (all platforms) | not applicable |

## Image Interface

```go
type Image interface {
    Exists(ctx context.Context, ref string) (bool, error)
    Pull(ctx context.Context, ref string, opts PullOptions) (<-chan PullProgress, error)
    Remove(ctx context.Context, ref string, force bool) error
    List(ctx context.Context) ([]ImageInfo, error)
}
```

Accessed via `c.Image()` on the top-level client. Nil when the Tai server has no container runtime.

| Implementation | Constructor | Backend | Notes |
|----------------|-------------|---------|-------|
| **Docker** | `NewDockerImage(cli)` | Docker SDK | Shared by Local and Docker-via-Tai modes |
| **K8s** | `NewK8sImage()` | No-op | Image pulling is handled by kubelet |

### DockerCli Helper

```go
func DockerCli(sb Sandbox) *client.Client
```

Extracts the underlying Docker SDK client from a `Sandbox` (Local or Docker). Returns `nil` for K8s sandboxes. Used internally to construct `NewDockerImage(DockerCli(sb))`.

### Types

```go
type PullOptions struct {
    Auth *RegistryAuth // nil = anonymous / public
}

type RegistryAuth struct {
    Username string
    Password string
    Server   string // e.g. "ghcr.io", "registry.example.com"
}

type PullProgress struct {
    Status  string // "Pulling fs layer", "Downloading", "Extracting", "Pull complete", etc.
    Layer   string // layer digest / short ID
    Current int64  // bytes completed
    Total   int64  // bytes total (0 if unknown)
    Error   string // non-empty on failure
}

type ImageInfo struct {
    ID      string
    Tags    []string
    Size    int64
    Created time.Time
}
```

### Image Example

```go
c, _ := tai.New("tai://192.168.1.100")
defer c.Close()

progress, _ := c.Image().Pull(ctx, "alpine:latest", sandbox.PullOptions{})
for p := range progress {
    fmt.Printf("%s %s %d/%d\n", p.Status, p.Layer, p.Current, p.Total)
}

images, _ := c.Image().List(ctx)
for _, img := range images {
    fmt.Printf("%s %v\n", img.ID[:12], img.Tags)
}
```

## Sandbox Example

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
