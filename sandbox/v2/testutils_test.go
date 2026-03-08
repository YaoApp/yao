package sandbox_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	sandbox "github.com/yaoapp/yao/sandbox/v2"
	"github.com/yaoapp/yao/tai"
	"github.com/yaoapp/yao/tai/registry"
	taisandbox "github.com/yaoapp/yao/tai/sandbox"
	"github.com/yaoapp/yao/workspace"
)

// k8sSem limits concurrent K8s pod creation to avoid overwhelming the cluster.
var k8sSem = make(chan struct{}, 2)

// k8sCleanupMu serialises K8s pod cleanup to prevent overlapping API calls
// when many tests finish at once.
var k8sCleanupMu sync.Mutex

func TestMain(m *testing.M) {
	purgeStaleContainers()
	os.Exit(m.Run())
}

// purgeStaleContainers removes leftover sb-* containers/pods from previous
// test runs across all configured nodes (Docker + K8s).
func purgeStaleContainers() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	type target struct {
		name string
		addr string
		opts []tai.Option
	}

	var targets []target
	targets = append(targets, target{name: "local", addr: testLocalAddr()})

	if addr := os.Getenv("SANDBOX_TEST_REMOTE_ADDR"); addr != "" {
		targets = append(targets, target{name: "remote", addr: addr})
	}
	if host := os.Getenv("TAI_TEST_CONTAINERIZED_HOST"); host != "" {
		grpcPort := envPort("TAI_TEST_CONTAINERIZED_GRPC_PORT", 9200)
		targets = append(targets, target{name: "containerized", addr: fmt.Sprintf("tai://%s:%d", host, grpcPort)})
	}
	if host := os.Getenv("TAI_TEST_K8S_HOST"); host != "" {
		kubeconfig := os.Getenv("TAI_TEST_KUBECONFIG")
		if kubeconfig != "" {
			grpcPort := envPort("TAI_TEST_K8S_GRPC_PORT", envPort("TAI_TEST_GRPC_PORT", 19100))
			opts := []tai.Option{
				tai.K8s,
				tai.WithKubeConfig(kubeconfig),
				tai.WithPorts(tai.Ports{K8s: envPort("TAI_TEST_K8S_PORT", 6443), GRPC: grpcPort}),
			}
			if ns := os.Getenv("TAI_TEST_K8S_NAMESPACE"); ns != "" {
				opts = append(opts, tai.WithNamespace(ns))
			}
			targets = append(targets, target{name: "k8s", addr: fmt.Sprintf("tai://%s:%d", host, grpcPort), opts: opts})
		}
	}

	for _, tgt := range targets {
		client, err := tai.New(tgt.addr, tgt.opts...)
		if err != nil {
			continue
		}
		sb := client.Sandbox()
		if sb == nil {
			client.Close()
			continue
		}
		containers, err := sb.List(ctx, taisandbox.ListOptions{All: true})
		if err != nil {
			client.Close()
			continue
		}
		for _, c := range containers {
			id := c.Name
			if id == "" {
				id = c.ID
			}
			if !strings.HasPrefix(id, "sb-") && !strings.HasPrefix(c.Labels["sandbox-id"], "sb-") {
				continue
			}
			sb.Remove(ctx, id, true)
			log.Printf("[purge] %s: removed stale container %s", tgt.name, id)
		}
		client.Close()
	}
}

type nodeConfig struct {
	Name    string // human-readable label for t.Run (e.g. "remote", "k8s")
	Addr    string
	TaiID   string // actual registry key, filled after tai.New
	Options []tai.Option
}

