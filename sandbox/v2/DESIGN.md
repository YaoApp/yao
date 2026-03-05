# Sandbox V2 Design

## Positioning

Sandbox is a **standalone infrastructure module** in Yao, on the same level as `process`, `store`, and `fs`. It provides isolated execution environments with standard file I/O. Any module can use it — Agent, JSAPI scripts, Process handlers, API endpoints.

```
Yao Infrastructure
├── process    — process execution
├── store      — KV storage
├── fs         — host filesystem
├── stream     — streaming execution (planned)
└── sandbox    — isolated execution environments ← this module
```

Sandbox does NOT import or depend on Agent. Agent is one of many consumers.

## Architecture

```
┌─────────────────────────────────────────────────┐
│  Consumers (know nothing about tai/Docker/K8s)  │
│  ├── JSAPI: Sandbox("my-app")                   │
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
│  ├── Create / Get / Start / Stop / Remove       │
│  ├── List / Cleanup / Close                     │
│  └── guard rails (limits, TTL) + Box factory    │
│                                                  │
│  Box (per-instance)                              │
│  ├── Exec(cmd) → ExecResult                     │
│  ├── Stream(cmd) → ExecStream (real-time I/O)   │
│  ├── Attach(port) → ServiceConn (WS/SSE/TCP)   │
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
│  ├── "k8s"    → tai.New("tai://k8s")  (K8s)    │
│  └── ...                                        │
│                                                  │
│  Each tai.Client provides:                       │
│  ├── Sandbox()   → CRUD + Exec + ExecStream     │
│  ├── Volume()    → file I/O (local disk / gRPC) │
│  ├── Workspace() → fs.FS                        │
│  ├── Proxy()     → URL resolve + Connect        │
│  └── VNC()       → VNC WebSocket                │
└─────────────────────────────────────────────────┘
```

## Dependency Rules

```
sandbox/v2 → tai           ✓  (sole runtime dependency)
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
// Pool defines a named tai.Client endpoint with its own policy.
type Pool struct {
    Name    string       // unique name, e.g. "local", "gpu", "k8s-prod"
    Addr    string       // tai.New() address: "local", "tai://host", "docker:///path"
    Options []tai.Option // tai.WithPorts(), tai.WithKubeConfig(), etc.

    // Guard rails (per-pool)
    MaxPerUser  int           // max boxes per user on this pool, 0 = unlimited
    MaxTotal    int           // max boxes total on this pool, 0 = unlimited

    // Default lifecycle (overridable per-box via CreateOptions)
    IdleTimeout time.Duration // 0 = no timeout
    MaxLifetime time.Duration // 0 = no limit
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
    max_per_user: 1              # GPU is expensive, 1 per user
    max_total: 4
    idle_timeout: 10m            # reclaim fast
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
package sandbox

var mgr *Manager

// Init initializes the global Manager.
// Config contains everything: pool definitions + guard rails.
// At least one Pool entry is required. The first entry is the default.
// Pass empty Pool list to disable sandbox (methods return ErrNotAvailable).
func Init(cfg Config) error

// M returns the global Manager. Panics if Init was not called.
func M() *Manager
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

`Init` creates the Manager from config (pool definitions + guard rails). `Start` connects to pools, discovers existing containers, and starts the cleanup loop. Two-step so that gRPC server is ready before Start (containers may send heartbeats immediately).

Pool connections are created lazily on first use and reused across all Box instances.

### Config

Pool definitions only. Guard rails and lifecycle defaults are per-pool. Per-instance settings (image, memory, workdir, etc.) are in `CreateOptions`.

```go
type Config struct {
    Pool []Pool // runtime endpoints; first is default
}
```

Container gRPC env vars (`YAO_GRPC_ADDR`, `YAO_GRPC_UPSTREAM`, etc.) are derived automatically at creation time — local address from Yao's gRPC config (`config.Conf.GRPC`), remote relay from pool's tai address. No manual configuration needed.

Per-instance settings (image, memory, CPU, workdir, env, pool) are passed via `CreateOptions` by the caller — assistant config, JSAPI parameters, or Process arguments. The Manager doesn't impose defaults for container specs; that's the caller's responsibility.

### Core API

```go
type Manager struct {
    pool       map[string]*tai.Client // name → connection (lazy-initialized)
    poolDefs   []Pool                 // pool definitions
    defaultPool string                // first pool name
    config     Config
    boxes      sync.Map   // id → *Box
    mu         sync.Mutex // creation serialization
}

// --- Bootstrap ---

// Start discovers existing containers from all pools, rebuilds the boxes map,
// and starts the cleanup loop. Called once after Init.
func (m *Manager) Start(ctx context.Context) error

// --- Pool management ---

// AddPool registers a new pool at runtime. Connects lazily on first use.
func (m *Manager) AddPool(ctx context.Context, p Pool) error

// RemovePool removes a pool by name. Fails if any running boxes are on it.
// Use force=true to stop all boxes on the pool first, then remove.
func (m *Manager) RemovePool(ctx context.Context, name string, force bool) error

// Pools returns all registered pool names and their status (connected/disconnected).
func (m *Manager) Pools() []PoolInfo

// --- Heartbeat (called by gRPC handler, not by consumers) ---

// Heartbeat updates the box's last heartbeat timestamp.
// Called by the gRPC Heartbeat handler when a container reports in.
// Returns ErrNotFound if sandbox_id is unknown (container orphaned or already removed).
func (m *Manager) Heartbeat(sandboxID string, active bool, processCount int) error

// --- CRUD ---

// Create creates and starts a new sandbox.
func (m *Manager) Create(ctx context.Context, opts CreateOptions) (*Box, error)

// Get returns an existing sandbox by ID. Returns ErrNotFound if not exists.
func (m *Manager) Get(ctx context.Context, id string) (*Box, error)

// GetOrCreate returns existing sandbox or creates a new one.
func (m *Manager) GetOrCreate(ctx context.Context, opts CreateOptions) (*Box, error)

// List returns all sandboxes, optionally filtered.
func (m *Manager) List(ctx context.Context, opts ListOptions) ([]*Box, error)

// Remove stops and removes a sandbox.
func (m *Manager) Remove(ctx context.Context, id string) error

// Cleanup removes idle/expired sandboxes. Called periodically.
func (m *Manager) Cleanup(ctx context.Context) error

