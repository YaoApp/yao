# Workspace Design Document

> **Status**: Draft
> **Module**: `workspace` (top-level, parallel to `sandbox/v2`)
> **Depends on**: `tai` SDK (Volume, VolumeProvider, Sandbox), `sandbox/v2` Manager

---

## Overview

Workspace is a **first-class, persistent storage entity** independent of containers, chat sessions, and user sessions. It represents a user's project files — source code, configs, build artifacts — that can be mounted into any number of ephemeral containers.

Workspace is the **anchor point** for container scheduling: when a Workspace is created on a specific Tai node (host machine), all subsequent containers that reference it are automatically routed to the same node, because bind mounts require co-location on the same physical host.

---

## Problem

Current design: `Box.Workspace()` returns `workspace.FS` keyed by `box.id` — workspace and container are 1:1, same lifecycle. This couples file storage to container lifetime.

Real usage pattern:

```
User creates a project → uploads files → works on it across multiple chat sessions
  → attaches a long-running dev server → destroys/rebuilds containers freely
  → project files must survive all of this
```

Workspace must outlive containers. It is the persistent artifact; containers are disposable compute.

---

## Architecture

```
┌──────────────────────────────────────────────────┐
│                  Application Layer                │
│                                                   │
│  Workspace Management UI        Chat Interface    │
│  ┌─────────────────────┐    ┌─────────────────┐  │
│  │ Create / Delete / UI │    │ Select Workspace│  │
│  │ Browse / Upload     │    │ Start Chat      │  │
│  └─────────┬───────────┘    └────────┬────────┘  │
│            │                         │            │
└────────────┼─────────────────────────┼────────────┘
             │                         │
             ▼                         ▼
┌──────────────────────────────────────────────────┐
│                  Yao Engine                        │
│                                                   │
│  workspace.Manager              sandbox.Manager   │
│  ┌────────────────┐          ┌─────────────────┐  │
│  │ CRUD            │◄────────│ Mount workspace │  │
│  │ File I/O        │         │ Route to node   │  │
│  │ Node binding    │         │ Create container│  │
│  └────────┬───────┘          └────────┬────────┘  │
│           │                           │           │
└───────────┼───────────────────────────┼───────────┘
            │                           │
            ▼                           ▼
┌──────────────────────────────────────────────────┐
│                  Tai Node (Host)                   │
│                                                   │
│  Volume gRPC             Container Runtime        │
│  ┌──────────────┐      ┌─────────────────────┐   │
│  │ ReadFile      │      │ Container A (rw)    │   │
│  │ WriteFile     │      │  └─ /workspace ─┐   │   │
│  │ ListDir       │      │                  │   │   │
│  │ SyncPush/Pull │      │ Container B (ro) │   │   │
│  └──────┬───────┘      │  └─ /workspace ─┐│   │   │
│         │               └────────────────┼┼───┘   │
│         │                                ││       │
│         ▼                                ▼▼       │
│  ┌──────────────────────────────────────────┐     │
│  │  /data/ws/{workspace-id}/                │     │
│  │    ├── .workspace.json  (metadata)       │     │
│  │    ├── src/                              │     │
│  │    ├── package.json                      │     │
│  │    └── ...                               │     │
│  └──────────────────────────────────────────┘     │
│                                                   │
│  VolumeProvider                                   │
│  ┌─────────────┬──────────────┬──────────────┐   │
│  │ BindMount   │ DockerVolume │ K8s PVC      │   │
│  │ (default)   │              │              │   │
│  └─────────────┴──────────────┴──────────────┘   │
└───────────────────────────────────────────────────┘
```

---

## Core Design

### Node Binding

Workspace is physically stored on a Tai node's disk. **Bind mount requires Workspace and container to be on the same host.** Therefore:

- **Workspace binds to a specific Tai node at creation time.** This binding is immutable.
- When a container references a Workspace (`CreateOptions.WorkspaceID`), the container is **automatically routed to the same Tai node** — the caller does not (and should not) specify a Pool.
- One Tai node = one Pool = one host machine. These are equivalent in the current architecture.

