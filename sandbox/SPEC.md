# Sandbox Functional Specification

Detailed interfaces, types, and behavior for the sandbox refactoring.
Architecture and rationale: see [DESIGN.md](./DESIGN.md).

---

## 1. sandbox.Manager

Replaces the current Docker-only Manager. Backed by a single `tai.Client`.

### Config

```go
type Config struct {
    Image            string        // container image, default "yaoapp/workspace:latest"
    MaxContainers    int           // global limit, default 100
    IdleTimeout      time.Duration // default cleanup interval, default 30m
    MaxMemory        string        // per-container, e.g. "2g"
    MaxCPU           float64       // per-container, e.g. 1.0
    ContainerWorkDir string        // mount target inside container, default "/workspace"
    ContainerUser    string        // empty = image default
}
```

Environment variable overrides remain the same (`YAO_SANDBOX_IMAGE`, etc.). `WorkspaceRoot` and `IPCDir` are removed — local paths derived from `tai.Client.IsLocal()` at runtime; remote mode uses `tai.Client.Volume()`.

### Constructor

```go
func NewManager(client *tai.Client, cfg *Config) (*Manager, error)
```

- Validates `client` is non-nil and healthy (calls `client.Sandbox().List()` as connectivity check)
- Starts background cleanup goroutine
- Returns ready Manager

### Manager struct

```go
type Manager struct {
    client     *tai.Client
    config     *Config
    sandboxes  sync.Map       // name → *Sandbox
    running    atomic.Int32
    ipc        *IPCRouter     // local: Unix socket manager, remote: gRPC stub
    cleanup    *time.Ticker
    done       chan struct{}
}
```

### Public methods

```go
// Lifecycle
func (m *Manager) GetOrCreate(ctx context.Context, opts GetOrCreateOptions) (*Sandbox, error)
func (m *Manager) Get(ctx context.Context, name string) (*Sandbox, error)
func (m *Manager) Start(ctx context.Context, name string) error
func (m *Manager) Stop(ctx context.Context, name string, timeout time.Duration) error
func (m *Manager) Remove(ctx context.Context, name string) error
func (m *Manager) List(ctx context.Context, filter ListFilter) ([]*Sandbox, error)

// Execution
func (m *Manager) Exec(ctx context.Context, name string, cmd []string, opts ExecOptions) (*ExecResult, error)
func (m *Manager) Stream(ctx context.Context, name string, cmd []string, opts ExecOptions) (io.ReadCloser, error)
func (m *Manager) KillProcess(ctx context.Context, name string, pattern string) error

// File operations (routes local/remote internally)
func (m *Manager) ReadFile(ctx context.Context, name string, path string) ([]byte, error)
func (m *Manager) WriteFile(ctx context.Context, name string, path string, data []byte) error
func (m *Manager) ListDir(ctx context.Context, name string, path string) ([]FileInfo, error)
func (m *Manager) Stat(ctx context.Context, name string, path string) (*FileInfo, error)
func (m *Manager) MkDir(ctx context.Context, name string, path string) error
func (m *Manager) RemoveFile(ctx context.Context, name string, path string) error
func (m *Manager) CopyToContainer(ctx context.Context, name string, hostPath, containerPath string) error
func (m *Manager) CopyFromContainer(ctx context.Context, name string, containerPath, hostPath string) error

// Info
func (m *Manager) IsLocal() bool
func (m *Manager) Close() error
```

---

## 2. Types

### Sandbox

```go
type Sandbox struct {
    Name       string
    UserID     string
    ChatID     string
    Image      string
    Status     Status
    Lifecycle  Lifecycle
    CreatedAt  time.Time
    LastUsedAt time.Time
    IP         string
}
```

### Status

```go
type Status string

const (
    StatusCreated Status = "created"
    StatusRunning Status = "running"
    StatusStopped Status = "stopped"
)
```

### Lifecycle

```go
type Lifecycle string

const (
    LifecycleOneShot     Lifecycle = "one-shot"     // destroyed after execution
    LifecycleSession     Lifecycle = "session"       // alive while user active, idle timeout
    LifecycleLongRunning Lifecycle = "long-running"  // hours/days, recoverable
    LifecyclePersistent  Lifecycle = "persistent"    // never auto-cleaned
)
```

### GetOrCreateOptions

```go
type GetOrCreateOptions struct {
    UserID    string
    ChatID    string
    Image     string            // override Config.Image
    Lifecycle Lifecycle         // default: LifecycleSession
    Env       map[string]string // injected into container
    Cmd       []string          // override entrypoint
    Memory    string            // override Config.MaxMemory
    CPU       float64           // override Config.MaxCPU
}
```

