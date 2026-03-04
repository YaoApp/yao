# Yao gRPC Server

General-purpose gRPC gateway for the Yao process. Shares OAuth + ACL scope system with openapi — one token, two protocols.

## Services

| Layer | Method | Purpose | Scope |
|-------|--------|---------|-------|
| **Base** | `Run` | Execute Yao process, return result | `grpc:run` |
| | `Stream` | Execute Yao process, stream output | `grpc:stream` |
| | `Shell` | Execute system command, wait for result | `grpc:shell` |
| | `ShellStream` | Execute system command, stream stdout/stderr | `grpc:shell` |
| **API** | `API` | Proxy to openapi, any endpoint | openapi's own scopes |
| **MCP** | `MCPListTools` | List MCP tools for a session | `grpc:mcp` |
| | `MCPCallTool` | Call MCP tool → process.Exec() | `grpc:mcp` |
| | `MCPListResources` | List MCP resources | `grpc:mcp` |
| | `MCPReadResource` | Read MCP resource | `grpc:mcp` |
| **LLM** | `ChatCompletions` | Send messages to LLM, get response | `grpc:llm` |
| | `ChatCompletionsStream` | Stream LLM response (SSE → gRPC stream) | `grpc:llm` |
| **Agent** | `AgentStream` | Call agent, stream response | `grpc:agent` |

## Clients

- Container MCP tools (via Tai gRPC relay)
- `yao run` CLI (after `yao login`)
- Yao-to-Yao (cross-node process execution)

## Auth

Same as openapi. gRPC auth interceptor reuses the same `guard.Authenticate` logic — including automatic token refresh when access token is expired but refresh token is valid.

```
metadata (Bearer + x-refresh-token)
    → VerifyToken
    → expired? → TryRefresh (same as guard.go) → new tokens in response metadata
    → extract scopes → acl.Scope.Check(method, path, scopes)
```

### Infrastructure reuse assessment

Existing openapi/oauth infrastructure can be reused for gRPC with **zero modifications**:

| Component | Reusable as-is | Notes |
|-----------|---------------|-------|
| `VerifyToken(token string)` | Yes | Pure string input, no Gin dependency |
| `MakeAccessToken(clientID, scope, subject, expiresIn, extraClaims...)` | Yes | Supports custom scope/subject for container tokens |
| `MakeRefreshToken(...)` | Yes | Same as above |
| `Revoke(ctx, token, tokenTypeHint)` | Yes | For container token cleanup on Remove |
| `ScopeManager.Check(req *AccessRequest)` | Yes | Only needs `(Method, Path, Scopes)` — no Gin dependency |
| `acl.Register(...)` | Yes | gRPC scopes registered via same pattern |

The `authorized.SetInfo` / `authorized.GetInfo` are Gin-bound but **not needed** — gRPC interceptor builds `AccessRequest` directly from JWT claims. Full `Enforce` chain (client/team/member) is HTTP multi-tenant only; gRPC uses `VerifyToken → ScopeManager.Check` which is sufficient.

New code required: ~80 lines (interceptor + scope registration). Existing code changes: **zero**.

### CLI auth: `yao login` / `yao logout`

OAuth 2.0 Device Authorization Grant. No `--remote` flag needed — logged in = gRPC, not logged in = local.

```
$ yao login --server https://yao.example.com
请访问: https://yao.example.com/device
输入代码: ABCD-1234
等待授权... ✓ (token saved to ~/.yao/credentials)

$ yao run models.user.Find '{"id":1}'    ← auto gRPC
$ yao logout
```

Requires two new openapi endpoints:
- `POST /oauth/device/authorize` — issue device_code + user_code
- `POST /oauth/device/token` — poll for access_token

Token scope: based on user's role, e.g. `grpc:run grpc:stream grpc:shell grpc:llm grpc:agent grpc:mcp`.

**Implementation cost**: ~190 lines new code, ~10 lines changes to existing code.
Scaffolding already in place — `types.DeviceAuthorizationResponse`, `GrantTypeDeviceCode`, error codes (`ErrorAuthorizationPending`, `ErrorSlowDown`), `DeviceCodeLifetime` config, `DeviceAuthorization()` method signature, and HTTP route are all pre-defined. Core work:

1. Implement `DeviceAuthorization()` in `device.go` (currently returns `nil, nil`)
2. Add device_code store/get/consume helpers in `token.go`
3. Add `GrantTypeDeviceCode` case to `Token()` switch in `core.go` (1 case branch)
4. Implement `handleDeviceCodeGrant()` in `core.go`
5. Add user authorization callback handler
6. Fix discovery endpoint path inconsistency (`/oauth/device` vs `/oauth/device_authorization`)

Risk: **very low** — all additions are in isolated code paths, no changes to existing `authorization_code` / `client_credentials` / `refresh_token` flows.

### Container token

Container images and `yao-grpc` (`yao/tai/grpc/`) are ours — it handles token refresh automatically.

```
Manager creates container
    ├─ oauth.MakeAccessToken(subject=userID, scope="grpc:mcp grpc:run")
    ├─ oauth.MakeRefreshToken(...)
    └─ tai.Client.Sandbox().Create(CreateRequest{
           Env: {
               YAO_TOKEN, YAO_REFRESH_TOKEN, YAO_SANDBOX_ID,
               YAO_GRPC_ADDR,                 // where to connect
               YAO_GRPC_UPSTREAM,             // remote only: where Tai should forward to
           },
       })

       Local:  YAO_GRPC_ADDR=127.0.0.1:9099    (direct to Yao, no upstream needed)
       Remote: YAO_GRPC_ADDR=tai-host:9100      YAO_GRPC_UPSTREAM=yao-host:9099

yao-grpc (tai/grpc/, container-internal)
    ├─ reads YAO_GRPC_ADDR + YAO_TOKEN + YAO_REFRESH_TOKEN + YAO_SANDBOX_ID from env
    ├─ if YAO_GRPC_UPSTREAM set: attaches x-grpc-upstream metadata (tells Tai where to forward)
    ├─ every call: Bearer token + x-refresh-token + x-sandbox-id in gRPC metadata
    ├─ server auth interceptor reuses guard.Authenticate logic:
    │   token valid → pass through
    │   token expired + refresh token present → auto rotate (same as HTTP guard)
    │   new tokens returned via response metadata (x-access-token, x-refresh-token)
    ├─ yao-grpc reads response metadata, updates tokens in memory
    └─ transparent to caller, no separate refresh RPC needed
```

- access_token: short TTL (15m)
- refresh_token: no expiry (valid until container removed)
- Manager revokes refresh_token on container Remove
- Tai does NOT know Yao address at startup — yao-grpc carries target in request metadata

### Virtual endpoint mapping

| gRPC | Virtual endpoint |
|------|-----------------|
| Run("models.user.Find") | `POST /grpc/run/models.user.Find` |
| Stream("flows.report") | `POST /grpc/stream/flows.report` |
| Shell | `POST /grpc/shell` |
| ShellStream | `POST /grpc/shell` (same) |
| API(POST, /kb/collections) | `POST /kb/collections` (real openapi path) |
| MCPListTools | `GET /grpc/mcp/tools` |
| MCPCallTool("search") | `POST /grpc/mcp/call/search` |
| MCPListResources | `GET /grpc/mcp/resources` |
| MCPReadResource("uri") | `GET /grpc/mcp/resources/read` |
| ChatCompletions | `POST /grpc/llm/completions` |
| ChatCompletionsStream | `POST /grpc/llm/completions` (same) |
| AgentStream("robot-id") | `POST /grpc/agent/robot-id` |

API method uses the **actual openapi path** — no virtual mapping needed, scope check is identical to HTTP.

### Scope registration

```go
func init() {
    acl.Register(
        &acl.ScopeDefinition{Name: "grpc:run",    Endpoints: []string{"POST /grpc/run/*"}},
        &acl.ScopeDefinition{Name: "grpc:stream", Endpoints: []string{"POST /grpc/stream/*"}},
        &acl.ScopeDefinition{Name: "grpc:shell",  Endpoints: []string{"POST /grpc/shell"}},
        &acl.ScopeDefinition{Name: "grpc:mcp",    Endpoints: []string{"GET /grpc/mcp/tools", "POST /grpc/mcp/call/*", "GET /grpc/mcp/resources", "GET /grpc/mcp/resources/read"}},
        &acl.ScopeDefinition{Name: "grpc:llm",    Endpoints: []string{"POST /grpc/llm/completions"}},
        &acl.ScopeDefinition{Name: "grpc:agent", Endpoints: []string{"POST /grpc/agent/*"}},
    )
}
```

