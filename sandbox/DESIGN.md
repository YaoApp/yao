# Sandbox Refactoring Design

## Background

The current `sandbox.Manager` was built as a quick prototype for the Claude coding agent. It directly depends on the local Docker client, uses bind mounts for file IO, and Unix sockets for IPC. This limits it to single-node, local-only operation.

This document outlines the refactoring plan to make sandbox a production-grade, multi-node capable system built on top of the Tai SDK (`yao/tai`).

## Architecture

```
tai.Client                   Single connection to a Tai endpoint (or local Docker)
    │                        sandbox / volume / proxy / vnc low-level APIs
    │
sandbox.Manager              Business layer
                             Lifecycle, user isolation, TTL, cleanup, IPC
```

`sandbox.Manager` takes a single `tai.Client` at construction time. Scaling is handled externally by the container runtime — K8s scheduler for pod placement and node scaling, Docker for single-host. The SDK does not manage multiple endpoints or do any scheduling.

### Why Tai stays in the K8s path

K8s handles pod scheduling and container lifecycle, but it does **not** provide:

| Capability | K8s native? | What you'd need without Tai |
|-----------|-------------|---------------------------|
| File sync to/from container | No | PVC + init container or sidecar |
| HTTP preview proxy | No | Ingress + Service per sandbox |
| VNC access | No | VNC sidecar + Service + Ingress |
| gRPC IPC relay (container → Yao) | No | Pod must reach Yao directly (network policy, Service) |

Tai bundles all four behind a single endpoint. Bypassing Tai to "direct-connect" K8s only covers pod CRUD and exec — you'd still need to solve file IO, preview, VNC, and IPC separately, which means either deploying Tai anyway or assembling equivalent infrastructure from K8s primitives.

The SDK's `NewK8s()` already supports direct kube-apiserver connection (pass empty `addr`), but this is only useful for bare compute scenarios with no file sync or web preview requirements.

### sandbox.Manager (yao/sandbox)

High-level business layer on top of `tai.Client`. Manages container lifecycle, user/session isolation, file operations, and IPC.

**Responsibilities:**
- Create / get / start / stop / remove sandboxes
- Lifecycle policies: one-shot, session-bound, long-running, persistent
- Per-user and global container limits
- Idle timeout and cleanup
- File operations (via `tai.Client.Volume()` for remote, bind mount for local)
- IPC relay to Yao gRPC server

### Yao gRPC Server (yao/grpc) — ✅ Implemented

General-purpose gRPC gateway exposed by the Yao process. Not limited to sandbox IPC — it exposes process execution, shell, API proxy, MCP, LLM, and Agent capabilities to any gRPC client. 14 RPCs defined; V1 (unary + LLM/Agent streaming) complete, V2 (base streaming via `gou/stream`) pending.

**Clients:**
- Container-internal `tai call` (via Tai Gateway relay or direct)
- `yao run` CLI (after `yao login`)
- Other Yao instances (future node-to-node)

**IPC path (replacing Unix socket):**
```
Local:   Container → tai call (tai repo) → Yao gRPC 127.0.0.1:9099
Remote:  Container → tai call (tai repo) → Tai Gateway (:9100 gRPC) → Yao gRPC Server (:9099)
```

All modes use gRPC — no Unix socket fallback. `yao-grpc` reads `YAO_GRPC_ADDR` from env and connects. Local containers point directly at the Yao gRPC server on loopback; remote containers point at the Tai relay. Tai does **not** know Yao gRPC address at startup — `yao-grpc` carries target in `x-grpc-upstream` request metadata. This keeps Tai stateless and allows one Tai to serve multiple Yao instances.

## Authentication

The gRPC server reuses the existing `openapi/oauth` service — no new auth system needed.

### What already exists

| Capability | Module | Reuse | Needs changes |
|------------|--------|-------|---------------|
| JWT sign (RS256) | `oauth.MakeAccessToken()` | Issue tokens for gRPC clients | None — supports custom scope/subject/extraClaims |
| JWT verify | `oauth.VerifyToken(token string)` | Validate Bearer token in interceptor | None — pure string input, no Gin dependency |
| Signing certs | `oauth.SigningCertificates` | Same keypair for HTTP and gRPC | None |
| Identity | `TokenClaims` (Subject/ClientID/Scope) | gRPC request context | None |
| Scope/ACL | `acl.Scope.Check(*AccessRequest)` | Method-level access control | None — only needs `(Method, Path, Scopes)`, no Gin dependency |
| Scope registration | `acl.Register(...)` | gRPC scopes via same pattern | None — add `grpc:*` scope definitions in `init()` |
| Client auth | `ClientProvider` | `client_credentials` grant for CLI/containers | None |
| Token revocation | `oauth.Revoke(ctx, token, hint)` | Container token cleanup | None |
| Device Flow | `DeviceAuthorization()`, `AuthorizeDevice()`, device_code store, `GrantTypeDeviceCode` grant | CLI `yao login` | ✅ Implemented |

