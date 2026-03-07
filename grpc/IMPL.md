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
grpc/client/                    // gRPC client (moved from tai/grpc/ to grpc/client/)
├── client.go                   // gRPC client, Dial, method wrappers
└── token.go                    // read env tokens, attach metadata, handle refresh

tai repo: tai/call/             // container-side binary (replaces yao-grpc)
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

### Phase 4: Tai gateway change (Tai repo) ✅

Depends on: Phase 1 (need proto definitions for testing). `tai call` (tai repo) depends on this.

Tai gateway receives the upstream address during registration (`SetUpstream`). All gRPC requests are forwarded to the configured upstream — no per-request metadata required.

| Task | Detail | Status |
|------|--------|--------|
| Tai `gateway/gateway.go` | Removed fixed `upstream *grpc.ClientConn`. `SetUpstream` configures the forwarding target. `sync.Map` cache for connections (key = address string). | ✅ Done |
| Tai `server/server.go` | Remove `YaoUpstream` from `Config`. Gateway uses `SetUpstream` after registration. | ✅ Done |
| Tai `main.go` | Remove `--yao` flag, `TAI_YAO_UPSTREAM` env var, YAML `yao` field, and required check. | ✅ Done |
| Tai `gateway/gateway_test.go` | Updated tests: SetUpstream routing, no-upstream → Unavailable, metadata forwarding, upstream error propagation, upstream switching, connection cache. | ✅ Done |

Connection cache: `sync.Map[string, *grpc.ClientConn]` — lazy dial on first request per upstream, reuse thereafter. No eviction needed (upstream count ≈ 1 in practice). `Close` closes all cached connections.

Deliverable: Tai receives upstream via registration. Forwards all requests to configured upstream.

### Phase 5: yao-grpc container client ✅

Depends on: Phase 1 (server + auth), Phase 4 (Tai gateway accepts `x-grpc-upstream`).

| Task | Detail | Status |
|------|--------|--------|
| `tai call` (tai repo) | In-container gRPC bridge. Reads `YAO_TOKEN` / `YAO_REFRESH_TOKEN` / `YAO_SANDBOX_ID` / `YAO_GRPC_ADDR` from env. Attaches auth metadata on every call via unary + stream interceptors. Auto-refresh from response headers. | ✅ Done |

Container token issuance uses existing `oauth.MakeAccessToken` / `oauth.MakeRefreshToken` — called by sandbox Manager at container creation, injected as env vars. Revoke on Remove. No new auth code needed on the issuance side.

Deliverable: `tai call` subcommand (part of Tai binary).

### Phase 6: Device Flow + CLI auth ✅

Depends on: Phase 1. Three sub-phases with sequential dependency: 6.1 → 6.2 → 6.3.

#### Phase 6.1: OAuth Device Flow backend ✅

Backend endpoints for RFC 8628 Device Authorization Grant. Scaffolding already in place (`types.DeviceAuthorizationResponse`, `GrantTypeDeviceCode`, error codes, route registration).

| Task | Detail | Status |
|------|--------|--------|
| `engine/machine.go` + platform files | `GetMachineID()` Go API + `utils.app.MachineID` process — cross-platform (macOS/Linux/Windows) deterministic machine fingerprint | ✅ Done |
| `oauth/token.go` | `deviceCodeKey`, `userCodeKey`, `storeDeviceCode`, `getDeviceCodeData`, `authorizeDeviceCode`, `consumeDeviceCode` — device_code + user_code storage/retrieval/consumption helpers | ✅ Done |
| `oauth/device.go` | Implement `DeviceAuthorization()` + `AuthorizeDevice()` + `generateUserCode()` — generate codes (gonanoid, XXXX-XXXX format), validate client + grant type, store, return `DeviceAuthorizationResponse` | ✅ Done |
| `oauth/core.go` | Add `case types.GrantTypeDeviceCode` → `handleDeviceCodeGrant()` — poll returns `authorization_pending` / `expired_token` / token | ✅ Done |
| `openapi/oauth.go` | Replace stub `oauthDeviceAuthorization` handler → call `DeviceAuthorization()`. Add `POST /oauth/device/authorize` → `oauthDeviceAuthorize` (bearer token + user_code → authorize device). | ✅ Done |
| `oauth/discovery.go` | Fix path: `/oauth/device` → `/oauth/device_authorization` | ✅ Done |
| `oauth/oauth.go` | Config defaults: `DeviceCodeLength=8`, `UserCodeLength=8`, `DeviceCodeInterval=5s`, `DeviceFlowEnabled=true`, `DynamicClientRegistrationEnabled=true` | ✅ Done |
| `openapi/tests/oauth/device_test.go` | Full test suite: device auth success/error, token polling (pending/invalid), end-to-end flow | ✅ Done |

Deliverable: Device flow endpoints functional — `POST /oauth/device_authorization` issues codes, `POST /oauth/token` with `grant_type=device_code` polls status. `POST /oauth/device/authorize` allows authenticated user to authorize device.

#### Phase 6.2: CUI auth/device page (frontend) ✅

Depends on: Phase 6.1 (backend endpoints). Frontend-only task in **CUI repo**.

Route: `/auth/device` (Umi convention-based routing → `pages/auth/device/index.tsx`)

