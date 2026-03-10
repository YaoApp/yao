# Sandbox V2 — Go API Reference

Package: `github.com/yaoapp/yao/sandbox/v2`

Sandbox V2 manages sandboxes through a set of Tai nodes. Two primary abstractions:

- **Box** — a container (Docker or K8s pod). Created via `Manager.Create`.
- **Host** — the Tai host machine itself. Obtained via `Manager.Host` (no Create needed).

Supports workspace mounting, VNC, WebSocket proxying, and HostExec.

---

## Initialization

### Init

```go
func Init()
```

Initializes the global Manager singleton. Must be called once at startup.
No configuration is needed — node discovery is handled by `tai/registry`.

```go
sandbox.Init()
```

### M

```go
func M() *Manager
```

Returns the global Manager. Panics if `Init` was not called.

```go
mgr := sandbox.M()
```

---

## Node Discovery

Sandbox V2 no longer uses static node configuration. Nodes are discovered dynamically
through `tai/registry`. Each Tai node registers itself with a unique **TaiID** (e.g.
`"192.168.1.10-19100"` for direct mode, `"local"` for Docker). The TaiID is used as the
`NodeID` identifier in `CreateOptions`, `ListOptions`, `Host()`, `ImageExists()`, etc.

---

## Lifecycle Policies

```go
type LifecyclePolicy string

const (
    OneShot     LifecyclePolicy = "oneshot"     // removed after first Exec
    Session     LifecyclePolicy = "session"     // removed after idle timeout
    LongRunning LifecyclePolicy = "longrunning" // stopped after idle, removed after max lifetime
    Persistent  LifecyclePolicy = "persistent"  // never auto-cleaned
)
```

---

## Manager

### Start

```go
func (m *Manager) Start(ctx context.Context) error
```

Recovers existing containers from all nodes and starts the background cleanup loop (1 min interval).

```go
ctx := context.Background()
err := sandbox.M().Start(ctx)
```

### Close

```go
func (m *Manager) Close() error
```

Stops the cleanup loop and closes all node connections.

### Create

```go
func (m *Manager) Create(ctx context.Context, opts CreateOptions) (*Box, error)
```

Creates and starts a new sandbox container. Returns a `Box` handle.

```go
box, err := sandbox.M().Create(ctx, sandbox.CreateOptions{
    Image:   "alpine:latest",
    Owner:   "user-123",
    NodeID:  "192.168.1.10-19100",  // TaiID from registry
    Policy:  sandbox.Session,
    WorkDir: "/workspace",
    Env:     map[string]string{"LANG": "en_US.UTF-8"},
    Memory:  512 * 1024 * 1024, // 512MB
    CPUs:    1.0,
    VNC:     true,
    Labels:  map[string]string{"project": "demo"},
    Ports: []sandbox.PortMapping{
        {ContainerPort: 8080, HostPort: 0, Protocol: "tcp"},
    },
    IdleTimeout: 15 * time.Minute,
    StopTimeout: 3 * time.Second,
    WorkspaceID: "ws-abc",
    MountMode:   "rw",
    MountPath:   "/workspace",
})
```

### Host

```go
func (m *Manager) Host(ctx context.Context, nodeID string) (*Host, error)
```

Returns a `Host` handle for the given node (identified by TaiID). Unlike `Create`, no
container is provisioned — the Host is available as long as the Tai server reports
`host_exec` capability. Returns `ErrNodeNotFound` if the TaiID is not registered,
`ErrNodeMissing` if the nodeID argument is empty, or an error if the node has no `host_exec`.

```go
host, err := sandbox.M().Host(ctx, "192.168.1.10-19100")
```

### Get

```go
func (m *Manager) Get(ctx context.Context, id string) (*Box, error)
```

Returns an existing sandbox by ID. Returns `ErrNotFound` if absent.

```go
box, err := sandbox.M().Get(ctx, "sb-12345")
```

### GetOrCreate

```go
func (m *Manager) GetOrCreate(ctx context.Context, opts CreateOptions) (*Box, error)
```

Returns existing sandbox by `opts.ID` or creates a new one.

```go
box, err := sandbox.M().GetOrCreate(ctx, sandbox.CreateOptions{
    ID:    "sb-session-xyz",
    Image: "alpine:latest",
    Owner: "user-123",
})
```

### List

```go
func (m *Manager) List(ctx context.Context, opts ListOptions) ([]*Box, error)
```

Returns all sandboxes matching the given filters. Empty fields = no filter.

```go
boxes, err := sandbox.M().List(ctx, sandbox.ListOptions{
    Owner: "user-123",
    NodeID: "192.168.1.10-19100",
    Labels: map[string]string{"project": "demo"},
})
```

### Remove

```go
func (m *Manager) Remove(ctx context.Context, id string) error
```

