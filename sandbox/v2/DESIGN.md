# Sandbox V2 Design

## Positioning

Sandbox is a **standalone infrastructure module** in Yao, on the same level as `process`, `store`, and `fs`. It provides isolated execution environments with standard file I/O. Any module can use it — Agent, JSAPI scripts, Process handlers, API endpoints.

```
Yao Infrastructure
├── process    — process execution
├── store      — KV storage
├── fs         — host filesystem
├── stream     — streaming execution (planned)
├── workspace  — persistent user storage ← new in V2
└── sandbox    — isolated execution environments ← this module
```

Sandbox does NOT import or depend on Agent. Agent is one of many consumers.

## Architecture

```
┌─────────────────────────────────────────────────┐
│  Consumers (know nothing about tai/Docker/K8s)  │
│  ├── JSAPI: sandbox.Create/Get/List/Delete      │
│  ├── JSAPI: workspace.Create/Get/List/Delete     │
│  ├── Process: sandbox.Create, sandbox.Exec      │
│  ├── Agent: uses sandbox via interface           │
│  └── API: /api/__yao/sandbox/*                   │
└──────────────────┬──────────────────────────────┘
                   │ sandbox.Manager (public API)
                   ▼
┌─────────────────────────────────────────────────┐
│  sandbox/v2                                      │
│                                                  │
│  Manager (global singleton)                      │
│  ├── Create / Get / GetOrCreate / List / Remove │
│  ├── Start / Cleanup / Close                    │
│  ├── Heartbeat (idle tracking)                  │
│  ├── AddPool / RemovePool / Pools               │
│  ├── SetWorkspaceManager (workspace integration)│
│  ├── EnsureImage / ImageExists / PullImage      │
│  └── guard rails (limits, TTL) + Box factory    │
│                                                  │
│  Box (per-instance)                              │
│  ├── Exec(cmd) → ExecResult                     │
│  ├── Stream(cmd) → ExecStream (real-time I/O)   │
│  ├── Attach(port) → ServiceConn (WS/SSE)        │
│  ├── Workspace() → workspace.FS                 │
│  ├── VNC() → url                                │
│  ├── Proxy(port) → url                          │
│  └── Start / Stop / Remove / Info               │
└──────────────────┬──────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────┐
│  tai.Client pool (lazy-initialized)             │
│  ├── "local"  → tai.New("local")      (Docker)  │
│  ├── "gpu"    → tai.New("tai://gpu")  (Remote)  │
│  ├── "k8s"    → tai.New("tai://k8s",K8s)(K8s)  │
│  └── ...                                        │
│                                                  │
│  Each tai.Client provides:                       │
│  ├── Sandbox()   → CRUD + Exec + ExecStream     │
│  ├── Image()     → Exists + Pull + Remove + List│
│  ├── Volume()    → file I/O (local disk / gRPC) │
│  ├── Workspace() → fs.FS                        │
│  ├── Proxy()     → URL resolve + Connect        │
│  └── VNC()       → VNC WebSocket                │
└─────────────────────────────────────────────────┘
```

## Dependency Rules

```
sandbox/v2 → tai           ✓  (sole runtime dependency)
sandbox/v2 → workspace     ✓  (workspace integration, optional)
sandbox/v2 → agent         ✗  NEVER
sandbox/v2 → docker        ✗  NEVER (tai handles it)
agent      → sandbox/v2    ✓  (consumer, via Manager API)
jsapi      → sandbox/v2    ✓  (consumer, via Manager API)
process    → sandbox/v2    ✓  (consumer, via Manager API)
```

## Manager

Global singleton. Manages a **pool of named `tai.Client` connections** — each pool entry targets a different runtime endpoint (local Docker, remote Tai, K8s cluster). Caller picks which pool to create a sandbox on.

### Pool

```go
type Pool struct {
    Name        string
    Addr        string        // tai.New() address: "local", "tai://host", "docker:///path"
    Options     []tai.Option  // tai.K8s, tai.WithKubeConfig(), tai.WithPorts(), etc.
    MaxPerUser  int           // max boxes per user on this pool, 0 = unlimited
    MaxTotal    int           // max boxes total on this pool, 0 = unlimited
    IdleTimeout time.Duration // 0 = no timeout
    MaxLifetime time.Duration // 0 = no limit
    StopTimeout time.Duration // SIGTERM grace period before SIGKILL; 0 = DefaultStopTimeout (2s)
}
```

Example configuration:

```
pool:
  - name: local
    addr: "local"
    max_total: 20
    idle_timeout: 30m

  - name: gpu
    addr: "tai://gpu-server.internal"
    max_per_user: 1
    max_total: 4
    idle_timeout: 10m
    max_lifetime: 2h

  - name: k8s
    addr: "tai://k8s-proxy.internal"
    max_total: 100
    idle_timeout: 1h
    options:
      runtime: k8s
      kubeconfig: /etc/yao/kubeconfig.yml
```

### Initialization

```go
var mgr *Manager

func Init(cfg Config) error   // create Manager from Config; at least one Pool required
func M() *Manager             // return global singleton; panics if Init not called
```

Startup sequence in `cmd/start.go`:

```
config.Load
sandbox.Init(config.Conf.Sandbox)  // create Manager with pool + guard rails
engine.Load
...
service.Start                      // HTTP
grpc.Start                         // gRPC
sandbox.M().Start(ctx)             // discover existing containers, start cleanup loop
```

`Init` creates the Manager from config (pool definitions + guard rails). `Start` connects to pools, discovers existing containers, and starts the cleanup loop. Two-step so that gRPC server is ready before Start.