```
创建 Workspace:
    用户选择节点 "gpu-server" → workspace.Create(opts)
    → Tai "gpu-server" 上创建 /data/ws/ws-123/

创建容器（选了 Workspace）:
    → sandbox.Create(opts, WorkspaceID: "ws-123")
    → Manager 查到 ws-123 绑在 "gpu-server"
    → 自动路由到 "gpu-server" Pool
    → bind mount /data/ws/ws-123:/workspace:rw  ✓ 同机

创建容器（没选 Workspace）:
    → 按原逻辑选 Pool（用户指定或默认）
```

This makes Workspace the **scheduling anchor**: once a Workspace is chosen, the node is determined.

### Workspace struct

```go
type Workspace struct {
    ID        string            // unique identifier, e.g. "ws-abc123"
    Name      string            // human-readable, e.g. "my-react-app"
    Owner     string            // user ID
    Node      string            // Tai node name (= Pool name); set at creation, immutable
    Labels    map[string]string // arbitrary metadata
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

`Node` is the critical field: it pins this Workspace to a specific machine. All container operations referencing this Workspace are routed to this node.

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

---

## API Design

### workspace.Manager

Workspace has its own manager, separate from `sandbox.Manager`. It owns Workspace CRUD and file I/O.

```go
package workspace

type Manager struct {
    pools map[string]*tai.Client // node name → tai client (shared with sandbox.Manager)
}

// NewManager creates a workspace manager with the given pools.
// Pools are shared with sandbox.Manager — both reference the same tai.Client instances.
func NewManager(pools map[string]*tai.Client) *Manager
```

### Workspace CRUD

```go
type CreateOptions struct {
    ID     string            // explicit ID; empty = auto-generate (uuid)
    Name   string            // human-readable name
    Owner  string            // user ID
    Node   string            // target Tai node (required)
    Labels map[string]string
}

type ListOptions struct {
    Owner string // filter by owner; empty = all
    Node  string // filter by node; empty = all
}

// Create allocates storage on the target node and persists metadata.
func (m *Manager) Create(ctx context.Context, opts CreateOptions) (*Workspace, error)

// Get returns a workspace by ID.
// Checks the metadata file on the bound node.
func (m *Manager) Get(ctx context.Context, id string) (*Workspace, error)

// List returns workspaces, optionally filtered.
func (m *Manager) List(ctx context.Context, opts ListOptions) ([]*Workspace, error)

// Delete removes workspace storage from the node.
// Fails if containers currently mount it (unless force=true).
func (m *Manager) Delete(ctx context.Context, id string, force bool) error

// Update modifies workspace metadata (Name, Labels).
// Node and Owner are immutable after creation.
func (m *Manager) Update(ctx context.Context, id string, opts UpdateOptions) (*Workspace, error)

type UpdateOptions struct {
    Name   *string            // nil = no change
    Labels map[string]string  // nil = no change; non-nil replaces all
}
```

### File I/O (no container needed)

File operations go through the Tai `Volume` gRPC service, using the Workspace ID as the session identifier. No container is needed.

```go
// FS returns an fs.FS view of the workspace, backed by Tai Volume gRPC.
func (m *Manager) FS(ctx context.Context, id string) (workspace.FS, error)

// ReadFile reads a file from the workspace.
func (m *Manager) ReadFile(ctx context.Context, id string, path string) ([]byte, error)

// WriteFile writes a file to the workspace.
func (m *Manager) WriteFile(ctx context.Context, id string, path string, data []byte, perm os.FileMode) error

// ListDir lists entries in a workspace directory.
func (m *Manager) ListDir(ctx context.Context, id string, path string) ([]DirEntry, error)

// Remove deletes a file or directory from the workspace.
func (m *Manager) Remove(ctx context.Context, id string, path string) error