Force-removes a sandbox (SIGKILL + delete).

```go
err := sandbox.M().Remove(ctx, "sb-12345")
```

### Cleanup

```go
func (m *Manager) Cleanup(ctx context.Context) error
```

Removes idle/expired sandboxes based on lifecycle policies. Called automatically by
the cleanup loop, but can also be invoked manually.

### Heartbeat

```go
func (m *Manager) Heartbeat(sandboxID string, active bool, processCount int) error
```

Updates a sandbox's last-active timestamp. Called by the gRPC heartbeat service.

```go
err := sandbox.M().Heartbeat("sb-12345", true, 3)
```

### Nodes

```go
func (m *Manager) Nodes() []registry.NodeSnapshot
```

Returns all registered Tai nodes from the `tai/registry`.

```go
for _, n := range sandbox.M().Nodes() {
    fmt.Printf("tai_id=%s mode=%s addr=%s status=%s\n",
        n.TaiID, n.Mode, n.Addr, n.Status)
}
```

### ImageExists

```go
func (m *Manager) ImageExists(ctx context.Context, nodeID, ref string) (bool, error)
```

Reports whether the given image ref exists on the target node.
Returns `(true, nil)` when the node has no image service (e.g. K8s — kubelet handles pulls).

```go
exists, err := sandbox.M().ImageExists(ctx, "192.168.1.10-19100", "alpine:latest")
```

### PullImage

```go
func (m *Manager) PullImage(ctx context.Context, nodeID, ref string, opts ImagePullOptions) (<-chan taisandbox.PullProgress, error)
```

Pulls an image to the target node. Returns a channel of `taisandbox.PullProgress`
(from `github.com/yaoapp/yao/tai/sandbox`). Returns `(nil, nil)` when the node has no image
service (e.g. K8s).

`PullProgress` fields: `Status string`, `Layer string`, `Current int64`, `Total int64`, `Error string`.

```go
ch, err := sandbox.M().PullImage(ctx, "192.168.1.10-19100", "myapp:v2", sandbox.ImagePullOptions{
    Auth: &sandbox.RegistryAuth{
        Username: "user",
        Password: "pass",
        Server:   "registry.example.com",
    },
})
for p := range ch {
    fmt.Printf("pull: %s layer=%s %d/%d\n", p.Status, p.Layer, p.Current, p.Total)
}
```

### EnsureImage

```go
func (m *Manager) EnsureImage(ctx context.Context, nodeID, ref string, opts ImagePullOptions) error
```

Checks if the image exists; if not, pulls it and blocks until complete.

```go
err := sandbox.M().EnsureImage(ctx, "192.168.1.10-19100", "alpine:latest", sandbox.ImagePullOptions{})
```

---

## Box

A `Box` is a handle to a running sandbox container.

### Accessors

```go
func (b *Box) ID() string
func (b *Box) Owner() string
func (b *Box) ContainerID() string
func (b *Box) NodeID() string
func (b *Box) WorkspaceID() string
```

### Exec

```go
func (b *Box) Exec(ctx context.Context, cmd []string, opts ...ExecOption) (*ExecResult, error)
```

Runs a command and waits for completion. If the box policy is `OneShot`, the box is
auto-removed after execution.

```go
result, err := box.Exec(ctx, []string{"python3", "-c", "print('hello')"},
    sandbox.WithWorkDir("/workspace"),
    sandbox.WithEnv(map[string]string{"PYTHONPATH": "/lib"}),
    sandbox.WithTimeout(30*time.Second),
)
fmt.Printf("exit=%d stdout=%s stderr=%s\n", result.ExitCode, result.Stdout, result.Stderr)
```

### Stream

```go
func (b *Box) Stream(ctx context.Context, cmd []string, opts ...ExecOption) (*ExecStream, error)
```

Runs a command with real-time streaming I/O.

```go
stream, err := box.Stream(ctx, []string{"bash"})
go io.Copy(os.Stdout, stream.Stdout)
go io.Copy(os.Stderr, stream.Stderr)
fmt.Fprintln(stream.Stdin, "echo hello")
stream.Stdin.Close()
exitCode, _ := stream.Wait()
```

### Attach

```go
func (b *Box) Attach(ctx context.Context, port int, opts ...AttachOption) (*ServiceConn, error)
```

Connects to a service running inside the sandbox via WebSocket proxy.

```go
conn, err := box.Attach(ctx, 8080,
    sandbox.WithProtocol("ws"),
    sandbox.WithPath("/api/stream"),
    sandbox.WithHeaders(map[string]string{"Authorization": "Bearer xxx"}),
)
defer conn.Close()
conn.Write([]byte(`{"action":"subscribe"}`))
data, _ := conn.Read()
```

### VNC

```go
func (b *Box) VNC(ctx context.Context) (string, error)
```

