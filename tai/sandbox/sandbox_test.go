package sandbox

import (
	"context"
	"os"
	"testing"
	"time"
)

func taiTestDocker() string {
	if addr := os.Getenv("TAI_TEST_DOCKER"); addr != "" {
		return addr
	}
	return "tcp://127.0.0.1:2375"
}

func TestHelpers(t *testing.T) {
	t.Run("envSlice", func(t *testing.T) {
		if got := envSlice(nil); got != nil {
			t.Errorf("envSlice(nil) = %v", got)
		}
		s := envSlice(map[string]string{"A": "1", "B": "2"})
		if len(s) != 2 {
			t.Errorf("len = %d, want 2", len(s))
		}
	})

	t.Run("proto", func(t *testing.T) {
		if got := proto(""); got != "tcp" {
			t.Errorf("proto empty = %q", got)
		}
		if got := proto("udp"); got != "udp" {
			t.Errorf("proto udp = %q", got)
		}
	})

	t.Run("hostIP", func(t *testing.T) {
		if got := hostIP(""); got != "127.0.0.1" {
			t.Errorf("hostIP empty = %q", got)
		}
		if got := hostIP("10.0.0.1"); got != "10.0.0.1" {
			t.Errorf("hostIP explicit = %q", got)
		}
	})
}