// SyncPush uploads a local directory tree to the workspace.
func (m *Manager) SyncPush(ctx context.Context, id string, localPath string) error

// SyncPull downloads the workspace to a local directory.
func (m *Manager) SyncPull(ctx context.Context, id string, localPath string) error
```

These are thin wrappers around `tai.Client.Volume().{ReadFile,WriteFile,ListDir,...}` — the Tai SDK already implements all of these.

---

## Integration with Sandbox

### sandbox.CreateOptions changes

```go
type CreateOptions struct {
    // ... existing fields ...

    WorkspaceID string    // workspace to mount; empty = no workspace
    MountMode   MountMode // "rw" (default) or "ro"
    MountPath   string    // container path; default "/workspace"
}
```

### Container creation flow

When `WorkspaceID` is set in `CreateOptions`, the sandbox Manager:

```
Manager.Create(ctx, CreateOptions{
    Image:       "yaoapp/workspace:latest",
    WorkspaceID: "ws-abc123",
    MountMode:   MountRW,
})

  1. Validate CreateOptions (image required, etc.)
  2. If WorkspaceID is set:
     a. ws := workspaceManager.Get(ctx, workspaceID)
     b. Force Pool = ws.Node  (override any user-specified Pool)
     c. spec := taiClient.VolumeProvider().MountSpec(workspaceID)
     d. Inject mount into container create:
        - Docker: opts.Binds = ["/data/ws/ws-abc123:/workspace:rw"]
        - K8s:    opts.Volumes + opts.VolumeMounts (PVC)
  3. Create container via tai.Client.Sandbox().Create()
  4. Start container
  5. Return Box
```

### Box.Workspace() behavior change

```go
func (b *Box) Workspace() workspace.FS {
    sessionID := b.workspaceID
    if sessionID == "" {
        sessionID = b.id // backward compatible
    }
    client, _ := b.manager.getPool(b.pool)
    return client.Workspace(sessionID)
}
```

Multiple boxes mounting the same workspace -> same `sessionID` -> same files via Volume API.

---

## Metadata Storage

Workspace metadata (ID, Name, Owner, Node, Labels, timestamps) is stored as a JSON file inside the workspace directory.

### Storage path

```
/data/ws/{id}/.workspace.json
```

### Schema

```json
{
  "id": "ws-abc123",
  "name": "my-react-app",
  "owner": "user-001",
  "node": "gpu-server",
  "labels": {"project": "frontend"},
  "created_at": "2026-03-05T10:00:00Z",
  "updated_at": "2026-03-05T12:30:00Z"
}
```

### Operations

| Operation | Implementation |
|-----------|---------------|
| Create | `Volume.WriteFile(id, ".workspace.json", json)` + `Volume.ResolvePath(id)` |
| Get | `Volume.ReadFile(id, ".workspace.json")` → unmarshal |
| List | `Volume.ListDir("")` → iterate dirs → read `.workspace.json` each |
| Update | Read → merge → `Volume.WriteFile(id, ".workspace.json", json)` |
| Delete | `Volume.Cleanup(id)` (removes entire dir) |

Phase 1 strategy: simple JSON files, zero external dependencies. Can migrate to SQLite or Yao's built-in DB if query/filter performance becomes a bottleneck.

---

## Node Management

### Listing available nodes

Application layer needs to present available nodes when user creates a Workspace. This comes from the sandbox Manager's pool configuration:

```go
// In workspace.Manager or sandbox.Manager
func (m *Manager) Nodes() []NodeInfo

type NodeInfo struct {
    Name     string // pool name = node name, e.g. "gpu-server"
    Addr     string // tai:// address
    Online   bool   // is tai client connected
    // Can be extended with capacity info later
}
```

### Dynamic node configuration

Nodes are configured at the application level (Yao settings/config). When a node is added or removed, both `workspace.Manager` and `sandbox.Manager` share the updated pool map. The Pool configuration API (from `sandbox/v2`) handles this — Workspace inherits it.

```
Application Config:
    nodes:
      - name: "local"
        addr: "tai://localhost"
      - name: "gpu-server"
        addr: "tai://192.168.1.100:9527"