**Key insight**: `authorized.SetInfo/GetInfo` are Gin-bound, but gRPC does NOT need them. The gRPC interceptor builds `AccessRequest` directly from JWT claims and calls `ScopeManager.Check` — bypasses the full `Enforce` chain (client/team/member), which is HTTP multi-tenant only.

**Status**: All auth infrastructure is implemented and working — gRPC interceptor, scope registration, Device Flow (backend + CUI page), CLI commands (`yao login`/`yao logout`).

### gRPC interceptor

```go
func authInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
    md, _ := metadata.FromIncomingContext(ctx)
    token := extractBearer(md)
    claims, err := oauth.OAuth.VerifyToken(token)
    if err != nil {
        return nil, status.Errorf(codes.Unauthenticated, "invalid token")
    }
    ctx = withClaims(ctx, claims)
    return handler(ctx, req)
}
```

`oauth.OAuth` is a global singleton initialized at Yao startup. The gRPC server simply references it — same signing keys, same token format, same user/client model.

### Token flow by client type

| Client | How it gets a token |
|--------|-------------------|
| Container MCP tool | `oauth.MakeAccessToken(clientID, "grpc:mcp grpc:run", userID, 900)` injected as `YAO_TOKEN` env var. `yao-grpc` auto-refreshes via response metadata. Manager revokes refresh token on container Remove. |
| `yao run` CLI | ✅ `yao login --server <url>` → OAuth Device Authorization Grant (RFC 8628) → dynamic client registration via machine ID → token saved to `~/.yao/credentials` (base64 JSON). Logged in = gRPC, not logged in = local. |
| Yao-to-Yao | Pre-shared service token or `client_credentials` |

## Network Security

### Yao ↔ Tai communication

Tai exposes Docker Engine API (:2375), K8s API (:6443), gRPC Volume (:9100), HTTP proxy (:8080), and VNC (:6080). These are raw protocol proxies with **no built-in auth** — security is handled at the network layer.

| Deployment | Strategy |
|-----------|----------|
| Same host (local) | Bind to `127.0.0.1` or Unix socket, no exposure |
| Same VPC / LAN | Firewall rules / security groups, private subnet only |
| Cross-network | VPN / WireGuard tunnel, or mTLS termination at Tai |

### gRPC Server listen policy

The Yao gRPC server (:9099) supports configurable listen address:

| Scenario | Listen | Why |
|----------|--------|-----|
| Local dev | `127.0.0.1:9099` | Only local containers reach it |
| Production (same host) | `127.0.0.1:9099` | Tai on same machine forwards via loopback |
| Production (multi-node) | `0.0.0.0:9099` + IP allowlist | Remote Tai nodes need access |

### IP allowlist (gRPC server)

For multi-node deployment where gRPC must listen on `0.0.0.0`, the server should support an IP/CIDR allowlist:

```
YAO_GRPC_LISTEN=0.0.0.0:9099
YAO_GRPC_ALLOW=10.0.0.0/8,172.16.0.0/12,192.168.0.0/16
```

Enforcement is a simple gRPC interceptor that runs **before** the auth interceptor:

```go
func ipAllowInterceptor(allowedCIDRs []*net.IPNet) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
        peer, _ := peer.FromContext(ctx)
        if !isAllowed(peer.Addr, allowedCIDRs) {
            return nil, status.Errorf(codes.PermissionDenied, "ip not allowed")
        }
        return handler(ctx, req)
    }
}
```

Defense in depth: IP allowlist is the first gate, OAuth token is the second. Both must pass.

### Tai side security

Tai itself does not need auth — it trusts its network boundary. Recommended:
- Docker: Tai container runs on a private network, ports not exposed to public
- K8s: Tai runs as a DaemonSet or Deployment, service only accessible within cluster
- If Tai must be exposed, put it behind a reverse proxy (nginx/envoy) with mTLS or VPN

### Tai high availability

Tai is a single endpoint, but all its services except VNC WebSocket are stateless. Avoiding single-point-of-failure is a deployment concern, not an SDK concern.