func TestLocalSandbox(t *testing.T) {
	sb, err := NewLocal("")
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer sb.Close()

	ctx := context.Background()
	var containerID string

	t.Run("Create", func(t *testing.T) {
		id, err := sb.Create(ctx, CreateOptions{
			Name:  "tai-sdk-test",
			Image: "alpine:latest",
			Cmd:   []string{"sleep", "30"},
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if id == "" {
			t.Fatal("expected non-empty ID")
		}
		containerID = id
	})

	t.Run("Start", func(t *testing.T) {
		if containerID == "" {
			t.Skip("no container")
		}
		if err := sb.Start(ctx, containerID); err != nil {
			t.Fatalf("Start: %v", err)
		}
	})

	t.Run("Inspect", func(t *testing.T) {
		if containerID == "" {
			t.Skip("no container")
		}
		info, err := sb.Inspect(ctx, containerID)
		if err != nil {
			t.Fatalf("Inspect: %v", err)
		}
		if info.Status != "running" {
			t.Errorf("status = %q, want running", info.Status)
		}
		if info.Image != "alpine:latest" {
			t.Errorf("image = %q", info.Image)
		}
	})

	t.Run("Exec", func(t *testing.T) {
		if containerID == "" {
			t.Skip("no container")
		}
		result, err := sb.Exec(ctx, containerID, []string{"echo", "hello"}, ExecOptions{})
		if err != nil {
			t.Fatalf("Exec: %v", err)
		}
		if result.ExitCode != 0 {
			t.Errorf("exitCode = %d", result.ExitCode)
		}
		if result.Stdout != "hello\n" {
			t.Errorf("stdout = %q, want %q", result.Stdout, "hello\n")
		}
	})

	t.Run("List", func(t *testing.T) {
		if containerID == "" {
			t.Skip("no container")
		}
		containers, err := sb.List(ctx, ListOptions{All: true})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		found := false
		for _, c := range containers {
			if c.ID == containerID {
				found = true
				break
			}
		}
		if !found {
			t.Error("container not found in list")
		}
	})

	t.Run("Stop", func(t *testing.T) {
		if containerID == "" {
			t.Skip("no container")
		}
		if err := sb.Stop(ctx, containerID, 5*time.Second); err != nil {
			t.Fatalf("Stop: %v", err)
		}
	})

	t.Run("Remove", func(t *testing.T) {
		if containerID == "" {
			t.Skip("no container")
		}
		if err := sb.Remove(ctx, containerID, true); err != nil {
			t.Fatalf("Remove: %v", err)
		}
	})
}

func TestLocalCreateWithPorts(t *testing.T) {
	sb, err := NewLocal("")
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer sb.Close()

	ctx := context.Background()
	id, err := sb.Create(ctx, CreateOptions{
		Name:   "tai-sdk-port-test",
		Image:  "alpine:latest",
		Cmd:    []string{"sleep", "5"},
		Memory: 64 * 1024 * 1024,
		CPUs:   0.5,
		Ports: []PortMapping{
			{ContainerPort: 8080, HostPort: 0, Protocol: "tcp"},
		},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer sb.Remove(ctx, id, true)

	if err := sb.Start(ctx, id); err != nil {
		t.Fatalf("Start: %v", err)
	}

	info, err := sb.Inspect(ctx, id)
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}

	found := false
	for _, p := range info.Ports {
		if p.ContainerPort == 8080 {
			found = true
			if p.HostPort == 0 {
				t.Error("HostPort should be resolved")
			}
		}
	}
	if !found {
		t.Error("port 8080 not in Ports")
	}
}

func TestLocalCreateWithVNC(t *testing.T) {
	sb, err := NewLocal("")
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer sb.Close()

	ctx := context.Background()
	id, err := sb.Create(ctx, CreateOptions{
		Name:   "tai-sdk-vnc-test",
		Image:  "alpine:latest",
		Cmd:    []string{"sleep", "5"},
		Memory: 512 * 1024 * 1024,
		VNC:    true,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer sb.Remove(ctx, id, true)
}

func TestLocalCreateWithEnvAndWorkDir(t *testing.T) {
	sb, err := NewLocal("")
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer sb.Close()

	ctx := context.Background()
	id, err := sb.Create(ctx, CreateOptions{
		Name:       "tai-sdk-env-test",
		Image:      "alpine:latest",
		Cmd:        []string{"sleep", "5"},
		WorkingDir: "/tmp",
		Env:        map[string]string{"FOO": "bar"},
		Binds:      []string{},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer sb.Remove(ctx, id, true)

	if err := sb.Start(ctx, id); err != nil {
		t.Fatalf("Start: %v", err)
	}
	result, err := sb.Exec(ctx, id, []string{"printenv", "FOO"}, ExecOptions{WorkDir: "/tmp"})
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if result.Stdout != "bar\n" {
		t.Errorf("FOO = %q, want %q", result.Stdout, "bar\n")
	}
}

func TestDockerSandboxViaTai(t *testing.T) {
	addr := taiTestDocker()
	sb, err := NewDocker(addr)
	if err != nil {
		t.Skipf("Tai Docker proxy not available at %s: %v", addr, err)
	}
	defer sb.Close()

	ctx := context.Background()

	id, err := sb.Create(ctx, CreateOptions{
		Name:  "tai-docker-proxy-test",
		Image: "alpine:latest",
		Cmd:   []string{"sleep", "10"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer sb.Remove(ctx, id, true)

	if err := sb.Start(ctx, id); err != nil {
		t.Fatalf("Start: %v", err)
	}

	info, err := sb.Inspect(ctx, id)
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if info.Status != "running" {
		t.Errorf("status = %q", info.Status)
	}

	result, err := sb.Exec(ctx, id, []string{"echo", "via-tai"}, ExecOptions{})
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if result.Stdout != "via-tai\n" {
		t.Errorf("stdout = %q", result.Stdout)
	}

	containers, err := sb.List(ctx, ListOptions{All: true})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	found := false
	for _, c := range containers {
		if c.ID == id {
			found = true
		}
	}
	if !found {
		t.Error("container not in list")
	}

	if err := sb.Stop(ctx, id, 5*time.Second); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}

func TestListWithLabels(t *testing.T) {
	sb, err := NewLocal("")
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer sb.Close()

	// List with non-matching labels should return empty
	result, err := sb.List(context.Background(), ListOptions{
		Labels: map[string]string{"tai-test-nonexist": "true"},
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0, got %d", len(result))
	}
}

func TestNewLocalInvalidAddr(t *testing.T) {
	_, err := NewLocal("tcp://192.168.254.254:1")
	if err == nil {
		t.Error("expected error for unreachable Docker")
	}
}

func TestPortStr(t *testing.T) {
	if got := portStr(0); got != "" {
		t.Errorf("portStr(0) = %q", got)
	}
	if got := portStr(8080); got != "8080" {
		t.Errorf("portStr(8080) = %q", got)
	}
}