→ Both managers share:
    pools["local"]      = tai.Client("tai://localhost")
    pools["gpu-server"] = tai.Client("tai://192.168.1.100:9527")
```

### Node failure handling

If a Tai node goes offline:
- Workspace CRUD for that node: returns error (node unreachable)
- Container creation referencing a Workspace on that node: returns error
- Workspaces on that node are not lost — data is still on the node's disk, will be available when node comes back online
- No automatic migration (Phase 1). Can add migration (rsync between nodes) later if needed.

---

## User Flows

### Flow 1: Workspace management UI

```
1. User opens Workspace management UI
   → API: workspace.List(owner: "user-001")
   → Returns list of workspaces with metadata

2. User creates workspace
   → UI shows available nodes (from Nodes() API)
   → User selects "gpu-server"
   → API: workspace.Create({ name: "my-project", node: "gpu-server" })
   → Directory /data/ws/ws-123/ created on gpu-server
   → .workspace.json written

3. User uploads files
   → API: workspace.WriteFile("ws-123", "src/main.go", data)
   → File written to /data/ws/ws-123/src/main.go via Volume gRPC

4. User browses files
   → API: workspace.ListDir("ws-123", "src/")
   → Returns file listing

5. User deletes workspace
   → API: workspace.Delete("ws-123")
   → Checks no active mounts → removes /data/ws/ws-123/
```

### Flow 2: Chat with Workspace

```
1. User opens Chat
   → Chat UI shows workspace selector
   → User picks "my-project" (ws-123, on node "gpu-server")

2. Agent needs a container:
   → sandbox.Create({
       image: "yaoapp/workspace:latest",
       workspace_id: "ws-123",
       mount_mode: "rw",
     })
   → Manager resolves ws-123.node = "gpu-server"
   → Container created on "gpu-server" Pool
   → -v /data/ws/ws-123:/workspace:rw
   → Agent can exec "ls /workspace/src/" inside container

3. Chat ends, container destroyed
   → Workspace files persist in /data/ws/ws-123/

4. User opens new Chat, selects same workspace
   → New container, same workspace, all files still there
```

### Flow 3: Long-running Runtime + Chat

```
1. User starts Runtime container for workspace:
   → sandbox.Create({
       image: "node:20",
       workspace_id: "ws-123",
       mount_mode: "rw",
       policy: "persistent",
       ports: [{ container: 3000 }],
     })
   → Container starts on "gpu-server"
   → -v /data/ws/ws-123:/workspace:rw
   → Inside: cd /workspace && npm install && npm run dev

2. User accesses dev server via proxy
   → box.Proxy(ctx, 3000, "/")

3. User opens Chat with same workspace:
   → Second container created on "gpu-server"
   → Same workspace mounted
   → Agent modifies source → Runtime hot-reloads

4. Chat ends, chat container destroyed
   → Runtime container keeps running
   → Workspace files persist
```

---

## Process & JSAPI

### Process registration

| Process | Args | Returns |
|---------|------|---------|
| `workspace.Create` | `options` (CreateOptions JSON) | Workspace |
| `workspace.Get` | `id` | Workspace |
| `workspace.List` | `options` (ListOptions JSON) | []Workspace |
| `workspace.Update` | `id`, `options` (UpdateOptions JSON) | Workspace |
| `workspace.Delete` | `id`, `force?` | — |
| `workspace.ReadFile` | `id`, `path` | file content |
| `workspace.WriteFile` | `id`, `path`, `data` | — |
| `workspace.ListDir` | `id`, `path` | []DirEntry |
| `workspace.Remove` | `id`, `path` | — |
| `workspace.Nodes` | — | []NodeInfo |

### JSAPI

```javascript
// Workspace CRUD
var ws = Workspace.Create({ name: "my-project", node: "gpu-server" })
var ws = Workspace.Get("ws-abc123")
var list = Workspace.List({ owner: "user-001" })
Workspace.Update("ws-abc123", { name: "new-name" })
Workspace.Delete("ws-abc123")

