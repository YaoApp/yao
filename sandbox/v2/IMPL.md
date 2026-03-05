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
| `manager.go` | Manager: Create/Get/GetOrCreate/List/Remove/Cleanup/Close, Start (container recovery), AddPool/RemovePool/Pools, Heartbeat, SetGRPCPort, SetWorkspaceManager, ImageExists/PullImage/EnsureImage | DONE |
| `box.go` | Box: Exec, Stream, Attach, Workspace, VNC, Proxy, Start/Stop/Remove, Info, touch/lastActiveTime/idleTimeout/maxLifetime/stopTimeout | DONE |
| `types.go` | LifecyclePolicy (OneShot/Session/LongRunning/Persistent), Pool, PoolInfo, PortMapping, CreateOptions (with WorkspaceID/MountMode/MountPath), ListOptions, ExecOption/ExecResult/ExecStream, AttachOption/ServiceConn, ImagePullOptions/RegistryAuth, BoxInfo, DefaultStopTimeout | DONE |
| `config.go` | Config struct | DONE |
| `errors.go` | ErrNotAvailable, ErrNotFound, ErrLimitExceeded, ErrPoolNotFound, ErrPoolInUse | DONE |
| `grpc.go` | CreateContainerTokens, RevokeContainerTokens, BuildGRPCEnv | DONE |

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
| `testutils_test.go` | testPools (local/remote/k8s), setupManager, createTestBox, ensureTestImage | DONE |
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

## Phase 2: Process + JSAPI — PENDING

| Task | Package | Detail |
|------|---------|--------|
| `process.go` | `sandbox/v2` | Register `sandbox.*` process namespace (sandbox.Create, sandbox.Exec, sandbox.ReadFile, etc.) |
| `process.go` | `workspace` | Register `workspace.*` process namespace |
| `jsapi/sandbox.go` | `sandbox/v2/jsapi` | V8 JSAPI `Sandbox()` constructor (registered in gou runtime) |
| `cmd/start.go` integration | `yao` | Call `sandbox.Init(config.Conf.Sandbox)` + `sandbox.M().Start(ctx)` in startup sequence |
| Heartbeat bridge | `yao/grpc` | Wire gRPC Heartbeat handler → `sandbox.M().Heartbeat()` |
| Wire `openapi/oauth` | `sandbox/v2/grpc.go` | `CreateContainerTokens` currently generates random strings; `RevokeContainerTokens` is a no-op. Replace with real `openapi/oauth` issue/revoke calls |

### Process Registration (planned)

```
sandbox.pool.Add        sandbox.pool.Remove     sandbox.pool.List
sandbox.Create          sandbox.Get             sandbox.GetOrCreate
sandbox.Remove          sandbox.List
sandbox.Start           sandbox.Stop
sandbox.Exec            sandbox.Stream          sandbox.Attach
sandbox.ReadFile        sandbox.WriteFile       sandbox.ListDir
sandbox.RemoveFile      sandbox.MkdirAll
sandbox.VNC             sandbox.Proxy
sandbox.EnsureImage     sandbox.ImageExists     sandbox.PullImage

workspace.Create        workspace.Get           workspace.List
workspace.Update        workspace.Delete
workspace.ReadFile      workspace.WriteFile     workspace.ListDir
workspace.Remove        workspace.FS
workspace.Nodes
```

### JSAPI (planned)

```javascript
// Sandbox
var sb = Sandbox("my-workspace", {
    image: "yaoapp/workspace:latest",
    owner: "user-123"
})
sb.Exec(["go", "build", "./..."])
sb.ReadFile("src/main.go")
sb.WriteFile("src/main.go", "package main\n...")
sb.Stream(["npm", "run", "dev"], function(chunk) { ... })
var conn = sb.Attach(3000, { protocol: "ws", path: "/ws" })
sb.Info()
sb.Stop()
sb.Start()
sb.Remove()

// Workspace
var ws = Workspace("my-workspace")
ws.ReadFile("src/main.go")
ws.WriteFile("src/main.go", "package main\n...")
ws.ListDir("src/")
ws.Remove("tmp.txt")
```

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
sandbox-pool=<pool>
sandbox-policy=<policy>
workspace-id=<workspace-id>    (if WorkspaceID set)
```

Used by `Manager.Start()` to discover and recover existing containers after restart.

### Workspace Bind Mount

When `CreateOptions.WorkspaceID` is set:

```
1. NodeForWorkspace(wsID) → node name
2. Force pool = node name
3. MountPath(wsID) → hostDir
4. Bind: hostDir:/workspace:rw
```

### Multi-Mode Testing

`testPools()` returns all available pool configurations:

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
        t.Run(pc.Name, func(t *testing.T) {
            m := setupManagerForPool(t, pc)
            // test logic
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

### sandbox/v2 (7 source + 10 test = 17 files)

| File | Lines | Purpose |
|------|-------|---------|
| `sandbox.go` | ~25 | Global singleton |
| `manager.go` | ~620 | Manager implementation |
| `box.go` | ~230 | Box implementation |
| `types.go` | ~170 | Type definitions |
| `config.go` | ~5 | Config struct |
| `errors.go` | ~10 | Error definitions |
| `grpc.go` | ~55 | Token/env injection |
| `testutils_test.go` | ~130 | Test helpers |
| `sandbox_test.go` | ~30 | Singleton tests |
| `manager_test.go` | ~250 | CRUD tests |
| `manager_lifecycle_test.go` | ~120 | Lifecycle tests |
| `box_test.go` | ~200 | Box tests |
| `box_attach_test.go` | ~260 | Attach/VNC tests |
| `box_workspace_test.go` | ~285 | Workspace tests |
| `box_image_test.go` | ~120 | Image tests |
| `grpc_test.go` | ~80 | Token tests |
| `bench_test.go` | ~230 | Benchmarks |

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