Pool connections are created lazily on first use and reused across all Box instances.

### Config

```go
type Config struct {
    Pool []Pool
}
```

Container gRPC env vars (`YAO_GRPC_ADDR`, `YAO_GRPC_UPSTREAM`, etc.) are derived automatically at creation time. Per-instance settings (image, memory, CPU, workdir, env, pool) are passed via `CreateOptions`.

### Core API

```go
type Manager struct {
    pool        map[string]*tai.Client // name → connection (lazy-initialized)
    poolDefs    []Pool
    defaultPool string                 // first pool name
    config      Config
    boxes       sync.Map               // id → *Box
    mu          sync.Mutex
    cancel      context.CancelFunc
    grpcPort    int
    wsManager   *workspace.Manager     // optional workspace integration
}

// --- Bootstrap ---
func (m *Manager) Start(ctx context.Context) error
func (m *Manager) Close() error
func (m *Manager) SetGRPCPort(port int)
func (m *Manager) SetWorkspaceManager(wm *workspace.Manager)

// --- Pool management ---
func (m *Manager) AddPool(ctx context.Context, p Pool) error
func (m *Manager) RemovePool(ctx context.Context, name string, force bool) error
func (m *Manager) Pools() []PoolInfo

// --- Heartbeat ---
func (m *Manager) Heartbeat(sandboxID string, active bool, processCount int) error

// --- CRUD ---
func (m *Manager) Create(ctx context.Context, opts CreateOptions) (*Box, error)
func (m *Manager) Get(ctx context.Context, id string) (*Box, error)
func (m *Manager) GetOrCreate(ctx context.Context, opts CreateOptions) (*Box, error)
func (m *Manager) List(ctx context.Context, opts ListOptions) ([]*Box, error)
func (m *Manager) Remove(ctx context.Context, id string) error
func (m *Manager) Cleanup(ctx context.Context) error

// --- Image management ---
func (m *Manager) ImageExists(ctx context.Context, pool, ref string) (bool, error)
func (m *Manager) PullImage(ctx context.Context, pool, ref string, opts ImagePullOptions) (<-chan PullProgress, error)
func (m *Manager) EnsureImage(ctx context.Context, pool, ref string, opts ImagePullOptions) error
```

### CreateOptions

```go
type CreateOptions struct {
    ID          string
    Owner       string
    Labels      map[string]string
    Pool        string              // which tai.Client to use; empty = default pool

    // Container spec
    Image       string              // required
    WorkDir     string              // default "/workspace"
    User        string
    Env         map[string]string
    Memory      int64               // bytes, 0 = no limit
    CPUs        float64             // 0 = no limit
    VNC         bool
    Ports       []PortMapping

    // Lifecycle
    Policy      LifecyclePolicy     // default: Session
    IdleTimeout time.Duration       // override pool default; 0 = use pool default
    StopTimeout time.Duration       // SIGTERM grace period; 0 = pool default or DefaultStopTimeout

    // Workspace integration
    WorkspaceID string              // workspace to mount; empty = no workspace
    MountMode   string              // "rw" (default) or "ro"
    MountPath   string              // container path; default "/workspace"
}
```

When `WorkspaceID` is set, the Manager resolves the workspace's bound node via `workspace.Manager.NodeForWorkspace()` and forces the container onto that node. The workspace directory is bind-mounted into the container at `MountPath`.

### LifecyclePolicy

```go
type LifecyclePolicy string

const (
    OneShot     LifecyclePolicy = "oneshot"     // destroyed after first Exec
    Session     LifecyclePolicy = "session"     // alive while active, cleaned on idle
    LongRunning LifecyclePolicy = "longrunning" // user workspace, extended TTL
    Persistent  LifecyclePolicy = "persistent"  // never auto-cleaned
)

const DefaultStopTimeout = 2 * time.Second
```

## Box

A `Box` is a single sandbox instance. All operations go through it.

```go
type Box struct {
    id            string
    containerID   string
    pool          string
    owner         string
    policy        LifecyclePolicy
    labels        map[string]string
    lastCall      atomic.Int64  // last external API call
    lastHeartbeat atomic.Int64  // last container heartbeat
    processCount  atomic.Int32  // user processes inside container
    idleTimeoutD  time.Duration
    stopTimeoutD  time.Duration
    createdAt     time.Time
    refreshToken  string
    vnc           bool
    image         string
    workspaceID   string
    ws            workspace.FS  // lazy-initialized, cached
    manager       *Manager
}

// --- Identity ---
func (b *Box) ID() string
func (b *Box) Owner() string
func (b *Box) ContainerID() string
func (b *Box) Pool() string
func (b *Box) WorkspaceID() string

// --- Execution ---
func (b *Box) Exec(ctx context.Context, cmd []string, opts ...ExecOption) (*ExecResult, error)
func (b *Box) Stream(ctx context.Context, cmd []string, opts ...ExecOption) (*ExecStream, error)
func (b *Box) Attach(ctx context.Context, port int, opts ...AttachOption) (*ServiceConn, error)

// --- Filesystem ---
func (b *Box) Workspace() workspace.FS

// --- Network ---
func (b *Box) VNC(ctx context.Context) (string, error)
func (b *Box) Proxy(ctx context.Context, port int, path string) (string, error)

// --- Lifecycle ---
func (b *Box) Start(ctx context.Context) error
func (b *Box) Stop(ctx context.Context) error
func (b *Box) Remove(ctx context.Context) error
func (b *Box) Info(ctx context.Context) (*BoxInfo, error)
```

### ExecOption / ExecResult / ExecStream

