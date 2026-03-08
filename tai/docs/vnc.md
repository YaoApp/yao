# Package `vnc`

VNC WebSocket URL resolution. Resolves WebSocket URLs for VNC sessions running inside containers, enabling remote desktop access to sandbox environments.

## Interface

```go
type VNC interface {
    URL(ctx context.Context, containerID string) (string, error)
    Ping(ctx context.Context, containerID string) error
}
```

## Implementations

| Implementation | Constructor | Mode | URL Pattern |
|----------------|-------------|------|-------------|
| **Remote** | `NewRemote(host, port, hc)` | Via Tai VNC router | `ws://tai-host:16080/vnc/{containerID}/ws` |
| **Local** | `NewLocal(sb)` | Direct host port lookup | `ws://127.0.0.1:{hostPort}/ws` |
| **Tunnel** | `NewTunnel(taiID, yaoBase)` | Via Yao reverse proxy | `ws(s)://{yaoHost}/tai/{taiID}/vnc/{containerID}/ws` |

## Constructors

### NewRemote

```go
func NewRemote(host string, port int, hc *http.Client) VNC
```

Creates a VNC that routes through Tai's VNC WebSocket router.

- `host` — Tai server hostname/IP
- `port` — Tai VNC router port (default 16080)
- `hc` — custom HTTP client for Ping, `nil` uses `http.DefaultClient`

### NewLocal

```go
func NewLocal(sb sandbox.Sandbox) VNC
```

Creates a VNC that resolves URLs by inspecting the container's port mappings. Looks for container port **6080** (the standard noVNC port) in the port mappings.

Returns an error if port 6080 is not mapped. On macOS and Windows (Docker Desktop), the Local sandbox automatically maps port 6080 when `CreateOptions.VNC` is `true`.

### NewTunnel

```go
func NewTunnel(taiID, yaoBase string) VNC
```

Creates a VNC that routes through Yao's HTTP reverse proxy for tunnel-mode connections.

- `taiID` — the Tai node identifier in the registry
- `yaoBase` — the Yao server base URL (e.g. `"http://yao-server:5099"`)

## Methods

### URL

```go
URL(ctx context.Context, containerID string) (string, error)
```

Returns a WebSocket URL for connecting to the container's VNC session.

**Remote:** `ws://tai-host:16080/vnc/abc123/ws`
**Local:** `ws://127.0.0.1:32769/ws`

### Ping

```go
Ping(ctx context.Context, containerID string) error
```

Checks if the VNC endpoint is reachable by making an HTTP GET request to the WebSocket URL. Useful for verifying that the VNC server inside the container is ready before connecting a client.

- **Remote**: sends GET to `http://tai-host:16080/vnc/{containerID}/ws`
- **Local**: resolves the host port via Inspect, then sends GET
- **Tunnel**: always returns `nil` (no direct network path to probe)

## Example

```go
c, _ := tai.New("tai://192.168.1.100")
defer c.Close()

// Create a sandbox with VNC enabled
id, _ := c.Sandbox().Create(ctx, sandbox.CreateOptions{
    Name:  "desktop",
    Image: "yaoapp/sandbox-claude:latest",
    VNC:   true,
})
c.Sandbox().Start(ctx, id)

// Wait for VNC to be ready
for i := 0; i < 10; i++ {
    if err := c.VNC().Ping(ctx, id); err == nil {
        break
    }
    time.Sleep(time.Second)
}

// Get the WebSocket URL for a noVNC client
url, _ := c.VNC().URL(ctx, id)
fmt.Println(url) // ws://192.168.1.100:16080/vnc/desktop/ws
```
