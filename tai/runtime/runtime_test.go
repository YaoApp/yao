package runtime

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	dockertypes "github.com/docker/docker/api/types"
	corev1 "k8s.io/api/core/v1"
)

// buildContextFromDockerfile creates a tar archive containing a Dockerfile,
// suitable for passing to Docker's ImageBuild API.
func buildContextFromDockerfile(content string) (io.Reader, error) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	err := tw.WriteHeader(&tar.Header{
		Name: "Dockerfile",
		Size: int64(len(content)),
		Mode: 0644,
	})
	if err != nil {
		return nil, err
	}
	if _, err := tw.Write([]byte(content)); err != nil {
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	return &buf, nil
}

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

// ---------------------------------------------------------------------------
// image_docker.go tests
// ---------------------------------------------------------------------------

func TestDockerImage_ExistsAndInspect(t *testing.T) {
	sb, err := NewLocal("")
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer sb.Close()

	img := NewDockerImage(DockerCli(sb))
	ctx := context.Background()

	t.Run("Exists_true", func(t *testing.T) {
		ok, err := img.Exists(ctx, "alpine:latest")
		if err != nil {
			t.Fatalf("Exists: %v", err)
		}
		if !ok {
			t.Fatal("expected alpine:latest to exist")
		}
	})

	t.Run("Exists_false", func(t *testing.T) {
		ok, err := img.Exists(ctx, "nonexistent-image-xyz:99.99.99")
		if err != nil {
			t.Fatalf("Exists: %v", err)
		}
		if ok {
			t.Fatal("expected nonexistent image to not exist")
		}
	})

	t.Run("Inspect_alpine", func(t *testing.T) {
		meta, err := img.Inspect(ctx, "alpine:latest")
		if err != nil {
			t.Fatalf("Inspect: %v", err)
		}
		if meta.OS != "linux" {
			t.Errorf("OS = %q, want linux", meta.OS)
		}
		if meta.Arch == "" {
			t.Error("Arch should not be empty")
		}
		if meta.Shell == "" {
			t.Error("Shell should not be empty")
		}
	})

	t.Run("Inspect_error", func(t *testing.T) {
		_, err := img.Inspect(ctx, "nonexistent-image-xyz:99.99.99")
		if err == nil {
			t.Fatal("expected error for nonexistent image")
		}
	})

	t.Run("Inspect_withShellAndEnv", func(t *testing.T) {
		// Build a temporary test image with SHELL instruction and SHELL env var.
		cli := DockerCli(sb)
		dockerfile := `FROM alpine:latest
SHELL ["/bin/sh", "-c"]
ENV SHELL=/bin/sh
`
		buildCtx, err := buildContextFromDockerfile(dockerfile)
		if err != nil {
			t.Fatalf("build context: %v", err)
		}
		resp, err := cli.ImageBuild(ctx, buildCtx, dockertypes.ImageBuildOptions{
			Tags:        []string{"tai-test-shell:latest"},
			Remove:      true,
			ForceRemove: true,
		})
		if err != nil {
			t.Fatalf("ImageBuild: %v", err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		t.Cleanup(func() {
			img.Remove(ctx, "tai-test-shell:latest", true)
		})

		meta, err := img.Inspect(ctx, "tai-test-shell:latest")
		if err != nil {
			t.Fatalf("Inspect: %v", err)
		}
		// The image has SHELL ["/bin/sh", "-c"] so Config.Shell[0] = "/bin/sh"
		if meta.Shell != "/bin/sh" {
			t.Errorf("Shell = %q, want /bin/sh", meta.Shell)
		}
	})
}

func TestDockerImage_PullAndRemove(t *testing.T) {
	sb, err := NewLocal("")
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer sb.Close()

	img := NewDockerImage(DockerCli(sb))
	ctx := context.Background()
	ref := "alpine:3.18"

	// Remove first if it exists, ignore error
	img.Remove(ctx, ref, true)

	t.Run("Pull", func(t *testing.T) {
		ch, err := img.Pull(ctx, ref, PullOptions{})
		if err != nil {
			t.Fatalf("Pull: %v", err)
		}
		count := 0
		for p := range ch {
			if p.Error != "" {
				t.Fatalf("pull error: %s", p.Error)
			}
			count++
		}
		if count == 0 {
			t.Error("expected at least one progress event")
		}
	})

	t.Run("Pull_withAuth_error", func(t *testing.T) {
		// Pull with dummy auth — Docker Hub rejects fake credentials,
		// which exercises both encodeAuth and the Pull error path.
		_, err := img.Pull(ctx, ref, PullOptions{
			Auth: &RegistryAuth{
				Username: "testuser",
				Password: "testpass",
				Server:   "https://index.docker.io/v1/",
			},
		})
		if err == nil {
			t.Fatal("expected error for pull with invalid auth")
		}
	})

	t.Run("List", func(t *testing.T) {
		imgs, err := img.List(ctx)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(imgs) == 0 {
			t.Fatal("expected at least one image")
		}
		found := false
		for _, i := range imgs {
			for _, tag := range i.Tags {
				if tag == ref {
					found = true
				}
			}
			if i.ID == "" {
				t.Error("image ID should not be empty")
			}
		}
		if !found {
			t.Errorf("expected %s in image list", ref)
		}
	})

	t.Run("Remove", func(t *testing.T) {
		err := img.Remove(ctx, ref, true)
		if err != nil {
			t.Fatalf("Remove: %v", err)
		}
		ok, _ := img.Exists(ctx, ref)
		if ok {
			t.Error("image should not exist after Remove")
		}
	})

	t.Run("Remove_nonexistent", func(t *testing.T) {
		err := img.Remove(ctx, "nonexistent-image-xyz:99.99.99", true)
		if err == nil {
			t.Fatal("expected error for removing nonexistent image")
		}
	})
}

func TestDecodePullStream(t *testing.T) {
	t.Run("normal_events", func(t *testing.T) {
		events := []dockerPullEvent{
			{Status: "Pulling fs layer", ID: "abc123"},
			{Status: "Downloading", ID: "abc123", ProgressDetail: struct {
				Current int64 `json:"current"`
				Total   int64 `json:"total"`
			}{Current: 500, Total: 1000}},
			{Status: "Pull complete", ID: "abc123"},
		}
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		for _, ev := range events {
			enc.Encode(ev)
		}

		ch := make(chan PullProgress, 32)
		decodePullStream(&buf, ch)
		close(ch)

		got := make([]PullProgress, 0)
		for p := range ch {
			got = append(got, p)
		}
		if len(got) != 3 {
			t.Fatalf("expected 3 events, got %d", len(got))
		}
		if got[0].Status != "Pulling fs layer" || got[0].Layer != "abc123" {
			t.Errorf("event 0: %+v", got[0])
		}
		if got[1].Current != 500 || got[1].Total != 1000 {
			t.Errorf("event 1 progress: %+v", got[1])
		}
	})

	t.Run("error_event", func(t *testing.T) {
		ev := dockerPullEvent{Status: "Error", Error: "pull access denied"}
		var buf bytes.Buffer
		json.NewEncoder(&buf).Encode(ev)

		ch := make(chan PullProgress, 32)
		decodePullStream(&buf, ch)

		got := <-ch
		if got.Error != "pull access denied" {
			t.Errorf("error = %q, want %q", got.Error, "pull access denied")
		}
	})

	t.Run("malformed_json", func(t *testing.T) {
		buf := bytes.NewBufferString("not json\n")
		ch := make(chan PullProgress, 32)
		decodePullStream(buf, ch)

		got := <-ch
		if got.Error == "" {
			t.Error("expected error for malformed JSON")
		}
	})

	t.Run("empty_stream", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		ch := make(chan PullProgress, 32)
		decodePullStream(buf, ch)

		// decodePullStream returns without sending anything on empty input.
		// Channel should have no items.
		if len(ch) != 0 {
			t.Error("expected no events for empty stream")
		}
	})
}

func TestEncodeAuth(t *testing.T) {
	auth := &RegistryAuth{
		Username: "user",
		Password: "pass",
		Server:   "https://registry.example.com",
	}
	encoded, err := encodeAuth(auth)
	if err != nil {
		t.Fatalf("encodeAuth: %v", err)
	}
	if encoded == "" {
		t.Fatal("expected non-empty encoded string")
	}

	decoded, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	var cfg struct {
		Username      string `json:"username"`
		Password      string `json:"password"`
		ServerAddress string `json:"serveraddress"`
	}
	if err := json.Unmarshal(decoded, &cfg); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if cfg.Username != "user" || cfg.Password != "pass" || cfg.ServerAddress != "https://registry.example.com" {
		t.Errorf("decoded auth: %+v", cfg)
	}
}

// ---------------------------------------------------------------------------
// client_accessor.go tests
// ---------------------------------------------------------------------------

func TestDockerCli_Local(t *testing.T) {
	sb, err := NewLocal("")
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer sb.Close()

	cli := DockerCli(sb)
	if cli == nil {
		t.Fatal("expected non-nil Docker client for local runtime")
	}
}

func TestDockerCli_DockerSandbox(t *testing.T) {
	addr := taiTestDocker()
	sb, err := NewDocker(addr)
	if err != nil {
		t.Skipf("Tai Docker proxy not available at %s: %v", addr, err)
	}
	defer sb.Close()

	cli := DockerCli(sb)
	if cli == nil {
		t.Fatal("expected non-nil Docker client for docker sandbox")
	}
}

func TestDockerCli_K8s(t *testing.T) {
	host := taiTestK8sHost()
	port := taiTestK8sPort()
	kubeconfig := taiTestKubeConfig()
	if host == "" || port == "" || kubeconfig == "" {
		t.Skip("TAI_TEST_K8S_HOST, TAI_TEST_K8S_PORT, or TAI_TEST_KUBECONFIG not set")
	}

	addr := host + ":" + port
	k8s, err := NewK8s(addr, K8sOption{
		Namespace:  "default",
		KubeConfig: kubeconfig,
	})
	if err != nil {
		t.Skipf("K8s not available: %v", err)
	}
	defer k8s.Close()

	cli := DockerCli(k8s)
	if cli != nil {
		t.Fatal("expected nil Docker client for K8s runtime")
	}
}

// ---------------------------------------------------------------------------
// docker_core.go coverage gaps
// ---------------------------------------------------------------------------

func TestExec_InvalidContainer(t *testing.T) {
	sb, err := NewLocal("")
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer sb.Close()

	_, err = sb.Exec(context.Background(), "nonexistent-container-id-12345", []string{"echo", "hi"}, ExecOptions{})
	if err == nil {
		t.Fatal("expected error for exec on nonexistent container")
	}
}

func TestCreate_InvalidImage(t *testing.T) {
	sb, err := NewLocal("")
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer sb.Close()

	_, err = sb.Create(context.Background(), CreateOptions{
		Name:  "tai-invalid-image-test",
		Image: "nonexistent-image-xyz:99.99.99",
		Cmd:   []string{"sleep", "1"},
	})
	if err == nil {
		t.Fatal("expected error for invalid image")
	}
}

func TestInspect_InvalidContainer(t *testing.T) {
	sb, err := NewLocal("")
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer sb.Close()

	_, err = sb.Inspect(context.Background(), "nonexistent-container-id-12345")
	if err == nil {
		t.Fatal("expected error for inspect on nonexistent container")
	}
}

func TestList_NoLabels(t *testing.T) {
	sb, err := NewLocal("")
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer sb.Close()

	// List without label filter — exercises the len(opts.Labels) == 0 branch
	result, err := sb.List(context.Background(), ListOptions{All: true})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	// Should return at least some containers (or zero, both valid)
	_ = result
}

func TestExecWithUser(t *testing.T) {
	sb, err := NewLocal("")
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer sb.Close()

	ctx := context.Background()
	id, err := sb.Create(ctx, CreateOptions{
		Name:  "tai-exec-user-test",
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

	// Exec with User and Env to cover those branches in exec()
	result, err := sb.Exec(ctx, id, []string{"id", "-u"}, ExecOptions{
		User:    "0",
		WorkDir: "/tmp",
		Env:     map[string]string{"TEST_VAR": "hello"},
	})
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("exit code = %d", result.ExitCode)
	}
}

func TestExecStream_WithUserAndEnv(t *testing.T) {
	sb, err := NewLocal("")
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer sb.Close()

	ctx := context.Background()
	id, err := sb.Create(ctx, CreateOptions{
		Name:  "tai-stream-user-env",
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

	stream, err := sb.ExecStream(ctx, id, []string{"echo", "ok"}, ExecOptions{
		User:    "0",
		WorkDir: "/tmp",
		Env:     map[string]string{"X": "1"},
	})
	if err != nil {
		t.Fatalf("ExecStream: %v", err)
	}

	out, _ := io.ReadAll(stream.Stdout)
	if string(out) != "ok\n" {
		t.Errorf("stdout = %q", string(out))
	}
	code, _ := stream.Wait()
	if code != 0 {
		t.Errorf("exit code = %d", code)
	}
}

func TestInspect_StoppedContainer(t *testing.T) {
	sb, err := NewLocal("")
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer sb.Close()

	ctx := context.Background()
	id, err := sb.Create(ctx, CreateOptions{
		Name:  "tai-inspect-stopped",
		Image: "alpine:latest",
		Cmd:   []string{"true"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer sb.Remove(ctx, id, true)

	// Inspect a created-but-never-started container — exercises
	// branches where NetworkSettings may have no networks.
	info, err := sb.Inspect(ctx, id)
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if info.ID != id {
		t.Errorf("ID mismatch: got %q, want %q", info.ID, id)
	}
}

func TestNewDockerInvalidAddr(t *testing.T) {
	_, err := NewDocker("tcp://192.168.254.254:1")
	if err == nil {
		t.Error("expected error for unreachable Tai proxy")
	}
}