// Close stops the cleanup loop and releases all pool connections.
func (m *Manager) Close() error
```

### CreateOptions

All per-instance settings live here. Caller decides everything about the container.

```go
type CreateOptions struct {
    // Identity
    ID     string // explicit ID; empty = auto-generate
    Owner  string // user ID for isolation and limits
    Labels map[string]string

    // Runtime target
    Pool string // which tai.Client to use; empty = default pool

    // Container spec
    Image   string            // required
    WorkDir string            // container working directory, default "/workspace"
    User    string            // container user
    Env     map[string]string // additional env vars
    Memory  int64             // bytes, 0 = no limit
    CPUs    float64           // 0 = no limit
    VNC     bool              // enable VNC
    Ports   []PortMapping     // extra port mappings

    // Lifecycle
    Policy      LifecyclePolicy // default: Session
    IdleTimeout time.Duration   // override Manager default; 0 = use Manager default
}

type LifecyclePolicy string

const (
    OneShot     LifecyclePolicy = "oneshot"     // destroyed after first Exec
    Session     LifecyclePolicy = "session"     // alive while active, cleaned on idle
    LongRunning LifecyclePolicy = "longrunning" // user workspace, extended TTL
    Persistent  LifecyclePolicy = "persistent"  // never auto-cleaned
)
```

## Box

A `Box` is a single sandbox instance. All operations go through it.

```go
type Box struct {
    id            string
    containerID   string
    pool          string       // which tai.Client this box runs on
    owner         string
    policy        LifecyclePolicy
    labels        map[string]string
    lastCall      atomic.Int64 // last external API call (Exec/Workspace/VNC/Proxy)
    lastHeartbeat atomic.Int64 // last container heartbeat
    processCount  atomic.Int32 // user processes inside container (from heartbeat)
    ws            workspace.FS // lazy-initialized, cached
    manager       *Manager
}

// lastActiveTime returns max(lastCall, lastHeartbeat).
func (b *Box) lastActiveTime() time.Time

// ID returns the sandbox identifier.
func (b *Box) ID() string

// Owner returns the user who owns this sandbox.
func (b *Box) Owner() string

// ContainerID returns the underlying container ID.
func (b *Box) ContainerID() string

// --- Execution ---

// Exec runs a command and waits for it to finish.
func (b *Box) Exec(ctx context.Context, cmd []string, opts ...ExecOption) (*ExecResult, error)

// Stream runs a command with real-time streaming I/O.
func (b *Box) Stream(ctx context.Context, cmd []string, opts ...ExecOption) (*ExecStream, error)

// Attach connects to a service running inside the sandbox on the given container port.
func (b *Box) Attach(ctx context.Context, port int, opts ...AttachOption) (*ServiceConn, error)

// --- Filesystem (fs.FS compatible) ---

// Workspace returns an fs.FS-compatible filesystem for this sandbox.
// Supports: Open, Stat, ReadFile, ReadDir, WriteFile, Remove, Rename, MkdirAll.
// Internally calls tai.Client.Workspace(box.id) — uses sandbox ID as volume session.
func (b *Box) Workspace() workspace.FS

// --- Network ---

// VNC returns the VNC WebSocket URL. Error if VNC not enabled.
func (b *Box) VNC(ctx context.Context) (string, error)

// Proxy returns the HTTP URL for a service running on the given port inside the sandbox.
func (b *Box) Proxy(ctx context.Context, port int, path string) (string, error)

// --- Lifecycle ---

// Start starts a stopped sandbox.
func (b *Box) Start(ctx context.Context) error

// Stop stops the sandbox without removing it.
func (b *Box) Stop(ctx context.Context) error

// Remove stops and removes the sandbox.
func (b *Box) Remove(ctx context.Context) error

// Info returns current sandbox status.
func (b *Box) Info(ctx context.Context) (*BoxInfo, error)
```

### ExecOption / ExecResult

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
```

### ExecStream

```go
type ExecStream struct {
    Stdout io.ReadCloser   // real-time stdout
    Stderr io.ReadCloser   // real-time stderr
    Stdin  io.WriteCloser  // write to process stdin (nil if not interactive)
    Wait   func() (int, error) // block until exit, return exit code
    Cancel func()              // kill the process
}
```

Usage:

```go
// Interactive CLI (e.g. Claude)
s, _ := box.Stream(ctx, []string{"claude", "--chat"})
go io.Copy(os.Stdout, s.Stdout)
s.Stdin.Write([]byte("help\n"))
code, _ := s.Wait()

// Long-running process (e.g. dev server)
s, _ := box.Stream(ctx, []string{"npm", "run", "dev"})
go io.Copy(logWriter, s.Stdout)  // continuous output
// ... later
s.Cancel()
```

### AttachOption / ServiceConn

```go
type AttachOption func(*attachConfig)

func WithProtocol(proto string) AttachOption   // "ws", "sse", "tcp"; default "ws"
func WithPath(path string) AttachOption        // URL path, e.g. "/v1/chat"
func WithHeaders(h map[string]string) AttachOption

type ServiceConn struct {
    // Bidirectional (WebSocket, TCP)
    Read  func() ([]byte, error)
    Write func(data []byte) error

    // Server-push (SSE)
    Events <-chan []byte  // nil if not SSE mode

    // Common
    URL   string         // resolved URL for reference
    Close func() error
}
```

`port` is the port the service listens on **inside the container** (e.g. 3000 for a Node server). Routing to that port — Docker port mapping (local) or Tai HTTP proxy (remote) — is handled internally.

**Local mode caveat**: `tai/proxy.NewLocal` resolves host ports via `Inspect()` → `PortMapping`. The container must have the port mapped at creation time (`CreateOptions.Ports`). If the port was not mapped, `Proxy()` and `Attach()` return an error. Remote mode has no such restriction — Tai HTTP proxy routes by container IP directly.

Usage:

```go
// WebSocket — connect to Cursor Server inside sandbox
conn, _ := box.Attach(ctx, 3000, WithProtocol("ws"), WithPath("/ws"))
conn.Write([]byte(`{"type":"edit","file":"main.go"}`))
msg, _ := conn.Read()
conn.Close()

// SSE — connect to Claude API inside sandbox
conn, _ := box.Attach(ctx, 8080, WithProtocol("sse"), WithPath("/v1/messages"))
for event := range conn.Events {
    fmt.Println(string(event))
}
```

### PoolInfo

```go
type PoolInfo struct {
    Name        string        // pool name
    Addr        string        // tai address
    Connected   bool          // tai.Client connection established
    Boxes       int           // number of boxes on this pool
    MaxPerUser  int
    MaxTotal    int
    IdleTimeout time.Duration
    MaxLifetime time.Duration
}
```

### BoxInfo

