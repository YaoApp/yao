package runtime

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
)

func taiTestDocker() string {
	if addr := os.Getenv("TAI_TEST_DOCKER"); addr != "" {
		return addr
	}
	return "tcp://127.0.0.1:2375"
}

func taiTestK8sHost() string { return os.Getenv("TAI_TEST_K8S_HOST") }
func taiTestK8sPort() string { return os.Getenv("TAI_TEST_K8S_PORT") }

func taiTestKubeConfig() string { return os.Getenv("TAI_TEST_KUBECONFIG") }

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

func TestLocalRuntime(t *testing.T) {
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

func TestDockerRuntimeViaTai(t *testing.T) {
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

func TestK8sRuntime(t *testing.T) {
	host := taiTestK8sHost()
	port := taiTestK8sPort()
	kubeconfig := taiTestKubeConfig()
	if host == "" || port == "" || kubeconfig == "" {
		t.Skip("TAI_TEST_K8S_HOST, TAI_TEST_K8S_PORT, or TAI_TEST_KUBECONFIG not set")
	}

	addr := host + ":" + port
	sb, err := NewK8s(addr, K8sOption{
		Namespace:  "default",
		KubeConfig: kubeconfig,
	})
	if err != nil {
		t.Skipf("K8s not available at %s: %v", addr, err)
	}
	defer sb.Close()

	ctx := context.Background()
	var podName string

	t.Run("Create", func(t *testing.T) {
		id, err := sb.Create(ctx, CreateOptions{
			Name:  "tai-k8s-test",
			Image: "alpine:latest",
			Cmd:   []string{"sleep", "60"},
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if id == "" {
			t.Fatal("expected non-empty name")
		}
		podName = id
	})

	t.Run("Start", func(t *testing.T) {
		if podName == "" {
			t.Skip("no pod")
		}
		if err := sb.Start(ctx, podName); err != nil {
			t.Fatalf("Start (wait for Running): %v", err)
		}
	})

	t.Run("Inspect", func(t *testing.T) {
		if podName == "" {
			t.Skip("no pod")
		}
		info, err := sb.Inspect(ctx, podName)
		if err != nil {
			t.Fatalf("Inspect: %v", err)
		}
		if info.Status != "Running" {
			t.Errorf("status = %q, want Running", info.Status)
		}
		if info.Image != "alpine:latest" {
			t.Errorf("image = %q", info.Image)
		}
	})

	t.Run("Exec", func(t *testing.T) {
		if podName == "" {
			t.Skip("no pod")
		}
		result, err := sb.Exec(ctx, podName, []string{"echo", "k8s-hello"}, ExecOptions{})
		if err != nil {
			t.Fatalf("Exec: %v", err)
		}
		if result.ExitCode != 0 {
			t.Errorf("exitCode = %d", result.ExitCode)
		}
		if result.Stdout != "k8s-hello\n" {
			t.Errorf("stdout = %q", result.Stdout)
		}
	})

	t.Run("List", func(t *testing.T) {
		if podName == "" {
			t.Skip("no pod")
		}
		pods, err := sb.List(ctx, ListOptions{})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		found := false
		for _, p := range pods {
			if p.Name == podName {
				found = true
				break
			}
		}
		if !found {
			t.Error("pod not found in list")
		}
	})

	t.Run("Remove", func(t *testing.T) {
		if podName == "" {
			t.Skip("no pod")
		}
		if err := sb.Remove(ctx, podName, true); err != nil {
			t.Fatalf("Remove: %v", err)
		}
	})
}

func TestNewK8sMissingKubeConfig(t *testing.T) {
	_, err := NewK8s("127.0.0.1:6443")
	if err == nil {
		t.Error("expected error for missing kubeconfig")
	}
}

func TestNewK8sBadKubeConfig(t *testing.T) {
	_, err := NewK8s("127.0.0.1:6443", K8sOption{KubeConfig: "/nonexistent/kubeconfig.yml"})
	if err == nil {
		t.Error("expected error for bad kubeconfig path")
	}
}

func TestK8sBuildResources(t *testing.T) {
	r := buildResources(512*1024*1024, 1.5)
	mem := r.Limits[corev1.ResourceMemory]
	if mem.Value() != 512*1024*1024 {
		t.Errorf("memory = %d, want %d", mem.Value(), 512*1024*1024)
	}
	cpu := r.Limits[corev1.ResourceCPU]
	if cpu.MilliValue() != 1500 {
		t.Errorf("cpu = %dm, want 1500m", cpu.MilliValue())
	}
}

func TestK8sBuildResourcesPartial(t *testing.T) {
	r := buildResources(0, 0.5)
	if _, ok := r.Limits[corev1.ResourceMemory]; ok {
		t.Error("memory should not be set when 0")
	}
	cpu := r.Limits[corev1.ResourceCPU]
	if cpu.MilliValue() != 500 {
		t.Errorf("cpu = %dm, want 500m", cpu.MilliValue())
	}
}

func TestK8sRuntimeStopAndRemove(t *testing.T) {
	host := taiTestK8sHost()
	port := taiTestK8sPort()
	kubeconfig := taiTestKubeConfig()
	if host == "" || port == "" || kubeconfig == "" {
		t.Skip("TAI_TEST_K8S_HOST, TAI_TEST_K8S_PORT, or TAI_TEST_KUBECONFIG not set")
	}

	addr := host + ":" + port
	sb, err := NewK8s(addr, K8sOption{
		Namespace:  "default",
		KubeConfig: kubeconfig,
	})
	if err != nil {
		t.Skipf("K8s not available: %v", err)
	}
	defer sb.Close()

	ctx := context.Background()

	id, err := sb.Create(ctx, CreateOptions{
		Name:  "tai-k8s-stop-test",
		Image: "alpine:latest",
		Cmd:   []string{"sleep", "60"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := sb.Start(ctx, id); err != nil {
		t.Fatalf("Start: %v", err)
	}

	if err := sb.Stop(ctx, id, 5*time.Second); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	// Remove should succeed even if already deleted by Stop
	if err := sb.Remove(ctx, id, true); err != nil {
		t.Logf("Remove after Stop: %v (expected if already deleted)", err)
	}
}

func TestK8sCreateWithResources(t *testing.T) {
	host := taiTestK8sHost()
	port := taiTestK8sPort()
	kubeconfig := taiTestKubeConfig()
	if host == "" || port == "" || kubeconfig == "" {
		t.Skip("TAI_TEST_K8S_HOST, TAI_TEST_K8S_PORT, or TAI_TEST_KUBECONFIG not set")
	}

	addr := host + ":" + port
	sb, err := NewK8s(addr, K8sOption{
		Namespace:  "default",
		KubeConfig: kubeconfig,
	})
	if err != nil {
		t.Skipf("K8s not available: %v", err)
	}
	defer sb.Close()

	ctx := context.Background()
	id, err := sb.Create(ctx, CreateOptions{
		Name:   "tai-k8s-res-test",
		Image:  "alpine:latest",
		Cmd:    []string{"sleep", "10"},
		Memory: 64 * 1024 * 1024,
		CPUs:   0.5,
		Env:    map[string]string{"FOO": "bar"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer sb.Remove(ctx, id, true)

	if err := sb.Start(ctx, id); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Exec with WorkDir and Env
	result, err := sb.Exec(ctx, id, []string{"echo", "hi"}, ExecOptions{
		WorkDir: "/tmp",
		Env:     map[string]string{"BAR": "baz"},
	})
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("exitCode = %d", result.ExitCode)
	}
}

func TestK8sRemoveNonExistent(t *testing.T) {
	host := taiTestK8sHost()
	port := taiTestK8sPort()
	kubeconfig := taiTestKubeConfig()
	if host == "" || port == "" || kubeconfig == "" {
		t.Skip("TAI_TEST_K8S_HOST, TAI_TEST_K8S_PORT, or TAI_TEST_KUBECONFIG not set")
	}

	addr := host + ":" + port
	sb, err := NewK8s(addr, K8sOption{
		Namespace:  "default",
		KubeConfig: kubeconfig,
	})
	if err != nil {
		t.Skipf("K8s not available: %v", err)
	}
	defer sb.Close()

	// Remove non-existent should not error
	err = sb.Remove(context.Background(), "nonexistent-pod-12345", false)
	if err != nil {
		t.Errorf("Remove non-existent should return nil, got: %v", err)
	}
}

func TestNewK8sRelativeKubeConfig(t *testing.T) {
	kubeconfig := taiTestKubeConfig()
	if kubeconfig == "" {
		t.Skip("TAI_TEST_KUBECONFIG not set")
	}

	// NewK8s with empty addr should still work (uses kubeconfig's server)
	_, err := NewK8s("", K8sOption{
		KubeConfig: kubeconfig,
	})
	if err != nil {
		t.Skipf("K8s not available: %v", err)
	}
}

func TestCreateWithLabels(t *testing.T) {
	sb, err := NewLocal("")
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer sb.Close()

	ctx := context.Background()
	labels := map[string]string{
		"sandbox-id":    "test-123",
		"sandbox-owner": "user1",
	}

	id, err := sb.Create(ctx, CreateOptions{
		Name:   "tai-label-test",
		Image:  "alpine:latest",
		Cmd:    []string{"sleep", "10"},
		Labels: labels,
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
	for k, v := range labels {
		if info.Labels[k] != v {
			t.Errorf("label %q = %q, want %q", k, info.Labels[k], v)
		}
	}

	listed, err := sb.List(ctx, ListOptions{
		Labels: map[string]string{"sandbox-id": "test-123"},
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	found := false
	for _, c := range listed {
		if c.ID == id {
			found = true
			if c.Labels["sandbox-owner"] != "user1" {
				t.Errorf("list labels missing sandbox-owner")
			}
		}
	}
	if !found {
		t.Error("labeled container not found in filtered list")
	}
}

func TestCreateWithUser(t *testing.T) {
	sb, err := NewLocal("")
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer sb.Close()

	ctx := context.Background()
	id, err := sb.Create(ctx, CreateOptions{
		Name:  "tai-user-test",
		Image: "alpine:latest",
		Cmd:   []string{"sleep", "10"},
		User:  "1000:1000",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer sb.Remove(ctx, id, true)

	if err := sb.Start(ctx, id); err != nil {
		t.Fatalf("Start: %v", err)
	}

	result, err := sb.Exec(ctx, id, []string{"id", "-u"}, ExecOptions{})
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if result.Stdout != "1000\n" {
		t.Errorf("user id = %q, want %q", result.Stdout, "1000\n")
	}
}

func TestExecStream_ShortCommand(t *testing.T) {
	sb, err := NewLocal("")
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer sb.Close()

	ctx := context.Background()
	id, err := sb.Create(ctx, CreateOptions{
		Name:  "tai-stream-short",
		Image: "alpine:latest",
		Cmd:   []string{"sleep", "30"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer sb.Remove(ctx, id, true)

	if err := sb.Start(ctx, id); err != nil {
		t.Fatalf("Start: %v", err)
	}

	stream, err := sb.ExecStream(ctx, id, []string{"echo", "hello-stream"}, ExecOptions{})
	if err != nil {
		t.Fatalf("ExecStream: %v", err)
	}

	out, err := io.ReadAll(stream.Stdout)
	if err != nil {
		t.Fatalf("ReadAll stdout: %v", err)
	}
	if string(out) != "hello-stream\n" {
		t.Errorf("stdout = %q, want %q", string(out), "hello-stream\n")
	}

	code, err := stream.Wait()
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
}

func TestExecStream_Stdin(t *testing.T) {
	sb, err := NewLocal("")
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer sb.Close()

	ctx := context.Background()
	id, err := sb.Create(ctx, CreateOptions{
		Name:  "tai-stream-stdin",
		Image: "alpine:latest",
		Cmd:   []string{"sleep", "30"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer sb.Remove(ctx, id, true)

	if err := sb.Start(ctx, id); err != nil {
		t.Fatalf("Start: %v", err)
	}

	stream, err := sb.ExecStream(ctx, id, []string{"cat"}, ExecOptions{})
	if err != nil {
		t.Fatalf("ExecStream: %v", err)
	}

	_, err = stream.Stdin.Write([]byte("from-stdin\n"))
	if err != nil {
		t.Fatalf("Write stdin: %v", err)
	}
	stream.Stdin.Close()

	out, err := io.ReadAll(stream.Stdout)
	if err != nil {
		t.Fatalf("ReadAll stdout: %v", err)
	}
	if string(out) != "from-stdin\n" {
		t.Errorf("stdout = %q, want %q", string(out), "from-stdin\n")
	}

	code, err := stream.Wait()
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
}

func TestExecStream_ExitCode(t *testing.T) {
	sb, err := NewLocal("")
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer sb.Close()

	ctx := context.Background()
	id, err := sb.Create(ctx, CreateOptions{
		Name:  "tai-stream-exit",
		Image: "alpine:latest",
		Cmd:   []string{"sleep", "30"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer sb.Remove(ctx, id, true)

	if err := sb.Start(ctx, id); err != nil {
		t.Fatalf("Start: %v", err)
	}

	stream, err := sb.ExecStream(ctx, id, []string{"sh", "-c", "exit 42"}, ExecOptions{})
	if err != nil {
		t.Fatalf("ExecStream: %v", err)
	}

	io.ReadAll(stream.Stdout)
	code, err := stream.Wait()
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if code != 42 {
		t.Errorf("exit code = %d, want 42", code)
	}
}

func TestExecStream_Stderr(t *testing.T) {
	sb, err := NewLocal("")
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer sb.Close()

	ctx := context.Background()
	id, err := sb.Create(ctx, CreateOptions{
		Name:  "tai-stream-stderr",
		Image: "alpine:latest",
		Cmd:   []string{"sleep", "30"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer sb.Remove(ctx, id, true)

	if err := sb.Start(ctx, id); err != nil {
		t.Fatalf("Start: %v", err)
	}

	stream, err := sb.ExecStream(ctx, id, []string{"sh", "-c", "echo err-msg >&2"}, ExecOptions{})
	if err != nil {
		t.Fatalf("ExecStream: %v", err)
	}

	stderr, err := io.ReadAll(stream.Stderr)
	if err != nil {
		t.Fatalf("ReadAll stderr: %v", err)
	}
	if !strings.Contains(string(stderr), "err-msg") {
		t.Errorf("stderr = %q, want to contain %q", string(stderr), "err-msg")
	}

	code, _ := stream.Wait()
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
}

func TestExecStream_Cancel(t *testing.T) {
	sb, err := NewLocal("")
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer sb.Close()

	ctx := context.Background()
	id, err := sb.Create(ctx, CreateOptions{
		Name:  "tai-stream-cancel",
		Image: "alpine:latest",
		Cmd:   []string{"sleep", "30"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer sb.Remove(ctx, id, true)

	if err := sb.Start(ctx, id); err != nil {
		t.Fatalf("Start: %v", err)
	}

	stream, err := sb.ExecStream(ctx, id, []string{"sleep", "300"}, ExecOptions{})
	if err != nil {
		t.Fatalf("ExecStream: %v", err)
	}

	stream.Cancel()

	done := make(chan struct{})
	go func() {
		stream.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Error("Wait did not return after Cancel within 5s")
	}
}

func TestParseUID(t *testing.T) {
	tests := []struct {
		input string
		want  int64
		ok    bool
	}{
		{"1000", 1000, true},
		{"1000:1000", 1000, true},
		{"0", 0, true},
		{"abc", 0, false},
	}
	for _, tt := range tests {
		got, err := parseUID(tt.input)
		if tt.ok && err != nil {
			t.Errorf("parseUID(%q): unexpected error %v", tt.input, err)
		}
		if !tt.ok && err == nil {
			t.Errorf("parseUID(%q): expected error", tt.input)
		}
		if tt.ok && got != tt.want {
			t.Errorf("parseUID(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