## Network

### Server listen config

| Env | Default | Purpose |
|-----|---------|---------|
| `YAO_GRPC_HOST` | `127.0.0.1` | Comma-separated bind addresses. |
| `YAO_GRPC_PORT` | `9099` | Listen port (shared by all addresses). |
| `YAO_GRPC` | _(unset)_ | Set `off` to explicitly disable gRPC server. |

gRPC server **defaults to enabled** (`127.0.0.1:9099`) — sandbox container callbacks depend on it.

`YAO_GRPC_HOST` accepts one or more addresses separated by `,`. Each address gets its own `net.Listener`; all listeners feed into the same `grpc.Server` (gRPC supports multiple `Serve` calls on one server).

| Scenario | Config | Effect |
|----------|--------|--------|
| Local dev / default | _(nothing to set)_ | `127.0.0.1:9099` — loopback, sandbox works out of box |
| LAN multi-NIC | `YAO_GRPC_HOST=192.168.10.1,10.0.0.1` | Binds each internal IP |
| Open | `YAO_GRPC_HOST=0.0.0.0` | All interfaces |
| Disabled | `YAO_GRPC=off` | gRPC server not started (pure API gateway, no sandbox) |

When multiple addresses are given, the server creates one goroutine per listener. Shutdown (`grpc.GracefulStop`) drains all listeners.

Config lives in `config.Config.GRPC` (type `GRPCConfig`), same pattern as `Host`/`Port` for HTTP.

### Startup

gRPC server starts **after** HTTP server in `cmd/start.go`, as a parallel goroutine:

```
engine.Load → itask.Start → ischedule.Start → service.Start (HTTP) → grpc.StartServer (gRPC)
```

gRPC server starts by default. Set `YAO_GRPC=off` to explicitly disable (no-op startup). Any other value or unset means enabled.

Shutdown: `defer grpc.Stop()` in `cmd/start.go`, called before HTTP stop for graceful drain.

### Access control

Local: containers and CLI connect via loopback. Remote: only Tai relay connects (address known from `YAO_TAI_ADDR`). All callers carry OAuth tokens — no IP allowlist needed.

Interceptor chain: auth → ACL → handler.

Public methods (skip auth): `Healthz`. Auth interceptor checks method name and passes through.

## IPC Path (replacing Unix socket)

All modes use gRPC — no Unix socket fallback. One code path, local and remote.

```
Local:   Container → yao-grpc → Yao gRPC 127.0.0.1:9099
Remote:  Container → yao-grpc → Tai :9100 relay → Yao gRPC :9099
```

`yao-grpc` reads `YAO_GRPC_ADDR` from env and connects. Local containers point directly at the Yao gRPC server on loopback; remote containers point at the Tai relay. No mode switch, no branching.

### Tai relay routing

Tai does **not** know the Yao gRPC address at startup. yao-grpc tells Tai where to forward on every request via metadata:

```
Manager.Create(sandbox)
    ├─ oauth.MakeAccessToken(...)
    ├─ oauth.MakeRefreshToken(...)
    └─ tai.Client.Sandbox().Create(CreateRequest{
           Env: {
               YAO_TOKEN, YAO_REFRESH_TOKEN,
               YAO_GRPC_ADDR: "tai-host:9100",
               YAO_GRPC_UPSTREAM: "yao-host:9099",
           },
       })
```

yao-grpc reads `YAO_GRPC_UPSTREAM` from env and attaches it as `x-grpc-upstream` metadata on every request to Tai. Tai gateway reads this metadata and forwards to the specified address. No per-container state in Tai, no lookup table — pure transparent proxy. One Tai can serve containers from different Yao instances because each request carries its own target.

For local mode, no Tai relay — Manager injects `YAO_GRPC_ADDR=127.0.0.1:9099` directly (no `YAO_GRPC_UPSTREAM` needed).

