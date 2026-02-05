package vncproxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExtractSandboxID(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "VNC status path",
			path:     "/v1/sandbox/abc123/vnc",
			expected: "abc123",
		},
		{
			name:     "VNC client path",
			path:     "/v1/sandbox/user-chat-123/vnc/client",
			expected: "user-chat-123",
		},
		{
			name:     "VNC websocket path",
			path:     "/v1/sandbox/test-sandbox-id/vnc/ws",
			expected: "test-sandbox-id",
		},
		{
			name:     "Complex ID",
			path:     "/v1/sandbox/user_123-chat_456/vnc",
			expected: "user_123-chat_456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			got := extractSandboxID(req)
			if got != tt.expected {
				t.Errorf("extractSandboxID() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestConfigDefaults(t *testing.T) {
	config := DefaultConfig()

	if config.DockerNetwork != "bridge" {
		t.Errorf("DockerNetwork = %q, want %q", config.DockerNetwork, "bridge")
	}
	if config.ContainerNoVNCPort != 6080 {
		t.Errorf("ContainerNoVNCPort = %d, want %d", config.ContainerNoVNCPort, 6080)
	}
	if config.ContainerVNCPort != 5900 {
		t.Errorf("ContainerVNCPort = %d, want %d", config.ContainerVNCPort, 5900)
	}
	if config.ContainerNamePrefix != "yao-sandbox-" {
		t.Errorf("ContainerNamePrefix = %q, want %q", config.ContainerNamePrefix, "yao-sandbox-")
	}
}

func TestConfigInit(t *testing.T) {
	config := &Config{}
	config.Init()

	// Should have defaults after Init
	if config.DockerNetwork != "bridge" {
		t.Errorf("DockerNetwork = %q, want %q", config.DockerNetwork, "bridge")
	}
	if config.ContainerNoVNCPort != 6080 {
		t.Errorf("ContainerNoVNCPort = %d, want %d", config.ContainerNoVNCPort, 6080)
	}
}

// Integration tests require Docker - skip if not available
func TestProxyCreation(t *testing.T) {
	// This will fail if Docker is not available, which is expected in CI
	proxy, err := NewProxy(nil)
	if err != nil {
		t.Skipf("Skipping test: Docker not available: %v", err)
	}
	defer proxy.Close()

	if proxy.config == nil {
		t.Error("Proxy config should not be nil")
	}
}
