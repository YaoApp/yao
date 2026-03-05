# Sandbox V2 — Implementation Plan

Phase 1 implementation. Covers tai SDK prerequisites + sandbox/v2 core Go API.
No JSAPI, no Process registration — those are Phase 2.

Reference: [DESIGN.md](./DESIGN.md)

## Execution Order

```
Step 0: tai/sandbox — Labels, User, ContainerInfo.Labels  (no deps)
Step 1: tai/sandbox — ExecStream                          (no deps)
Step 2: tai/proxy   — Connect                             (no deps)
Step 3: yao/grpc    — Heartbeat RPC (proto + handler)     (no deps)
Step 4: tai/grpc    — yao-grpc heartbeat goroutine        (depends on Step 3 proto)
Step 4.5: docker    — build v2 test images                (depends on Steps 1–4)
Step 5: sandbox/v2  — core module                         (depends on Steps 0–4)
Step 6: tests                                             (depends on Steps 5 + 4.5)
```

Steps 0–3 are independent and can be parallelized.
Step 4.5 (images) depends on tai SDK + yao-grpc changes being compiled into binaries.

---

## Step 0: `tai` — Labels, User, ContainerInfo.Labels + `tai.New("local")`

**Files:** `tai/tai.go`, `tai/sandbox/sandbox.go`, `tai/sandbox/docker_core.go`, `tai/sandbox/k8s.go`

### 0.0 `tai.New("")` → error, add `"local"` / `"127.0.0.1"` aliases

```go
// tai/tai.go — parseAddr changes:
// - addr == "" → return error ("use local")
// - addr == "local" || addr == "127.0.0.1" → return "docker", "", "" (platform default socket)
```

All callers must use explicit addresses. `"local"` means platform-default Docker daemon.

### 0.1 Add `Labels` and `User` to `CreateOptions`

```go
// sandbox.go — add two fields to existing struct
type CreateOptions struct {
    // ... existing fields ...
    Labels map[string]string
    User   string
}
```

### 0.2 Wire into Docker create

```go
// docker_core.go — in create(), after building cfg:
cfg.Labels = opts.Labels
if opts.User != "" {
    cfg.User = opts.User
}
```

### 0.3 Wire into K8s create

```go
// k8s.go — in Create(), set pod labels:
pod.ObjectMeta.Labels = mergeLabels(pod.ObjectMeta.Labels, opts.Labels)

// For User, parse and set SecurityContext.RunAsUser
```

### 0.4 Add `Labels` to `ContainerInfo`

```go
// sandbox.go
type ContainerInfo struct {
    // ... existing fields ...
    Labels map[string]string
}
```

### 0.5 Populate Labels in Docker list/inspect

```go
// docker_core.go — in list():
ci.Labels = c.Labels

// docker_core.go — in inspect():
ci.Labels = info.Config.Labels
```

### 0.6 Populate Labels in K8s list

```go
// k8s.go — in List():
ci.Labels = pod.Labels
```

### 0.7 Tests

- `TestCreateWithLabels` — create container with labels, list with label filter, verify match
- `TestCreateWithUser` — create container with user, exec `whoami`, verify
- `TestListLabels` — create 2 containers with different labels, list with filter, verify count

**Estimated: ~20 lines code + ~60 lines tests**

---

## Step 1: `tai/sandbox` — ExecStream

**Files:** `tai/sandbox/sandbox.go`, `tai/sandbox/docker_core.go`, `tai/sandbox/k8s.go`

### 1.1 Add to Sandbox interface

```go
// sandbox.go
type ExecStream struct {
    Stdout io.ReadCloser
    Stderr io.ReadCloser
    Stdin  io.WriteCloser
    Wait   func() (int, error)
    Cancel func()
}

// Add to Sandbox interface:
ExecStream(ctx context.Context, id string, cmd []string, opts ExecOptions) (*ExecStream, error)
```

### 1.2 Docker implementation