### yao-grpc (container client)

`yao-grpc` is the in-container gRPC client binary. Replaces the old `yao-bridge`. Lives in `yao/tai/grpc/`:

```
yao/tai/grpc/
├── grpc.go             // gRPC client: connect, forward MCP/process calls
├── auth.go             // token management: read env, auto-refresh
├── grpc_test.go
└── cmd/
    └── main.go
```

Rationale for placing in `yao/tai`:
- Consumes Tai relay — same layer as `tai/proxy`, `tai/volume`
- Shares gRPC deps already in `yao/tai`
- Version-locked with Tai SDK and server protocol
- Built in same CI: `go build -o yao-grpc ./tai/grpc/cmd`

Pure client — no signing keys, no `oauth` package dependency. Reads `YAO_TOKEN` + `YAO_REFRESH_TOKEN` + `YAO_SANDBOX_ID` from env, attaches all three as gRPC metadata on every call. Token refresh is transparent — server auto-rotates expired tokens (same logic as HTTP guard) and returns new tokens via response metadata.

## Proto

```protobuf
service Yao {
  // Base
  rpc Run(RunRequest) returns (RunResponse);
  rpc Stream(RunRequest) returns (stream Chunk);
  rpc Shell(ShellRequest) returns (ShellResponse);
  rpc ShellStream(ShellRequest) returns (stream Chunk);

  // API gateway
  rpc API(APIRequest) returns (APIResponse);

  // MCP
  rpc MCPListTools(MCPListRequest) returns (MCPListResponse);
  rpc MCPCallTool(MCPCallRequest) returns (MCPCallResponse);
  rpc MCPListResources(MCPListRequest) returns (MCPResourcesResponse);
  rpc MCPReadResource(MCPResourceRequest) returns (MCPResourceResponse);

  // AI - LLM
  rpc ChatCompletions(ChatRequest) returns (ChatResponse);
  rpc ChatCompletionsStream(ChatRequest) returns (stream ChatChunk);

  // AI - Agent
  rpc AgentStream(AgentRequest) returns (stream AgentChunk);

  // Health
  rpc Healthz(Empty) returns (HealthzResponse);
}
```

### LLM layer

`ChatCompletions` and `ChatCompletionsStream` call the existing `llm.ChatCompletions` process (`agent/llm/process.go`). It auto-detects connector type (openai/anthropic/etc.), selects the appropriate provider, and returns OpenAI-compatible format.

```
gRPC ChatCompletions(connector, messages, opts)
    → process.Exec("llm.ChatCompletions", connector, messages, opts)
    → agent/llm.New(conn, opts) → provider.Stream/Post → response

gRPC ChatCompletionsStream(connector, messages, opts)
    → same path, with streaming callback → gRPC stream chunks
```

The caller specifies a connector ID. The `llm.ChatCompletions` process resolves it via `connector.Select()`, creates the LLM instance, and executes. Streaming version passes a callback that forwards chunks to the gRPC stream.

### Agent layer

`AgentStream` wraps `agent/robots/:id/completions` — resolves robot → host assistant → runs agent pipeline → streams output. Only stream method — agent output is inherently streamed; non-stream callers simply consume all chunks. Internally calls `assistant.Stream()` with `ctx.Writer` set to nil (or noop) when the caller doesn't need incremental output.

```
gRPC AgentStream(agent_id, messages) → resolve robot → assistant.Stream() → stream chunks
```

This enables container-internal agents to call other agents without HTTP, and remote `yao` instances to orchestrate agent pipelines cross-node.

`AgentChunk` carries `agent/output/message.Message` — the same DSL used by HTTP SSE streaming. Each chunk is one JSON-serialized `Message`:

```protobuf
message AgentChunk {
  bytes data = 1;  // JSON-encoded agent/output/message.Message
  bool  done = 2;
}
```

The `Message` structure uses `Type` + `Props` to express all content types (text, thinking, tool_call, error, action, event, image, audio, video). Streaming control fields (`chunk_id`, `message_id`, `block_id`, `thread_id`) and delta fields (`delta`, `delta_path`, `delta_action`) are preserved as-is over gRPC — the client merges chunks using the same logic as CUI's SSE consumer.

