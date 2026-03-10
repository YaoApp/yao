# Sandbox V2 — Implementation Status

Reference: [DESIGN.md](./DESIGN.md)

---

## Phase 1: Core Module — DONE

### tai SDK Prerequisites — DONE

| Step | Package | What | Status |
|------|---------|------|--------|
| 0 | `tai/sandbox` | Labels, User in CreateOptions + ContainerInfo | DONE |
| 1 | `tai/sandbox` | ExecStream (Docker + K8s) | DONE |
| 2 | `tai/proxy` | Connect (WS/SSE, Local + Remote) | DONE |
| 3 | `tai/sandbox` | Image interface (Exists, Pull, Remove, List) | DONE |
| 4 | `tai/tai.go` | Client: Sandbox(), Image(), Proxy(), VNC(), Volume(), Workspace() | DONE |
| 5 | `yao/grpc` | Heartbeat RPC (proto + handler) | DONE |

### sandbox/v2 Core — DONE

| File | What | Status |
|------|------|--------|
| `sandbox.go` | `Init()`, `M()`, global singleton | DONE |
| `manager.go` | Manager: Create/Get/GetOrCreate/List/Remove/Cleanup/Close, Start (container recovery), Nodes, Heartbeat, ImageExists/PullImage/EnsureImage | DONE |
| `box.go` | Box: Exec, Stream, Attach, Workspace, VNC, Proxy, Start/Stop/Remove, Info, touch/lastActiveTime/idleTimeout/maxLifetime/stopTimeout | DONE |
| `types.go` | LifecyclePolicy (OneShot/Session/LongRunning/Persistent), NodeID, PortMapping, CreateOptions (with WorkspaceID/MountMode/MountPath), ListOptions, ExecOption/ExecResult/ExecStream, AttachOption/ServiceConn, ImagePullOptions/RegistryAuth, BoxInfo, DefaultStopTimeout | DONE |
| `errors.go` | ErrNotAvailable, ErrNotFound, ErrNodeNotFound, ErrNodeMissing | DONE |
| `grpc.go` | BuildGRPCEnv (sandbox ID + gRPC addr only; token injection is caller's responsibility via Env) | DONE |

### workspace Module — DONE

| File | What | Status |
|------|------|--------|
| `workspace.go` | Workspace struct, CreateOptions, ListOptions, UpdateOptions, NodeInfo, MountMode, metadata marshal/unmarshal | DONE |
| `manager.go` | Manager: Create/Get/List/Update/Delete, ReadFile/WriteFile/ListDir/Remove/FS, Nodes/AddPool/RemovePool, NodeForWorkspace/MountPath | DONE |
| `errors.go` | ErrNotFound, ErrNodeMissing, ErrNodeOffline, ErrHasMounts | DONE |

### Tests — DONE

| File | Coverage | Status |
|------|----------|--------|
| **sandbox/v2** | | |
| `sandbox_test.go` | Init, M, singleton | DONE |
| `manager_test.go` | Create, Get, GetOrCreate, List, Remove, pool limits (MaxTotal, MaxPerUser), multi-pool | DONE |
| `manager_lifecycle_test.go` | Start (recovery), Cleanup, idle tracking, Heartbeat | DONE |
| `box_test.go` | Exec, Info, Workspace (ReadFile/WriteFile), status | DONE |
| `box_attach_test.go` | Attach WS echo, Attach SSE events, VNC URL, VNC Connect (RFB handshake) | DONE |
| `box_workspace_test.go` | Workspace mount, file I/O through Box, invalid ID | DONE |
| `box_image_test.go` | ImageExists (Docker+K8s), PullImage (progress+K8s no-op), EnsureImage, bad ref | DONE |
| `grpc_test.go` | Token creation/revocation, env var building (local vs remote) | DONE |
| `bench_test.go` | ContainerLifecycle, Create, Exec, ExecHeavy, Remove, Info, StopStart, WorkspaceReadWrite | DONE |
| `testutils_test.go` | testNodes (local/remote/k8s), setupManager, setupManagerForNode, createTestBox, ensureTestImage | DONE |
| `export_test.go` | ResetForTest | DONE |
| **workspace** | | |
| `workspace_test.go` | Create (auto/explicit ID, labels, invalid node), Get, List (owner/node filter), Update (name/labels), Delete, Nodes, NodeForWorkspace, AddPool/RemovePool, MountPath | DONE |
| `fileio_test.go` | ReadWriteFile, nested paths, ListDir, Remove, fs.FS (ReadFile, WriteFile, MkdirAll, Rename, WalkDir, Remove, NotFound) | DONE |
| `bench_test.go` | WriteFile, ReadFile, ReadWriteCycle, WriteLargeFile, ListDir, FSWalkDir, CreateDelete | DONE |
| `testutils_test.go` | testPools, setupManagerForPool, clientForPool, localClient, setupManagerMultiNode, createWorkspace | DONE |

### CI — DONE

| Job | Contents | Status |
|-----|----------|--------|
| `SandboxV2Test` | Consolidated: image pre-pull → tai-test → sandbox/v2 (local+remote+k8s) → workspace (local+remote) | DONE |
| `BenchmarkSandboxV2` | Parallel: performance tests for sandbox/v2 + workspace | DONE |
| `GRPCTest` | Independent: gRPC tests (unchanged) | DONE |

### Performance Optimizations — DONE

| Optimization | Before | After | Impact |
|-------------|--------|-------|--------|
| Remove redundant Stop in Manager.Remove() | 2.14s | 177ms | 12x faster Docker remove |
| Container CMD trap SIGTERM | 2s+ stop | near-instant | Graceful shutdown on Stop |
| K8s Start: respect ctx deadline | 30s hardcoded | ctx-aware + 60s default | Proper timeout propagation |
| K8s Pod spec: Args vs Command | CMD overridden | ENTRYPOINT preserved | Correct container behavior |

---

## Phase 2: JSAPI + Computer Unification — DONE

### Unified Computer Interface — DONE

Box and Host now share a single `Computer` interface (`types.go`). Both `sandbox.Create()` and `sandbox.Host()` return the same JS `Computer` object; `kind` property distinguishes them. Box-only methods (`Info`, `Start`, `Stop`, `Remove`) throw at runtime when called on a host.

| Step | Package | What | Status |
|------|---------|------|--------|
| Computer interface | `sandbox/v2/types.go` | `Computer` interface: Exec, Stream, VNC, Proxy, ComputerInfo, BindWorkplace, Workplace | DONE |
| Host implementation | `sandbox/v2/host.go` | `Host` struct implements `Computer` via tai HostExec + VNC/Proxy | DONE |
| ComputerInfo | `sandbox/v2/types.go` | `ComputerInfo` struct with Kind, NodeID, TaiID, System, Capabilities, box-specific fields | DONE |

### JSAPI — DONE

| File | What | Status |
|------|------|--------|
| `jsapi/jsapi.go` | Static methods: `sandbox.Create`, `Get`, `List`, `Delete` | DONE |
| `jsapi/computer.go` | `NewComputerObject` factory (11 methods + 4 properties), `sbHost`, helpers | DONE |
| `jsapi/node.go` | `sandbox.GetNode`, `Nodes`, `NodesByTeam`, `snapshotToJS` | DONE |
| `jsapi/API.md` | Full JavaScript API reference | DONE |

Design decisions:
- **No Go objects in V8**: closures capture only `kind` (string) and `identifier` (string); `getComputer()` re-fetches from Manager on each call — prevents memory leaks across runtimes.
- **Stream**: blocking with callback `function(type, data)`, goroutines feed a channel, main V8 thread drains it.
- **Workplace()**: delegates to `workspace/jsapi.NewFSObject()` — reuses existing WorkspaceFS JSAPI.

```javascript
// Unified Computer — same API for box and host
const pc = sandbox.Create({ image: "node:20", owner: "user-123" })
pc.Exec(["node", "-e", "console.log('hello')"])
pc.Stream(["npm", "run", "dev"], function(type, data) {
    if (type === "stdout") console.log(data)
    if (type === "exit") console.log("exited:", data)
})
pc.VNC()                          // → "ws://host:port/vnc/{id}/ws"
pc.Proxy(3000, "/api")            // → "http://host:port/{id}:3000/api"
pc.ComputerInfo()                 // → { kind, pool, system, ... }
pc.BindWorkplace("ws-abc")
pc.Workplace().ReadFile("main.go")
pc.Info()                         // box-only
pc.Remove()                       // box-only

// Host — same interface, no container
const host = sandbox.Host("gpu")
host.Exec(["nvidia-smi"])
host.VNC()                        // → "ws://host:port/vnc/__host__/ws"
host.Proxy(8080)                  // → "http://host:port/__host__:8080/"
host.kind                         // "host"
host.Info()                       // throws: "not supported: Info() requires a box computer"

// Nodes (registry read-only query)
const nodes = sandbox.Nodes()
const node  = sandbox.GetNode("tai-abc123")
const team  = sandbox.NodesByTeam("team-001")
```

### JSAPI Tests — DONE

| Test | Coverage | Status |
|------|----------|--------|
| `TestCreate` | Create box, verify kind/id | DONE |
| `TestGet` | Get existing box | DONE |
| `TestGetNotFound` | Get non-existent → null | DONE |
| `TestDelete` | Delete + verify gone | DONE |
| `TestList` | List with owner filter | DONE |
| `TestExec` | Exec echo, verify stdout | DONE |
| `TestExecWithOptions` | Exec with workdir option | DONE |
| `TestStream` | Stream with callback, verify chunks + exit code | DONE |
| `TestComputerInfo` | Verify kind field | DONE |
| `TestBoxInfo` | Box-only Info() | DONE |
| `TestHostBoxMethodsThrow` | Host.Info() throws "not supported" | DONE |
| `TestComputerKind` | kind property = "box" | DONE |
| `TestNodes` | Nodes() returns array | DONE |
| `TestGetNodeNotFound` | GetNode non-existent → null | DONE |

All 14 tests pass in both local and remote modes.

### OAuth Decoupling — DONE

Token injection (YAO_TOKEN, YAO_REFRESH_TOKEN) has been **removed from sandbox Manager**.
`CreateContainerTokens`, `RevokeContainerTokens`, and the `Box.refreshToken` field have been deleted.
`BuildGRPCEnv` now only sets `YAO_SANDBOX_ID` and `YAO_GRPC_ADDR`.

Token provisioning is the **caller's responsibility** via `CreateOptions.Env`:
- The caller (e.g. Agent Hook) already holds an OAuth context
- It calls `oauth.OAuth.MakeAccessToken(...)` to issue a scoped token
- Passes it in `CreateOptions.Env["YAO_TOKEN"]` / `Env["YAO_REFRESH_TOKEN"]`
- `opts.Env` takes priority over `BuildGRPCEnv` output (caller can override anything)

### Remaining (Startup)

| Task | Package | Status | Detail |
|------|---------|--------|--------|
| `engine/load.go` integration | `yao` | **DONE** | `sandbox.Init()` + `sandbox.M().Start(ctx)` added as a `loadStep("Sandbox", ...)` right after Registry init |
| Heartbeat bridge | `yao/cmd` | **DONE** | `cmd/start.go` calls `yaogrpc.SetSandboxOnBeat(...)` before `service.Start`, forwarding gRPC heartbeats to `sandbox.M().Heartbeat()` |

---

## Phase 3: Agent Integration — PENDING

| Task | Detail |
|------|--------|
| Agent creates Box via `sandbox.M().GetOrCreate()` | Replace `infraSandbox.Manager` |
| Agent uses `Box.Workspace()` for file I/O | Replace Docker Copy/bind mount reads |
| Agent uses `Box.Exec()` for commands | Replace Docker exec |
| Agent uses `Box.VNC()` / `Box.Proxy()` | Replace vncproxy |
| Agent injects `Box` as `SandboxExecutor` | `ctx.sandbox` JSAPI unchanged for hooks |

---

## Phase 4: Cutover — PENDING

| Task | Detail |
|------|--------|
| Move `sandbox/v2` → `sandbox` | Rename package |
| Delete old sandbox code | manager.go, ipc/, bridge/, vncproxy/, docker/ |
| Delete `DESIGN-REMOTE.md` | Superseded by tai.Client |
| Update `cmd/start.go` | Use new init path |
| `sandbox/process.go` | Register `sandbox.*` process namespace (post-cutover) |
| `workspace/process.go` | Register `workspace.*` process namespace (post-cutover) |

---

## Implementation Details

### Container CMD

All V2 containers use a SIGTERM-aware sleep as PID 1:

```bash
sh -c "trap 'exit 0' TERM; while :; do sleep 86400 & wait $!; done"
```

This ensures:
- Container stays alive indefinitely (no hardcoded `sleep infinity`)
- Exits immediately on SIGTERM (no 2s wait)
- Works on both Docker and K8s

### Container Labels

Manager injects these labels at creation time:

```
managed-by=yao-sandbox
sandbox-id=<id>
sandbox-owner=<owner>
sandbox-node-id=<nodeID>
sandbox-policy=<policy>
workspace-id=<workspace-id>    (if WorkspaceID set)
```

Used by `Manager.Start()` to discover and recover existing containers after restart.

### Workspace Bind Mount

When `CreateOptions.WorkspaceID` is set:

```
1. NodeForWorkspace(wsID) → node name
2. Force nodeID = node name
3. MountPath(wsID) → hostDir
4. Bind: hostDir:/workspace:rw
```

### Multi-Mode Testing

`testNodes()` returns all available node configurations:

```go
func testPools() []poolConfig {
    pools := []poolConfig{{Name: "local", Addr: testLocalAddr()}}
    if addr := os.Getenv("SANDBOX_TEST_REMOTE_ADDR"); addr != "" {
        pools = append(pools, poolConfig{Name: "remote", Addr: addr})
    }
    if host := os.Getenv("TAI_TEST_K8S_HOST"); host != "" {
        // ... K8s pool with kubeconfig, namespace, ports
        pools = append(pools, poolConfig{Name: "k8s", ...})
    }
    return pools
}
```

Every test iterates over all available pools:

```go
func TestSomething(t *testing.T) {
    for _, pc := range testPools() {
        pc := pc
        t.Run(pc.Name, func(t *testing.T) {
            m := setupManagerForPool(t, &pc)
            // test logic — use pc.TaiID as pool identifier
        })
    }
}
```

### Benchmark Helpers

```go
func setupManagerForBench(b *testing.B, pc poolConfig) *sandbox.Manager
func ensureTestImageBench(b *testing.B, m *sandbox.Manager, pool string)
func createBoxForBench(b *testing.B, m *sandbox.Manager) *sandbox.Box
```

K8s-specific behavior:
- `BenchmarkStopStart`: skipped (K8s Stop deletes Pod)
- Create/Lifecycle benchmarks: 120s timeout for K8s Pod scheduling

---

## File Inventory

### sandbox/v2 (9 source + 10 test = 19 files)

| File | Lines | Purpose |
|------|-------|---------|
| `sandbox.go` | ~25 | Global singleton |
| `manager.go` | ~620 | Manager implementation |
| `box.go` | ~317 | Box implementation (Computer interface) |
| `host.go` | ~232 | Host implementation (Computer interface) |
| `types.go` | ~247 | Type definitions (Computer, ComputerInfo, ExecOption, etc.) |
| `config.go` | ~5 | Config struct |
| `errors.go` | ~10 | Error definitions |
| `grpc.go` | ~50 | BuildGRPCEnv (sandbox ID + addr) |
| `export_test.go` | ~6 | ResetForTest |
| `testutils_test.go` | ~364 | Test helpers (multi-pool, host exec targets) |
| `sandbox_test.go` | ~30 | Singleton tests |
| `manager_test.go` | ~250 | CRUD tests |
| `manager_lifecycle_test.go` | ~120 | Lifecycle tests |
| `box_test.go` | ~200 | Box tests |
| `box_attach_test.go` | ~260 | Attach/VNC tests |
| `box_workspace_test.go` | ~285 | Workspace tests |
| `box_image_test.go` | ~120 | Image tests |
| `grpc_test.go` | ~40 | BuildGRPCEnv tests |
| `bench_test.go` | ~230 | Benchmarks |

### sandbox/v2/jsapi (3 source + 1 test + 1 doc = 5 files)

| File | Lines | Purpose |
|------|-------|---------|
| `jsapi.go` | ~286 | Static methods (Create/Get/List/Delete) + V8 registration |
| `computer.go` | ~472 | NewComputerObject factory, sbHost, helpers |
| `node.go` | ~143 | Node query methods (GetNode/Nodes/NodesByTeam) + snapshotToJS |
| `jsapi_test.go` | ~430 | 14 test cases (local + remote modes) |
| `API.md` | ~604 | JavaScript API reference |

### workspace (3 source + 4 test = 7 files)

| File | Lines | Purpose |
|------|-------|---------|
| `workspace.go` | ~80 | Types + metadata |
| `manager.go` | ~320 | Manager implementation |
| `errors.go` | ~10 | Error definitions |
| `testutils_test.go` | ~90 | Test helpers |
| `workspace_test.go` | ~325 | CRUD tests |
| `fileio_test.go` | ~235 | File I/O tests |
| `bench_test.go` | ~150 | Benchmarks |

### workspace/jsapi (2 source + 1 test + 1 doc = 4 files)

| File | Lines | Purpose |
|------|-------|---------|
| `jsapi.go` | ~100 | Static methods (Create/Get/List/Delete) + V8 registration |
| `fs.go` | ~630 | NewFSObject factory (WorkspaceFS methods) |
| `jsapi_test.go` | ~460 | JSAPI tests (local + remote modes) |
| `API.md` | ~220 | Workspace JavaScript API reference |