```go
type ExecOption func(*execConfig)

func WithWorkDir(dir string) ExecOption
func WithEnv(env map[string]string) ExecOption
func WithTimeout(d time.Duration) ExecOption

type ExecResult struct {
    ExitCode int
    Stdout   string
    Stderr   string
}

type ExecStream struct {
    Stdout io.ReadCloser
    Stderr io.ReadCloser
    Stdin  io.WriteCloser
    Wait   func() (int, error) // block until exit, return exit code
    Cancel func()              // kill the process
}
```

### AttachOption / ServiceConn

```go
type AttachOption func(*attachConfig)

func WithProtocol(proto string) AttachOption   // "ws", "sse"; default "ws"
func WithPath(path string) AttachOption
func WithHeaders(h map[string]string) AttachOption

type ServiceConn struct {
    Read   func() ([]byte, error) // read next message (WS mode)
    Write  func(data []byte) error
    Events <-chan []byte           // SSE event channel
    URL    string
    Close  func() error
}
```

`port` is the port the service listens on **inside the container**. Routing — Docker port mapping (local) or Tai HTTP proxy (remote) — is handled internally.

### Image Management

```go
type ImagePullOptions struct {
    Auth *RegistryAuth
}

type RegistryAuth struct {
    Username string
    Password string
    Server   string
}
```

`EnsureImage` first checks `ImageExists`; if not present, calls `PullImage` and blocks until complete. For K8s pools this is a no-op — kubelet manages image pulling natively via `imagePullPolicy`.

### BoxInfo / PoolInfo

```go
type BoxInfo struct {
    ID           string
    ContainerID  string
    Pool         string
    Owner        string
    Status       string // "running", "stopped", "creating"
    Policy       LifecyclePolicy
    Labels       map[string]string
    Image        string
    CreatedAt    time.Time
    LastActive   time.Time
    ProcessCount int
    VNC          bool
}

type PoolInfo struct {
    Name        string
    Addr        string
    Connected   bool
    Boxes       int
    MaxPerUser  int
    MaxTotal    int
    IdleTimeout time.Duration
    MaxLifetime time.Duration
}
```

## Workspace Integration

Sandbox V2 integrates with the workspace module via `Manager.SetWorkspaceManager()` and `CreateOptions.WorkspaceID`:

```go
// Link workspace manager at startup
sbm.SetWorkspaceManager(wsm)

// Create sandbox with workspace mount
box, err := sbm.Create(ctx, sandbox.CreateOptions{
    Image:       "yaoapp/workspace:latest",
    WorkspaceID: "ws-abc123",
    MountMode:   "rw",       // default
    MountPath:   "/workspace", // default
})
```

When `WorkspaceID` is set:
1. Manager calls `workspace.Manager.NodeForWorkspace()` to resolve the workspace's bound node
2. Forces the container onto that node's pool
3. Calls `workspace.Manager.MountPath()` to get the host-side directory
4. Adds a Docker bind mount: `hostPath:mountPath:mode`
5. Box.Workspace() uses the workspace ID as the volume session key

This guarantees that a workspace's container always runs on the same host where its storage lives.

## Container Setup — Manager.Create()

When Manager creates a sandbox, it:

1. Validates `CreateOptions` (Image required)
2. Generates sandbox ID (or uses provided one)
3. Resolves workspace node binding (if WorkspaceID set)
4. Checks user limits (`MaxPerUser`) and total limits (`MaxTotal`)
5. Resolves pool (by name or default)
6. Creates OAuth token pair for container IPC
7. Builds `tai.sandbox.CreateOptions`:
   - Injects management labels: `managed-by`, `sandbox-id`, `sandbox-owner`, `sandbox-pool`, `sandbox-policy`, `workspace-id`
   - Sets container CMD to graceful-shutdown-aware sleep: `sh -c "trap 'exit 0' TERM; while :; do sleep 86400 & wait $!; done"`
   - Merges caller's Env with gRPC env vars (`YAO_SANDBOX_ID`, `YAO_TOKEN`, `YAO_REFRESH_TOKEN`, `YAO_GRPC_ADDR`, etc.)
   - Adds workspace bind mount if WorkspaceID is set
8. Calls `tai.Client.Sandbox().Create()` then `Start()`
9. Wraps in a `Box`, registers in `boxes` map

## Lifecycle Management

### Idle Tracking — Dual Source

```go
box.lastActive = max(lastExternalCall, lastHeartbeat)
```

| Source | What it tracks | Updated by |
|--------|---------------|------------|
| External call | Caller is using the sandbox | `Box.Exec()`, `Box.Stream()`, `Box.Workspace()`, `Box.VNC()`, `Box.Proxy()`, `Box.Attach()` |
| Container heartbeat | Processes running inside the container | gRPC `Heartbeat` RPC |

### Cleanup Loop

Runs every 60 seconds. Policy behavior:

| Policy | Idle | Max Lifetime | Auto |
|--------|------|-------------|------|
| OneShot | — | — | Removed after first Exec completes |
| Session | Remove | Remove | Default for agent chats |
| LongRunning | Stop (keep data) | Remove | User workspaces |
| Persistent | Never | Never | User-managed |

### Container Stop Behavior

`DefaultStopTimeout = 2s`. Docker `ContainerStop` sends SIGTERM, waits the timeout, then SIGKILL. The V2 container CMD (`trap 'exit 0' TERM; ...`) exits immediately on SIGTERM, so actual stop time is near-instant.

`Manager.Remove()` calls `Sandbox().Remove(force=true)` directly (SIGKILL + delete) — no redundant Stop call. This keeps remove latency under 200ms.