```go
type BoxInfo struct {
    ID            string
    ContainerID   string
    Pool          string
    Owner         string
    Status        string // "running", "stopped", "creating"
    Policy        LifecyclePolicy
    Labels        map[string]string
    Image         string
    CreatedAt     time.Time
    LastActive    time.Time // max(lastCall, lastHeartbeat)
    ProcessCount  int       // user processes inside container (0 = idle)
    VNC           bool
}
```

## Workspace — fs.FS Interface

`Box.Workspace()` returns `workspace.FS` from `tai/workspace`. This is the standard Go `fs.FS` interface extended with write operations.

```go
// tai/workspace.FS — already implemented
type FS interface {
    fs.FS         // Open(name) (fs.File, error)
    fs.StatFS     // Stat(name) (fs.FileInfo, error)
    fs.ReadFileFS // ReadFile(name) ([]byte, error)
    fs.ReadDirFS  // ReadDir(name) ([]fs.DirEntry, error)
    io.Closer

    WriteFile(name string, data []byte, perm os.FileMode) error
    Remove(name string) error
    RemoveAll(name string) error
    Rename(oldname, newname string) error
    MkdirAll(name string, perm os.FileMode) error
}
```

100% compatible with Go standard library:

```go
ws := box.Workspace()

// Standard fs functions work
data, _ := fs.ReadFile(ws, "main.go")
fs.WalkDir(ws, ".", func(path string, d fs.DirEntry, err error) error { ... })
info, _ := fs.Stat(ws, "go.mod")

// Extended write operations
ws.WriteFile("main.go", []byte("package main"), 0644)
ws.MkdirAll("src/pkg", 0755)
ws.Remove("tmp.txt")
ws.Rename("old.go", "new.go")
```

Local mode: reads/writes go directly to host disk via bind mount.
Remote mode: reads/writes go through tai Volume gRPC with lz4 compression.
Caller doesn't know or care which mode.

## Container gRPC (already implemented)

Container processes communicate with Yao via gRPC. No Unix sockets.

```
Local:   Container → yao-grpc → Yao gRPC 127.0.0.1:9099
Remote:  Container → yao-grpc → Tai :9100 relay → Yao gRPC :9099
```

Manager injects env vars at container creation:

```
# Local
YAO_GRPC_ADDR=127.0.0.1:9099
YAO_TOKEN=<access_token>
YAO_REFRESH_TOKEN=<refresh_token>
YAO_SANDBOX_ID=<sandbox_id>

# Remote (adds Tai relay)
YAO_GRPC_TAI=enable
YAO_GRPC_UPSTREAM=yao-host:9099
```

Token issuance uses existing `openapi/oauth`. Manager creates token pair at container creation, revokes refresh token on Remove.

## Process Registration

Sandbox operations are exposed as Yao Processes under the `sandbox` namespace.

```go
func init() {
    process.Register("sandbox", handler)
}
```

| Process | Args | Returns |
|---------|------|---------|
| `sandbox.pool.Add` | `pool` (Pool JSON) | PoolInfo |
| `sandbox.pool.Remove` | `name`, `force?` | — |
| `sandbox.pool.List` | — | []PoolInfo |
| `sandbox.Create` | `options` (CreateOptions JSON) | BoxInfo |
| `sandbox.Get` | `id` | BoxInfo |
| `sandbox.GetOrCreate` | `options` | BoxInfo |
| `sandbox.Remove` | `id` | — |
| `sandbox.List` | `options` (ListOptions JSON) | []BoxInfo |
| `sandbox.Start` | `id` | — |
| `sandbox.Stop` | `id` | — |
| `sandbox.Exec` | `id`, `cmd[]`, `options?` | ExecResult |
| `sandbox.Stream` | `id`, `cmd[]`, `options?` | stream (chunked output) |
| `sandbox.Attach` | `id`, `port`, `options?` | ServiceConn info |
| `sandbox.ReadFile` | `id`, `path` | file content (string) |
| `sandbox.WriteFile` | `id`, `path`, `content` | — |
| `sandbox.ListDir` | `id`, `path` | []FileInfo |
| `sandbox.RemoveFile` | `id`, `path` | — |
| `sandbox.MkdirAll` | `id`, `path` | — |
| `sandbox.VNC` | `id` | URL string |
| `sandbox.Proxy` | `id`, `port`, `path?` | URL string |

This allows any Yao script, Flow, or API to use sandbox:

```json
{
  "process": "sandbox.Exec",
  "args": ["sb-001", ["go", "build", "./..."]]
}
```

## JSAPI

Global constructor function registered in `gou/runtime/v8`, following the `FS()` / `Store()` pattern.

```javascript
// Pool management
Sandbox.AddPool({ name: "gpu2", addr: "tai://gpu2.internal" })
Sandbox.RemovePool("gpu2")
var pools = Sandbox.Pools()
// [{ name: "local", addr: "local", connected: true, boxes: 3 }, ...]

// Get or create a sandbox
var sb = Sandbox("my-workspace", {
    image: "yaoapp/workspace:latest",
    owner: "user-123"
})

// File operations (fs.FS semantics)
var content = sb.ReadFile("src/main.go")
sb.WriteFile("src/main.go", "package main\n...")
var entries = sb.ListDir("src/")
var info = sb.Stat("src/main.go")
sb.MkdirAll("src/components")
sb.Remove("tmp.txt")
sb.Rename("old.go", "new.go")

// Command execution — wait for result
var result = sb.Exec(["go", "build", "./..."])
// result.exit_code, result.stdout, result.stderr

// Streaming execution — real-time output
sb.Stream(["npm", "run", "dev"], function(chunk) {
    log.Info(chunk)   // real-time stdout/stderr
    return 1          // 1=continue, 0=stop
})

// Connect to a service inside the sandbox
var conn = sb.Attach(3000, { protocol: "ws", path: "/ws" })
conn.Write('{"type":"ping"}')
var msg = conn.Read()
conn.Close()

// Network
var vncUrl = sb.VNCUrl()
var previewUrl = sb.ProxyUrl(3000, "/")

// Info
var info = sb.Info()
// info.id, info.status, info.owner, info.created_at

// Lifecycle
sb.Stop()
sb.Start()
sb.Remove()

// Properties
sb.id          // sandbox ID
sb.workdir     // container working directory
```

Registration in `gou/runtime/v8/isolate.go`:

```go
template.Set("Sandbox", sandboxT.New().ExportFunction(iso))
```

Implementation: `gou/runtime/v8/objects/sandbox/sandbox.go` — wraps `sandbox.M().GetOrCreate()` + `Box` methods, using `bridge.GoValue` / `bridge.JsValue` for type conversion.

## Bootstrap — Manager.Start()

On `Manager.Start()`, the Manager recovers all existing sandboxes and starts the cleanup loop:

```
1. For each pool:
     tai.Client.Sandbox().List(labels: {"managed-by": "yao-sandbox"})
       → discover running/stopped containers

2. For each discovered container:
     Parse labels → extract sandbox ID, owner, policy, pool name
     Rebuild Box struct, register in boxes map
     Set lastCall = now (grace period after restart)

3. Start cleanupLoop goroutine
```

Containers are identified by the label `managed-by=yao-sandbox` plus `sandbox-id=<id>`. Manager injects these labels at creation time. On restart, it queries each pool for containers with `managed-by=yao-sandbox` and rebuilds the in-memory state.

**What happens to orphaned containers** (created by old Manager, no longer matching any pool):
- If a pool is removed from config, its containers are invisible to the new Manager
- They stay running in Docker/K8s until manually cleaned or TTL-expired by the runtime
- This is by design — Manager only manages containers it can reach

Startup sequence in `cmd/start.go`:

```
sandbox.Init(config.Conf.Sandbox) // create Manager with pool + guard rails
sandbox.M().Start(ctx)            // discover existing containers, start cleanup loop
```

## Container Setup — Manager.Create()

When Manager creates a sandbox, it:

1. Validates `CreateOptions` (Image required)
2. Generates sandbox ID (or uses provided one)
3. Checks user limits (`MaxPerUser`) and total limits (`MaxTotal`)
4. Resolves pool (by name or default)
5. Creates OAuth token pair for container IPC via `openapi/oauth`
6. Builds `tai.sandbox.CreateOptions` from caller's `CreateOptions`:
   - Image, Cmd (`sleep infinity`), User — all from caller
   - Field name mapping: v2 `WorkDir` → tai `WorkingDir`
   - Merges caller's Env with IPC env vars:
     - `YAO_GRPC_ADDR`, `YAO_TOKEN`, `YAO_REFRESH_TOKEN`, `YAO_SANDBOX_ID`
     - Remote mode: `YAO_GRPC_TAI=enable`, `YAO_GRPC_UPSTREAM`
   - Memory/CPU limits, VNC flag, port mappings — all from caller
   - Injects management labels:
     - `managed-by=yao-sandbox`
     - `sandbox-id=<id>`
     - `sandbox-owner=<owner>`
     - `sandbox-pool=<pool>`
     - `sandbox-policy=<policy>`
7. Calls `tai.Client.Sandbox().Create()` then `Start()`
8. Wraps in a `Box`, registers in `boxes` map
9. Starts idle tracking

## Lifecycle Management

### Idle Tracking — Dual Source

Idle is determined by two sources, taking the most recent of both:

```go
box.lastActive = max(lastExternalCall, lastHeartbeat)
```

| Source | What it tracks | Updated by |
|--------|---------------|------------|
| External call | Caller is using the sandbox | `Box.Exec()`, `Box.Workspace()`, `Box.VNC()`, `Box.Proxy()` |
| Container heartbeat | Processes running inside the container | `yao-grpc` → gRPC `Heartbeat` RPC |

**Why both**: external calls alone miss "user walked away but `npm run build` is still running". Heartbeat alone misses "user is reading output, hasn't issued a new command yet". Together they cover all cases.

### Heartbeat — Container Side

`yao-grpc` (already running inside every container) runs a background goroutine:

```
Every 30 seconds:
  1. Count user processes (ps aux, exclude sleep/init/yao-grpc)
  2. Count gRPC calls forwarded in last 30s (internal counter)
  3. If either > 0 → send Heartbeat(sandbox_id, active=true, process_count=N)
     else → don't send (silent = idle)
```

~30 lines added to `tai/grpc/cmd/main.go`. Zero new dependencies.

### Heartbeat — Server Side

New gRPC RPC in `yao.proto`:

```protobuf
rpc Heartbeat(HeartbeatRequest) returns (HeartbeatResponse);

message HeartbeatRequest {
  string sandbox_id = 1;
  bool   active = 2;
  int32  process_count = 3;
}
message HeartbeatResponse {}
```

Handler (~20 lines in `grpc/sandbox/`): looks up Box by `sandbox_id`, updates `lastHeartbeat`. Auth: reuses container's `YAO_TOKEN`, no new scope needed (piggyback on existing `grpc:mcp`).

### Idle Decision Matrix

| External calls | Heartbeat | Judgment | Action |
|---------------|-----------|----------|--------|
| Recent | Recent | Active | None |
| Recent | Silent | Active | None (user reading output) |
| None | Recent | Active | None (build/server still running) |
| None | Silent | **Idle** | Policy-based stop/remove |

### Cleanup Loop

```go
func (m *Manager) cleanupLoop(ctx context.Context) {
    ticker := time.NewTicker(1 * time.Minute)
    for {
        select {
        case <-ticker.C:
            m.Cleanup(ctx)
        case <-ctx.Done():
            return
        }
    }
}

func (m *Manager) Cleanup(ctx context.Context) error {
    now := time.Now()
    m.boxes.Range(func(key, value any) bool {
        box := value.(*Box)
        idle := now.Sub(box.lastActiveTime()) // max(external, heartbeat)

        switch box.policy {
        case OneShot:
            // already removed after Exec
        case Session:
            if idle > box.idleTimeout() { box.Remove(ctx) }
        case LongRunning:
            if idle > box.idleTimeout() { box.Stop(ctx) }
            if lifetime > box.maxLifetime() { box.Remove(ctx) }
        case Persistent:
            // never auto-cleaned
        }
        return true
    })
    return nil
}
```

### Policy Behavior

| Policy | Idle | Max Lifetime | Auto |
|--------|------|-------------|------|
| OneShot | — | — | Removed after first Exec completes |
| Session | Stop + Remove | Remove | Default for agent chats |
| LongRunning | Stop (keep data) | Remove | User workspaces |
| Persistent | Never | Never | User-managed |

## Package Structure

```
sandbox/v2/
├── sandbox.go          // Init, M(), global singleton
├── manager.go          // Manager struct, Create/Get/List/Remove/Cleanup
├── box.go              // Box struct, Exec/Workspace/VNC/Proxy/lifecycle
├── config.go           // Config, env parsing
├── types.go            // CreateOptions, ExecResult, BoxInfo, enums
├── errors.go           // sentinel errors
├── process.go          // Yao Process registration (sandbox.*)
├── grpc.go             // token creation + gRPC env var injection for containers
├── jsapi/
│   └── sandbox.go      // V8 JSAPI: Sandbox() constructor (lives in gou)
└── DESIGN.md           // this document
```

## Tai SDK Changes Required

