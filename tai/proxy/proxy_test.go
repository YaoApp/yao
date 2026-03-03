package proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/yaoapp/yao/tai/sandbox"
)

func TestRemoteURL(t *testing.T) {
	p := NewRemote("10.0.0.1", 8080, nil)
	ctx := context.Background()

	url, err := p.URL(ctx, "abc123", 3000, "/api/health")
	if err != nil {
		t.Fatalf("URL: %v", err)
	}
	want := "http://10.0.0.1:8080/abc123:3000/api/health"
	if url != want {
		t.Errorf("got %q, want %q", url, want)
	}
}

func TestRemoteURLNoLeadingSlash(t *testing.T) {
	p := NewRemote("host", 8080, nil)
	ctx := context.Background()

	url, _ := p.URL(ctx, "id", 80, "path")
	want := "http://host:8080/id:80/path"
	if url != want {
		t.Errorf("got %q, want %q", url, want)
	}
}

func TestRemoteHealthz(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	// parse host and port from srv.URL
	p := &remoteProxy{base: srv.URL, client: srv.Client()}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := p.Healthz(ctx); err != nil {
		t.Fatalf("Healthz: %v", err)
	}
}

func TestRemoteHealthzFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	p := &remoteProxy{base: srv.URL, client: srv.Client()}
	if err := p.Healthz(context.Background()); err == nil {
		t.Error("expected error for 503")
	}
}

func TestLocalURL(t *testing.T) {
	mock := &mockSandbox{
		inspectFn: func(ctx context.Context, id string) (*sandbox.ContainerInfo, error) {
			return &sandbox.ContainerInfo{
				ID: id,
				Ports: []sandbox.PortMapping{
					{ContainerPort: 3000, HostPort: 32768, HostIP: "127.0.0.1", Protocol: "tcp"},
					{ContainerPort: 8080, HostPort: 32769, HostIP: "127.0.0.1", Protocol: "tcp"},
				},
			}, nil
		},
	}

	p := NewLocal(mock)
	ctx := context.Background()

	url, err := p.URL(ctx, "c1", 3000, "/api")
	if err != nil {
		t.Fatalf("URL: %v", err)
	}
	want := "http://127.0.0.1:32768/api"
	if url != want {
		t.Errorf("got %q, want %q", url, want)
	}
}

func TestLocalURLPortNotFound(t *testing.T) {
	mock := &mockSandbox{
		inspectFn: func(ctx context.Context, id string) (*sandbox.ContainerInfo, error) {
			return &sandbox.ContainerInfo{ID: id}, nil
		},
	}

	p := NewLocal(mock)
	_, err := p.URL(context.Background(), "c1", 9999, "/")
	if err == nil {
		t.Error("expected error for unmapped port")
	}
}

func TestLocalURLInspectError(t *testing.T) {
	mock := &mockSandbox{
		inspectFn: func(ctx context.Context, id string) (*sandbox.ContainerInfo, error) {
			return nil, fmt.Errorf("not found")
		},
	}

	p := NewLocal(mock)
	_, err := p.URL(context.Background(), "c1", 80, "/")
	if err == nil {
		t.Error("expected error for inspect failure")
	}
}

func TestLocalHealthz(t *testing.T) {
	p := NewLocal(&mockSandbox{})
	if err := p.Healthz(context.Background()); err != nil {
		t.Errorf("Healthz should return nil: %v", err)
	}
}

func TestHostIP(t *testing.T) {
	if got := hostIP(""); got != "127.0.0.1" {
		t.Errorf("hostIP empty = %q", got)
	}
	if got := hostIP("10.0.0.1"); got != "10.0.0.1" {
		t.Errorf("hostIP explicit = %q", got)
	}
}

// mockSandbox implements sandbox.Sandbox for testing.
type mockSandbox struct {
	inspectFn func(ctx context.Context, id string) (*sandbox.ContainerInfo, error)
}

func (m *mockSandbox) Create(ctx context.Context, opts sandbox.CreateOptions) (string, error) {
	return "", nil
}
func (m *mockSandbox) Start(ctx context.Context, id string) error { return nil }
func (m *mockSandbox) Stop(ctx context.Context, id string, timeout time.Duration) error {
	return nil
}
func (m *mockSandbox) Remove(ctx context.Context, id string, force bool) error { return nil }
func (m *mockSandbox) Exec(ctx context.Context, id string, cmd []string, opts sandbox.ExecOptions) (*sandbox.ExecResult, error) {
	return nil, nil
}
func (m *mockSandbox) Inspect(ctx context.Context, id string) (*sandbox.ContainerInfo, error) {
	if m.inspectFn != nil {
		return m.inspectFn(ctx, id)
	}
	return &sandbox.ContainerInfo{ID: id}, nil
}
func (m *mockSandbox) List(ctx context.Context, opts sandbox.ListOptions) ([]sandbox.ContainerInfo, error) {
	return nil, nil
}
func (m *mockSandbox) Close() error { return nil }