## Tai SDK Interface

Sandbox V2 depends on these tai sub-package interfaces:

### tai.Client

```go
func New(addr string, opts ...Option) (*Client, error)
func (c *Client) Sandbox() sandbox.Sandbox
func (c *Client) Image() sandbox.Image
func (c *Client) Volume() volume.Volume
func (c *Client) Workspace(sessionID string) workspace.FS
func (c *Client) Proxy() proxy.Proxy
func (c *Client) VNC() vnc.VNC
func (c *Client) DataDir() string
func (c *Client) IsLocal() bool
func (c *Client) Close() error
```

Address schemes: `"local"` (Docker default), `"docker://..."` (explicit Docker), `"tai://host"` (remote Tai Server). Remote mode auto-discovers service ports via ServerInfo gRPC, with `WithPorts()` taking precedence.

### sandbox.Sandbox

```go
type Sandbox interface {
    Create(ctx, opts CreateOptions) (string, error)
    Start(ctx, id string) error
    Stop(ctx, id string, timeout time.Duration) error
    Remove(ctx, id string, force bool) error
    Exec(ctx, id string, cmd []string, opts ExecOptions) (*ExecResult, error)
    ExecStream(ctx, id string, cmd []string, opts ExecOptions) (*StreamHandle, error)
    Inspect(ctx, id string) (*ContainerInfo, error)
    List(ctx, opts ListOptions) ([]ContainerInfo, error)
    Close() error
}
```

Implementations: `docker_core.go` (local Docker), `docker.go` (remote Docker via Tai proxy), `k8s.go` (Kubernetes via Tai proxy).

### sandbox.Image

```go
type Image interface {
    Exists(ctx, ref string) (bool, error)
    Pull(ctx, ref string, opts PullOptions) (<-chan PullProgress, error)
    Remove(ctx, ref string, force bool) error
    List(ctx) ([]ImageInfo, error)
}
```

Docker implementation pulls via Docker SDK with real-time progress streaming. K8s implementation is a no-op — kubelet handles image pulling.

### proxy.Proxy

```go
type Proxy interface {
    URL(ctx, containerID string, port int, path string) (string, error)
    Connect(ctx, containerID string, opts ConnectOptions) (*Connection, error)
    Healthz(ctx) error
}
```

Local: resolves host ports via `Inspect()`. Remote: routes through Tai HTTP proxy which handles WebSocket upgrade and SSE streaming natively.

## gRPC Token Injection

```go
func CreateContainerTokens(sandboxID, owner string, scopes []string) (access, refresh string, err error)
func RevokeContainerTokens(refresh string) error
func BuildGRPCEnv(pool *Pool, sandboxID, access, refresh string, grpcPort int) map[string]string
```

Environment variables injected into each container:

```
# All modes
YAO_SANDBOX_ID=<sandbox_id>
YAO_TOKEN=<access_token>
YAO_REFRESH_TOKEN=<refresh_token>
YAO_GRPC_ADDR=127.0.0.1:9099

# Remote mode (tai://) adds:
YAO_GRPC_TAI=enable
YAO_GRPC_ADDR=<tai-host>:9100
YAO_GRPC_UPSTREAM=127.0.0.1:9099
```

## Errors

```go
var (
    ErrNotAvailable  = errors.New("sandbox: not available (no pools configured)")
    ErrNotFound      = errors.New("sandbox: not found")
    ErrLimitExceeded = errors.New("sandbox: limit exceeded")
    ErrPoolNotFound  = errors.New("sandbox: pool not found")
    ErrPoolInUse     = errors.New("sandbox: pool has running boxes")
)
```

## Package Structure

```
sandbox/v2/
├── sandbox.go              // Init, M(), global singleton
├── manager.go              // Manager: CRUD, pool management, image ops, cleanup
├── box.go                  // Box: Exec, Stream, Attach, Workspace, VNC, Proxy, lifecycle
├── types.go                // CreateOptions, ExecResult, ExecStream, ServiceConn, BoxInfo, etc.
├── config.go               // Config struct
├── errors.go               // sentinel errors
├── grpc.go                 // token creation/revocation, gRPC env var injection
├── jsapi/                  // (Phase 2) V8 JSAPI sandbox.* namespace
│   ├── jsapi.go            // RegisterObject("sandbox"), Create/Get/List/Delete
│   └── box.go              // Box JS object: Exec/Attach/VNC/Proxy/Workspace/Info/Start/Stop/Remove
├── export_test.go          // ResetForTest() for test isolation
├── testutils_test.go       // shared test helpers (multi-pool setup)
├── sandbox_test.go         // Init/M singleton tests
├── manager_test.go         // Manager CRUD tests
├── manager_lifecycle_test.go // Heartbeat, Cleanup, idle tracking tests
├── box_test.go             // Box Exec/Workspace/Info tests
├── box_attach_test.go      // Attach WS/SSE/VNC tests
├── box_workspace_test.go   // Workspace integration tests
├── box_image_test.go       // Image Pull API tests
├── bench_test.go           // Performance benchmarks
├── grpc_test.go            // Token/env building tests
├── DESIGN.md               // this document
└── IMPL.md                 // implementation status and plan
```

---

# Workspace Module

## Positioning

Workspace is a **top-level module** (`workspace/`), parallel to `sandbox/v2`. It provides persistent, user-managed storage that is decoupled from container lifecycle. Workspaces are pinned to a specific Tai node; containers referencing a workspace are automatically routed to that node.