### ExecOptions / ExecResult

```go
type ExecOptions struct {
    WorkDir string
    Env     map[string]string
    Stdin   io.Reader
    Timeout time.Duration
}

type ExecResult struct {
    ExitCode int
    Stdout   string
    Stderr   string
}
```

### ListFilter

```go
type ListFilter struct {
    UserID    string    // empty = all users
    Status    Status    // empty = all statuses
    Lifecycle Lifecycle // empty = all policies
}
```

### FileInfo

```go
type FileInfo struct {
    Name    string
    Path    string
    Size    int64
    Mode    os.FileMode
    ModTime time.Time
    IsDir   bool
}
```

### Errors

```go
var (
    ErrTooManyContainers  = errors.New("sandbox: container limit reached")
    ErrNotFound           = errors.New("sandbox: not found")
    ErrNotRunning         = errors.New("sandbox: not running")
    ErrAlreadyExists      = errors.New("sandbox: already exists")
)
```

---

## 3. Lifecycle State Machine

```
                    GetOrCreate()
                         │
                         ▼
                    ┌─────────┐
          ┌────────│ Created  │
          │        └────┬────┘
          │   Start()   │
          │             ▼
          │        ┌─────────┐    idle timeout / Stop()
          │        │ Running  │──────────────────┐
          │        └────┬────┘                   │
          │             │                        ▼
          │             │                   ┌─────────┐
          │             │                   │ Stopped  │
          │             │                   └────┬────┘
          │             │            Start()     │
          │             │    ┌───────────────────┘
          │             │    │
          │             ▼    ▼
          │        Remove() from any state
          │             │
          │             ▼
          │        [Destroyed]
          │
          └── one-shot: auto Remove() after Exec/Stream returns
```

### Cleanup rules

| Lifecycle | Trigger | Action |
|-----------|---------|--------|
| one-shot | Exec/Stream completes | Manager.Remove() immediately |
| session | `IdleTimeout` since `LastUsedAt` | Manager.Stop() then Remove() |
| long-running | `IdleTimeout * 24` since `LastUsedAt` | Manager.Stop() (not removed, can restart) |
| persistent | Never | No automatic action |

Background goroutine runs every `Config.IdleTimeout / 2`, scans `sandboxes`, applies rules.

### Touch

Every `Exec`, `Stream`, `ReadFile`, `WriteFile`, `ListDir` call updates `LastUsedAt`.

---

## 4. File Operations Routing

```go
func (m *Manager) ReadFile(ctx context.Context, name string, path string) ([]byte, error) {
    if m.client.IsLocal() {
        hostPath := m.hostPath(name, path)
        return os.ReadFile(hostPath)
    }
    sessionID := m.sessionID(name)
    ws := m.client.Workspace(sessionID)
    f, err := ws.Open(path)
    // ... read and return
}
```

| Operation | Local | Remote |
|-----------|-------|--------|
| ReadFile | `os.ReadFile(hostPath)` | `client.Workspace(session).Open(path)` |
| WriteFile | `os.WriteFile(hostPath)` | `client.Volume().Write(session, path, data)` |
| ListDir | `os.ReadDir(hostPath)` | `client.Volume().ReadDir(session, path)` |
| Stat | `os.Stat(hostPath)` | `client.Volume().Stat(session, path)` |
| MkDir | `os.MkdirAll(hostPath)` | `client.Volume().MkDir(session, path)` |
| RemoveFile | `os.RemoveAll(hostPath)` | `client.Volume().Remove(session, path)` |
| CopyToContainer | bind mount (noop, already on host) | `client.Volume().Write()` streamed |
| CopyFromContainer | bind mount (direct read) | `client.Volume().Read()` streamed |

`hostPath` = `dataDir/{userID}/{chatID}/{containerRelativePath}`

`sessionID` = `{userID}/{chatID}` (maps to volume session on Tai)

---

## 5. IPC Router

Abstracts local Unix socket vs remote gRPC relay. Manager creates the right one based on `client.IsLocal()`.

### Interface

```go
type IPCRouter interface {
    Create(sessionID string, tools []MCPTool) (IPCSession, error)
    Get(sessionID string) (IPCSession, error)
    Close(sessionID string) error
    CloseAll() error
}

type IPCSession interface {
    SetTools(tools []MCPTool)
    SetContext(ctx *AgentContext)
    SocketPath() string   // local only, empty for remote
    GRPCAddr() string     // remote only, empty for local
    Close() error
}
```

### Local implementation