```go
// docker_core.go — new method
func (d *dockerCore) execStream(ctx context.Context, id string, cmd []string, opts ExecOptions) (*ExecStream, error) {
    execCfg := container.ExecOptions{
        Cmd:          cmd,
        WorkingDir:   opts.WorkDir,
        Env:          envSlice(opts.Env),
        AttachStdout: true,
        AttachStderr: true,
        AttachStdin:  true,
    }
    execResp, err := d.cli.ContainerExecCreate(ctx, id, execCfg)
    // ...
    resp, err := d.cli.ContainerExecAttach(ctx, execResp.ID, container.ExecAttachOptions{})
    // ...
    // Use io.Pipe + stdcopy.StdCopy in a goroutine to demux stdout/stderr
    // Wait: poll ContainerExecInspect until Running=false
    // Cancel: context cancel → close resp.Conn
}
```

Key: `ContainerExecAttach` returns `HijackedResponse` with multiplexed stream. Use `stdcopy.StdCopy` in a goroutine writing to `io.Pipe` pairs for stdout/stderr separation.

### 1.3 K8s implementation

```go
// k8s.go — new method
func (s *k8sSandbox) ExecStream(ctx context.Context, id string, cmd []string, opts ExecOptions) (*ExecStream, error) {
    // remotecommand.NewSPDYExecutor
    // StreamWithContext using io.Pipe for stdin/stdout/stderr
    // Wait: executor returns when process exits
    // Cancel: cancel the context
}
```

### 1.4 Tests

- `TestExecStream_ShortCommand` — `echo hello`, read stdout, verify Wait returns 0
- `TestExecStream_LongRunning` — `sleep 10`, Cancel after 1s, verify cleanup
- `TestExecStream_Stdin` — `cat`, write to stdin, read from stdout, verify echo
- `TestExecStream_ExitCode` — `exit 42`, verify Wait returns 42

**Estimated: ~80 lines code + ~100 lines tests**

---

## Step 2: `tai/proxy` — Connect

**Files:** `tai/proxy/proxy.go`, `tai/proxy/connect.go` (new)

### 2.1 Add to Proxy interface

```go
// proxy.go — extend interface
type Proxy interface {
    URL(ctx context.Context, containerID string, port int, path string) (string, error)
    Connect(ctx context.Context, containerID string, port int, opts ConnectOptions) (*Connection, error)
    Healthz(ctx context.Context) error
}

type ConnectOptions struct {
    Protocol string            // "ws", "sse", "tcp"; default "ws"
    Path     string
    Headers  map[string]string
}

type Connection struct {
    Read   func() ([]byte, error)
    Write  func(data []byte) error
    Events <-chan []byte
    URL    string
    Close  func() error
}
```

### 2.2 Implementation — `connect.go`

Local: resolve URL via `URL()`, then dial directly.
Remote: resolve URL via `URL()` (points to Tai HTTP proxy), then dial.

Both modes use the same dialing logic after URL resolution:
- **WebSocket**: `gorilla/websocket.Dialer.DialContext`
- **SSE**: `http.Get` + chunked body reader, parse `data:` lines into Events channel
- **TCP**: `net.Dial`

### 2.3 Tests

- `TestConnectWebSocket` — start a WS echo server in container, connect, send/receive
- `TestConnectSSE` — start an SSE server in container, connect, verify events arrive
- Skip TCP for now (less common use case)

**Estimated: ~120 lines code + ~80 lines tests**

---

## Step 3: `yao/grpc` — Heartbeat RPC

**Files:** `grpc/pb/yao.proto`, `grpc/sandbox/heartbeat.go` (new), `grpc/api/api.go`

### 3.1 Proto

```protobuf
// Add to service Yao:
rpc Heartbeat(HeartbeatRequest) returns (HeartbeatResponse);

message HeartbeatRequest {
  string sandbox_id = 1;
  bool   active = 2;
  int32  process_count = 3;
}
message HeartbeatResponse {}
```

Regenerate: `protoc --go_out=. --go-grpc_out=. grpc/pb/yao.proto`

### 3.2 Handler

```go
// grpc/sandbox/heartbeat.go
func (s *Server) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
    err := sandbox.M().Heartbeat(req.SandboxId, req.Active, int(req.ProcessCount))
    if err != nil {
        return nil, status.Errorf(codes.NotFound, "sandbox %s: %v", req.SandboxId, err)
    }
    return &pb.HeartbeatResponse{}, nil
}
```

### 3.3 Register in server

Wire into `grpc/api/api.go` server registration (same pattern as Healthz).

### 3.4 ACL virtual endpoint