Returns the VNC WebSocket URL for the sandbox (requires `VNC: true` at creation).

```go
url, err := box.VNC(ctx)
// url = "ws://tai-host:16080/vnc/xxx/ws"
```

### Proxy

```go
func (b *Box) Proxy(ctx context.Context, port int, path string) (string, error)
```

Returns the HTTP proxy URL for a service on the given port.

```go
url, err := box.Proxy(ctx, 3000, "/api/health")
// url = "http://tai-host:8099/container-id:3000/api/health"
```

### Workspace

```go
func (b *Box) Workspace() workspace.FS
```

Returns a `workspace.FS` interface (`github.com/yaoapp/yao/tai/workspace`) for file
operations on the sandbox's workspace volume. The interface embeds `fs.FS`, `fs.StatFS`,
`fs.ReadFileFS`, `fs.ReadDirFS`, `io.Closer`, and adds write methods (`WriteFile`,
`Remove`, `RemoveAll`, `Rename`, `MkdirAll`).

```go
ws := box.Workspace()
data, _ := ws.ReadFile("main.py")
ws.WriteFile("output.txt", []byte("result"), 0644)
ws.MkdirAll("src/pkg", 0755)
ws.Remove("tmp.log")
```

### Start / Stop / Remove

```go
func (b *Box) Start(ctx context.Context) error
func (b *Box) Stop(ctx context.Context) error
func (b *Box) Remove(ctx context.Context) error
```

```go
box.Stop(ctx)   // SIGTERM with grace period, then SIGKILL
box.Start(ctx)  // restart a stopped sandbox
box.Remove(ctx) // force remove
```

### Info

```go
func (b *Box) Info(ctx context.Context) (*BoxInfo, error)
```

Returns current sandbox status from the underlying container runtime.

```go
info, err := box.Info(ctx)
fmt.Printf("status=%s processes=%d vnc=%v created=%s\n",
    info.Status, info.ProcessCount, info.VNC, info.CreatedAt)
```

---

## Host

A `Host` represents a Tai host machine execution environment, distinct from `Box` (containers).
No `Create` call is needed — a Host is available as long as the node's Tai server reports `host_exec`.

### Accessors

```go
func (h *Host) NodeID() string
```

### Exec

```go
func (h *Host) Exec(ctx context.Context, cmd string, args []string, opts ...HostExecOption) (*HostExecResult, error)
```

Runs a command directly on the Tai host machine via HostExec gRPC.

```go
host, _ := sandbox.M().Host(ctx, "192.168.1.10-19100")
result, err := host.Exec(ctx, "git", []string{"status"},
    sandbox.WithHostWorkDir("/data/repos/project"),
    sandbox.WithHostEnv(map[string]string{"GIT_AUTHOR_NAME": "bot"}),
    sandbox.WithHostTimeout(10000),        // 10s
    sandbox.WithHostMaxOutput(1024*1024),   // 1MB
)
fmt.Printf("exit=%d stdout=%s duration=%dms\n",
    result.ExitCode, string(result.Stdout), result.DurationMs)
```

### Stream

```go
func (h *Host) Stream(ctx context.Context, cmd string, args []string, opts ...HostExecOption) (*HostExecStream, error)
```

Runs a command on the Tai host and streams stdout/stderr in real time via HostExec gRPC
ExecStream. Returns a `HostExecStream` with separate channels for stdout and stderr.

```go
host, _ := sandbox.M().Host(ctx, "192.168.1.10-19100")
stream, err := host.Stream(ctx, "tail", []string{"-f", "/var/log/app.log"},
    sandbox.WithHostWorkDir("/data"),
    sandbox.WithHostTimeout(60000),
)
go func() {
    for chunk := range stream.Stderr {
        fmt.Fprintf(os.Stderr, "%s", chunk)
    }
}()
for chunk := range stream.Stdout {
    fmt.Printf("%s", chunk)
}
exitCode, err := stream.Wait()
```

To cancel a long-running stream early:

```go
stream.Cancel()
```

### Workspace

```go
func (h *Host) Workspace(sessionID string) workspace.FS
```

Returns a `workspace.FS` for the given session on the host. Files are stored under
`dataDir/{sessionID}/` on the Tai host, accessed via Volume gRPC (independent of container
bind mounts).

```go
ws := host.Workspace("ws-abc")
ws.WriteFile("input.txt", []byte("data"), 0644)
data, _ := ws.ReadFile("output.txt")
entries, _ := ws.ReadDir(".")
```

---

## ExecOption Functions

```go
func WithWorkDir(dir string) ExecOption
func WithEnv(env map[string]string) ExecOption
func WithTimeout(timeout time.Duration) ExecOption
```

## AttachOption Functions

