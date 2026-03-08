package tai

import (
	"fmt"
	"os"
	"strconv"
	"testing"
)

func taiTestHost() string {
	if h := os.Getenv("TAI_TEST_HOST"); h != "" {
		return h
	}
	return "127.0.0.1"
}

// taiRemoteAddr returns the tai:// address for remote tests (e.g. TestNewRemoteDocker).
// Uses TAI_TEST_HOST and, when set, TAI_TEST_GRPC_PORT so Tai on non-default port works.
func taiRemoteAddr() string {
	host := taiTestHost()
	if p := os.Getenv("TAI_TEST_GRPC_PORT"); p != "" {
		return "tai://" + host + ":" + p
	}
	return "tai://" + host
}

// taiTestPorts builds a Ports struct from TAI_TEST_*_PORT env vars.
// Only non-zero fields are set so they override ServerInfo-discovered values.
func taiTestPorts() Ports {
	return Ports{
		Docker: envPort("TAI_TEST_DOCKER_PORT", 0),
		HTTP:   envPort("TAI_TEST_HTTP_PORT", 0),
		VNC:    envPort("TAI_TEST_VNC_PORT", 0),
	}
}

func envPort(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			return p
		}
	}
	return fallback
}

func TestParseAddr(t *testing.T) {
	tests := []struct {
		addr         string
		wantScheme   string
		wantHost     string
		wantDocker   string
		wantGRPCPort int
		wantErr      bool
	}{
		{"", "", "", "", 0, true},
		{"local", "docker", "", "", 0, false},
		{"127.0.0.1", "docker", "", "", 0, false},
		{"localhost", "docker", "", "", 0, false},
		{"::1", "docker", "", "", 0, false},
		{"docker:///var/run/docker.sock", "docker", "", "docker:///var/run/docker.sock", 0, false},
		{"docker://192.168.1.50:2375", "docker", "", "docker://192.168.1.50:2375", 0, false},
		{"unix:///var/run/docker.sock", "docker", "", "unix:///var/run/docker.sock", 0, false},
		{"tcp://127.0.0.1:2375", "docker", "", "tcp://127.0.0.1:2375", 0, false},
		{"npipe:////./pipe/docker_engine", "docker", "", "npipe:////./pipe/docker_engine", 0, false},
		{"tai://192.168.1.100", "tai", "192.168.1.100", "", 0, false},
		{"tai://10.0.0.5:9200", "tai", "10.0.0.5", "", 9200, false},
		{"tai://", "", "", "", 0, true},
		{"ftp://host", "", "", "", 0, true},
		{"  tai://host  ", "tai", "host", "", 0, false},
		// Bare non-local host → auto-prepend tai://
		{"192.168.1.50", "tai", "192.168.1.50", "", 0, false},
		{"192.168.1.50:9200", "tai", "192.168.1.50", "", 9200, false},
		{"my-server", "tai", "my-server", "", 0, false},
		{"my-server:9200", "tai", "my-server", "", 9200, false},
	}

	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			scheme, host, dockerAddr, grpcPort, err := parseAddr(tt.addr)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if scheme != tt.wantScheme {
				t.Errorf("scheme = %q, want %q", scheme, tt.wantScheme)
			}
			if host != tt.wantHost {
				t.Errorf("host = %q, want %q", host, tt.wantHost)
			}
			if dockerAddr != tt.wantDocker {
				t.Errorf("dockerAddr = %q, want %q", dockerAddr, tt.wantDocker)
			}
			if grpcPort != tt.wantGRPCPort {
				t.Errorf("grpcPort = %d, want %d", grpcPort, tt.wantGRPCPort)
			}
		})
	}
}

