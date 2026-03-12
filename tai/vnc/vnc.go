package vnc

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/yaoapp/yao/tai/runtime"
)

const defaultVNCContainerPort = 6080

// VNC resolves VNC WebSocket URLs for containers.
// Remote routes through Tai VNC router; Local resolves host ports directly.
type VNC interface {
	URL(ctx context.Context, containerID string) (string, error)
	Ping(ctx context.Context, containerID string) error
}

// --- Remote implementation ---

type remoteVNC struct {
	host   string
	port   int
	client *http.Client
}

// NewRemote creates a VNC that routes through Tai's VNC router.
func NewRemote(host string, port int, hc *http.Client) VNC {
	if hc == nil {
		hc = http.DefaultClient
	}
	return &remoteVNC{host: host, port: port, client: hc}
}

func (r *remoteVNC) URL(_ context.Context, containerID string) (string, error) {
	return fmt.Sprintf("ws://%s:%d/vnc/%s/ws", r.host, r.port, containerID), nil
}

func (r *remoteVNC) Ping(ctx context.Context, containerID string) error {
	url := fmt.Sprintf("http://%s:%d/vnc/%s/ws", r.host, r.port, containerID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// --- Tunnel implementation ---

type tunnelVNC struct {
	taiID   string
	yaoBase string // e.g. "http://yao-host:5099"
}

// NewTunnel creates a VNC that routes through Yao's reverse proxy
// for tunnel-connected Tai instances.
func NewTunnel(taiID, yaoBase string) VNC {
	return &tunnelVNC{taiID: taiID, yaoBase: strings.TrimRight(yaoBase, "/")}
}

func (t *tunnelVNC) URL(_ context.Context, containerID string) (string, error) {
	base := strings.Replace(t.yaoBase, "http://", "ws://", 1)
	base = strings.Replace(base, "https://", "wss://", 1)
	return fmt.Sprintf("%s/tai/%s/vnc/%s/ws", base, t.taiID, containerID), nil
}

func (t *tunnelVNC) Ping(_ context.Context, _ string) error {
	return nil
}

// --- Local implementation ---

type localVNC struct {
	sb runtime.Runtime
}

// NewLocal creates a VNC that resolves host VNC ports via runtime.Inspect.
func NewLocal(sb runtime.Runtime) VNC {
	return &localVNC{sb: sb}
}

func (l *localVNC) URL(ctx context.Context, containerID string) (string, error) {
	info, err := l.sb.Inspect(ctx, containerID)
	if err != nil {
		return "", fmt.Errorf("inspect: %w", err)
	}
	for _, p := range info.Ports {
		if p.ContainerPort == defaultVNCContainerPort && p.HostPort != 0 {
			ip := p.HostIP
			if ip == "" {
				ip = "127.0.0.1"
			}
			return fmt.Sprintf("ws://%s:%d/ws", ip, p.HostPort), nil
		}
	}
	return "", fmt.Errorf("VNC port %d not mapped for container %s", defaultVNCContainerPort, containerID)
}

func (l *localVNC) Ping(ctx context.Context, containerID string) error {
	url, err := l.URL(ctx, containerID)
	if err != nil {
		return err
	}
	httpURL := "http" + url[2:]
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, httpURL, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