Same as current `ipc.Manager` — creates Unix socket per session, bind-mounts into container, yao-bridge connects to it.

### Remote implementation

No socket. Container receives `YAO_IPC_MODE=grpc` and `YAO_IPC_ADDR=tai-host:9100`. `yao-bridge` (`yao/tai/bridge/`) connects to Tai's gRPC relay, which forwards to Yao gRPC Server. Tai relay upstream is per-container via `CreateRequest.GRPCUpstream`, not a Tai startup parameter.

Tool registration: remote IPCSession sends tool list to Yao gRPC Server via a registration RPC at session creation.

### Container env injection

```go
func (m *Manager) buildContainerEnv(session IPCSession, userEnv map[string]string) map[string]string {
    env := maps.Clone(userEnv)
    if m.client.IsLocal() {
        env["YAO_IPC_MODE"] = "socket"
        env["YAO_IPC_ADDR"] = session.SocketPath()
    } else {
        env["YAO_IPC_MODE"] = "grpc"
        env["YAO_IPC_ADDR"] = session.GRPCAddr()
        env["YAO_TOKEN"] = m.issueAccessToken(session)
        env["YAO_REFRESH_TOKEN"] = m.issueRefreshToken(session)
    }
    return env
}

// CreateRequest also carries GRPCUpstream for Tai relay routing (per-container, not per-Tai)

```

---

## 6. Yao gRPC Server

### Proto definition

```protobuf
syntax = "proto3";
package yao.v1;

service Yao {
    rpc Exec(ExecRequest) returns (ExecResponse);
    rpc StreamExec(ExecRequest) returns (stream ExecChunk);

    // MCP tool registration (called by remote IPC sessions)
    rpc RegisterTools(RegisterToolsRequest) returns (RegisterToolsResponse);

    // Health
    rpc Healthz(HealthzRequest) returns (HealthzResponse);
}

message ExecRequest {
    string process = 1;     // e.g. "models.user.Find"
    bytes  args    = 2;     // JSON-encoded arguments
    string session = 3;     // sandbox session ID for context
}

message ExecResponse {
    bytes  result = 1;      // JSON-encoded result
    string error  = 2;
}

message ExecChunk {
    bytes data = 1;
    bool  done = 2;
}

message RegisterToolsRequest {
    string session = 1;
    repeated MCPToolDef tools = 2;
}

message MCPToolDef {
    string name        = 1;
    string description = 2;
    string process     = 3;    // Yao process to call
    bytes  input_schema = 4;   // JSON Schema
}

message RegisterToolsResponse {}

message HealthzRequest {}
message HealthzResponse {
    string status = 1;
}
```

### Server startup

```go
func StartGRPCServer(cfg GRPCConfig) (*grpc.Server, error)

type GRPCConfig struct {
    Listen    string   // "127.0.0.1:9099" or "0.0.0.0:9099"
    AllowCIDR []string // IP allowlist, empty = no restriction
}
```

Interceptor chain: `ipAllowInterceptor` → `authInterceptor` → handler.

### Exec handler

```go
func (s *yaoServer) Exec(ctx context.Context, req *pb.ExecRequest) (*pb.ExecResponse, error) {
    claims := claimsFromContext(ctx)
    // ACL check: does this token have permission to call this process?
    
    p := process.New(req.Process)
    var args []interface{}
    json.Unmarshal(req.Args, &args)
    
    result, err := p.Exec(args...)
    if err != nil {
        return &pb.ExecResponse{Error: err.Error()}, nil
    }
    
    data, _ := json.Marshal(result)
    return &pb.ExecResponse{Result: data}, nil
}
```

---

## 7. Agent Layer

### Assistant sandbox config

```yaml
# assistants/coder.yao
sandbox:
  enabled: true
  lifecycle: session
  idle_timeout: 30m
  image: yaoapp/workspace:latest
  command: claude
  memory: "4g"
  cpu: 2.0
```

### Parsed config type

```go
type AssistantSandboxConfig struct {
    Enabled     bool          `json:"enabled"`
    Lifecycle   Lifecycle     `json:"lifecycle"`
    IdleTimeout time.Duration `json:"idle_timeout"`
    Image       string        `json:"image"`
    Command     string        `json:"command"`
    Memory      string        `json:"memory"`
    CPU         float64       `json:"cpu"`
}
```

### Init flow (new)