func TestMergedPorts(t *testing.T) {
	p := mergedPorts(Ports{HTTP: 8888})
	if p.HTTP != 8888 {
		t.Errorf("HTTP = %d, want 8888", p.HTTP)
	}
	if p.GRPC != 19100 {
		t.Errorf("GRPC = %d, want 19100 (default)", p.GRPC)
	}
	if p.VNC != 16080 {
		t.Errorf("VNC = %d, want 16080 (default)", p.VNC)
	}
	if p.Docker != 0 {
		t.Errorf("Docker = %d, want 0 (unset)", p.Docker)
	}
	if p.K8s != 0 {
		t.Errorf("K8s = %d, want 0 (unset)", p.K8s)
	}
}

func TestMergedPortsAll(t *testing.T) {
	p := mergedPorts(Ports{GRPC: 1, HTTP: 2, VNC: 3, Docker: 4, K8s: 5})
	if p.GRPC != 1 || p.HTTP != 2 || p.VNC != 3 || p.Docker != 4 || p.K8s != 5 {
		t.Errorf("unexpected ports: %+v", p)
	}
}

func TestOptions(t *testing.T) {
	cfg := &config{ports: defaultPorts()}

	WithPorts(Ports{HTTP: 9999}).apply(cfg)
	if cfg.ports.HTTP != 9999 {
		t.Errorf("WithPorts: HTTP = %d", cfg.ports.HTTP)
	}
	if cfg.userPorts.HTTP != 9999 {
		t.Errorf("WithPorts: userPorts.HTTP = %d", cfg.userPorts.HTTP)
	}

	WithDataDir("/data").apply(cfg)
	if cfg.dataDir != "/data" {
		t.Errorf("WithDataDir = %q", cfg.dataDir)
	}

	WithHTTPClient(nil).apply(cfg)

	Docker.apply(cfg)
	if cfg.runtime != Docker {
		t.Error("Docker option failed")
	}
	K8s.apply(cfg)
	if cfg.runtime != K8s {
		t.Error("K8s option failed")
	}
}

func TestNewEmptyAddr(t *testing.T) {
	_, err := New("")
	if err == nil {
		t.Error("expected error for empty addr")
	}
}

func TestNewLocal(t *testing.T) {
	c, err := New("local")
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer c.Close()

	if !c.IsLocal() {
		t.Error("expected IsLocal = true")
	}
	if c.Volume() == nil {
		t.Error("Volume should not be nil")
	}
	if c.Sandbox() == nil {
		t.Error("Sandbox should not be nil")
	}
	if c.Proxy() == nil {
		t.Error("Proxy should not be nil")
	}
	if c.VNC() == nil {
		t.Error("VNC should not be nil")
	}

	// Test Workspace accessor
	ws := c.Workspace("test-session")
	if ws == nil {
		t.Error("Workspace should not be nil")
	}
}

func TestNewLocalWithDataDir(t *testing.T) {
	dir := t.TempDir()
	c, err := New("local", WithDataDir(dir))
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer c.Close()

	if !c.IsLocal() {
		t.Error("expected IsLocal = true")
	}
}

func TestNewLocalExplicitSocket(t *testing.T) {
	c, err := New("unix:///var/run/docker.sock")
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer c.Close()

	if !c.IsLocal() {
		t.Error("expected IsLocal = true for unix socket")
	}
}

func TestNewRemoteK8s(t *testing.T) {
	host := os.Getenv("TAI_TEST_K8S_HOST")
	kubeconfig := os.Getenv("TAI_TEST_KUBECONFIG")
	if host == "" || kubeconfig == "" {
		t.Skip("TAI_TEST_K8S_HOST or TAI_TEST_KUBECONFIG not set")
	}

	grpcPort := envPort("TAI_TEST_K8S_GRPC_PORT", envPort("TAI_TEST_GRPC_PORT", 19100))
	ports := Ports{
		K8s:  envPort("TAI_TEST_K8S_PORT", 16443),
		GRPC: grpcPort,
		HTTP: envPort("TAI_TEST_K8S_HTTP_PORT", 8099),
		VNC:  envPort("TAI_TEST_K8S_VNC_PORT", 16080),
	}

	c, err := New(fmt.Sprintf("tai://%s:%d", host, grpcPort), K8s,
		WithPorts(ports),
		WithKubeConfig(kubeconfig),
		WithNamespace("default"),
	)
	if err != nil {
		t.Skipf("Tai K8s not available: %v", err)
	}
	defer c.Close()

	if c.IsLocal() {
		t.Error("expected IsLocal = false")
	}
	if c.Sandbox() == nil {
		t.Error("Sandbox should not be nil")
	}
}