```go
func WithProtocol(protocol string) AttachOption  // "ws" (default) or "sse"
func WithPath(path string) AttachOption           // URL path on the target service
func WithHeaders(headers map[string]string) AttachOption
```

## HostExecOption Functions

```go
func WithHostWorkDir(dir string) HostExecOption
func WithHostEnv(env map[string]string) HostExecOption
func WithHostStdin(data []byte) HostExecOption
func WithHostTimeout(ms int64) HostExecOption
func WithHostMaxOutput(bytes int64) HostExecOption
```

---

## Types

### CreateOptions

```go
type CreateOptions struct {
    ID          string
    Owner       string
    Labels      map[string]string
    NodeID      string              // TaiID from registry (required unless WorkspaceID routes to a node)
    Image       string              // required
    WorkDir     string              // default "/workspace"
    User        string              // container user
    Env         map[string]string
    Memory      int64               // bytes; 0 = unlimited
    CPUs        float64             // 0 = unlimited
    VNC         bool
    Ports       []PortMapping
    Policy      LifecyclePolicy     // default Session
    IdleTimeout time.Duration       // 0 = no idle cleanup
    MaxLifetime time.Duration       // 0 = no max lifetime
    StopTimeout time.Duration       // SIGTERM grace period; 0 = DefaultStopTimeout (2s)
    WorkspaceID string              // workspace to mount; empty = none
    MountMode   string              // "rw" (default) or "ro"
    MountPath   string              // default "/workspace"
}
```

### ListOptions

```go
type ListOptions struct {
    Owner  string
    NodeID string
    Labels map[string]string
}
```

### PortMapping

```go
type PortMapping struct {
    ContainerPort int
    HostPort      int    // 0 = auto-assign
    HostIP        string
    Protocol      string // "tcp" (default), "udp"
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

### ExecStream

```go
type ExecStream struct {
    Stdout io.ReadCloser
    Stderr io.ReadCloser
    Stdin  io.WriteCloser
    Wait   func() (int, error) // blocks until exit; returns exit code
    Cancel func()              // kills the process
}
```

### ServiceConn

```go
type ServiceConn struct {
    Read   func() ([]byte, error)
    Write  func(data []byte) error
    Events <-chan []byte
    URL    string
    Close  func() error
}
```

### BoxInfo

```go
type BoxInfo struct {
    ID           string
    ContainerID  string
    NodeID       string
    Owner        string
    Status       string          // "running", "stopped", etc.
    Policy       LifecyclePolicy
    Labels       map[string]string
    Image        string
    CreatedAt    time.Time
    LastActive   time.Time
    ProcessCount int
    VNC          bool
}
```

### ImagePullOptions / RegistryAuth

```go
type ImagePullOptions struct {
    Auth *RegistryAuth // nil = anonymous
}

type RegistryAuth struct {
    Username string
    Password string
    Server   string
}
```

### HostExecResult

```go
type HostExecResult struct {
    ExitCode   int
    Stdout     []byte
    Stderr     []byte
    DurationMs int64
    Error      string
    Truncated  bool
}
```

### HostExecStream

```go
type HostExecStream struct {
    Stdout <-chan []byte
    Stderr <-chan []byte
    Wait   func() (int, error) // blocks until exit; returns exit code
    Cancel func()              // cancels the stream context
}
```

---

## Errors

```go
var (
    ErrNotAvailable = errors.New("sandbox: not available (no nodes registered)")
    ErrNotFound     = errors.New("sandbox: not found")
    ErrNodeNotFound = errors.New("sandbox: node not found")
    ErrNodeMissing  = errors.New("sandbox: node ID is required")
)
```

---

## Helper Functions

### BuildGRPCEnv

```go
func BuildGRPCEnv(mode, addr, sandboxID string) map[string]string
```

Builds environment variables injected into sandbox containers. The gRPC port is read from
`config.Conf.GRPC.Port` (defaults to `9099`).

- `mode` — the `TaiNode.Mode` (`"local"`, `"direct"`, `"tunnel"`)
- `addr` — the `TaiNode.Addr` (e.g. `"tai://192.168.1.10:19100"` for direct mode)
- `sandboxID` — the container's sandbox identifier

| Variable         | Description                        |
|------------------|------------------------------------|
| `YAO_SANDBOX_ID` | Sandbox identifier                 |
| `YAO_GRPC_ADDR`  | gRPC server address (auto-derived) |

Address derivation logic:
- `local` → `host.docker.internal:<grpcPort>`
- `direct` with `tai://host:port` → `host:port`
- `tunnel` → `127.0.0.1:<grpcPort>`

Token injection (`YAO_TOKEN`, `YAO_REFRESH_TOKEN`) is the **caller's responsibility** via
`CreateOptions.Env`. See IMPL.md "OAuth Decoupling" for details.