```
┌─────────────────────┐     ┌─────────────────────┐
│    sandbox/v2        │     │     workspace        │
│  (container runtime) │◄────│  (persistent storage)│
│                      │     │                      │
│  CreateOptions {     │     │  CRUD + File I/O     │
│    WorkspaceID ──────┼────►│  Node binding        │
│  }                   │     │  fs.FS interface      │
└──────────┬───────────┘     └──────────┬───────────┘
           │                            │
           └──────────┬─────────────────┘
                      ▼
              tai.Client pool
```

## Core Types

```go
type Workspace struct {
    ID        string
    Name      string
    Owner     string
    Node      string              // Tai node this workspace is pinned to
    Labels    map[string]string
    CreatedAt time.Time
    UpdatedAt time.Time
}

type CreateOptions struct {
    ID     string                 // explicit ID; empty = auto-generate (ws-<uuid>)
    Name   string
    Owner  string
    Node   string                 // target Tai node (required)
    Labels map[string]string
}

type ListOptions struct {
    Owner string
    Node  string
}

type UpdateOptions struct {
    Name   *string               // nil = no change
    Labels map[string]string     // nil = no change; non-nil replaces all labels
}

type NodeInfo struct {
    Name   string
    Addr   string
    Online bool
}

type DirEntry struct {
    Name  string
    IsDir bool
    Size  int64
}
```

## Manager API

```go
type Manager struct {
    pools map[string]*tai.Client
    mu    sync.RWMutex
}

func NewManager(pools map[string]*tai.Client) *Manager

// --- CRUD ---
func (m *Manager) Create(ctx, opts CreateOptions) (*Workspace, error)
func (m *Manager) Get(ctx, id string) (*Workspace, error)
func (m *Manager) List(ctx, opts ListOptions) ([]*Workspace, error)
func (m *Manager) Update(ctx, id string, opts UpdateOptions) (*Workspace, error)
func (m *Manager) Delete(ctx, id string, force bool) error

// --- File I/O ---
func (m *Manager) ReadFile(ctx, id string, path string) ([]byte, error)
func (m *Manager) WriteFile(ctx, id string, path string, data []byte, perm os.FileMode) error
func (m *Manager) ListDir(ctx, id string, path string) ([]DirEntry, error)
func (m *Manager) Remove(ctx, id string, path string) error
func (m *Manager) FS(ctx, id string) (workspace.FS, error)

// --- Node management ---
func (m *Manager) Nodes() []NodeInfo
func (m *Manager) AddPool(name string, client *tai.Client)
func (m *Manager) RemovePool(name string)

// --- Sandbox integration ---
func (m *Manager) NodeForWorkspace(ctx, id string) (string, error)
func (m *Manager) MountPath(ctx, id string) (string, error)
```

## Metadata Storage

Workspace metadata is stored as `.workspace.json` inside the workspace's root directory on the Tai node:

```
<data-dir>/
├── ws-abc123/
│   ├── .workspace.json   ← metadata (ID, Name, Owner, Node, Labels, timestamps)
│   ├── src/
│   ├── go.mod
│   └── ...
├── ws-def456/
│   └── ...
```

This approach collocates metadata with data — no external database required. `List()` scans top-level directories and reads each `.workspace.json`. `Get()` scans all nodes until the workspace is found.

## Errors

```go
var (
    ErrNotFound    = errors.New("workspace: not found")
    ErrNodeMissing = errors.New("workspace: node is required")
    ErrNodeOffline = errors.New("workspace: node is offline or not configured")
    ErrHasMounts   = errors.New("workspace: workspace has active container mounts")
)
```

## Package Structure

```
workspace/
├── workspace.go         // types, metadata marshal/unmarshal
├── manager.go           // Manager: CRUD, file I/O, node management
├── errors.go            // sentinel errors
├── jsapi/               // (Phase 2) V8 JSAPI workspace.* namespace
│   ├── jsapi.go         // RegisterObject("workspace"), Create/Get/List/Delete
│   └── fs.go            // WorkspaceFS JS object: ReadFile/WriteFile/ReadDir/Stat/MkdirAll/Remove/RemoveAll/Rename
├── testutils_test.go    // shared test helpers
├── workspace_test.go    // CRUD tests (Create/Get/List/Update/Delete/Nodes)
├── fileio_test.go       // File I/O + fs.FS tests
├── bench_test.go        // Performance benchmarks
└── DESIGN.md            // detailed design document
```

---

# Testing

## Test Environment

Three pool modes configured via environment variables:

```bash
# Local — direct Docker daemon (always available)
SANDBOX_TEST_LOCAL_ADDR=local

# Remote — via Tai container (Docker backend)
SANDBOX_TEST_REMOTE_ADDR=tai://127.0.0.1:9100

# K8s — via Tai container (K8s backend)
TAI_TEST_K8S_HOST=<tai-host>
TAI_TEST_KUBECONFIG=<path>
TAI_TEST_K8S_PORT=6443
TAI_TEST_K8S_NAMESPACE=default

# Test image
SANDBOX_TEST_IMAGE=yaoapp/sandbox-v2-test:latest
```

Tests skip unavailable modes via `t.Skip`. Both sandbox/v2 and workspace tests iterate over all available pools.

## Test Coverage

### sandbox/v2

