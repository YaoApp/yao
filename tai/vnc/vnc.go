package vnc

import (
	"context"
	"fmt"
	"net/http"

	"github.com/yaoapp/yao/tai/sandbox"
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

// --- Local implementation ---

type localVNC struct {
	sb sandbox.Sandbox
}

// NewLocal creates a VNC that resolves host VNC ports via sandbox.Inspect.
func NewLocal(sb sandbox.Sandbox) VNC {
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
