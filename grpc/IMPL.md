# Yao gRPC Server — Implementation Plan

Design: [DESIGN.md](./DESIGN.md)

## Scope

**V1**: Auth + unary RPCs + LLM/Agent streaming + container client.

**V2**: Base streaming (`Stream`, `ShellStream`) + `gou/stream` package + V8 integration.

## Package Structure

```
grpc/
├── grpc.go                 // StartServer, config, server lifecycle
├── pb/
│   ├── yao.proto
│   ├── yao.pb.go           // generated
│   └── yao_grpc.pb.go      // generated
├── auth/
│   ├── guard.go            // unary + stream interceptor (calls oauth.VerifyToken, ScopeManager.Check)
│   ├── endpoint.go         // gRPC method → virtual HTTP endpoint mapping
│   └── scope.go            // init() acl.Register for grpc:* scopes
├── run/
│   └── run.go              // Run handler
├── shell/
│   └── shell.go            // Shell, ShellStream (V2) handlers
├── api/
│   └── api.go              // API proxy handler
├── mcp/
│   └── mcp.go              // MCPListTools, MCPCallTool, MCPListResources, MCPReadResource
├── llm/
│   └── llm.go              // ChatCompletions, ChatCompletionsStream
├── agent/
│   └── agent.go            // AgentStream
└── health/
    └── health.go           // Healthz
```

Container client:

```
tai/grpc/
├── grpc.go                 // gRPC client, Dial, method wrappers
├── auth.go                 // read env tokens, attach metadata, handle refresh
├── grpc_test.go
└── cmd/
    └── main.go             // yao-grpc binary entry
```

## V1 Phases

### Phase 0: Proto + codegen ✅

No dependency.

| Task | Detail | Status |
|------|--------|--------|
| `grpc/pb/yao.proto` | All 14 RPCs + all message types. V2 methods (`Stream`, `ShellStream`) included in proto, handler left `Unimplemented`. | ✅ Done |
| codegen | `protoc` → `pb/*.pb.go` + `pb/*_grpc.pb.go` | ✅ Done |

### Phase 1: Auth + server skeleton ✅

Depends on: Phase 0. Auth is ~80 lines new code calling existing `openapi/oauth` functions.

| Task | Detail | Status |
|------|--------|--------|
| `grpc/auth/scope.go` | `init()` — `acl.Register` 6 gRPC scope definitions | ✅ Done |
| `grpc/auth/endpoint.go` | Map gRPC method + request params → virtual HTTP endpoint for ACL (e.g. `Run("models.user.Find")` → `POST /grpc/run/models.user.Find`) | ✅ Done |
| `grpc/auth/guard.go` | Extract Bearer from metadata → `oauth.AuthenticateToken` (pure, no gin) → ACL scope check. Skip `Healthz`. New tokens via `SendHeader`. | ✅ Done |
| `openapi/oauth/authenticate.go` | `AuthenticateToken(AuthInput) → AuthResult` — gin-free auth core. `refreshTokenDirect`, `buildAuthInfo`. Shares `refreshGates` with `TryRefreshToken`. | ✅ Done |
| `grpc/grpc.go` | `StartServer(cfg)` — `grpc.NewServer` with interceptor, register service, listen. See **Server config & startup** below. | ✅ Done |
| `grpc/health/health.go` | `Healthz` → `{status: "ok"}` | ✅ Done |
| `config/types.go` | Add `GRPC` field to `Config` struct — see config below | ✅ Done |
| `cmd/start.go` | After `service.Start(config.Conf)` (HTTP ready), call `grpc.StartServer(config.Conf)` in goroutine. Print gRPC listen address in Access Points block. `defer grpc.Stop()` in shutdown path. | ✅ Done |

**Server config & startup:**

Config struct addition (`config/types.go`):

```go
type Config struct {
    // ... existing fields ...
    GRPC GRPCConfig `json:"grpc,omitempty"`
}

type GRPCConfig struct {
    Enabled string `json:"enabled,omitempty" env:"YAO_GRPC"`
    Host    string `json:"host,omitempty" env:"YAO_GRPC_HOST" envDefault:"127.0.0.1"`
    Port    int    `json:"port,omitempty" env:"YAO_GRPC_PORT" envDefault:"9099"`
}
```

- **Default** — `127.0.0.1:9099`, enabled. Sandbox callbacks work out of box.
- `YAO_GRPC_HOST=192.168.10.1,10.0.0.1` — comma-separated, binds each IP for multi-NIC LAN
- `YAO_GRPC_HOST=0.0.0.0` — all interfaces
- `YAO_GRPC=off` — explicitly disable gRPC server

`grpc.StartServer` implementation:

