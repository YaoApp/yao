package proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/yaoapp/yao/tai/runtime"
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
		inspectFn: func(ctx context.Context, id string) (*runtime.ContainerInfo, error) {
			return &runtime.ContainerInfo{
				ID: id,
				Ports: []runtime.PortMapping{
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
		inspectFn: func(ctx context.Context, id string) (*runtime.ContainerInfo, error) {
			return &runtime.ContainerInfo{ID: id}, nil
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
		inspectFn: func(ctx context.Context, id string) (*runtime.ContainerInfo, error) {
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

// ── Connect tests ─────────────────────────────────────────────────────────────

func TestConnect_UnsupportedProtocol(t *testing.T) {
	_, err := connect(context.Background(), "http://127.0.0.1:1234", "tcp", http.DefaultClient)
	if err == nil {
		t.Fatal("expected error for unsupported protocol")
	}
	if !strings.Contains(err.Error(), "unsupported connect protocol") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestConnectWS_EchoRoundtrip(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		for {
			mt, msg, err := c.ReadMessage()
			if err != nil {
				return
			}
			c.WriteMessage(mt, msg)
		}
	}))
	defer srv.Close()

	conn, err := connectWS(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("connectWS: %v", err)
	}
	defer conn.Close()

	if err := conn.Send([]byte("hello")); err != nil {
		t.Fatalf("Send: %v", err)
	}

	select {
	case msg := <-conn.Messages:
		if string(msg) != "hello" {
			t.Errorf("got %q, want %q", msg, "hello")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for echo")
	}
}

func TestConnectSSE_ReceiveEvents(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)
		for i := 0; i < 3; i++ {
			fmt.Fprintf(w, "data: event-%d\n\n", i)
			flusher.Flush()
		}
	}))
	defer srv.Close()

	conn, err := connectSSE(context.Background(), srv.URL, srv.Client())
	if err != nil {
		t.Fatalf("connectSSE: %v", err)
	}
	defer conn.Close()

	var events []string
	for msg := range conn.Messages {
		events = append(events, string(msg))
		if len(events) >= 3 {
			break
		}
	}
	if len(events) != 3 {
		t.Fatalf("got %d events, want 3", len(events))
	}
	for i, e := range events {
		want := fmt.Sprintf("event-%d", i)
		if e != want {
			t.Errorf("event[%d] = %q, want %q", i, e, want)
		}
	}
}

func TestConnectSSE_SendNotSupported(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "data: x\n\n")
	}))
	defer srv.Close()

	conn, err := connectSSE(context.Background(), srv.URL, srv.Client())
	if err != nil {
		t.Fatalf("connectSSE: %v", err)
	}
	defer conn.Close()

	if err := conn.Send([]byte("test")); err == nil {
		t.Error("expected error from SSE Send")
	}
}

func TestConnectSSE_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	_, err := connectSSE(context.Background(), srv.URL, srv.Client())
	if err == nil {
		t.Fatal("expected error for non-200")
	}
	if !strings.Contains(err.Error(), "status 503") {
		t.Errorf("unexpected error: %v", err)
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
