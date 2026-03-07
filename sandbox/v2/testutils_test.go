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
	taisandbox "github.com/yaoapp/yao/tai/sandbox"
	"github.com/yaoapp/yao/tai/volume"
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
// test runs across all configured pools (Docker + K8s).
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
			grpcPort := envPort("TAI_TEST_K8S_GRPC_PORT", envPort("TAI_TEST_GRPC_PORT", 9100))
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

type poolConfig struct {
	Name    string
	Addr    string
	Options []tai.Option
}

// testPools returns all available pool configurations for multi-mode testing.
//   - local:          always present (direct Docker daemon)
//   - remote:         when SANDBOX_TEST_REMOTE_ADDR is set (Tai on host → Docker)
//   - containerized:  when TAI_TEST_CONTAINERIZED_HOST is set (Tai in container → Docker)
//   - k8s:            when TAI_TEST_K8S_HOST + TAI_TEST_KUBECONFIG are set (Tai → K8s)
func testPools() []poolConfig {
	pools := []poolConfig{
		{Name: "local", Addr: testLocalAddr()},
	}
	if addr := os.Getenv("SANDBOX_TEST_REMOTE_ADDR"); addr != "" {
		pools = append(pools, poolConfig{Name: "remote", Addr: addr})
	}
	if host := os.Getenv("TAI_TEST_CONTAINERIZED_HOST"); host != "" {
		grpcPort := envPort("TAI_TEST_CONTAINERIZED_GRPC_PORT", 9200)
		addr := fmt.Sprintf("tai://%s:%d", host, grpcPort)
		// No WithPorts for HTTP/VNC — Tai self-inspects its container
		// and returns host-mapped ports via ServerInfo automatically.
		pools = append(pools, poolConfig{Name: "containerized", Addr: addr})
	}
	if host := os.Getenv("TAI_TEST_K8S_HOST"); host != "" {
		kubeconfig := os.Getenv("TAI_TEST_KUBECONFIG")
		if kubeconfig == "" {
			return pools
		}
		grpcPort := envPort("TAI_TEST_K8S_GRPC_PORT", envPort("TAI_TEST_GRPC_PORT", 9100))
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
		pools = append(pools, poolConfig{Name: "k8s", Addr: addr, Options: opts})
	}
	return pools
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
	IsWinNative bool
}

// hostExecTargets returns all Tai instances that support HostExec gRPC.
// No container creation needed — these are direct gRPC connections.
func hostExecTargets() []hostExecTarget {
	var targets []hostExecTarget
	if addr := os.Getenv("SANDBOX_TEST_REMOTE_ADDR"); addr != "" {
		addr = strings.TrimPrefix(addr, "tai://")
		targets = append(targets, hostExecTarget{Name: "remote", Addr: addr})
	}
	if host := os.Getenv("TAI_TEST_K8S_HOST"); host != "" {
		grpcPort := envPort("TAI_TEST_K8S_GRPC_PORT", envPort("TAI_TEST_GRPC_PORT", 9100))
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

// linuxCmd adapts a Linux command to the equivalent Windows command for
// Windows native Tai targets.
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

func setupManager(t *testing.T, pools ...sandbox.Pool) *sandbox.Manager {
	t.Helper()
	cfg := sandbox.Config{Pool: pools}
	if err := sandbox.Init(cfg); err != nil {
		t.Fatalf("Init: %v", err)
	}
	m := sandbox.M()
	t.Cleanup(func() {
		m.Close()
	})
	return m
}

func setupManagerForPool(t *testing.T, pc poolConfig, mutators ...func(*sandbox.Pool)) *sandbox.Manager {
	t.Helper()
	pool := sandbox.Pool{Name: pc.Name, Addr: pc.Addr, Options: pc.Options}
	for _, fn := range mutators {
		fn(&pool)
	}
	return setupManager(t, pool)
}

// setupManagerWithWorkspace creates a sandbox Manager with a linked workspace Manager.
// Returns both managers and a helper to create workspaces on the given pool's node.
func setupManagerWithWorkspace(t *testing.T, pc poolConfig) (*sandbox.Manager, *workspace.Manager) {
	t.Helper()
	sbm := setupManagerForPool(t, pc)

	var wsClient *tai.Client
	var err error
	if pc.Addr == "local" || pc.Addr == "" {
		dataDir := t.TempDir()
		vol := volume.NewLocal(dataDir)
		wsClient, err = tai.New("local", tai.WithVolume(vol), tai.WithDataDir(dataDir))
	} else {
		wsClient, err = tai.New(pc.Addr, pc.Options...)
	}
	if err != nil {
		t.Fatalf("tai.New for workspace: %v", err)
	}
	t.Cleanup(func() { wsClient.Close() })

	wsm := workspace.NewManager(map[string]*tai.Client{pc.Name: wsClient})
	sbm.SetWorkspaceManager(wsm)
	return sbm, wsm
}

// ensureTestImage guarantees testImage() is available on the given pool before
// container creation. Safe for all modes (Docker pull; K8s no-op).
func ensureTestImage(t *testing.T, m *sandbox.Manager, pool string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	if err := m.EnsureImage(ctx, pool, testImage(), sandbox.ImagePullOptions{}); err != nil {
		t.Fatalf("EnsureImage(%s, %s): %v", pool, testImage(), err)
	}
}

func createTestBox(t *testing.T, m *sandbox.Manager, opts ...func(*sandbox.CreateOptions)) *sandbox.Box {
	t.Helper()
	co := sandbox.CreateOptions{
		Image: testImage(),
		Owner: "test-user",
	}
	for _, fn := range opts {
		fn(&co)
	}

	pool := co.Pool
	if pool == "" {
		pools := m.Pools()
		if len(pools) > 0 {
			pool = pools[0].Name
		}
	}

	isK8s := pool == "k8s"
	if isK8s {
		k8sSem <- struct{}{}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if pool != "" {
		if err := m.EnsureImage(ctx, pool, co.Image, sandbox.ImagePullOptions{}); err != nil {
			if isK8s {
				<-k8sSem
			}
			t.Fatalf("EnsureImage(%s, %s): %v", pool, co.Image, err)
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