| File | Coverage |
|------|----------|
| `sandbox_test.go` | `Init()`, `M()`, singleton behavior |
| `manager_test.go` | Create, Get, GetOrCreate, List, Remove, pool management, limits |
| `manager_lifecycle_test.go` | Start (container discovery), Cleanup, idle tracking, Heartbeat |
| `box_test.go` | Exec, Info, Workspace (ReadFile/WriteFile), lifecycle |
| `box_attach_test.go` | Attach WS, Attach SSE, VNC URL, VNC Connect |
| `box_workspace_test.go` | Workspace file I/O through Box, workspace mount integration |
| `box_image_test.go` | ImageExists, PullImage (with progress), EnsureImage, K8s no-op |
| `grpc_test.go` | Token creation/revocation, env var building |
| `bench_test.go` | ContainerLifecycle, Create, Exec, ExecHeavy, Remove, Info, StopStart, WorkspaceReadWrite |

### workspace

| File | Coverage |
|------|----------|
| `workspace_test.go` | Create (auto/explicit ID, labels, invalid node), Get, List (filter owner/node), Update (name/labels), Delete, Nodes, NodeForWorkspace, AddPool, RemovePool, MountPath |
| `fileio_test.go` | ReadWriteFile, nested paths, ListDir, Remove, fs.FS (ReadFile, WriteFile, MkdirAll, Rename, WalkDir, Remove) |
| `bench_test.go` | WriteFile, ReadFile, ReadWriteCycle, WriteLargeFile, ListDir, FSWalkDir, CreateDelete |

## CI Integration

Consolidated into two CI jobs:

| Job | Contents |
|-----|----------|
| `SandboxV2Test` | Image pre-pull → tai-test → sandbox/v2 (local+remote+k8s) → workspace (local+remote) |
| `BenchmarkSandboxV2` | Performance tests for sandbox/v2 + workspace (parallel with SandboxV2Test) |

## Benchmark Results (Reference)

| Benchmark | Local | Remote | K8s |
|-----------|-------|--------|-----|
| ContainerLifecycle | ~300ms | ~200ms | ~10s |
| Create | ~100ms | ~80ms | ~8s |
| Exec | ~30ms | ~50ms | ~150ms |
| Remove | ~180ms | ~120ms | ~220ms |
| Info | ~5ms | ~10ms | ~30ms |
| StopStart | ~2.2s | ~2.2s | N/A (skip) |

K8s `StopStart` is skipped because K8s `Stop` deletes the Pod; a subsequent `Start` cannot restart a deleted Pod.

Docker `StopStart` ~2.2s is expected: `DefaultStopTimeout = 2s` and Docker waits the full timeout before SIGKILL unless PID 1 exits on SIGTERM first.

---

# Migration Plan

## Phase 1: Core (DONE)

- tai SDK: Sandbox, ExecStream, Image, Proxy.Connect, Labels, User
- sandbox/v2: Manager, Box, all CRUD + Exec + Stream + Attach + Workspace + VNC + Proxy + Image
- workspace: Manager, CRUD, file I/O, node binding, sandbox integration
- gRPC: Heartbeat RPC (proto + handler)
- Tests: unit + integration + benchmarks
- CI: consolidated SandboxV2Test + BenchmarkSandboxV2

## Phase 2: JSAPI + OAuth + Auth (PENDING)

### Prerequisites

| Task | Detail |
|------|--------|
| Wire `openapi/oauth` | `grpc.go` currently uses random token placeholders; replace with real OAuth issue/revoke |

### JSAPI Design

All JSAPI methods are static — no constructors, no Go objects in V8, no bridge/Release.
JS objects only hold string IDs, delegate everything to Go singletons (`sandbox.M()`, `workspace.M()`).

#### sandbox namespace (`RegisterObject("sandbox")`)

Static methods:

| JS | Go | Returns |
|----|-----|---------|
| `sandbox.Create(opts)` | `Manager.Create(ctx, CreateOptions)` | `Box` |
| `sandbox.Create(opts)` (opts.id set) | `Manager.GetOrCreate(ctx, CreateOptions)` | `Box` |
| `sandbox.Get(id)` | `Manager.Get(ctx, id)` | `Box \| null` |
| `sandbox.List(filter?)` | `Manager.List(ctx, ListOptions)` → `Box.Info()` | `BoxInfo[]` |
| `sandbox.Delete(id)` | `Manager.Remove(ctx, id)` | `void` |

`sandbox.Create(options)` — JS options → Go `CreateOptions`:

```
{
  id:           string   →  CreateOptions.ID           // optional; triggers GetOrCreate
  owner:        string   →  CreateOptions.Owner        // required
  pool:         string   →  CreateOptions.Pool         // default: first pool
  image:        string   →  CreateOptions.Image        // required
  workdir:      string   →  CreateOptions.WorkDir
  user:         string   →  CreateOptions.User         // e.g. "1000:1000"
  env:          object   →  CreateOptions.Env          // map[string]string
  memory:       number   →  CreateOptions.Memory       // bytes (int64)
  cpus:         number   →  CreateOptions.CPUs         // float64
  vnc:          boolean  →  CreateOptions.VNC
  ports:        array    →  CreateOptions.Ports        // [{container, host, host_ip, protocol}] → []PortMapping
  policy:       string   →  CreateOptions.Policy       // "oneshot"|"session"|"longrunning"|"persistent"
  idle_timeout: number   →  CreateOptions.IdleTimeout  // ms → time.Duration
  stop_timeout: number   →  CreateOptions.StopTimeout  // ms → time.Duration
  workspace_id: string   →  CreateOptions.WorkspaceID
  mount_mode:   string   →  CreateOptions.MountMode    // "rw"|"ro"
  mount_path:   string   →  CreateOptions.MountPath
  labels:       object   →  CreateOptions.Labels       // map[string]string
}
```

`sandbox.List(filter?)` — JS filter → Go `ListOptions`:

```
{
  owner:  string  →  ListOptions.Owner   // empty = all
  pool:   string  →  ListOptions.Pool    // empty = all
  labels: object  →  ListOptions.Labels
}
```

