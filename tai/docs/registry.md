# Package `registry`

In-memory registry for Tai nodes. Manages both **direct** (network-reachable) and **tunnel** (WebSocket-bridged) connections. Used server-side by Yao to track all connected Tai instances.

## Architecture

```
Direct Mode:     Yao  ── TCP ──>  Tai (gRPC/HTTP/Docker/VNC)
Tunnel Mode:     Yao  <── WS ──  Tai (control channel)
                 Yao  <── WS ──  Tai (data channels, on-demand)
```

## Types

### TaiNode

```go
type TaiNode struct {
    TaiID        string
    MachineID    string
    Version      string
    Auth         AuthInfo
    System       SystemInfo
    Mode         string            // "direct" | "tunnel"
    Addr         string            // direct: "tai-host"; tunnel: empty
    YaoBase      string            // tunnel: Yao server base URL
    Ports        map[string]int    // {"grpc":19100, "http":8099, ...}
    Capabilities map[string]bool   // {"docker":true, "host_exec":true}
    ControlConn  *websocket.Conn   // tunnel: WS control channel
    Status       string            // "online" | "offline" | "connecting"
    ConnectedAt  time.Time
    LastPing     time.Time
    DisplayName  string
}
```

### NodeSnapshot

Read-only copy of `TaiNode` safe to use outside locks. Returned by `Get()` and `List()`.

```go
type NodeSnapshot struct {
    TaiID, MachineID, Version string
    Auth         AuthInfo
    System       SystemInfo
    Mode, Addr, YaoBase string
    Ports        map[string]int
    Capabilities map[string]bool
    Status       string
    ConnectedAt, LastPing time.Time
    DisplayName  string
}
```

### AuthInfo

```go
type AuthInfo struct {
    Subject  string
    UserID   string
    ClientID string
    Scope    string
    TeamID   string
    TenantID string
}
```

### SystemInfo

```go
type SystemInfo struct {
    OS       string `json:"os"`
    Arch     string `json:"arch"`
    Hostname string `json:"hostname"`
    NumCPU   int    `json:"num_cpu"`
    TotalMem int64  `json:"total_mem,omitempty"`
}
```

## Registry API

### Init / Global

```go
func Init(logger *slog.Logger)
func Global() *Registry
```

`Init` creates the global singleton (once). `Global` returns it (nil before Init).

### Register / Unregister

```go
func (r *Registry) Register(node *TaiNode)
func (r *Registry) Unregister(taiID string)
```

`Register` adds or replaces a node, setting `Status="online"` and recording timestamps. `Unregister` closes all tunnel listeners and the control connection, then removes the node.

### Query

```go
func (r *Registry) Get(taiID string) (*NodeSnapshot, bool)
func (r *Registry) List() []NodeSnapshot
func (r *Registry) ListByTeam(teamID string) []NodeSnapshot
```

### Heartbeat

```go
func (r *Registry) UpdatePing(taiID string)
```

### Health Check

```go
func (r *Registry) StartHealthCheck(done <-chan struct{}, interval, timeout, cleanupAfter time.Duration)
```

Runs a background goroutine that:
1. Marks direct-mode nodes as `"offline"` if `LastPing` exceeds `timeout`
2. Auto-unregisters nodes that stay offline longer than `timeout + cleanupAfter`

## Tunnel API

For tunnel-mode nodes, the registry manages on-demand TCP-over-WebSocket channels.

### RequestChannel

```go
func (r *Registry) RequestChannel(taiID string, targetPort int) (channelID string, result chan net.Conn, err error)
```

Sends an `{"type":"open", "channel_id":"...", "target_port":...}` command to the node's control WebSocket. Returns a channel that receives the `net.Conn` when Tai connects back with the data channel. Times out after 30 seconds.

### AcceptDataChannel

```go
func (r *Registry) AcceptDataChannel(channelID, taiID string, conn net.Conn) error
```

Called when Tai establishes a data WebSocket for a pending channel. Validates `taiID` ownership and delivers the connection to the waiting `RequestChannel` caller.

### WriteControlJSON

```go
func (r *Registry) WriteControlJSON(taiID string, v interface{}) error
```

Thread-safe JSON write to a node's control WebSocket.

### OpenLocalListener

```go
func (r *Registry) OpenLocalListener(taiID string, targetPort int) (net.Listener, error)
```

Creates a `127.0.0.1:0` TCP listener. Every accepted connection is automatically bridged through the tunnel to `targetPort` on the Tai node. Returns the listener so the caller can read `ln.Addr()` to get the ephemeral port.

## Connection Flow (Tunnel)

```
1. Tai → Yao: WebSocket upgrade to GET /ws/tai (Bearer auth)
2. Tai → Yao: sends {"type":"register", "tai_id":"xxx", ...} on WS
3. Yao: Register(node) with Mode="tunnel", ControlConn=ws
4. Yao → Tai: sends {"type":"registered", "tai_id":"xxx"}
5. Client → Yao: tai.New("tunnel://tai-abc123")
6. Yao: OpenLocalListener("tai-abc123", 19100) → 127.0.0.1:54321
7. Yao: grpc.Dial("passthrough:///127.0.0.1:54321") → triggers accept
8. Yao: RequestChannel("tai-abc123", 19100) → sends {"type":"open"} on control WS
9. Tai: receives "open", dials localhost:19100, connects data WS to GET /ws/tai/data/:channel_id
10. Yao: AcceptDataChannel(channelID, taiID, conn) → bridges local TCP ↔ data WS
11. gRPC traffic flows transparently through the tunnel
```

### Keep-alive

Tai sends `{"type":"ping"}` periodically on the control channel. Yao replies `{"type":"pong"}` and updates `LastPing`.
