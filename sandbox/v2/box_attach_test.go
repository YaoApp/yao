package sandbox_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	sandbox "github.com/yaoapp/yao/sandbox/v2"
)

func waitForPort(t *testing.T, box *sandbox.Box, port int, timeout time.Duration) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	url, err := box.Proxy(ctx, port, "/")
	if err != nil {
		t.Fatalf("Proxy URL: %v", err)
	}

	host := url[len("http://"):]
	if i := len(host) - 1; host[i] == '/' {
		host = host[:i]
	}
	for i := 0; i < len(host); i++ {
		if host[i] == '/' {
			host = host[:i]
			break
		}
	}

	deadline := time.After(timeout)
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			t.Fatalf("port %d not ready within %v", port, timeout)
		case <-ticker.C:
			conn, err := net.DialTimeout("tcp", host, time.Second)
			if err == nil {
				conn.Close()
				time.Sleep(200 * time.Millisecond)
				return
			}
			fmt.Printf("waiting for %s: %v\n", host, err)
		}
	}
}

func TestAttachWS(t *testing.T) {
	skipIfNoDocker(t)

	img := testImage()
	if img == "alpine:latest" {
		t.Skip("WebSocket test requires sandbox-v2-test image with ws-echo service")
	}

	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			box := createTestBox(t, m, func(co *sandbox.CreateOptions) {
				co.Ports = []sandbox.PortMapping{
					{ContainerPort: 9800, HostPort: 0, Protocol: "tcp"},
				}
			})

			waitForPort(t, box, 9800, 30*time.Second)

			conn, err := box.Attach(t.Context(), 9800, sandbox.WithProtocol("ws"), sandbox.WithPath("/"))
			if err != nil {
				t.Fatalf("Attach WS: %v", err)
			}
			defer conn.Close()

			if err := conn.Write([]byte("ping")); err != nil {
				t.Fatalf("Write: %v", err)
			}

			msg, err := conn.Read()
			if err != nil {
				t.Fatalf("Read: %v", err)
			}
			if string(msg) != "ping" {
				t.Errorf("echo = %q, want %q", string(msg), "ping")
			}
		})
	}
}

func TestAttachSSE(t *testing.T) {
	skipIfNoDocker(t)

	img := testImage()
	if img == "alpine:latest" {
		t.Skip("SSE test requires sandbox-v2-test image with sse-server service")
	}

	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			box := createTestBox(t, m, func(co *sandbox.CreateOptions) {
				co.Ports = []sandbox.PortMapping{
					{ContainerPort: 9801, HostPort: 0, Protocol: "tcp"},
				}
			})

			waitForPort(t, box, 9801, 30*time.Second)

			conn, err := box.Attach(t.Context(), 9801, sandbox.WithProtocol("sse"), sandbox.WithPath("/events"))
			if err != nil {
				t.Fatalf("Attach SSE: %v", err)
			}
			defer conn.Close()

			count := 0
			for event := range conn.Events {
				if len(event) > 0 {
					count++
				}
				if count >= 2 {
					break
				}
			}
			if count < 2 {
				t.Errorf("received %d events, want >= 2", count)
			}
		})
	}
}

func TestVNCURL(t *testing.T) {
	skipIfNoDocker(t)

	img := testImage()
	if img == "alpine:latest" {
		t.Skip("VNC test requires sandbox-v2-test image with VNC desktop")
	}

	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			box := createTestBox(t, m, func(co *sandbox.CreateOptions) {
				co.VNC = true
			})

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			url, err := box.VNC(ctx)
			if err != nil {
				t.Fatalf("VNC URL: %v", err)
			}
			if !strings.HasPrefix(url, "ws://") {
				t.Fatalf("VNC URL = %q, want ws:// prefix", url)
			}
			t.Logf("VNC URL: %s", url)
		})
	}
}

func TestVNCConnect(t *testing.T) {
	skipIfNoDocker(t)

	img := testImage()
	if img == "alpine:latest" {
		t.Skip("VNC test requires sandbox-v2-test image with VNC desktop")
	}

	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			box := createTestBox(t, m, func(co *sandbox.CreateOptions) {
				co.VNC = true
			})

			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()

			vncURL, err := box.VNC(ctx)
			if err != nil {
				t.Fatalf("VNC URL: %v", err)
			}
			t.Logf("VNC URL: %s", vncURL)

			waitForWSEndpoint(t, vncURL, 30*time.Second)

			dialer := websocket.Dialer{
				Subprotocols:     []string{"binary"},
				HandshakeTimeout: 10 * time.Second,
			}
			ws, resp, err := dialer.DialContext(ctx, vncURL, http.Header{})
			if err != nil {
				extra := ""
				if resp != nil {
					extra = fmt.Sprintf(" (status %d)", resp.StatusCode)
				}
				t.Fatalf("VNC dial: %v%s", err, extra)
			}
			defer ws.Close()

			ws.SetReadDeadline(time.Now().Add(10 * time.Second))
			_, msg, err := ws.ReadMessage()
			if err != nil {
				t.Fatalf("VNC read: %v", err)
			}
			if !strings.HasPrefix(string(msg), "RFB ") {
				t.Fatalf("VNC banner = %q, want RFB prefix", string(msg))
			}
			t.Logf("VNC banner: %s", strings.TrimSpace(string(msg)))
		})
	}
}

func waitForWSEndpoint(t *testing.T, wsURL string, timeout time.Duration) {
	t.Helper()
	httpURL := "http" + strings.TrimPrefix(wsURL, "ws")
	if idx := strings.LastIndex(httpURL, "/ws"); idx > 0 {
		httpURL = httpURL[:idx]
	}

	host := strings.TrimPrefix(httpURL, "http://")
	if i := strings.Index(host, "/"); i > 0 {
		host = host[:i]
	}

	deadline := time.After(timeout)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			t.Fatalf("VNC endpoint %s not ready within %v", host, timeout)
		case <-ticker.C:
			conn, err := net.DialTimeout("tcp", host, time.Second)
			if err == nil {
				conn.Close()
				time.Sleep(500 * time.Millisecond)
				return
			}
		}
	}
}