### Shell execution context

`Shell` and `ShellStream` execute commands in the **Yao host process**, not inside a sandbox container. This is by design — the scope `grpc:shell` is a privileged capability, not granted to container tokens by default. Container-internal commands run via `tai.Client.Sandbox().Exec()`, which is a different path (not exposed as a gRPC method).

See [pb/yao.proto](./pb/yao.proto) for full message definitions.

## Process & Stream (gou foundation)

gRPC `Run` and `Stream` map to two parallel systems in `gou`:

```
gou/process/   — execute once, return result     → gRPC Run
gou/stream/    — execute once, push chunks        → gRPC Stream
```

### gou/process (existing, unchanged)

```go
type Handler func(process *Process) interface{}

process.Register("scripts", handler)
p := process.New("scripts.foo.bar", args...)
p.Execute()
result := p.Value()
```

### gou/stream (new package, parallel to process)

```go
type Handler func(ctx context.Context, process *Process, send func([]byte) error) error

stream.Register("scripts", handler)
s := stream.New("scripts.foo.bar", args...)
s.Execute(ctx, func(chunk []byte) error { ... })
```

`stream.Process` mirrors `process.Process` fields (Name, Group, Method, ID, Args, Global, Sid, Authorized) but `ctx` is a first-class parameter, not buried in a struct field.

`send` returns error when the receiver disconnects — handler should stop.

### Fallback

If a stream handler is not registered for a name but a process handler exists, `stream.Execute` falls back to: run the process handler once, JSON-marshal the result, call `send` once.

### Registration

```go
// gou/process — existing
process.Register("models", modelsHandler)
process.Register("scripts", scriptsHandler)

// gou/stream — new, same namespace
stream.Register("scripts", scriptsStreamHandler)
stream.Register("llm", llmStreamHandler)
```

Same naming convention. A process name can have both a process handler and a stream handler.

### gRPC mapping

```go
func (s *yaoServer) Run(ctx context.Context, req *pb.RunRequest) (*pb.RunResponse, error) {
    p := process.NewWithContext(ctx, req.Process, args...)
    if err := p.Execute(); err != nil { return nil, err }
    data, _ := json.Marshal(p.Value())
    return &pb.RunResponse{Result: data}, nil
}

func (s *yaoServer) Stream(req *pb.RunRequest, grpcStream pb.Yao_StreamServer) error {
    st := stream.New(req.Process, args...)
    return st.Execute(grpcStream.Context(), func(chunk []byte) error {
        return grpcStream.Send(&pb.Chunk{Data: chunk})
    })
}
```

### V8 integration

Both are exposed as top-level globals in JavaScript, parallel:

```go
// gou/runtime/v8/isolate.go MakeTemplate
template.Set("Process", processModule.ExportFunction(iso))        // existing
template.Set("Stream",  streamModule.ExportFunction(iso))         // new
```

**JS calling Go stream** (JS is consumer):

```javascript
Stream("llm.chat.completions", function(chunk) {
    log.Info(chunk)
    return 1  // 1=continue, 0=stop
}, { model: "gpt-4", messages: [...] })
```

**JS script as stream handler** (JS is producer):

```javascript
// scripts/report.js — registered via stream.Register("scripts", ...)
function generate(args, send) {
    send("part 1")
    send("part 2")
}
```

V8 runtime registers both:

```go
func init() {
    process.Register("scripts", processScripts)   // existing
    stream.Register("scripts", processScriptsStream) // new
}
```

`processScriptsStream` calls `script.ExecStream(ctx, p, send)` which injects `send` into the V8 global before executing the script method.

### Impact on existing code

| Component | Changes |
|-----------|---------|
| `gou/process/` | None |
| `gou/stream/` | New package (~150 lines) |
| `gou/runtime/v8/process.go` | +1 line: `stream.Register(...)` |
| `gou/runtime/v8/script.go` | +`ExecStream` method |
| `gou/runtime/v8/isolate.go` | +1 line: `template.Set("Stream", ...)` |
| `gou/runtime/v8/functions/` | +`stream/` module for JS→Go stream consumption |