| Deployment | HA strategy |
|-----------|-------------|
| Docker single-host | Docker restart policy (`--restart=always`), Tai failure = transient |
| K8s Deployment | `replicas: N` + K8s Service load balancing, liveness probe on `/healthz` |
| K8s DaemonSet | One Tai per node, pod talks to local Tai via node-local Service |

VNC uses WebSocket long connections — if Tai restarts, active VNC sessions drop and the client reconnects. Stateless services (K8s proxy, Docker proxy, Volume gRPC, HTTP proxy) recover transparently behind a Service.

The SDK `tai.Client` connects to a single address. In K8s this address is a Service VIP — Tai replicas behind it are invisible to the SDK.

## Container Lifecycle

Lifecycle is managed by `sandbox.Manager`, not by tai.Client.

| Policy | TTL | Behavior |
|--------|-----|----------|
| One-shot | 0 | Destroyed immediately after execution |
| Session | Minutes | Alive while user is active, cleaned up on idle timeout |
| Long-running | Hours/Days | User workspace, recoverable, cleaned up after extended idle |
| Persistent | None | User-managed, never auto-cleaned |

## File Operations

| Mode | tai.Client | File IO |
|------|-----------|---------|
| Local | `tai.New("local")` | Bind mount, direct host filesystem |
| Remote | `tai.New("tai://host")` | `tai.Client.Volume()` via gRPC |

Local mode preserves bind mount for performance. Remote mode uses `tai/volume` (gRPC + lz4 compression). `sandbox.Manager` routes based on `client.IsLocal()`.

## Agent Layer Adaptation

The agent layer (`agent/assistant`, `agent/sandbox`, `agent/context`) currently hardcodes local-only assumptions. It needs to be adapted to work with the new `sandbox.Manager` backed by `tai.Client`.

### Current coupling

```
agent/assistant/sandbox.go
    │
    ├─ GetSandboxManager()          Global singleton, local Docker only
    ├─ initSandbox()                Creates executor, calls manager.GetOrCreate()
    ├─ BuildMCPConfigForSandbox()   Hardcodes /tmp/yao.sock for yao-bridge
    └─ loadMCPToolsForIPC()         Loads MCP tools, injects into IPC session

agent/sandbox/claude/executor.go
    │
    ├─ manager.GetOrCreate()        Direct Docker container creation
    ├─ manager.Stream()             Docker exec + attach
    └─ manager.Remove()             Docker container removal

agent/context/jsapi_sandbox.go
    │
    ├─ ReadFile()                   Host filesystem via bind mount path translation
    ├─ WriteFile()                  Docker CopyToContainer
    └─ Exec()                       Docker exec
```

### What changes

| Component | Before | After |
|-----------|--------|-------|
| `GetSandboxManager()` | Global singleton, `docker.NewClientWithOpts(FromEnv)` | Initialized with a `tai.Client` from Yao config |
| Container creation | `dockerClient.ContainerCreate()` | `tai.Client.Sandbox().Create()` |
| Container exec | `dockerClient.ContainerExecCreate/Start/Attach` | `tai.Client.Sandbox().Exec()` |
| File read | Host path via bind mount (`containerPathToHost`) | Local: bind mount (same). Remote: `tai.Client.Volume().Read()` |
| File write | `dockerClient.CopyToContainer` | Local: bind mount. Remote: `tai.Client.Volume().Write()` |
| IPC | Unix socket bind mount + yao-bridge | All modes: `tai call` → gRPC (direct or via Tai gateway). No Unix socket. |
| MCP config | `{args: ["/tmp/yao.sock"]}` hardcoded | `YAO_GRPC_ADDR` + `YAO_TOKEN` env vars. Local: direct. Remote: via Tai gateway. |
| VNC | `vncproxy.NewProxy(nil)` local assumption | `tai.Client.VNC().URL()` |
| Cleanup | `dockerClient.ContainerRemove` | `tai.Client.Sandbox().Remove()` |

### IPC migration detail

All modes use gRPC — no Unix socket fallback, one code path for local and remote.

```
Local:   Container → tai call → Yao gRPC 127.0.0.1:9099
Remote:  Container → tai call → Tai :9100 gateway → Yao gRPC :9099
```

`tai call` is the in-container gRPC bridge (part of the Tai binary). It reads env vars and bridges JSON-RPC/stdio to the Yao gRPC server.

Mode determined by env vars injected by Manager at container creation:

```
# Local: direct to Yao
YAO_GRPC_ADDR=127.0.0.1:9099

# Remote: via Tai gateway
YAO_GRPC_ADDR=tai-host:9100
```