Sandbox V2 needs changes in `tai/` and `yao/grpc` before Phase 1 can fully work. These are **prerequisites** — the sandbox module itself has zero Docker/K8s awareness, so all runtime capabilities must exist in tai; heartbeat support requires additions to both the gRPC server and the in-container client.

### 1. `tai/sandbox` — Add `ExecStream` (streaming exec)

Current `Exec()` buffers all output and returns `ExecResult` after the process exits. `Box.Stream()` needs a streaming variant.

```go
// tai/sandbox — new method on the Sandbox interface
type ExecStream struct {
    Stdout io.ReadCloser
    Stderr io.ReadCloser
    Stdin  io.WriteCloser
    Wait   func() (int, error) // blocks until exit, returns exit code
    Cancel func()              // kills the exec process
}

func (s *Sandbox) ExecStream(ctx context.Context, containerID string, cmd []string, opts ...ExecOption) (*ExecStream, error)
```

Implementation per runtime:

| Runtime | How |
|---------|-----|
| **Docker** (`docker_core.go`) | `ContainerExecCreate` + `ContainerExecAttach` — already returns a `HijackedResponse` with a raw stream. Current code pipes it into buffers; change to expose `io.ReadCloser` directly. `Cancel` calls `ContainerExecInspect` loop → kill. ~40 lines changed. |
| **K8s** (`k8s.go`) | `remotecommand.NewSPDYExecutor` + `StreamWithContext` — already supports streaming. Current code passes `bytes.Buffer`; change to pass `io.Pipe()`. ~30 lines changed. |

Both runtimes already have the raw streaming capability — the change is to **stop buffering** and expose the stream directly.

### 2. `tai/proxy` — Add `Connect` (bidirectional connection)

Current `proxy.Proxy` only returns a URL string (`Resolve()`). `Box.Attach()` needs an actual connection.

```go
// tai/proxy — new method
type ConnectOptions struct {
    Protocol string            // "ws", "sse", "tcp"; default "ws"
    Path     string            // URL path, e.g. "/v1/chat"
    Headers  map[string]string // extra request headers
}

type Connection struct {
    Read   func() ([]byte, error)  // read next message/event
    Write  func(data []byte) error // send data (no-op for SSE)
    Events <-chan []byte           // non-nil for SSE mode
    URL    string                  // resolved URL for reference
    Close  func() error
}

func (p *Proxy) Connect(ctx context.Context, containerID string, port int, opts ConnectOptions) (*Connection, error)
```

Implementation:

| Mode | How |
|------|-----|
| **Local** | Direct dial to `containerIP:port`. WebSocket via `gorilla/websocket` or `nhooyr.io/websocket`. SSE via `http.Get` + chunked read. TCP via `net.Dial`. |
| **Remote** | Dial through Tai HTTP proxy: `http://tai-host:8080/{containerID}:{port}/{path}`. Tai proxy already handles WebSocket upgrade and SSE streaming natively (`http.Hijacker` for WS, `FlushInterval: -1` for SSE). No Tai server changes needed. |

The Tai HTTP proxy server (`tai/httpproxy/router.go`) already supports:
- **WebSocket**: detects `Upgrade: websocket` header, does TCP-level bidirectional relay
- **SSE**: reverse proxy with `FlushInterval: -1`, streams through transparently
- **Regular HTTP**: standard `httputil.ReverseProxy`

So the `Connect` implementation in `tai/proxy` is a **client-side** addition only. The server side is ready.

### 3. `tai/sandbox` — Add `Labels` and `User` to `CreateOptions`

Current `tai/sandbox.CreateOptions` is missing two fields Manager needs:

- **`Labels`**: for container discovery on restart (`managed-by=yao-sandbox`, `sandbox-id`, etc.)
- **`User`**: to run container processes as a specific user

```go
// tai/sandbox — add to existing CreateOptions struct
type CreateOptions struct {
    // ... existing fields (Name, Image, Cmd, Env, Binds, WorkingDir, Memory, CPUs, VNC, Ports) ...
    Labels map[string]string // container/pod labels for discovery and management
    User   string            // container user, e.g. "1000:1000"
}
```

Implementation:

| Runtime | Field | How |
|---------|-------|-----|
| **Docker** | `Labels` | Set `cfg.Labels = opts.Labels` in `create()`. ~1 line. |
| **Docker** | `User` | Set `cfg.User = opts.User` in `create()`. ~1 line. |
| **K8s** | `Labels` | Set `pod.ObjectMeta.Labels` in `CreatePod`. ~1 line. |
| **K8s** | `User` | Set `SecurityContext.RunAsUser` in pod spec. ~3 lines. |

`List` with label filtering is **already implemented** in both runtimes:
- Docker: `filters.NewArgs("label", k+"="+v)` in `docker_core.go:175`
- K8s: `metav1.ListOptions{LabelSelector: ...}` in `k8s.go:255`

`ListOptions.Labels` field also already exists in `sandbox.go:68`. No changes needed for List.

Also needed: **`ContainerInfo` must include `Labels`**. Current `ContainerInfo` struct has no `Labels` field. `Manager.Start()` discovers existing containers via `List()` and needs to read labels (`sandbox-id`, `sandbox-owner`, `sandbox-policy`, `sandbox-pool`) to rebuild Box state.

```go
// tai/sandbox — add to existing ContainerInfo struct
type ContainerInfo struct {
    // ... existing fields (ID, Name, Image, Status, IP, Ports) ...
    Labels map[string]string // container/pod labels
}
```

| Runtime | How |
|---------|-----|
| **Docker** | `list()`: read `c.Labels` from `ContainerList` response. `inspect()`: read `info.Config.Labels`. ~1 line each. |
| **K8s** | `list()`: read `pod.Labels` from `PodList` response. ~1 line. |

### 4. `yao/grpc` + `tai/grpc` — Heartbeat RPC

Manager uses dual idle tracking (external API calls + container heartbeat). The heartbeat path requires additions on both sides: the gRPC server (new RPC) and `yao-grpc` in-container client (new background goroutine).

#### Server side — `yao/grpc`

New RPC in `grpc/pb/yao.proto`:

```protobuf
rpc Heartbeat(HeartbeatRequest) returns (HeartbeatResponse);

message HeartbeatRequest {
  string sandbox_id = 1;
  bool   active = 2;       // true if user processes detected
  int32  process_count = 3; // number of user processes
}
message HeartbeatResponse {}
```

Handler in `grpc/sandbox/` (~20 lines):

```go
func (s *Server) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
    box, err := sandbox.M().Get(ctx, req.SandboxId)
    if err != nil {
        return nil, status.Errorf(codes.NotFound, "sandbox %s not found", req.SandboxId)
    }
    sandbox.M().Heartbeat(req.SandboxId, req.Active, int(req.ProcessCount))
    return &pb.HeartbeatResponse{}, nil
}
```

