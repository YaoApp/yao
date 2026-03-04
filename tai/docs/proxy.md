# Package `proxy`

HTTP reverse proxy URL resolution. Resolves service URLs for containers so that HTTP services running inside sandboxes can be accessed from the host.

## Interface

```go
type Proxy interface {
    URL(ctx context.Context, containerID string, port int, path string) (string, error)
    Healthz(ctx context.Context) error
}
```

## Implementations

| Implementation | Constructor | Mode | URL Pattern |
|----------------|-------------|------|-------------|
| **Remote** | `NewRemote(host, port, hc)` | Via Tai HTTP proxy | `http://tai-host:8080/{containerID}:{port}/{path}` |
| **Local** | `NewLocal(sb)` | Direct host port lookup | `http://127.0.0.1:{hostPort}/{path}` |

## Constructors

### NewRemote

```go
func NewRemote(host string, port int, hc *http.Client) Proxy
```

Creates a Proxy that routes through Tai's HTTP reverse proxy. URLs are constructed by combining the Tai server address with the container ID and port.

- `host` — Tai server hostname/IP
- `port` — Tai HTTP proxy port (default 8080)
- `hc` — custom HTTP client, `nil` uses `http.DefaultClient`

### NewLocal

```go
func NewLocal(sb sandbox.Sandbox) Proxy
```

Creates a Proxy that resolves URLs by inspecting the container's port mappings via `sandbox.Inspect`. Looks up the host port bound to the requested container port.

Returns an error if the requested port is not mapped.

## Methods

### URL

```go
URL(ctx context.Context, containerID string, port int, path string) (string, error)
```

Resolves an HTTP URL to reach a service running on `port` inside the given container.

**Remote example:** container `abc123` port `3000` path `/api/health`
→ `http://tai-host:8080/abc123:3000/api/health`

**Local example:** container `abc123` port `3000` mapped to host port `32768`
→ `http://127.0.0.1:32768/api/health`

### Healthz

```go
Healthz(ctx context.Context) error
```

Checks the health of the proxy backend.

- **Remote**: sends `GET /healthz` to the Tai HTTP proxy server
- **Local**: always returns `nil` (no external dependency)

## Example

```go
c, _ := tai.New("tai://192.168.1.100")
defer c.Close()

// Get URL for a web service running on port 3000
url, _ := c.Proxy().URL(ctx, containerID, 3000, "/api/status")
resp, _ := http.Get(url)

// Health check
if err := c.Proxy().Healthz(ctx); err != nil {
    log.Fatal("Tai HTTP proxy is down:", err)
}
```