// testNodes returns all available node configurations for multi-mode testing.
func testNodes() []nodeConfig {
	nodes := []nodeConfig{
		{Name: "local", Addr: testLocalAddr()},
	}
	if addr := os.Getenv("SANDBOX_TEST_REMOTE_ADDR"); addr != "" {
		nodes = append(nodes, nodeConfig{Name: "remote", Addr: addr})
	}
	if host := os.Getenv("TAI_TEST_CONTAINERIZED_HOST"); host != "" {
		grpcPort := envPort("TAI_TEST_CONTAINERIZED_GRPC_PORT", 9200)
		addr := fmt.Sprintf("tai://%s:%d", host, grpcPort)
		nodes = append(nodes, nodeConfig{Name: "containerized", Addr: addr})
	}
	if host := os.Getenv("TAI_TEST_K8S_HOST"); host != "" {
		kubeconfig := os.Getenv("TAI_TEST_KUBECONFIG")
		if kubeconfig == "" {
			return nodes
		}
		grpcPort := envPort("TAI_TEST_K8S_GRPC_PORT", envPort("TAI_TEST_GRPC_PORT", 19100))
		addr := fmt.Sprintf("tai://%s:%d", host, grpcPort)
		opts := []tai.Option{
			tai.K8s,
			tai.WithKubeConfig(kubeconfig),
			tai.WithPorts(tai.Ports{
				K8s:  envPort("TAI_TEST_K8S_PORT", 6443),
				GRPC: grpcPort,
			}),
		}
		if ns := os.Getenv("TAI_TEST_K8S_NAMESPACE"); ns != "" {
			opts = append(opts, tai.WithNamespace(ns))
		}
		nodes = append(nodes, nodeConfig{Name: "k8s", Addr: addr, Options: opts})
	}
	return nodes
}

func skipIfNoDocker(t *testing.T) {
	t.Helper()
	addr := testLocalAddr()
	if addr == "" {
		t.Skip("SANDBOX_TEST_LOCAL_ADDR not set, skipping Docker tests")
	}
}

func skipIfNoTai(t *testing.T) {
	t.Helper()
	if os.Getenv("SANDBOX_TEST_REMOTE_ADDR") == "" {
		t.Skip("SANDBOX_TEST_REMOTE_ADDR not set, skipping Tai proxy tests")
	}
}

type hostExecTarget struct {
	Name        string
	Addr        string // host:port (without tai:// prefix)
	TaiID       string // filled after registration
	IsWinNative bool
}

func hostExecTargets() []hostExecTarget {
	var targets []hostExecTarget
	if addr := os.Getenv("SANDBOX_TEST_REMOTE_ADDR"); addr != "" {
		addr = strings.TrimPrefix(addr, "tai://")
		targets = append(targets, hostExecTarget{Name: "remote", Addr: addr})
	}
	if host := os.Getenv("TAI_TEST_K8S_HOST"); host != "" {
		grpcPort := envPort("TAI_TEST_K8S_GRPC_PORT", envPort("TAI_TEST_GRPC_PORT", 19100))
		targets = append(targets, hostExecTarget{Name: "k8s", Addr: fmt.Sprintf("%s:%d", host, grpcPort)})
	}
	if addr := os.Getenv("TAI_TEST_WIN_HOSTEXEC_LINUX"); addr != "" {
		targets = append(targets, hostExecTarget{Name: "win-linux", Addr: addr})
	}
	if addr := os.Getenv("TAI_TEST_WIN_HOSTEXEC_NATIVE"); addr != "" {
		targets = append(targets, hostExecTarget{Name: "win-native", Addr: addr, IsWinNative: true})
	}
	return targets
}

func skipIfNoHostExec(t *testing.T) {
	t.Helper()
	if len(hostExecTargets()) == 0 {
		t.Skip("no HostExec targets configured")
	}
}

func linuxCmd(tgt hostExecTarget, cmd string, args ...string) (string, []string) {
	if tgt.IsWinNative {
		switch cmd {
		case "echo":
			return "cmd.exe", append([]string{"/c", "echo"}, args...)
		case "pwd":
			return "cmd.exe", []string{"/c", "cd"}
		case "env":
			return "cmd.exe", []string{"/c", "set"}
		case "sleep":
			return "cmd.exe", []string{"/c", "ping", "-n", "10", "127.0.0.1"}
		case "cat":
			return "cmd.exe", []string{"/c", "more"}
		case "sh":
			if len(args) >= 2 && args[0] == "-c" {
				return "cmd.exe", []string{"/c", args[1]}
			}
			return "cmd.exe", append([]string{"/c"}, args...)
		default:
			return cmd, args
		}
	}
	return cmd, args
}

