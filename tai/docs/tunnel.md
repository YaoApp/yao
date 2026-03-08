# Package `tunnel`

WebSocket tunnel handlers for Tai nodes that cannot be reached directly (e.g. behind NAT/firewall). Provides Gin HTTP handlers mounted on the Yao server.

## Handlers

| Method | Route | Handler | Description |
|--------|-------|---------|-------------|
| `GET` | `/ws/tai` | `HandleControl` | Control channel WebSocket |
| `GET` | `/ws/tai/data/:channel_id` | `HandleData` | Data channel WebSocket |
| `ANY` | `/tai/:taiID/proxy/*path` | `HandleProxy` | HTTP reverse proxy via tunnel |
| `GET` | `/tai/:taiID/vnc/*path` | `HandleVNC` | VNC WebSocket proxy via tunnel |

All WebSocket endpoints require `Authorization: Bearer <token>` header.

## HandleControl

Manages the long-lived control WebSocket for a Tai node.

**Flow:**
1. Authenticate Bearer token via OAuth
2. Upgrade to WebSocket
3. Read `register` message (JSON):

```json
{
  "type": "register",
  "tai_id": "tai-abc123",
  "machine_id": "m-001",
  "version": "1.2.0",
  "server": "http://yao-server:5099",
  "ports": {"grpc": 19100, "http": 8099, "vnc": 16080, "docker": 12375},
  "capabilities": {"docker": true, "host_exec": true},
  "system": {"os": "linux", "arch": "amd64", "hostname": "host-01", "num_cpu": 8}
}
```

4. Register node in global registry with `Mode="tunnel"`
5. Reply `{"type": "registered", "tai_id": "tai-abc123"}`
6. Loop reading messages:
   - `{"type": "ping"}` → update heartbeat, reply `{"type": "pong"}`
7. On disconnect → unregister node

## HandleData

Handles on-demand data channel connections from Tai.

**Flow:**
1. Authenticate Bearer token
2. Extract `:channel_id` from URL
3. Upgrade to WebSocket
4. Wrap WebSocket as `net.Conn` (bidirectional byte bridge)
5. Call `registry.AcceptDataChannel(channelID, clientID, conn)`

The `channel_id` must match a pending `RequestChannel` call. The `clientID` (from token) must match the Tai node that owns the channel.

## HandleProxy

HTTP reverse proxy for tunnel-connected Tai nodes.

**Flow:**
1. Look up Tai node from `:taiID` in registry
2. Get the node's HTTP port (from `node.Ports["http"]`, default 8099)
3. Open a tunnel data channel to that port via `RequestChannel`
4. Forward the incoming HTTP request through the tunnel
5. Read the response and stream it back to the client

## HandleVNC

VNC WebSocket proxy for tunnel-connected Tai nodes.

**Flow:**
1. Look up Tai node from `:taiID` in registry
2. Get the node's VNC port (from `node.Ports["vnc"]`, default 16080)
3. Open a tunnel data channel to that port via `RequestChannel`
4. Upgrade the client connection to WebSocket
5. Bridge client WebSocket ↔ tunnel data channel (binary messages)

## Internal Types

### wsConn

`wsConn` wraps `gorilla/websocket.Conn` to implement `net.Conn` for bidirectional byte bridging. This allows tunnel data channels to be treated as standard TCP connections by the registry's bridge logic.

```go
type wsConn struct { ... }
func (c *wsConn) Read(p []byte) (int, error)   // reads WS binary messages
func (c *wsConn) Write(p []byte) (int, error)  // writes WS binary messages
func (c *wsConn) Close() error
// Also implements: LocalAddr, RemoteAddr, SetDeadline, SetReadDeadline, SetWriteDeadline
```
