//go:build unit

package webproxy

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"strings"
	"testing"
	"time"
)

// TestBindingTunnelProxy_ReverseProxy_Routes verifies that non-WebSocket
// requests are handled by the reverseProxy field when Mode == ModeTaiProxy.
func TestBindingTunnelProxy_ReverseProxy_Routes(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello from backend"))
	}))
	defer backend.Close()

	// Parse the backend address for dialing
	backendAddr := backend.Listener.Addr().String()

	dialer := &tunnelDialer{warmCh: make(chan net.Conn, 1)}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return net.DialTimeout("tcp", backendAddr, 2*time.Second)
		},
		MaxIdleConns:        6,
		MaxIdleConnsPerHost: 6,
	}
	rp := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = "http"
			req.URL.Host = backendAddr
			stripProxyCookie(req)
		},
		Transport:     transport,
		FlushInterval: -1,
	}

	binding := &Binding{
		Mode:         ModeTaiProxy,
		transport:    transport,
		reverseProxy: rp,
		dialer:       dialer,
	}

	handler := binding.buildHandler()

	req := httptest.NewRequest("GET", "/test-path", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if body != "hello from backend" {
		t.Fatalf("unexpected body: %q", body)
	}
}

// TestBindingTunnelProxy_ErrorHandler_502 verifies that when the tunnel
// transport fails, the ErrorHandler returns 502.
func TestBindingTunnelProxy_ErrorHandler_502(t *testing.T) {
	dialer := &tunnelDialer{warmCh: make(chan net.Conn, 1)}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return nil, io.ErrUnexpectedEOF
		},
	}
	rp := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = "http"
			req.URL.Host = "tunnel"
			stripProxyCookie(req)
		},
		Transport: transport,
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
		},
	}

	binding := &Binding{
		Mode:         ModeTaiProxy,
		transport:    transport,
		reverseProxy: rp,
		dialer:       dialer,
	}

	handler := binding.buildHandler()
	req := httptest.NewRequest("GET", "/fail", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", rec.Code)
	}
}

// TestBindingTunnelProxy_WebSocket_UsesHijack verifies that WebSocket
// upgrade requests in ModeTaiProxy are routed to handleTunnelWebSocket
// (which attempts hijack) rather than the ReverseProxy.
func TestBindingTunnelProxy_WebSocket_UsesHijack(t *testing.T) {
	dialer := &tunnelDialer{warmCh: make(chan net.Conn, 1)}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			t.Fatal("ReverseProxy should not be called for WebSocket")
			return nil, nil
		},
	}
	rp := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = "http"
			req.URL.Host = "tunnel"
		},
		Transport: transport,
	}

	binding := &Binding{
		Mode:         ModeTaiProxy,
		TaiID:        "test-tai",
		ContainerID:  "test-container",
		TargetPort:   8080,
		transport:    transport,
		reverseProxy: rp,
		dialer:       dialer,
	}

	handler := binding.buildHandler()

	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// httptest.ResponseRecorder doesn't support Hijack, so handleTunnelWebSocket
	// will return 500 "hijack not supported" — this proves it took the WS path
	// instead of going through reverseProxy (which would have called t.Fatal).
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 (hijack not supported), got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "hijack not supported") {
		t.Fatalf("expected hijack error, got: %s", rec.Body.String())
	}
}