Add to `grpc/auth/endpoints.go`:

```go
// Heartbeat → POST /grpc/heartbeat (reuse existing container token scope)
```

### 3.5 Tests

- `TestHeartbeat_Success` — create sandbox, send heartbeat, verify no error
- `TestHeartbeat_NotFound` — send heartbeat with unknown sandbox_id, verify NotFound
- `TestHeartbeat_Auth` — verify token auth works (reuse testutils)

**Estimated: ~40 lines code + ~50 lines tests**

---

## Step 4: `tai/grpc` — yao-grpc heartbeat goroutine

**Files:** `tai/grpc/cmd/main.go` (or equivalent entry point), `tai/grpc/heartbeat.go` (new)

### 4.1 Heartbeat loop

```go
// tai/grpc/heartbeat.go
func heartbeatLoop(ctx context.Context, client *grpc.Client, sandboxID string) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    for {
        select {
        case <-ticker.C:
            count := countUserProcesses()
            if count > 0 {
                client.Heartbeat(ctx, sandboxID, true, int32(count))
            }
        case <-ctx.Done():
            return
        }
    }
}

func countUserProcesses() int {
    // exec: ps -eo comm --no-headers
    // filter out: sleep, init, yao-grpc, sh, bash (if parent is sleep)
    // return count
}
```

### 4.2 Wire into main

```go
// In main() or NewFromEnv(), after client is connected:
sandboxID := os.Getenv("YAO_SANDBOX_ID")
if sandboxID != "" {
    go heartbeatLoop(ctx, client, sandboxID)
}
```

Note: `heartbeatLoop` calls `client.Heartbeat()` (new public method on `tai/grpc.Client`), not `client.svc` (private).

### 4.3 Tests

- `TestCountUserProcesses` — unit test for process filtering logic
- `TestHeartbeatLoop_SendsWhenActive` — mock gRPC client, start background process, verify heartbeat sent
- `TestHeartbeatLoop_SilentWhenIdle` — no user processes, verify no RPC calls

**Estimated: ~40 lines code + ~40 lines tests**

---

## Step 4.5: Docker — V2 Test Images

**Depends on:** Steps 1–4 (tai SDK ExecStream, proxy Connect, yao-grpc heartbeat)

Sandbox V2 tests need containers that have `yao-grpc` (with heartbeat) pre-installed. Also need a test-specific image with Nginx for Attach WS/SSE testing.

**Files:** `sandbox/docker/v2/` (new directory)

### Image hierarchy

```
sandbox-v2-base            ← base + yao-grpc + claude-proxy
sandbox-v2-test            ← v2-base + nginx (WS echo + SSE endpoint)
```

### 4.5.1 `sandbox/docker/v2/Dockerfile.base`

```dockerfile
FROM yaoapp/sandbox-base:latest

# Replace yao-bridge with yao-grpc
ARG TARGETARCH
COPY yao-grpc-${TARGETARCH} /usr/local/bin/yao-grpc
RUN chmod +x /usr/local/bin/yao-grpc

# Claude API proxy (OpenAPI-compatible)
COPY claude-proxy-${TARGETARCH} /usr/local/bin/claude-proxy
RUN chmod +x /usr/local/bin/claude-proxy

# yao-grpc auto-start: if YAO_SANDBOX_ID is set, start heartbeat + serve
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

WORKDIR /workspace
USER sandbox
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["sleep", "infinity"]
```

### 4.5.2 `sandbox/docker/v2/entrypoint.sh`

```bash
#!/bin/bash
# Start yao-grpc in background if sandbox env vars are present
if [ -n "$YAO_SANDBOX_ID" ] && [ -n "$YAO_GRPC_ADDR" ]; then
    yao-grpc serve &
fi

# Start claude-proxy if config exists
if [ -n "$CLAUDE_PROXY_BACKEND" ] || [ -f /workspace/.claude-proxy.json ]; then
    claude-proxy &
fi

exec "$@"
```

### 4.5.3 `sandbox/docker/v2/Dockerfile.test`

For unit tests — adds Nginx with a simple WS echo server and SSE endpoint.

