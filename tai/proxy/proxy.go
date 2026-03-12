package proxy

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/yaoapp/yao/tai/runtime"
)

// Proxy resolves HTTP service URLs for containers.
// Remote routes through Tai HTTP proxy; Local resolves host ports directly.
type Proxy interface {
	URL(ctx context.Context, containerID string, port int, path string) (string, error)
	Connect(ctx context.Context, containerID string, opts ConnectOptions) (*Connection, error)
	Healthz(ctx context.Context) error
}

// ConnectOptions configures a persistent connection to a container service.
type ConnectOptions struct {
	Port     int    // container port
	Path     string // URL path (e.g. "/ws" or "/events")
	Protocol string // "ws" or "sse"
}

// Connection represents a persistent connection to a container service.
type Connection struct {
	// Messages receives incoming data. Channel is closed when the connection ends.
	Messages <-chan []byte
	// Send writes data to the connection (only valid for "ws" protocol).
	Send func(data []byte) error
	// Close terminates the connection.
	Close func() error
}

// --- Remote implementation ---

type remoteProxy struct {
	base   string // "http://host:port"
	client *http.Client
}

// NewRemote creates a Proxy that routes through Tai's HTTP proxy.
func NewRemote(host string, port int, hc *http.Client) Proxy {
	if hc == nil {
		hc = http.DefaultClient
	}
	return &remoteProxy{
		base:   fmt.Sprintf("http://%s:%d", host, port),
		client: hc,
	}
}

func (r *remoteProxy) URL(_ context.Context, containerID string, port int, path string) (string, error) {
	path = strings.TrimPrefix(path, "/")
	return fmt.Sprintf("%s/%s:%d/%s", r.base, containerID, port, path), nil
}

func (r *remoteProxy) Healthz(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.base+"/healthz", nil)
	if err != nil {
		return err
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("healthz: status %d", resp.StatusCode)
	}
	return nil
}

// --- Tunnel implementation ---

type tunnelProxy struct {
	taiID   string
	yaoBase string // e.g. "http://yao-host:5099"
}

// NewTunnel creates a Proxy that routes through Yao's reverse proxy for
// tunnel-connected Tai instances. URLs point to {yaoBase}/tai/{taiID}/proxy/*.
func NewTunnel(taiID, yaoBase string) Proxy {
	return &tunnelProxy{taiID: taiID, yaoBase: strings.TrimRight(yaoBase, "/")}
}

func (t *tunnelProxy) URL(_ context.Context, containerID string, port int, path string) (string, error) {
	path = strings.TrimPrefix(path, "/")
	return fmt.Sprintf("%s/tai/%s/proxy/%s:%d/%s", t.yaoBase, t.taiID, containerID, port, path), nil
}

func (t *tunnelProxy) Healthz(_ context.Context) error {
	return nil
}

// --- Local implementation ---

type localProxy struct {
	sb runtime.Runtime
}

// NewLocal creates a Proxy that resolves host ports via runtime.Inspect.
func NewLocal(sb runtime.Runtime) Proxy {
	return &localProxy{sb: sb}
}

func (l *localProxy) URL(ctx context.Context, containerID string, port int, path string) (string, error) {
	info, err := l.sb.Inspect(ctx, containerID)
	if err != nil {
		return "", fmt.Errorf("inspect: %w", err)
	}
	for _, p := range info.Ports {
		if p.ContainerPort == port && p.HostPort != 0 {
			path = strings.TrimPrefix(path, "/")
			return fmt.Sprintf("http://%s:%d/%s", hostIP(p.HostIP), p.HostPort, path), nil
		}
	}
	return "", fmt.Errorf("port %d not mapped for container %s", port, containerID)
}

func (l *localProxy) Healthz(_ context.Context) error {
	return nil
}

func hostIP(ip string) string {
	if ip == "" {
		return "127.0.0.1"
	}
	return ip
}