func testLocalAddr() string {
	if addr := os.Getenv("SANDBOX_TEST_LOCAL_ADDR"); addr != "" {
		return addr
	}
	return "local"
}

func testImage() string {
	if img := os.Getenv("SANDBOX_TEST_IMAGE"); img != "" {
		return img
	}
	return "alpine:latest"
}

func envPort(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			return p
		}
	}
	return fallback
}

// registerNode creates a tai.Client and registers it in the global registry.
// It fills pc.TaiID with the actual registry key returned by tai.New.
func registerNode(t *testing.T, pc *nodeConfig) {
	t.Helper()

	reg := registry.Global()
	if reg == nil {
		registry.Init(nil)
	}

	client, err := tai.New(pc.Addr, pc.Options...)
	if err != nil {
		t.Fatalf("tai.New(%s): %v", pc.Addr, err)
	}
	pc.TaiID = client.TaiID()
	t.Cleanup(func() { client.Close() })
}

func setupManager(t *testing.T, nodes ...nodeConfig) (*sandbox.Manager, []nodeConfig) {
	t.Helper()

	reg := registry.Global()
	if reg == nil {
		registry.Init(nil)
	}
	_ = reg

	out := make([]nodeConfig, len(nodes))
	copy(out, nodes)
	for i := range out {
		client, err := tai.New(out[i].Addr, out[i].Options...)
		if err != nil {
			t.Fatalf("tai.New(%s): %v", out[i].Addr, err)
		}
		out[i].TaiID = client.TaiID()
	}

	sandbox.Init()
	m := sandbox.M()
	t.Cleanup(func() { m.Close() })
	return m, out
}

func setupManagerForNode(t *testing.T, pc *nodeConfig) *sandbox.Manager {
	t.Helper()
	m, registered := setupManager(t, *pc)
	*pc = registered[0]
	return m
}

// setupManagerWithWorkspace creates a sandbox Manager and returns
// the global workspace.Manager (which uses the registry for client lookups).
func setupManagerWithWorkspace(t *testing.T, pc *nodeConfig) (*sandbox.Manager, *workspace.Manager) {
	t.Helper()
	sbm := setupManagerForNode(t, pc)
	return sbm, workspace.M()
}

func ensureTestImage(t *testing.T, m *sandbox.Manager, nodeID string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	if err := m.EnsureImage(ctx, nodeID, testImage(), sandbox.ImagePullOptions{}); err != nil {
		t.Fatalf("EnsureImage(%s, %s): %v", nodeID, testImage(), err)
	}
}

func createTestBox(t *testing.T, m *sandbox.Manager, pc nodeConfig, opts ...func(*sandbox.CreateOptions)) *sandbox.Box {
	t.Helper()
	co := sandbox.CreateOptions{
		Image:  testImage(),
		Owner:  "test-user",
		NodeID: pc.TaiID,
	}
	for _, fn := range opts {
		fn(&co)
	}

	nodeID := co.NodeID
	if nodeID == "" {
		nodes := m.Nodes()
		if len(nodes) > 0 {
			nodeID = nodes[0].TaiID
			co.NodeID = nodeID
		}
	}

	isK8s := pc.Name == "k8s"
	if isK8s {
		k8sSem <- struct{}{}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if nodeID != "" {
		if err := m.EnsureImage(ctx, nodeID, co.Image, sandbox.ImagePullOptions{}); err != nil {
			if isK8s {
				<-k8sSem
			}
			t.Fatalf("EnsureImage(%s, %s): %v", nodeID, co.Image, err)
		}
	}

	box, err := m.Create(ctx, co)
	if err != nil {
		if isK8s {
			<-k8sSem
		}
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() {
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanCancel()
		if isK8s {
			k8sCleanupMu.Lock()
			defer k8sCleanupMu.Unlock()
		}
		if err := m.Remove(cleanCtx, box.ID()); err != nil {
			t.Logf("cleanup Remove(%s): %v", box.ID(), err)
		}
		if isK8s {
			<-k8sSem
		}
	})
	return box
}