Auth: reuses container's `YAO_TOKEN` — no new OAuth scope needed. The token is already issued with gRPC access when Manager creates the container.

#### Client side — `tai/grpc/cmd/main.go` (`yao-grpc`)

New background goroutine (~30 lines) added to `yao-grpc` startup:

```go
func heartbeatLoop(ctx context.Context, client *grpc.Client, sandboxID string) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    for {
        select {
        case <-ticker.C:
            count := countUserProcesses()     // ps aux, exclude sleep/init/yao-grpc
            active := count > 0
            if active {
                client.Heartbeat(ctx, sandboxID, true, int32(count))
            }
            // silent when idle — no heartbeat sent, Manager tracks absence
        case <-ctx.Done():
            return
        }
    }
}

func countUserProcesses() int {
    // exec `ps -eo comm`, filter out known system processes
    // (sleep, init, yao-grpc, sh -c sleep)
    // return count of remaining user processes
}
```

`yao-grpc` reads `YAO_SANDBOX_ID` from env (injected by Manager at container creation). If empty, heartbeat is disabled (container not managed by sandbox).

#### Heartbeat flow

```
Container (every 30s)                    Yao Server
─────────────────────                    ──────────
countUserProcesses()
  ├── active (count > 0)
  │   └── yao-grpc → Heartbeat RPC ──→ grpc/sandbox/Heartbeat()
  │                                      └── sandbox.M().Heartbeat(id, true, N)
  │                                           └── box.lastHeartbeat = now
  │                                               box.processCount = N
  └── idle (count == 0)
      └── (no RPC sent)                 Manager sees: no heartbeat in 30s+
                                         └── combined with no external calls → idle
```

Key behaviors:
- **Only sends when active** — idle containers are silent, reducing gRPC traffic
- **30s interval** — matches Manager cleanup loop granularity (1 min), two missed heartbeats = considered idle
- **Crash-safe** — if `yao-grpc` dies, heartbeats stop, Manager treats it as idle after timeout
- **Zero new dependencies** — `yao-grpc` already has the gRPC client connection; heartbeat piggybacks on it

### Summary

| Change | Package | Effort | Blocks |
|--------|---------|--------|--------|
| `ExecStream` | `tai/sandbox` | ~40 lines Docker + ~30 lines K8s | `Box.Stream()` |
| `Connect` | `tai/proxy` | ~80 lines (client-side only, server ready) | `Box.Attach()` |
| `Labels` + `User` in `CreateOptions` | `tai/sandbox` | ~6 lines (Docker + K8s) | `Manager.Create()` labeling + user |
| `Labels` in `ContainerInfo` | `tai/sandbox` | ~3 lines (Docker list/inspect + K8s list) | `Manager.Start()` container discovery |
| `Heartbeat` RPC | `yao/grpc` | ~20 lines handler + 3 lines proto | `Manager.Heartbeat()` |
| Heartbeat goroutine | `tai/grpc` (`yao-grpc`) | ~30 lines | Container → Server heartbeat |

`List` with label filtering is already implemented in both Docker and K8s runtimes — no changes needed.

All changes are additive (no breaking changes to existing APIs). `Box.Exec()` and `Box.Workspace()` work with current tai — only Stream, Attach, and idle tracking need the new methods.

## Migration Plan

### Phase 1: Core module

Build `sandbox/v2` as a standalone package. No agent dependency.

**Tai / gRPC prerequisites** (do first):

| Task | Detail |
|------|--------|
| `tai/sandbox`: `ExecStream` | Streaming exec for Docker + K8s (~70 lines total) |
| `tai/proxy`: `Connect` | Client-side WebSocket/SSE/TCP connection (~80 lines) |
| `tai/sandbox`: `Labels` + `User` in `CreateOptions` | Add fields + wire into Docker/K8s create (~6 lines). List filter already done. |
| `tai/sandbox`: `Labels` in `ContainerInfo` | Add field + populate in Docker list/inspect, K8s list (~3 lines) |
| `yao/grpc`: `Heartbeat` RPC | Proto + handler (~20 lines) |
| `tai/grpc` (`yao-grpc`): heartbeat goroutine | Process detection + periodic report (~30 lines) |

**Sandbox V2 module:**

| Task | Detail |
|------|--------|
| `sandbox.go` | `Init()`, `M()`, singleton lifecycle |
| `config.go` | Config struct, defaults |
| `types.go` | CreateOptions, ExecResult, BoxInfo, LifecyclePolicy, Pool |
| `errors.go` | ErrNotAvailable, ErrNotFound, ErrLimitExceeded |
| `manager.go` | Manager with tai.Client pool. Create/Get/GetOrCreate/List/Remove/Cleanup/Close |
| `box.go` | Box wrapping tai Sandbox/Volume/Workspace/Proxy/VNC. Dual idle tracking (lastCall + lastHeartbeat) |
| `grpc.go` | OAuth token pair creation, gRPC env var injection |
| Tests | Unit + integration (needs Docker for local mode) |

### Phase 2: Process + JSAPI

| Task | Detail |
|------|--------|
| `process.go` | Register `sandbox.*` process namespace |
| `jsapi/sandbox.go` | V8 `Sandbox()` constructor in gou |
| Tests | Process handler tests, JSAPI tests |

### Phase 3: Agent integration

In the Agent repo (not in sandbox/v2):

| Task | Detail |
|------|--------|
| Agent creates Box via `sandbox.M().GetOrCreate()` | Replace `infraSandbox.Manager` |
| Agent uses `Box.Workspace()` for file I/O | Replace Docker Copy/bind mount reads |
| Agent uses `Box.Exec()` for commands | Replace Docker exec |
| Agent uses `Box.VNC()` / `Box.Proxy()` | Replace vncproxy |
| Agent injects `Box` as `SandboxExecutor` | `ctx.sandbox` JSAPI unchanged for hooks |
| `BuildMCPConfigForSandbox()` uses Box env vars | No more hardcoded `/tmp/yao.sock` |

### Phase 4: Cutover

| Task | Detail |
|------|--------|
| Move `sandbox/v2` → `sandbox` | Rename package |
| Delete old sandbox code | manager.go, ipc/, bridge/, vncproxy/, docker/ |
| Delete `DESIGN-REMOTE.md` | Superseded by tai.Client |
| Update `cmd/start.go` | Use new init path |

## What Gets Deleted (Phase 4)

Everything in the current `sandbox/` that is replaced by tai:

| Old | Replaced by |
|-----|------------|
| `manager.go` (Docker `*client.Client`) | `tai.Client.Sandbox()` |
| `ipc/` (Unix socket manager) | gRPC via `yao/grpc` + `tai/grpc` |
| `bridge/` (stdio→socket bridge) | `yao-grpc` binary (`tai/grpc/cmd`) |
| `vncproxy/` (Docker-based VNC) | `tai.Client.VNC()` |
| `proxy/` (Claude API proxy) | separate concern, not sandbox |
| `docker/` (Dockerfiles) | kept, they're image build files |
| `DESIGN-REMOTE.md` (Runtime interface) | tai.Client is the abstraction |
| `config.go` (old config) | new config in v2 |
| `helpers.go` (Docker helpers) | not needed |

## Comparison: V1 vs V2

| Aspect | V1 (current) | V2 (this design) |
|--------|-------------|-------------------|
| **Positioning** | Agent's Claude executor | Yao infrastructure module |
| **Runtime** | Direct Docker SDK | tai.Client pool (Docker/K8s/Remote) |
| **Execution** | Exec + Stream | Exec + Stream + Attach (service connections) |
| **File I/O** | bind mount + Docker Copy | `workspace.FS` (fs.FS compatible) |
| **IPC** | Unix socket + yao-bridge | gRPC (yao-grpc, already done) |
| **Idle detection** | External calls only | Dual: external calls + container heartbeat |
| **Lifecycle** | Chat session only | Policy-based (oneshot/session/longrunning/persistent) |
| **Pool** | Single Docker daemon | Multi-pool with per-pool policies, dynamic add/remove |
| **Agent coupling** | Tightly coupled | Zero dependency |
| **JSAPI** | Only `ctx.sandbox` in hooks | Global `Sandbox()` + `ctx.sandbox` |
| **Process** | None | `sandbox.*` namespace |
| **Multi-node** | Local only | Local + Remote via Tai |
| **K8s** | Not supported | Supported via tai.Client |

## Workspace — First-Class Entity

### Problem

Current design: `Box.Workspace()` returns `workspace.FS` keyed by `box.id` — workspace and container are 1:1, same lifecycle. This couples file storage to container lifetime.

Real usage pattern:

```
User creates a project → uploads files → works on it across multiple chat sessions
  → attaches a long-running dev server → destroys/rebuilds containers freely
  → project files must survive all of this
```

Workspace must outlive containers. It is the persistent artifact; containers are disposable compute.

### Design

Workspace becomes an independent entity with its own CRUD, decoupled from both Chat sessions and containers.

```
Workspace (persistent, user-managed)
    ├── CRUD / file management UI
    ├── Mountable to 0~N containers simultaneously
    └── Referenced by 0~N Chat sessions

Chat Session
    └── Selects a Workspace (not a container)

Container (ephemeral compute)
    ├── Bind-mounts a Workspace to /workspace
    ├── rw or ro per mount
    └── Created/destroyed independently of Workspace
```

### Workspace struct

```go
type Workspace struct {
    ID        string            // unique identifier, e.g. "ws-abc123"
    Name      string            // human-readable, e.g. "my-react-app"
    Owner     string            // user ID
    Labels    map[string]string // arbitrary metadata
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

No container references stored here. Workspace is pure storage — it doesn't know or care about containers.

### MountMode

```go
type MountMode string

const (
    MountRW MountMode = "rw"   // read-write (default)
    MountRO MountMode = "ro"   // read-only
)
```

Rules:
- A Workspace can be mounted by multiple containers simultaneously
- Each mount independently specifies `rw` or `ro`
- No write-lock enforcement — caller manages concurrency
- Default is `rw`

Rationale: In practice, Chat containers write source code and Runtime containers write build artifacts/logs — different files, no real conflict. Enforcing locks adds complexity without solving a real problem in this use case.

### CreateOptions changes

```go
type CreateOptions struct {
    // ... existing fields ...

    // Workspace mount (new)
    WorkspaceID string    // workspace to mount; empty = no workspace
    MountMode   MountMode // "rw" (default) or "ro"
    MountPath   string    // container path; default "/workspace"
}
```

When `WorkspaceID` is set, Manager resolves the storage path via `VolumeProvider.MountSpec()` and injects the bind mount into the container create options.

### Manager API additions

```go
// --- Workspace CRUD ---

// CreateWorkspace creates a persistent workspace.
// Storage is allocated via VolumeProvider.ResolvePath().
func (m *Manager) CreateWorkspace(ctx context.Context, opts WorkspaceOptions) (*Workspace, error)

// GetWorkspace returns a workspace by ID.
func (m *Manager) GetWorkspace(ctx context.Context, id string) (*Workspace, error)

// ListWorkspaces returns workspaces, optionally filtered by owner.
func (m *Manager) ListWorkspaces(ctx context.Context, opts WorkspaceListOptions) ([]*Workspace, error)

// DeleteWorkspace removes workspace storage.
// Fails if any containers currently mount it (unless force=true).
func (m *Manager) DeleteWorkspace(ctx context.Context, id string, force bool) error

type WorkspaceOptions struct {
    ID     string            // explicit ID; empty = auto-generate
    Name   string            // human-readable name
    Owner  string
    Labels map[string]string
}

type WorkspaceListOptions struct {
    Owner string
}
```

### Container creation flow (updated)

```
Manager.Create(ctx, CreateOptions{
    Image:       "yaoapp/workspace:latest",
    WorkspaceID: "ws-abc123",      // ← new
    MountMode:   MountRW,          // ← new
})

  1. Validate CreateOptions (image required, etc.)
  2. If WorkspaceID set:
     a. Verify workspace exists
     b. spec := provider.MountSpec(workspaceID)
     c. Inject into tai CreateOptions:
        - Docker: opts.Binds = ["/data/ws/ws-abc123:/workspace:rw"]
        - K8s:    opts.Volumes + opts.VolumeMounts (PVC)
  3. Create container via tai.Client.Sandbox().Create()
  4. Start container
  5. Return Box
```

### Box.Workspace() behavior change

```go
func (b *Box) Workspace() workspace.FS {
    // If container has a workspace mounted, use the workspace ID as session.
    // Otherwise fall back to box ID (backward compatible).
    sessionID := b.workspaceID
    if sessionID == "" {
        sessionID = b.id
    }
    client, _ := b.manager.getPool(b.pool)
    return client.Workspace(sessionID)
}
```

Multiple boxes mounting the same workspace → same `sessionID` → same files via Volume API.

### Typical flows

**Flow 1: Workspace management UI**

```
1. User creates workspace "my-project"
   → Manager.CreateWorkspace(opts) → VolumeProvider.ResolvePath("ws-123")
   → Directory /data/ws/ws-123/ created

