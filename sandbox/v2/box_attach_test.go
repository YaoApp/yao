package sandbox_test

import (
	"context"
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

	proxyURL, err := box.Proxy(ctx, port, "/")
	if err != nil {
		t.Fatalf("Proxy URL: %v", err)
	}

	host := proxyURL[len("http://"):]
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
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			t.Fatalf("port %d not ready within %v", port, timeout)
		case <-ticker.C:
			conn, err := net.DialTimeout("tcp", host, 2*time.Second)
			if err != nil {
				continue
			}
			conn.Close()
			// TCP reachable — give the service process time to accept
			// application-layer connections (Python ws/sse servers in CI
			// may take 1-3s after the port opens before they're ready).
			time.Sleep(2 * time.Second)
			return
		}
	}
}

func TestAttachWS(t *testing.T) {
	skipIfNoDocker(t)

	img := testImage()
	if img == "alpine:latest" {
		t.Skip("WebSocket test requires tai-sandbox-test image with ws-echo service")
	}

	for _, pc := range testNodes() {
		pc := pc
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForNode(t, &pc)
			box := createTestBox(t, m, pc, func(co *sandbox.CreateOptions) {
				co.Ports = []sandbox.PortMapping{
					{ContainerPort: 9800, HostPort: 0, Protocol: "tcp"},
				}
			})

			waitForPort(t, box, 9800, 30*time.Second)

			var conn *sandbox.ServiceConn
			var err error
			for attempt := 0; attempt < 5; attempt++ {
				conn, err = box.Attach(t.Context(), 9800, sandbox.WithProtocol("ws"), sandbox.WithPath("/"))
				if err == nil {
					break
				}
				time.Sleep(time.Duration(attempt+1) * 500 * time.Millisecond)
			}
			if err != nil {
				t.Fatalf("Attach WS after retries: %v", err)
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
		t.Skip("SSE test requires tai-sandbox-test image with sse-server service")
	}

	for _, pc := range testNodes() {
		pc := pc
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForNode(t, &pc)
			box := createTestBox(t, m, pc, func(co *sandbox.CreateOptions) {
				co.Ports = []sandbox.PortMapping{
					{ContainerPort: 9801, HostPort: 0, Protocol: "tcp"},
				}
			})

			waitForPort(t, box, 9801, 30*time.Second)

			var conn *sandbox.ServiceConn
			var err error
			for attempt := 0; attempt < 5; attempt++ {
				conn, err = box.Attach(t.Context(), 9801, sandbox.WithProtocol("sse"), sandbox.WithPath("/events"))
				if err == nil {
					break
				}
				time.Sleep(time.Duration(attempt+1) * 500 * time.Millisecond)
			}
			if err != nil {
				t.Fatalf("Attach SSE after retries: %v", err)
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
		t.Skip("VNC test requires tai-sandbox-test image with VNC desktop")
	}

	for _, pc := range testNodes() {
		pc := pc
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForNode(t, &pc)
			box := createTestBox(t, m, pc, func(co *sandbox.CreateOptions) {
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
		t.Skip("VNC test requires tai-sandbox-test image with VNC desktop")
	}

	for _, pc := range testNodes() {
		pc := pc
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForNode(t, &pc)
			box := createTestBox(t, m, pc, func(co *sandbox.CreateOptions) {
				co.VNC = true
			})

			ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
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
			var ws *websocket.Conn
			for attempt := 0; attempt < 5; attempt++ {
				var resp *http.Response
				ws, resp, err = dialer.DialContext(ctx, vncURL, http.Header{})
				if err == nil {
					break
				}
				if resp != nil {
					resp.Body.Close()
				}
				time.Sleep(time.Duration(attempt+1) * time.Second)
			}
			if err != nil {
				t.Fatalf("VNC dial after retries: %v", err)
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
			conn, err := net.DialTimeout("tcp", host, 2*time.Second)
			if err != nil {
				continue
			}
			conn.Close()
			// VNC services (Xvfb → fluxbox → x11vnc → websockify) need time
			// after the TCP port is reachable. Give the process chain time to
			// stabilize before attempting the WebSocket handshake.
			time.Sleep(2 * time.Second)
			return
		}
	}
}