Returns `BoxInfo[]` — each element:

```
{
  id:            string   ←  BoxInfo.ID
  container_id:  string   ←  BoxInfo.ContainerID
  pool:          string   ←  BoxInfo.Pool
  owner:         string   ←  BoxInfo.Owner
  status:        string   ←  BoxInfo.Status
  image:         string   ←  BoxInfo.Image
  vnc:           boolean  ←  BoxInfo.VNC
  policy:        string   ←  BoxInfo.Policy
  labels:        object   ←  BoxInfo.Labels
  created_at:    string   ←  BoxInfo.CreatedAt   (ISO 8601)
  last_active:   string   ←  BoxInfo.LastActive   (ISO 8601)
  process_count: number   ←  BoxInfo.ProcessCount
}
```

#### Box object

Read-only properties:

| JS | Go |
|----|----|
| `box.id` | `Box.ID()` |
| `box.owner` | `Box.Owner()` |
| `box.pool` | `Box.Pool()` |

Methods:

| JS | Go | Returns |
|----|-----|---------|
| `box.Exec(cmd, opts?)` | `Box.Exec(ctx, cmd, ...ExecOption)` | `ExecResult` |
| `box.Stream(cmd, opts?)` | `Box.Stream(ctx, cmd, ...ExecOption)` | `ExecStream` |
| `box.Attach(port, opts?)` | `Box.Attach(ctx, port, ...AttachOption)` | `ServiceConn` |
| `box.VNC()` | `Box.VNC(ctx)` | `string` |
| `box.Proxy(port, path?)` | `Box.Proxy(ctx, port, path)` | `string` |
| `box.Workspace()` | `Box.WorkspaceID()` → `NewFSObject` | `WorkspaceFS` |
| `box.Info()` | `Box.Info(ctx)` | `BoxInfo` |
| `box.Start()` | `Box.Start(ctx)` | `void` |
| `box.Stop()` | `Box.Stop(ctx)` | `void` |
| `box.Remove()` | `Box.Remove(ctx)` | `void` |

`box.Exec(cmd, options?)`:

```
cmd:     string[]                         → cmd []string
options: {
  workdir: string,                        → WithWorkDir(dir)
  env:     object,                        → WithEnv(map[string]string)
  timeout: number                         → WithTimeout(ms → time.Duration)
}
returns: {
  exit_code: number,                      ← ExecResult.ExitCode
  stdout:    string,                      ← ExecResult.Stdout
  stderr:    string                       ← ExecResult.Stderr
}
```

`box.Stream(cmd, options?)`:

```
options: same as Exec
returns: {
  stdout: ReadableStream,                 ← ExecStream.Stdout
  stderr: ReadableStream,                 ← ExecStream.Stderr
  stdin:  WritableStream,                 ← ExecStream.Stdin
  wait:   function() → number,            ← ExecStream.Wait() (int, error)
  cancel: function() → void               ← ExecStream.Cancel()
}
```

`box.Attach(port, options?)`:

```
port:    number                           → port int
options: {
  protocol: "ws"|"sse",                  → WithProtocol(protocol)
  path:     string,                      → WithPath(path)
  headers:  object                       → WithHeaders(map[string]string)
}
returns: {
  url:    string,                         ← ServiceConn.URL
  read:   function() → Uint8Array,        ← ServiceConn.Read()
  write:  function(data) → void,          ← ServiceConn.Write(data)
  events: AsyncIterable<Uint8Array>,      ← ServiceConn.Events
  close:  function() → void               ← ServiceConn.Close()
}
```

`box.Info()` returns same structure as `BoxInfo[]` element above.

#### workspace namespace (`RegisterObject("workspace")`)

Static methods:

| JS | Go | Returns |
|----|-----|---------|
| `workspace.Create(opts)` | `Manager.Create(ctx, CreateOptions)` | `WorkspaceFS` |
| `workspace.Get(id)` | `Manager.Get(ctx, id)` | `WorkspaceFS \| null` |
| `workspace.List(filter?)` | `Manager.List(ctx, ListOptions)` | `WorkspaceInfo[]` |
| `workspace.Delete(id)` | `Manager.Delete(ctx, id, false)` | `void` |

`workspace.Create(options)` — JS options → Go `CreateOptions`:

```
{
  id:     string  →  CreateOptions.ID      // optional; auto-generated if empty
  name:   string  →  CreateOptions.Name    // required
  owner:  string  →  CreateOptions.Owner   // required
  node:   string  →  CreateOptions.Node    // required
  labels: object  →  CreateOptions.Labels  // map[string]string
}
```

`workspace.List(filter?)` — JS filter → Go `ListOptions`:

```
{
  owner: string  →  ListOptions.Owner  // empty = all
  node:  string  →  ListOptions.Node   // empty = all
}
```

Returns `WorkspaceInfo[]` — each element:

```
{
  id:         string  ←  Workspace.ID
  name:       string  ←  Workspace.Name
  owner:      string  ←  Workspace.Owner
  node:       string  ←  Workspace.Node
  labels:     object  ←  Workspace.Labels
  created_at: string  ←  Workspace.CreatedAt (ISO 8601)
  updated_at: string  ←  Workspace.UpdatedAt (ISO 8601)
}
```

#### WorkspaceFS object

Read-only properties:

| JS | Go |
|----|----|
| `ws.id` | workspace ID |
| `ws.name` | `Workspace.Name` |
| `ws.node` | `Workspace.Node` |

Methods (1:1 to Go `taiworkspace.FS` + `Manager` shortcuts):

