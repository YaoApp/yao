package sandbox_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

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