2. User uploads files via Workspace management UI
   → Volume.WriteFile(ctx, "ws-123", "src/main.go", data, 0644)
   → Files written to /data/ws/ws-123/src/main.go

3. User browses files
   → Volume.ListDir(ctx, "ws-123", "src/")
   → Returns file listing from /data/ws/ws-123/src/
```

**Flow 2: Chat with Workspace**

```
1. User opens Chat, selects workspace "my-project" (ws-123)

2. Agent needs a container:
   → Manager.Create(ctx, CreateOptions{
       Image: "yaoapp/workspace:latest",
       WorkspaceID: "ws-123",
       MountMode: MountRW,
     })
   → Container starts with -v /data/ws/ws-123:/workspace:rw
   → Agent can exec "ls /workspace/src/" inside container

3. Chat ends, container destroyed
   → Workspace files persist in /data/ws/ws-123/
```

**Flow 3: Long-running Runtime**

```
1. User starts Runtime container for workspace "my-project":
   → Manager.Create(ctx, CreateOptions{
       Image: "node:20",
       WorkspaceID: "ws-123",
       MountMode: MountRW,
       Policy: Persistent,
       Ports: [{ContainerPort: 3000}],
     })
   → Container starts with -v /data/ws/ws-123:/workspace:rw
   → Inside container: cd /workspace && npm install && npm run dev

2. User accesses dev server:
   → box.Proxy(ctx, 3000, "/") → "http://localhost:32768/"
   → Or box.VNC(ctx) for desktop preview

3. User opens Chat, selects same workspace:
   → Manager.Create(ctx, CreateOptions{
       Image: "yaoapp/agent:latest",
       WorkspaceID: "ws-123",
       MountMode: MountRW,
     })
   → Second container, same workspace mounted
   → Agent modifies source → Runtime hot-reloads

4. Chat ends, Chat container destroyed
   → Runtime container keeps running
   → Workspace files persist
```

### Storage backend (already implemented in Tai)

The `storage.VolumeProvider` interface in Tai Server already has three complete implementations:

```go
// tai/storage/provider.go
type VolumeProvider interface {
    ResolvePath(sessionID string) (string, error)
    MountSpec(sessionID string) MountConfig
    Cleanup(sessionID string) error
}

type MountConfig struct {
    Type   string // "bind" | "volume" | "pvc"
    Source string
    Target string // always /workspace
}
```

| Provider | Backend | MountSpec | Status |
|----------|---------|-----------|--------|
| `BindMountProvider` | Host directory (`/data/ws/{id}/`) | `type:"bind"` | Implemented, default |
| `DockerVolumeProvider` | Docker named volume (`tai-{id}`) | `type:"volume"` | Implemented |
| `K8sPVCProvider` | K8s PVC (`tai-{id}-pvc`, 10Gi RWO) | `type:"pvc"` | Implemented |

These are implemented but **not yet wired** into the container creation flow. The only work needed is calling `MountSpec()` during `Manager.Create()` and passing the result into `tai.sandbox.CreateOptions.Binds`.

For file operations (CRUD UI), Tai's `Volume` gRPC service already operates on the same `dataDir/{sessionID}/` paths. No additional work needed — `Volume.ReadFile("ws-123", "src/main.go")` reads from the same directory that gets bind-mounted into containers.

### Workspace metadata storage

Workspace metadata (ID, Name, Owner, Labels, timestamps) needs persistent storage.

Recommendation: **JSON file** (`/data/ws/{id}/.workspace.json`) for Phase 1. Each workspace directory contains its own metadata. Listing = scan directories + read metadata files. Zero dependencies, works everywhere.

```json
{
  "id": "ws-abc123",
  "name": "my-react-app",
  "owner": "user-001",
  "labels": {"project": "frontend"},
  "created_at": "2026-03-05T10:00:00Z",
  "updated_at": "2026-03-05T12:30:00Z"
}
```

Can migrate to SQLite or Yao DB later if query/filter requirements grow.

### Process registration additions

| Process | Args | Returns |
|---------|------|---------|
| `sandbox.workspace.Create` | `options` (WorkspaceOptions JSON) | Workspace |
| `sandbox.workspace.Get` | `id` | Workspace |
| `sandbox.workspace.List` | `options` (WorkspaceListOptions JSON) | []Workspace |
| `sandbox.workspace.Delete` | `id`, `force?` | — |

### JSAPI additions

```javascript
// Workspace CRUD
var ws = Sandbox.CreateWorkspace({ name: "my-project", owner: "user-001" })
var ws = Sandbox.GetWorkspace("ws-abc123")
var list = Sandbox.ListWorkspaces({ owner: "user-001" })
Sandbox.DeleteWorkspace("ws-abc123")

// File operations on workspace (without a container)
ws.ReadFile("src/main.go")
ws.WriteFile("src/main.go", "package main\n...")
ws.ListDir("src/")
ws.Remove("tmp.txt")

// Create container with workspace
var sb = Sandbox("my-box", {
    image: "node:20",
    workspace_id: ws.id,
    mount_mode: "rw",
})
```

### What changes from current design

| Aspect | Before | After |
|--------|--------|-------|
| Workspace lifecycle | Tied to Box (same ID, same lifetime) | Independent entity, outlives containers |
| Workspace identity | `sessionID = box.id` | `sessionID = workspace.id` (explicit) |
| Container ↔ Workspace | 1:1, implicit | N:1, explicit via `CreateOptions.WorkspaceID` |
| File persistence | Lost when container removed | Persists until workspace deleted |
| Multi-container access | Not possible | Multiple containers mount same workspace |
| Storage backend | Volume gRPC only (no mount) | Volume gRPC + bind mount into container |
| CRUD without container | Not possible | Via Volume API directly |

### Implementation plan

**Phase 1.5** (between current Phase 1 and Phase 2):

| Task | Detail |
|------|--------|
| `workspace.go` | Workspace struct, WorkspaceOptions, metadata JSON read/write |
| Manager: workspace CRUD | `CreateWorkspace` / `GetWorkspace` / `ListWorkspaces` / `DeleteWorkspace` via VolumeProvider + JSON metadata |
| Manager: `Create()` updated | Wire `WorkspaceID` → `VolumeProvider.MountSpec()` → `Binds` |
| `Box.Workspace()` updated | Use `workspaceID` as sessionID when set |
| Tai Server: wire `VolumeProvider` | Call `MountSpec()` in container creation path |
| Tests | Workspace CRUD + mount verification |

No breaking changes. Containers created without `WorkspaceID` work exactly as before (`sessionID = box.id`, no bind mount).