| JS | Go | Returns |
|----|-----|---------|
| `ws.ReadFile(path)` | `FS.ReadFile(name)` / `Manager.ReadFile(ctx, id, path)` | `string` |
| `ws.WriteFile(path, data, perm?)` | `FS.WriteFile(name, data, perm)` / `Manager.WriteFile(ctx, id, path, data, perm)` | `void` |
| `ws.ReadDir(path?)` | `FS.ReadDir(name)` / `Manager.ListDir(ctx, id, path)` | `DirEntry[]` |
| `ws.Stat(path)` | `FS.Stat(name)` | `FileInfo` |
| `ws.MkdirAll(path, perm?)` | `FS.MkdirAll(name, perm)` | `void` |
| `ws.Remove(path)` | `FS.Remove(name)` / `Manager.Remove(ctx, id, path)` | `void` |
| `ws.RemoveAll(path)` | `FS.RemoveAll(name)` | `void` |
| `ws.Rename(from, to)` | `FS.Rename(old, new)` | `void` |

Planned (not yet implemented):

| JS | Go | Returns | Note |
|----|-----|---------|------|
| `ws.ReadFileBase64(path)` | `FS.ReadFile` → `base64.StdEncoding.EncodeToString` | `string` | Avoids V8↔Go binary bridge overhead for images, archives, etc. |
| `ws.WriteFileBase64(path, b64, perm?)` | `base64.StdEncoding.DecodeString` → `FS.WriteFile` | `void` | Same — base64 string transfer is far more efficient than Uint8Array across the bridge |
| `ws.CopyFromHost(hostPath, destPath?)` | Host `os.Read` → `FS.WriteFile` / `FS.MkdirAll` per entry | `void` | Copy file/dir from Yao host into workspace; `destPath` defaults to basename |
| `ws.CopyFromHostArchive(hostPath, destPath?)` | Zip on host → Tai Volume upload → Tai-side unarchive | `void` | For large directory trees; requires Tai server-side unarchive support |

Return types:

```
DirEntry: { name: string, is_dir: boolean, size: number }
FileInfo: { name: string, size: number, is_dir: boolean, mod_time: string (ISO 8601) }
```

### Auth

JSAPI does not enforce permissions internally. The Go Manager methods execute operations directly without owner/admin checks.

Developers retrieve the current caller identity via the gou global `Authorized()` function (registered by `gou/runtime/v8/functions/authorized`, reads from `bridge.Share.Authorized` / `__yao_data.AUTHORIZED`) and implement permission logic in their JS scripts.

`Authorized()` returns `map[string]interface{}` (or null if not set). The exact fields depend on what the caller sets via `Context.WithAuthorized()`. There is no fixed schema — typical fields include `user_id`, `team_id`, `scope`, etc.

```javascript
const auth = Authorized()        // gou global — returns caller info or null
const box  = sandbox.Get(id)
// Developer decides permission logic — fields depend on application's auth setup
if (box.owner !== auth.user_id) {
    throw new Error("permission denied")
}
```

Permission control is the responsibility of the caller (JS scripts, Agent hooks, API middleware, etc.).

### Implementation Tasks

| Task | Detail |
|------|--------|
| `sandbox/v2/jsapi/` | `RegisterObject("sandbox")` with Create/Get/List/Delete + Box object |
| `workspace/jsapi/` | `RegisterObject("workspace")` with Create/Get/List/Delete + FS object |
| Integration with `cmd/start.go` | Call `sandbox.Init()` + `sandbox.M().Start()` |

## Phase 3: Agent Integration (PENDING)

| Task | Detail |
|------|--------|
| Agent creates Box via `sandbox.M().GetOrCreate()` | Replace `infraSandbox.Manager` |
| Agent uses `Box.Workspace()` for file I/O | Replace Docker Copy/bind mount reads |
| Agent uses `Box.Exec()` for commands | Replace Docker exec |
| Agent uses `Box.VNC()` / `Box.Proxy()` | Replace vncproxy |

## Phase 4: Cutover (PENDING)

| Task | Detail |
|------|--------|
| Move `sandbox/v2` → `sandbox` | Rename package |
| Delete old sandbox code | manager.go, ipc/, bridge/, vncproxy/, docker/ |
| Update `cmd/start.go` | Use new init path |
| `sandbox/process.go` | Register `sandbox.*` process namespace (post-cutover) |
| `workspace/process.go` | Register `workspace.*` process namespace (post-cutover) |

## V1 vs V2 Comparison

| Aspect | V1 (current) | V2 (this design) |
|--------|-------------|-------------------|
| **Positioning** | Agent's Claude executor | Yao infrastructure module |
| **Runtime** | Direct Docker SDK | tai.Client pool (Docker/K8s/Remote) |
| **Execution** | Exec + Stream | Exec + Stream + Attach (WS/SSE) |
| **File I/O** | bind mount + Docker Copy | `workspace.FS` (fs.FS compatible) |
| **IPC** | Unix socket + yao-bridge | gRPC (tai call) |
| **Idle detection** | External calls only | Dual: external calls + container heartbeat |
| **Lifecycle** | Chat session only | Policy-based (oneshot/session/longrunning/persistent) |
| **Pool** | Single Docker daemon | Multi-pool with per-pool policies |
| **Agent coupling** | Tightly coupled | Zero dependency |
| **Workspace** | None | Persistent, node-bound, decoupled from containers |
| **Image management** | None | EnsureImage + Pull with progress |
| **K8s** | Not supported | Supported via tai.Client |
| **Multi-node** | Local only | Local + Remote via Tai |