```dockerfile
FROM yaoapp/sandbox-v2-base:latest

USER root

# Nginx + test services
RUN apt-get update && apt-get install -y nginx python3 && rm -rf /var/lib/apt/lists/*

# WS echo server (Python, ~15 lines)
COPY ws-echo.py /opt/test/ws-echo.py

# SSE endpoint (Python, ~15 lines)
COPY sse-server.py /opt/test/sse-server.py

# Nginx config — proxy WS on :3000, SSE on :3001
COPY nginx-test.conf /etc/nginx/sites-available/default

# Test entrypoint — start nginx + test services + original entrypoint
COPY test-entrypoint.sh /usr/local/bin/test-entrypoint.sh
RUN chmod +x /usr/local/bin/test-entrypoint.sh

USER sandbox
WORKDIR /workspace
ENTRYPOINT ["/usr/local/bin/test-entrypoint.sh"]
CMD ["sleep", "infinity"]
```

### 4.5.4 Test services

**`ws-echo.py`** — WebSocket echo on port 3000:

```python
#!/usr/bin/env python3
import asyncio, websockets
async def echo(ws):
    async for msg in ws:
        await ws.send(msg)
asyncio.run(websockets.serve(echo, "0.0.0.0", 3000))
```

**`sse-server.py`** — SSE endpoint on port 3001:

```python
#!/usr/bin/env python3
from http.server import HTTPServer, BaseHTTPRequestHandler
import time
class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.send_header("Content-Type", "text/event-stream")
        self.send_header("Cache-Control", "no-cache")
        self.end_headers()
        for i in range(5):
            self.wfile.write(f"data: event-{i}\n\n".encode())
            self.wfile.flush()
            time.sleep(0.1)
HTTPServer(("0.0.0.0", 3001), Handler).serve_forever()
```

**`test-entrypoint.sh`**:

```bash
#!/bin/bash
python3 /opt/test/ws-echo.py &
python3 /opt/test/sse-server.py &
exec /usr/local/bin/entrypoint.sh "$@"
```

### 4.5.5 Build script update

Add `v2` and `v2-test` targets to `sandbox/docker/build.sh`:

```bash
v2)
    echo "=== Building V2 images ==="
    # Build yao-grpc binary (replaces yao-bridge)
    cd "$SCRIPT_DIR/../../tai/grpc/cmd"
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o "$SCRIPT_DIR/v2/yao-grpc-amd64" .
    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o "$SCRIPT_DIR/v2/yao-grpc-arm64" .
    cd "$SCRIPT_DIR"

    # Build claude-proxy binary
    cd "$SCRIPT_DIR/../proxy/cmd/claude-proxy"
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o "$SCRIPT_DIR/v2/claude-proxy-amd64" .
    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o "$SCRIPT_DIR/v2/claude-proxy-arm64" .
    cd "$SCRIPT_DIR"

    build_multiarch "sandbox-v2-base" "v2/Dockerfile.base" "$PUSH"
    build_multiarch "sandbox-v2-test" "v2/Dockerfile.test" "$PUSH"
    ;;
```

### 4.5.6 Image usage

| Image | Purpose | Used by |
|-------|---------|---------|
| `sandbox-v2-base` | Production base for V2 sandboxes. Has `yao-grpc` (heartbeat) + `claude-proxy`. | `Manager.Create()` default image candidate |
| `sandbox-v2-test` | Unit tests. Has WS echo + SSE server for Attach testing. | `SANDBOX_TEST_IMAGE` in CI and local dev |

### 4.5.7 Env update

```bash
# env.local.sh — change test image to v2-test
export SANDBOX_TEST_IMAGE="yaoapp/sandbox-v2-test:latest"
```

**Estimated: ~5 files (Dockerfiles + scripts + test services), ~100 lines**

---

## Step 5: `sandbox/v2` — Core Module

**Files:** all in `sandbox/v2/`

### 5.1 `errors.go`

```go
var (
    ErrNotAvailable  = errors.New("sandbox: not available (no pools configured)")
    ErrNotFound      = errors.New("sandbox: not found")
    ErrLimitExceeded = errors.New("sandbox: limit exceeded")
    ErrPoolNotFound  = errors.New("sandbox: pool not found")
    ErrPoolInUse     = errors.New("sandbox: pool has running boxes")
)
```

### 5.2 `types.go`