```go
func StartServer(cfg config.Config) error {
    if strings.ToLower(cfg.GRPC.Enabled) == "off" {
        log.Info("gRPC server disabled (YAO_GRPC=off)")
        return nil
    }
    hosts := strings.Split(cfg.GRPC.Host, ",")
    for _, host := range hosts {
        addr := net.JoinHostPort(strings.TrimSpace(host), strconv.Itoa(cfg.GRPC.Port))
        lis, err := net.Listen("tcp", addr)
        // ... error handling ...
        go server.Serve(lis)  // one goroutine per listener, same grpc.Server
    }
    return nil
}
```

Startup sequence in `cmd/start.go`:

```
engine.Load(cfg)
itask.Start()
ischedule.Start()
service.Start(cfg)          // HTTP server
grpc.StartServer(cfg)       // gRPC server (after HTTP, parallel goroutine)
// ... event loop ...
defer grpc.Stop()           // GracefulStop drains all listeners (no-op if not started)
```

`cmd/start.go` prints each gRPC listen address:

```
Listening  0.0.0.0:5099 (HTTP)
Listening  192.168.10.1:9099 (gRPC)
Listening  10.0.0.1:9099 (gRPC)
```

Deliverable: Server starts, Healthz works, unauthenticated calls rejected, token refresh via metadata works.

### Phase 2: Base + API + MCP handlers ✅

Depends on: Phase 1.

| Task | Detail | Status |
|------|--------|--------|
| `grpc/run/run.go` | `Run` — `process.New(req.Process, args...).Exec()`. Injects `AuthorizedInfo` via `p.WithSID()` + `p.WithAuthorized()`. | ✅ Done |
| `grpc/shell/shell.go` | `Shell` — `exec.CommandContext` in host process. **Security**: refuse execution if Yao process is running as root (`os.Getuid() == 0` → `PermissionDenied`). Timeout: use request `timeout` field, default 30s, capped by server max. | ✅ Done |
| `grpc/api/api.go` | `API` — build `http.Request`, call openapi internally | ✅ Done |
| `grpc/mcp/mcp.go` | `MCPListTools`, `MCPCallTool`, `MCPListResources`, `MCPReadResource` | ✅ Done |

Deliverable: Base + API + MCP methods work with valid tokens.

### Phase 3: LLM + Agent handlers ✅

Depends on: Phase 1. No code dependency on Phase 2 — can parallel.

| Task | Detail | Status |
|------|--------|--------|
| `grpc/llm/llm.go` | `ChatCompletions` / `ChatCompletionsStream` — direct call to `agent/llm` (`connector.Select` → `llm.New` → `Stream`). Uses `agent/llm.BuildCompletionOptions`. Constructs `agent/context.Context` with `AuthorizedInfo`. | ✅ Done |
| `grpc/agent/agent.go` | `AgentStream` — `assistant.Get` → `ast.Stream` with `grpcStreamWriter` adapter bridging `http.ResponseWriter` to gRPC `ServerStreamingServer`. Constructs `agent/context.Context` with `AuthorizedInfo`. | ✅ Done |

Deliverable: LLM (unary + stream) and Agent streaming via gRPC.

### Phase 4: Tai gateway change (Tai repo) ⏳

Depends on: Phase 1 (need proto definitions for testing). yao-grpc depends on this.

Tai gateway currently dials a fixed `YaoUpstream` at startup. New behavior: yao-grpc tells Tai where to forward via request metadata (`x-grpc-upstream`). Tai reads the target address and proxies to it — removes `YaoUpstream` startup config.

| Task | Detail | Status |
|------|--------|--------|
| Tai `gateway/gateway.go` | Remove fixed `upstream *grpc.ClientConn`. On each request, read `x-grpc-upstream` from metadata → lookup/create conn from `sync.Map` cache (key = address string) → forward. Typical deployment has 1 upstream, cache stays tiny. | ⏳ Pending |
| Tai `server/server.go` | Remove `YaoUpstream` from `Config`. Gateway init no longer needs an address. | ⏳ Pending |

Connection cache: `sync.Map[string, *grpc.ClientConn]` — lazy dial on first request per upstream, reuse thereafter. No eviction needed (upstream count ≈ 1 in practice). `GracefulStop` closes all cached connections.

Deliverable: Tai starts without Yao address. Forwards based on request metadata.

### Phase 5: yao-grpc container client ⏳

Depends on: Phase 1 (server + auth), Phase 4 (Tai gateway accepts `x-grpc-upstream`).

| Task | Detail | Status |
|------|--------|--------|
| `tai/grpc/grpc.go` | `Dial(YAO_GRPC_ADDR)`, method wrappers mirroring server | ⏳ Pending |
| `tai/grpc/auth.go` | Read `YAO_TOKEN` / `YAO_REFRESH_TOKEN` / `YAO_SANDBOX_ID` / `YAO_GRPC_UPSTREAM` from env. Attach as metadata on every call: Bearer token, `x-refresh-token`, `x-sandbox-id`, `x-grpc-upstream` (if set, for Tai relay). Read `SendHeader` for rotated tokens, update in memory. | ⏳ Pending |
| `tai/grpc/cmd/main.go` | Stdio MCP server: JSON-RPC → gRPC. Replaces `yao-bridge`. `yao-grpc version` prints version/commit/build time (via `-ldflags`), for container debugging. | ⏳ Pending |
| `tai/grpc/grpc_test.go` | Tests | ⏳ Pending |