func TestNewRemoteK8sMissingKubeConfig(t *testing.T) {
	_, err := New("tai://127.0.0.1", K8s)
	if err == nil {
		t.Error("expected error for missing kubeconfig")
	}
}

func TestWithKubeConfigAndNamespace(t *testing.T) {
	cfg := &config{ports: defaultPorts()}
	WithKubeConfig("/path/to/kubeconfig").apply(cfg)
	if cfg.kubeConfig != "/path/to/kubeconfig" {
		t.Errorf("WithKubeConfig = %q", cfg.kubeConfig)
	}
	WithNamespace("test-ns").apply(cfg)
	if cfg.namespace != "test-ns" {
		t.Errorf("WithNamespace = %q", cfg.namespace)
	}
}

func TestNewInvalidScheme(t *testing.T) {
	_, err := New("ftp://host")
	if err == nil {
		t.Error("expected error for ftp://")
	}
}

func TestNewRemoteDocker(t *testing.T) {
	addr := taiRemoteAddr()
	ports := taiTestPorts()
	c, err := New(addr, WithPorts(ports))
	if err != nil {
		t.Skipf("Tai not available at %s: %v", addr, err)
	}
	defer c.Close()

	t.Logf("remote docker: addr=%s ports=%+v", addr, c.ports)

	if c.IsLocal() {
		t.Error("expected IsLocal = false for tai://")
	}
	if c.Volume() == nil {
		t.Error("Volume should not be nil")
	}
	if c.Sandbox() == nil {
		t.Error("Sandbox should not be nil")
	}
	if c.Proxy() == nil {
		t.Error("Proxy should not be nil")
	}
	if c.VNC() == nil {
		t.Error("VNC should not be nil")
	}
	ws := c.Workspace("test")
	if ws == nil {
		t.Error("Workspace should not be nil")
	}
}

func TestDiscoverPorts(t *testing.T) {
	addr := taiRemoteAddr()
	c, err := New(addr)
	if err != nil {
		t.Skipf("Tai not available at %s: %v", addr, err)
	}
	defer c.Close()

	t.Logf("client resolved: GRPC=%d HTTP=%d VNC=%d Docker=%d K8s=%d",
		c.ports.GRPC, c.ports.HTTP, c.ports.VNC, c.ports.Docker, c.ports.K8s)

	if c.ports.GRPC == 0 {
		t.Error("GRPC port should be discovered (non-zero)")
	}
	if c.ports.HTTP == 0 {
		t.Error("HTTP port should be discovered (non-zero)")
	}
}

func TestDiscoverPortsWithUserOverride(t *testing.T) {
	addr := taiRemoteAddr()
	c, err := New(addr, WithPorts(Ports{HTTP: 9999}))
	if err != nil {
		t.Skipf("Tai not available at %s: %v", addr, err)
	}
	defer c.Close()

	if c.ports.HTTP != 9999 {
		t.Errorf("HTTP = %d, want 9999 (user override should take precedence)", c.ports.HTTP)
	}
	if c.ports.GRPC == 0 {
		t.Error("GRPC port should still be discovered (non-zero)")
	}
	t.Logf("ports: GRPC=%d HTTP=%d(user) VNC=%d Docker=%d",
		c.ports.GRPC, c.ports.HTTP, c.ports.VNC, c.ports.Docker)
}