```go
func (a *Assistant) initSandbox(ctx context.Context) (*agentsandbox.Executor, error) {
    mgr := GetSandboxManager()  // global, initialized with tai.Client at Yao startup
    
    sb, err := mgr.GetOrCreate(ctx, sandbox.GetOrCreateOptions{
        UserID:    a.userID,
        ChatID:    a.chatID,
        Image:     a.config.Sandbox.Image,
        Lifecycle: a.config.Sandbox.Lifecycle,
        Memory:    a.config.Sandbox.Memory,
        CPU:       a.config.Sandbox.CPU,
    })
    // ...
    executor := agentsandbox.New(mgr, sb, a.config.Sandbox.Command)
    return executor, nil
}
```

### Cleanup (new)

```go
func (a *Assistant) sandboxCleanup(executor *agentsandbox.Executor) {
    executor.Disconnect()
    // Manager handles actual removal based on lifecycle policy.
    // one-shot: already removed after Exec.
    // session: will be cleaned up by background goroutine after idle timeout.
    // long-running/persistent: stays.
}
```

### GetSandboxManager (new)

```go
var (
    managerOnce sync.Once
    manager     *sandbox.Manager
)

func GetSandboxManager() *sandbox.Manager {
    managerOnce.Do(func() {
        client := config.GetTaiClient()  // initialized at Yao startup from env/config
        mgr, err := sandbox.NewManager(client, loadSandboxConfig())
        if err != nil {
            log.Fatal("sandbox manager init: %v", err)
        }
        manager = mgr
    })
    return manager
}
```

### Executor factory

```go
// agent/sandbox/executor.go
func New(mgr *sandbox.Manager, sb *sandbox.Sandbox, command string) Executor {
    switch command {
    case "claude":
        return claude.NewExecutor(mgr, sb)
    default:
        return generic.NewExecutor(mgr, sb)
    }
}
```

### Executor interface (unchanged)

```go
type Executor interface {
    Stream(ctx context.Context, opts StreamOptions) (io.ReadCloser, error)
    Disconnect() error

    // Delegated to Manager internally
    ReadFile(ctx context.Context, path string) ([]byte, error)
    WriteFile(ctx context.Context, path string, data []byte) error
    ListDir(ctx context.Context, path string) ([]FileInfo, error)
    Exec(ctx context.Context, cmd []string) (string, error)
    GetWorkDir() string
    GetSandboxID() string
    GetVNCUrl() string
}
```

Each method delegates to `mgr.ReadFile(ctx, sb.Name, path)` etc. The executor is a thin wrapper that knows the sandbox name.

---

## 8. Naming Convention

| Entity | Pattern | Example |
|--------|---------|---------|
| Container/Pod name | `yao-sb-{userID}-{chatID}` | `yao-sb-u123-c456` |
| Volume session | `{userID}/{chatID}` | `u123/c456` |
| IPC session | `{chatID}` | `c456` |
| Host workspace (local) | `{dataDir}/{userID}/{chatID}/` | `/data/u123/c456/` |

Prefix shortened from `yao-sandbox-` to `yao-sb-` for K8s DNS name length limit (63 chars).

---

## 9. Environment Variables

### Yao process

| Variable | Purpose | Default |
|----------|---------|---------|
| `YAO_TAI_ADDR` | Tai endpoint, e.g. `tai://10.0.0.1` or empty for local Docker | `""` (local) |
| `YAO_TAI_RUNTIME` | `docker` or `k8s` | `docker` |
| `YAO_TAI_KUBECONFIG` | Path to kubeconfig (K8s only) | |
| `YAO_TAI_NAMESPACE` | K8s namespace | `default` |
| `YAO_GRPC_LISTEN` | gRPC server listen address | `127.0.0.1:9099` |
| `YAO_GRPC_ALLOW` | CIDR allowlist, comma-separated | (empty = no filter) |
| `YAO_SANDBOX_IMAGE` | Default container image | `yaoapp/workspace:latest` |
| `YAO_SANDBOX_MAX` | Max containers | `100` |
| `YAO_SANDBOX_IDLE_TIMEOUT` | Idle timeout duration | `30m` |
| `YAO_SANDBOX_MEMORY` | Memory limit | `2g` |
| `YAO_SANDBOX_CPU` | CPU limit | `1.0` |

### Container-internal

| Variable | Purpose | Set by |
|----------|---------|--------|
| `YAO_IPC_MODE` | `socket` or `grpc` | Manager at creation |
| `YAO_IPC_ADDR` | Socket path or gRPC host:port | Manager at creation |
| `YAO_TOKEN` | JWT access token for gRPC auth (remote only, short TTL 15m) | Manager at creation |
| `YAO_REFRESH_TOKEN` | JWT refresh token (remote only, no expiry, revoked on Remove) | Manager at creation |