Container token issuance uses existing `oauth.MakeAccessToken` / `oauth.MakeRefreshToken` — called by sandbox Manager at container creation, injected as env vars. Revoke on Remove. No new auth code needed on the issuance side.

Deliverable: `go build -o yao-grpc ./tai/grpc/cmd`.

### Phase 6: Device Flow — backend (`yao login`) ⏳

Depends on: Phase 1. Independent — can parallel with Phase 2-5.

| Task | Detail | Status |
|------|--------|--------|
| `oauth/device.go` | Implement `DeviceAuthorization()` — generate `device_code` + `user_code`, store with expiry | ⏳ Pending |
| `oauth/token.go` | Device code store/get/consume helpers | ⏳ Pending |
| `oauth/core.go` | Add `GrantTypeDeviceCode` case → `handleDeviceCodeGrant()` (poll returns `authorization_pending` / token) | ⏳ Pending |
| `cmd/yao/login.go` | `yao login --server <url>` → device flow → poll token endpoint → save `~/.yao/credentials` | ⏳ Pending |
| `cmd/yao/logout.go` | Revoke + delete credentials | ⏳ Pending |
| `cmd/yao/run.go` | Credentials exist → gRPC; otherwise local. Non-silent mode prints `⟶ user@host (gRPC)` header before execution (same line position as existing `Run: process.name`). Silent mode (`-s`) keeps pure output — no connection info, for shell scripting. | ⏳ Pending |

Deliverable: `yao login` + `yao run` via gRPC (backend complete, auth page in Phase 7).

### Phase 7: Device Flow — CUI auth page (frontend) ⏳

Depends on: Phase 6 (backend endpoints ready). This is a **frontend-only** task in the CUI repo.

Route: `/auth/device` (Umi convention-based routing → `pages/auth/device/index.tsx`)

| Task | Detail | Status |
|------|--------|--------|
| `pages/auth/device/index.tsx` | Device authorization page. User enters `user_code` and clicks Authorize. Uses `AuthLayout` + `AuthInput` + `AuthButton` from existing `pages/auth/components/`. | ⏳ Pending |
| `pages/auth/device/index.less` | Styles, follow `pages/auth/entry/index.less` pattern | ⏳ Pending |

**Implementation details:**

- Framework: React + UmiJS Max + Ant Design + MobX (same as all auth pages)
- Layout: Wrap with `AuthLayout` (logo + theme switch), same as `/auth/entry`
- Components reuse: `AuthInput` for `user_code` input, `AuthButton` for submit, from `pages/auth/components/`
- Page export: `export default observer(DeviceAuth)` (same pattern as `pages/auth/entry/index.tsx`)
- API: `window.$app.openapi` → call backend `POST /oauth/device/authorize` with `{ user_code }`, bearer token from current session
- Auth: User must be logged in (redirect to `/auth/entry` if not). After authorizing, show success message and close/redirect
- i18n: Use `useIntl()` hook for text, support `zh-CN` / `en-US`
- Flow: User opens URL from CLI prompt → logs in if needed → enters user_code → clicks Authorize → backend binds device_code to user → CLI poll gets token

Deliverable: `/auth/device` page in CUI. User can authorize CLI device login from browser.

## V2 Phases

### Phase 8: `gou/stream` package ⏳

| Task | Detail | Status |
|------|--------|--------|
| `gou/stream/` | ~150 lines. `Handler`, `Process`, `Register`, `New`, `Execute`. Fallback to process. | ⏳ Pending |
| V8 | `stream.Register("scripts", ...)`, `ExecStream`, `template.Set("Stream", ...)`, JS `Stream()` global | ⏳ Pending |

### Phase 9: Base streaming handlers ⏳

Depends on: Phase 8.

| Task | Detail | Status |
|------|--------|--------|
| `grpc/run/run.go` | Add `Stream` handler — `stream.New(req.Process).Execute(ctx, send)` | ⏳ Pending |
| `grpc/shell/shell.go` | Add `ShellStream` handler — piped stdout → gRPC stream | ⏳ Pending |

## Dependency Graph

```
Phase 0 (proto)             ✅
    │
    ▼
Phase 1 (auth + server)     ✅
    │
    ├───────────┬───────────┬──────────────┐
    ▼           ▼           ▼              ▼
Phase 2 ✅   Phase 3 ✅  Phase 4 (Tai)   Phase 6
(handlers)  (LLM/Agent)    │           (device backend)
                            ▼              │
                         Phase 5           ▼
                         (yao-grpc)     Phase 7
                                       (CUI auth page)

--- V2 ---

Phase 8 (gou/stream)
    │
    ▼
Phase 9 (Stream, ShellStream)
```