`tai call` reads `YAO_TOKEN` / `YAO_REFRESH_TOKEN` / `YAO_SANDBOX_ID` from env, attaches as gRPC metadata on every call, and handles automatic token refresh from response metadata. The Tai gateway uses the upstream address configured during Tai registration to forward requests to Yao.

Tai relay upstream is NOT configured at Tai startup. `yao-grpc` carries the target address per request — Tai reads `x-grpc-upstream` metadata and proxies dynamically. One Tai can serve containers from different Yao instances.

`BuildMCPConfigForSandbox()` sets the env vars based on `client.IsLocal()`.

### SandboxExecutor interface

The `agent/context/jsapi_sandbox.go` `SandboxExecutor` interface stays the same — it's already abstract. Implementation behind it changes:

```go
type SandboxExecutor interface {
    ReadFile(path string) ([]byte, error)
    WriteFile(path string, data []byte) error
    ListDir(path string) ([]FileInfo, error)
    Exec(cmd string, args ...string) (string, error)
    GetWorkDir() string
    GetSandboxID() string
    GetVNCUrl() (string, error)
}
```

Hooks (`ctx.sandbox.ReadFile()`, etc.) work unchanged. The executor routes to bind mount or `tai.Client.Volume()` internally.

### Agent lifecycle policy

Currently: sandbox created on chat start, removed on chat end (`defer sandboxCleanup`).

New: lifecycle policy set per-assistant config:

```yaml
sandbox:
  lifecycle: session     # one-shot | session | long-running | persistent
  idle_timeout: 30m
  image: yaoapp/workspace:latest
```

`initSandbox()` passes the policy to `sandbox.Manager`, which enforces TTL and cleanup. `sandboxCleanup()` only disconnects the executor — the Manager decides whether to actually remove the container based on policy.

## Migration Path

### Completed

1. **Yao gRPC server** (`yao/grpc`) — full gRPC gateway with 14 RPCs (Run, Stream, Shell, ShellStream, API, MCP×4, ChatCompletions, ChatCompletionsStream, AgentStream, Healthz). OAuth + ACL auth interceptor reusing existing openapi infrastructure. V1 all unary + LLM/Agent streaming done; V2 base streaming (Stream, ShellStream) pending `gou/stream` package. Details: [grpc/DESIGN.md](../grpc/DESIGN.md), [grpc/IMPL.md](../grpc/IMPL.md).

2. **Tai SDK** (`yao/tai`) — unified sandbox runtime SDK with Local/Remote modes. Sandbox (container lifecycle), Volume (file IO + sync with lz4), Workspace (`fs.FS` compatible), Proxy (HTTP reverse proxy), VNC (WebSocket). Remote mode connects via Tai gateway (gRPC :9100, Docker :2375, K8s :6443, HTTP :8080, VNC :6080). Details: [tai/docs/README.md](../tai/docs/README.md).

3. **Tai gateway dynamic routing** (Tai repo) — removed fixed `YaoUpstream` startup config. Tai receives upstream address during registration (`SetUpstream`) and forwards all gRPC requests to it. One Tai serves containers from multiple Yao instances.

4. **`tai call` container client** (Tai repo) — in-container gRPC bridge replacing `yao-bridge`. Reads `YAO_TOKEN`/`YAO_REFRESH_TOKEN`/`YAO_SANDBOX_ID` from env, auto-refreshes tokens via response metadata. `YAO_GRPC_ADDR` determines the target (local Yao or remote Tai gateway).

5. **OAuth Device Flow + CLI auth** — `yao login --server <url>` (RFC 8628 Device Authorization Grant), `yao logout`, credentials stored as base64 JSON in `~/.yao/credentials`. CUI `/auth/device` page for user authorization. Dynamic client registration via machine ID.

6. **`yao run` via gRPC** (`yao/cmd/run.go`) — no `--remote` flag; logged in = gRPC, not logged in = local. `--auth <path>` for alternate credentials. TUI status bar (lipgloss) shows user/scope in gRPC mode, hidden with `-s` (silent).

### Remaining

7. **`sandbox.Manager` refactoring** — replace Docker client with `tai.Client`, unified file ops (bind mount for local, `tai.Client.Volume()` for remote), new lifecycle model (one-shot / session / long-running / persistent).

8. **Agent layer adaptation** — executor uses new Manager, IPC mode switch (gRPC replaces Unix socket), lifecycle policy per-assistant config.

9. **`gou/stream` package** (V2) — streaming process execution foundation. ~150 lines. Enables gRPC `Stream` and `ShellStream` handlers, V8 `Stream()` global.

10. **Workspace persistence** — browser preview, service exposure, delivery.