All type definitions from DESIGN.md:
- `LifecyclePolicy`, `Pool`, `PoolInfo`, `PortMapping`
- `CreateOptions`, `ListOptions`
- `ExecOption`, `ExecResult`, `ExecStream`
- `AttachOption`, `ServiceConn`, `ConnectOptions`
- `BoxInfo`

### 5.3 `config.go`

```go
type Config struct {
    Pool []Pool
}
```

### 5.4 `sandbox.go` — singleton

```go
var mgr *Manager

func Init(cfg Config) error {
    m, err := newManager(cfg)
    if err != nil { return err }
    mgr = m
    return nil
}

func M() *Manager {
    if mgr == nil { panic("sandbox.Init not called") }
    return mgr
}
```

### 5.5 `manager.go`

Core implementation. Key methods:

| Method | Logic |
|--------|-------|
| `newManager(cfg)` | Parse pool defs, set default pool |
| `Start(ctx)` | For each pool: connect, list containers with `managed-by=yao-sandbox`, rebuild boxes map, start cleanupLoop |
| `Create(ctx, opts)` | Validate → check limits → resolve pool → lazy-connect tai.Client → create OAuth tokens → build tai.CreateOptions (merge env, labels, field mapping) → tai.Create → tai.Start → wrap Box → register |
| `Get(ctx, id)` | Lookup boxes map |
| `GetOrCreate(ctx, opts)` | Get by ID, if not found → Create |
| `List(ctx, opts)` | Filter boxes by owner/pool/labels |
| `Remove(ctx, id)` | Lookup box → tai.Stop → tai.Remove → revoke OAuth token → delete from map |
| `Cleanup(ctx)` | Range boxes, apply policy-based idle/lifetime rules |
| `Close()` | Cancel cleanup loop, close all tai.Clients |
| `Heartbeat(id, active, count)` | Lookup box → update lastHeartbeat + processCount atomics |
| `AddPool(ctx, p)` | Validate name unique → append to poolDefs |
| `RemovePool(ctx, name, force)` | Check no boxes (or force-remove them) → remove from poolDefs → close tai.Client if connected |
| `Pools()` | Return PoolInfo slice |

Internal helpers:
- `getPool(name)` — lazy-connect tai.Client from poolDefs
- `buildTaiCreateOptions(opts, pool)` — field mapping + env injection + label injection
- `recoverBoxes(ctx, pool, client)` — list + parse labels + rebuild Box structs

### 5.6 `box.go`

```go
type Box struct { /* fields from DESIGN.md */ }

func (b *Box) Exec(ctx, cmd, opts)     // b.touch() → tai.Sandbox().Exec(b.containerID, ...)
func (b *Box) Stream(ctx, cmd, opts)   // b.touch() → tai.Sandbox().ExecStream(b.containerID, ...)
func (b *Box) Attach(ctx, port, opts)  // b.touch() → tai.Proxy().Connect(b.containerID, port, ...)
func (b *Box) Workspace()              // lazy-init: tai.Client.Workspace(b.id)
func (b *Box) VNC(ctx)                 // b.touch() → tai.VNC().URL(b.containerID)
func (b *Box) Proxy(ctx, port, path)   // b.touch() → tai.Proxy().URL(b.containerID, port, path)
func (b *Box) Start(ctx)               // tai.Sandbox().Start(b.containerID)
func (b *Box) Stop(ctx)                // tai.Sandbox().Stop(b.containerID, 10s)
func (b *Box) Remove(ctx)              // b.manager.Remove(ctx, b.id)
func (b *Box) Info(ctx)                // tai.Sandbox().Inspect + merge with box metadata

func (b *Box) touch()                  // b.lastCall.Store(time.Now().UnixMilli())
func (b *Box) lastActiveTime() Time    // max(lastCall, lastHeartbeat)
func (b *Box) idleTimeout() Duration   // box-level override or pool default
func (b *Box) maxLifetime() Duration   // pool default
```

### 5.7 `grpc.go` — OAuth token injection

```go
func createContainerTokens(sandboxID, owner string) (access, refresh string, err error)
func revokeContainerTokens(refresh string) error
func buildGRPCEnv(pool *Pool, sandboxID, access, refresh string) map[string]string
```