| Task | Detail | Status |
|------|--------|--------|
| `pages/auth/device/index.tsx` | Device authorization page. User enters `user_code`, clicks Authorize. Uses `AuthLayout` + `AuthInput` + `AuthButton` from existing `pages/auth/components/`. Three states: input, success, error. i18n (zh/en), light/dark, system CSS variables only. | ✅ Done |
| `pages/auth/device/index.less` | Styles, follow `pages/auth/entry/mfa/index.less` pattern. Full responsive + dark theme. | ✅ Done |
| `openapi/user/auth.ts` | `AuthorizeDevice(userCode)` method — `POST /oauth/device/authorize` | ✅ Done |
| `layouts/index.tsx` | Register `['auth_device', '/auth/device']` in `STANDALONE_PAGES` | ✅ Done |

Implementation:

- Framework: React + UmiJS Max + Ant Design + MobX (same as all auth pages)
- Layout: `AuthLayout` (logo + theme switch), same as `/auth/entry`
- Components: reuse `AuthInput` for `user_code` input, `AuthButton` for submit
- Page export: `export default observer(DeviceAuth)`
- API: `window.$app.openapi` → `POST /oauth/device/authorize` with `{ user_code }`, bearer token from session
- Auth: must be logged in (redirect to `/auth/entry` if not). After authorizing, show success and close/redirect
- i18n: `useIntl()`, `zh-CN` / `en-US`

Deliverable: `/auth/device` page. User authorizes CLI device login from browser.

#### Phase 6.3: CLI commands + TUI status bar ✅

Depends on: Phase 6.1 (backend) + Phase 6.2 (CUI page for end-to-end `yao login`).

**Credentials file** (`~/.yao/credentials`): base64-encoded JSON.

```json
{
  "server": "https://yao.example.com",
  "access_token": "eyJ...",
  "refresh_token": "eyJ...",
  "scope": "grpc:run grpc:stream grpc:shell grpc:llm grpc:agent grpc:mcp",
  "user": "admin@example.com",
  "expires_at": "2026-03-05T10:00:00Z"
}
```

Stored as: `base64(json) → ~/.yao/credentials`. Prevents casual `cat` exposure.

| Task | Detail | Status |
|------|--------|--------|
| `cmd/credential.go` | `Credential` struct, `LoadCredential`, `LoadCredentialFrom`, `SaveCredential`, `RemoveCredential` — base64-encoded JSON read/write to `~/.yao/credentials` | ✅ Done |
| `cmd/login.go` | `yao login --server <url>` — compute machine ID → `POST /oauth/register` (dynamic client) → `POST /oauth/device_authorization` → color-print device code + verification URL → poll `POST /oauth/token` with interval + slow_down handling → save to `~/.yao/credentials` | ✅ Done |
| `cmd/logout.go` | `yao logout` — read credentials, best-effort `POST /oauth/revoke`, delete `~/.yao/credentials` | ✅ Done |
| `cmd/run.go` | Detect credentials → gRPC mode vs local mode. `--auth <path>` flag loads alternate credentials file. `-s` (silent) mode: no TUI. gRPC mode renders TUI status bar then calls remote (gRPC call wiring pending Phase 4/5 integration). Local mode unchanged. | ✅ Done |
| `cmd/tui_status.go` | lipgloss `RenderStatusBar(cred)` — one-line persistent bar: `user (gRPC) │ scope: run,stream,...`. Rounded border, colored connection info. Hidden in silent mode. | ✅ Done |
| `cmd/root.go` | Register `loginCmd`, `logoutCmd` in root command | ✅ Done |
| i18n | All new strings have zh-CN translations via `langs` map | ✅ Done |

**`yao run` behavior matrix:**

| Credentials | `-s` flag | `--auth` flag | Behavior |
|-------------|-----------|---------------|----------|
| None | — | — | Local execution (current behavior) |
| `~/.yao/credentials` | No | — | gRPC + TUI status bar |
| `~/.yao/credentials` | Yes | — | gRPC, no TUI, pure output |
| — | Yes | `<path>` | gRPC via specified credentials, no TUI, pure output |
| — | No | `<path>` | gRPC via specified credentials + TUI status bar |

**TUI status bar** (bubbletea, `cmd/tui_status.go`):

```
┌─ admin@yao.example.com (gRPC) │ scope: run,stream,shell,llm,agent,mcp ─┐
```

- Top-line, persistent during execution
- lipgloss styled (dim border, colored connection info)
- Process output renders below, unaffected
- Hidden in silent mode (`-s`)

Deliverable: `yao login` + `yao logout` + `yao run` via gRPC with TUI status bar.

## V2 Phases

### Phase 7: `gou/stream` package ⏳

| Task | Detail | Status |
|------|--------|--------|
| `gou/stream/` | ~150 lines. `Handler`, `Process`, `Register`, `New`, `Execute`. Fallback to process. | ⏳ Pending |
| V8 | `stream.Register("scripts", ...)`, `ExecStream`, `template.Set("Stream", ...)`, JS `Stream()` global | ⏳ Pending |

### Phase 8: Base streaming handlers ⏳

Depends on: Phase 7.

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
    ├───────────┬───────────┬──────────────────────┐
    ▼           ▼           ▼                      ▼
Phase 2 ✅   Phase 3 ✅  Phase 4 ✅           Phase 6 ✅ (device flow + CLI)
(handlers)  (LLM/Agent)  (Tai gateway)            │
                            │              ┌───────┴───────┐
                            ▼              ▼               ▼
                         Phase 5 ✅     6.1 ✅          6.2 ✅
                         (yao-grpc)    (OAuth backend)  (CUI page)
                                           │               │
                                           └───────┬───────┘
                                                   ▼
                                              6.3 ✅
                                         (CMD + TUI)

--- V2 ---

Phase 7 (gou/stream)
    │
    ▼
Phase 8 (Stream, ShellStream)
```