// File operations (no container needed)
var data = Workspace.ReadFile("ws-abc123", "src/main.go")
Workspace.WriteFile("ws-abc123", "src/main.go", "package main\n...")
var entries = Workspace.ListDir("ws-abc123", "src/")
Workspace.Remove("ws-abc123", "tmp.txt")

// List available nodes
var nodes = Workspace.Nodes()
// → [{ name: "local", addr: "tai://localhost", online: true },
//    { name: "gpu-server", addr: "tai://192.168.1.100:9527", online: true }]

// Create container with workspace (via Sandbox API)
var sb = Sandbox("my-box", {
    image: "node:20",
    workspace_id: ws.id,   // → auto-routes to ws.node
    mount_mode: "rw",
})
```

---

## Storage Backend (Tai)

The `storage.VolumeProvider` interface in Tai Server already has three implementations:

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

Default is `BindMountProvider` for Docker environments (direct host path access for file CRUD). K8s environments use `K8sPVCProvider`.

The Tai `Volume` gRPC service (`ReadFile`, `WriteFile`, `ListDir`, etc.) already operates on the same `dataDir/{sessionID}/` paths. No additional work needed — Workspace file operations reuse existing Volume gRPC endpoints.

---

## Comparison: Before vs After

| Aspect | Before | After |
|--------|--------|-------|
| Workspace lifecycle | Tied to Box (same ID, same lifetime) | Independent entity, outlives containers |
| Workspace identity | `sessionID = box.id` | `sessionID = workspace.id` (explicit) |
| Container ↔ Workspace | 1:1, implicit | N:1, explicit via `CreateOptions.WorkspaceID` |
| Container scheduling | User picks Pool | Workspace determines Pool (node binding) |
| File persistence | Lost when container removed | Persists until workspace deleted |
| Multi-container access | Not possible | Multiple containers mount same workspace |
| Storage backend | Volume gRPC only (no mount) | Volume gRPC + bind mount into container |
| CRUD without container | Not possible | Via Volume API directly |
| Module status | Part of sandbox/v2 | Top-level module, parallel to sandbox/v2 |

---

## Implementation Plan

### Phase 1: Core (target: week 1-2)

| Task | Detail |
|------|--------|
| `workspace/workspace.go` | Workspace struct, MountMode, CreateOptions, metadata JSON read/write |
| `workspace/manager.go` | Manager with CRUD + file I/O (thin wrapper over tai Volume) |
| `workspace/manager_test.go` | Unit tests for CRUD and file operations |
| Node binding | `Workspace.Node` field, `Nodes()` API |
| `sandbox/v2` integration | `CreateOptions.WorkspaceID` → resolve node → force Pool → inject mount |
| `Box.Workspace()` update | Use `workspaceID` as sessionID when set |

### Phase 2: Wire into Tai (target: week 2-3)

| Task | Detail |
|------|--------|
| Tai Server: `VolumeProvider.MountSpec()` | Wire into container creation path |
| Tai gRPC: workspace metadata endpoints | Optional — can use Volume gRPC directly for Phase 1 |
| Process + JSAPI registration | `workspace.*` processes, JS bindings |

### Phase 3: Advanced (target: week 3+)

| Task | Detail |
|------|--------|
| Active mount tracking | Track which containers mount which workspaces |
| Delete safety | Refuse delete if active mounts exist |
| Workspace migration | rsync between nodes (stretch goal) |
| Quota / size limits | Per-workspace storage limits |
| Snapshot / backup | Workspace snapshots for rollback |

### Backward Compatibility

No breaking changes. Containers created without `WorkspaceID` work exactly as before:
- `sessionID = box.id`
- No bind mount
- Workspace FS backed by Volume gRPC as today
