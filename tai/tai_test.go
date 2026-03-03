package tai

import (
	"os"
	"testing"
)

func taiTestHost() string {
	if h := os.Getenv("TAI_TEST_HOST"); h != "" {
		return h
	}
	return "127.0.0.1"
}

func TestParseAddr(t *testing.T) {
	tests := []struct {
		addr       string
		wantScheme string
		wantHost   string
		wantDocker string
		wantErr    bool
	}{
		{"", "docker", "", "", false},
		{"docker:///var/run/docker.sock", "docker", "", "docker:///var/run/docker.sock", false},
		{"docker://192.168.1.50:2375", "docker", "", "docker://192.168.1.50:2375", false},
		{"unix:///var/run/docker.sock", "docker", "", "unix:///var/run/docker.sock", false},
		{"tcp://127.0.0.1:2375", "docker", "", "tcp://127.0.0.1:2375", false},
		{"npipe:////./pipe/docker_engine", "docker", "", "npipe:////./pipe/docker_engine", false},
		{"tai://192.168.1.100", "tai", "192.168.1.100", "", false},
		{"tai://10.0.0.5:9100", "tai", "10.0.0.5", "", false},
		{"tai://", "", "", "", true},
		{"ftp://host", "", "", "", true},
		{"  tai://host  ", "tai", "host", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			scheme, host, dockerAddr, err := parseAddr(tt.addr)
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
		})
	}
}

func TestMergedPorts(t *testing.T) {
	p := mergedPorts(Ports{HTTP: 8888})
	if p.HTTP != 8888 {
		t.Errorf("HTTP = %d, want 8888", p.HTTP)
	}
	if p.GRPC != 9100 {
		t.Errorf("GRPC = %d, want 9100 (default)", p.GRPC)
	}
	if p.Docker != 2375 {
		t.Errorf("Docker = %d, want 2375 (default)", p.Docker)
	}
	if p.VNC != 6080 {
		t.Errorf("VNC = %d, want 6080 (default)", p.VNC)
	}
	if p.Containerd != 2376 {
		t.Errorf("Containerd = %d, want 2376 (default)", p.Containerd)
	}
	if p.K8s != 6443 {
		t.Errorf("K8s = %d, want 6443 (default)", p.K8s)
	}
}

func TestMergedPortsAll(t *testing.T) {
	p := mergedPorts(Ports{GRPC: 1, HTTP: 2, VNC: 3, Docker: 4, Containerd: 5, K8s: 6})
	if p.GRPC != 1 || p.HTTP != 2 || p.VNC != 3 || p.Docker != 4 || p.Containerd != 5 || p.K8s != 6 {
		t.Errorf("unexpected ports: %+v", p)
	}
}

func TestOptions(t *testing.T) {
	cfg := &config{ports: defaultPorts()}

	WithPorts(Ports{HTTP: 9999}).apply(cfg)
	if cfg.ports.HTTP != 9999 {
		t.Errorf("WithPorts: HTTP = %d", cfg.ports.HTTP)
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
	Containerd.apply(cfg)
	if cfg.runtime != Containerd {
		t.Error("Containerd option failed")
	}
	K8s.apply(cfg)
	if cfg.runtime != K8s {
		t.Error("K8s option failed")
	}
}

func TestNewLocal(t *testing.T) {
	c, err := New("")
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
	c, err := New("", WithDataDir(dir))
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

func TestNewRemoteContainerdNotImplemented(t *testing.T) {
	_, err := New("tai://127.0.0.1", Containerd)
	if err == nil {
		t.Error("expected error for unimplemented containerd")
	}
}

func TestNewRemoteK8sNotImplemented(t *testing.T) {
	_, err := New("tai://127.0.0.1", K8s)
	if err == nil {
		t.Error("expected error for unimplemented k8s")
	}
}

func TestNewInvalidScheme(t *testing.T) {
	_, err := New("ftp://host")
	if err == nil {
		t.Error("expected error for ftp://")
	}
}

func TestNewRemoteDocker(t *testing.T) {
	addr := "tai://" + taiTestHost()
	c, err := New(addr)
	if err != nil {
		t.Skipf("Tai not available at %s: %v", addr, err)
	}
	defer c.Close()

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

func TestNewRemoteWithPorts(t *testing.T) {
	addr := "tai://" + taiTestHost()
	c, err := New(addr, WithPorts(Ports{HTTP: 8888}))
	if err != nil {
		t.Skipf("Tai not available at %s: %v", addr, err)
	}
	defer c.Close()
}