Uses `openapi/oauth` to create token pairs. Local mode: `YAO_GRPC_ADDR=127.0.0.1:<port>`. Remote mode: adds `YAO_GRPC_TAI=enable` + `YAO_GRPC_UPSTREAM`.

**Estimated: ~600 lines code total**

---

## Step 6: Tests

### 6.1 Test environment

Two pools configured via env vars (reuse existing CI infrastructure):

```
# Local pool — direct Docker
SANDBOX_TEST_LOCAL_ADDR=local                           (default Docker daemon)

# Remote pool — via Tai container (same as tai-test job)
SANDBOX_TEST_REMOTE_ADDR=tai://127.0.0.1               (uses TAI_TEST_* ports)
```

Skip tests when Docker/Tai unavailable: `t.Skipf`.

### 6.2 Test files

| File | Coverage |
|------|----------|
| `sandbox_test.go` | `Init()`, `M()`, singleton behavior |
| `manager_test.go` | Create, Get, GetOrCreate, List, Remove, pool management |
| `manager_lifecycle_test.go` | Start (container discovery), Cleanup, idle tracking |
| `box_test.go` | Exec, Stream, Workspace (ReadFile/WriteFile/MkdirAll), Proxy, VNC, lifecycle |
| `box_attach_test.go` | Attach with WS/SSE (requires service in container) |
| `grpc_test.go` | Token creation/revocation, env var building |

### 6.3 Key test scenarios

| Test | What it verifies |
|------|-----------------|
| `TestCreateAndExec` | Create box → exec `echo hello` → verify stdout → remove |
| `TestCreateWithLabels` | Create → inspect labels → list with label filter |
| `TestWorkspace` | Create → WriteFile → ReadFile → verify content match |
| `TestIdleCleanup` | Create with Session + 1s idle timeout → wait → verify removed |
| `TestStartRecovery` | Create → restart Manager → Start → verify box recovered from labels |
| `TestPoolLimits` | Set MaxTotal=1 → create 1 → create 2nd → verify ErrLimitExceeded |
| `TestHeartbeatUpdates` | Create → call Heartbeat → verify lastActive updated |
| `TestStream` | Create → stream `sh -c "echo a; sleep 0.1; echo b"` → verify chunks arrive |
| `TestMultiPool` | Create on local → create on remote → verify both work |

### 6.4 CI integration

Add `sandbox-v2-test` job to `unit-test.yml` (same pattern as existing `sandbox-test` + `tai-test`):

```yaml
sandbox-v2-test:
  runs-on: ubuntu-latest
  services:
    # MongoDB (for Yao runtime)
  steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
    - name: Start Tai container
      run: |
        docker run -d --name tai \
          -v /var/run/docker.sock:/var/run/docker.sock \
          -p 2375:2375 -p 9100:9100 -p 8080:8080 \
          yaoapp/tai:latest
    - name: Build V2 test image
      run: |
        cd sandbox/docker
        bash build.sh v2
    - name: Run tests
      env:
        SANDBOX_TEST_LOCAL_ADDR: "local"
        SANDBOX_TEST_REMOTE_ADDR: "tai://127.0.0.1"
        SANDBOX_TEST_IMAGE: "yaoapp/sandbox-v2-test:latest"
        TAI_TEST_HOST: "127.0.0.1"
      run: go test -v -count=1 ./sandbox/v2/...
```

**Estimated: ~400 lines tests**

---

## Summary

| Step | Package | Lines (code) | Lines (test) | Depends on |
|------|---------|-------------|-------------|------------|
| 0 | `tai/sandbox` | ~20 | ~60 | — |
| 1 | `tai/sandbox` | ~80 | ~100 | — |
| 2 | `tai/proxy` | ~120 | ~80 | — |
| 3 | `yao/grpc` | ~40 | ~50 | — |
| 4 | `tai/grpc` | ~40 | ~40 | Step 3 |
| 4.5 | `sandbox/docker/v2` | ~100 | — | Steps 1–4 |
| 5 | `sandbox/v2` | ~600 | — | Steps 0–4 |
| 6 | `sandbox/v2` | — | ~400 | Steps 5 + 4.5 |
| **Total** | | **~1000** | **~730** | |

Steps 0–3 can start in parallel. Step 4 needs Step 3's proto. Step 5 needs all prerequisites done. Step 6 runs after Step 5.
