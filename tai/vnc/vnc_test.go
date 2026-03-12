package vnc

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/yaoapp/yao/tai/runtime"
)

func TestRemoteURL(t *testing.T) {
	v := NewRemote("10.0.0.1", 6080, nil)
	ctx := context.Background()

	url, err := v.URL(ctx, "container-123")
	if err != nil {
		t.Fatalf("URL: %v", err)
	}
	want := "ws://10.0.0.1:6080/vnc/container-123/ws"
	if url != want {
		t.Errorf("got %q, want %q", url, want)
	}
}

func TestRemotePing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Parse host:port from test server URL for real remoteVNC
	u := srv.URL // "http://127.0.0.1:PORT"
	host := u[len("http://"):]
	colonIdx := 0
	for i, c := range host {
		if c == ':' {
			colonIdx = i
			break
		}
	}
	hostStr := host[:colonIdx]
	portStr := host[colonIdx+1:]
	port := 0
	for _, c := range portStr {
		port = port*10 + int(c-'0')
	}

	v := &remoteVNC{host: hostStr, port: port, client: srv.Client()}
	if err := v.Ping(context.Background(), "c1"); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestRemotePingError(t *testing.T) {
	v := &remoteVNC{host: "192.168.254.254", port: 1, client: &http.Client{Timeout: 100 * time.Millisecond}}
	if err := v.Ping(context.Background(), "c1"); err == nil {
		t.Error("expected error for unreachable host")
	}
}

func TestLocalURL(t *testing.T) {
	mock := &mockSandbox{
		inspectFn: func(ctx context.Context, id string) (*runtime.ContainerInfo, error) {
			return &runtime.ContainerInfo{
				ID: id,
				Ports: []runtime.PortMapping{
					{ContainerPort: 6080, HostPort: 49152, HostIP: "127.0.0.1", Protocol: "tcp"},
				},
			}, nil
		},
	}

	v := NewLocal(mock)
	url, err := v.URL(context.Background(), "c1")
	if err != nil {
		t.Fatalf("URL: %v", err)
	}
	want := "ws://127.0.0.1:49152/ws"
	if url != want {
		t.Errorf("got %q, want %q", url, want)
	}
}

func TestLocalURLEmptyHostIP(t *testing.T) {
	mock := &mockSandbox{
		inspectFn: func(ctx context.Context, id string) (*runtime.ContainerInfo, error) {
			return &runtime.ContainerInfo{
				ID: id,
				Ports: []runtime.PortMapping{
					{ContainerPort: 6080, HostPort: 49152, HostIP: "", Protocol: "tcp"},
				},
			}, nil
		},
	}

	v := NewLocal(mock)
	url, err := v.URL(context.Background(), "c1")
	if err != nil {
		t.Fatalf("URL: %v", err)
	}
	want := "ws://127.0.0.1:49152/ws"
	if url != want {
		t.Errorf("got %q, want %q", url, want)
	}
}

func TestLocalURLPortNotFound(t *testing.T) {
	mock := &mockSandbox{
		inspectFn: func(ctx context.Context, id string) (*runtime.ContainerInfo, error) {
			return &runtime.ContainerInfo{ID: id}, nil
		},
	}

	v := NewLocal(mock)
	_, err := v.URL(context.Background(), "c1")
	if err == nil {
		t.Error("expected error for missing VNC port")
	}
}

func TestLocalURLInspectError(t *testing.T) {
	mock := &mockSandbox{
		inspectFn: func(ctx context.Context, id string) (*runtime.ContainerInfo, error) {
			return nil, fmt.Errorf("not found")
		},
	}

	v := NewLocal(mock)
	_, err := v.URL(context.Background(), "c1")
	if err == nil {
		t.Error("expected error for inspect failure")
	}
}

func TestLocalPingSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Parse port from test server
	u := srv.URL[len("http://"):]
	colonIdx := 0
	for i, c := range u {
		if c == ':' {
			colonIdx = i
			break
		}
	}
	portStr := u[colonIdx+1:]
	port := 0
	for _, c := range portStr {
		port = port*10 + int(c-'0')
	}

	mock := &mockSandbox{
		inspectFn: func(ctx context.Context, id string) (*runtime.ContainerInfo, error) {
			return &runtime.ContainerInfo{
				ID: id,
				Ports: []runtime.PortMapping{
					{ContainerPort: 6080, HostPort: port, HostIP: "127.0.0.1", Protocol: "tcp"},
				},
			}, nil
		},
	}

	v := NewLocal(mock)
	if err := v.Ping(context.Background(), "c1"); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestLocalPingError(t *testing.T) {
	mock := &mockSandbox{
		inspectFn: func(ctx context.Context, id string) (*runtime.ContainerInfo, error) {
			return nil, fmt.Errorf("not found")
		},
	}

	v := NewLocal(mock)
	if err := v.Ping(context.Background(), "c1"); err == nil {
		t.Error("expected error")
	}
}

// mockSandbox implements runtime.Sandbox for testing.
type mockSandbox struct {
	inspectFn func(ctx context.Context, id string) (*runtime.ContainerInfo, error)
}

func (m *mockSandbox) Create(ctx context.Context, opts runtime.CreateOptions) (string, error) {
	return "", nil
}
func (m *mockSandbox) Start(ctx context.Context, id string) error { return nil }
func (m *mockSandbox) Stop(ctx context.Context, id string, timeout time.Duration) error {
	return nil
}
func (m *mockSandbox) Remove(ctx context.Context, id string, force bool) error { return nil }
func (m *mockSandbox) Exec(ctx context.Context, id string, cmd []string, opts runtime.ExecOptions) (*runtime.ExecResult, error) {
	return nil, nil
}
func (m *mockSandbox) ExecStream(ctx context.Context, id string, cmd []string, opts runtime.ExecOptions) (*runtime.StreamHandle, error) {
	return nil, nil
}
func (m *mockSandbox) Inspect(ctx context.Context, id string) (*runtime.ContainerInfo, error) {
	if m.inspectFn != nil {
		return m.inspectFn(ctx, id)
	}
	return &runtime.ContainerInfo{ID: id}, nil
}
func (m *mockSandbox) List(ctx context.Context, opts runtime.ListOptions) ([]runtime.ContainerInfo, error) {
	return nil, nil
}
func (m *mockSandbox) Close() error { return nil }
