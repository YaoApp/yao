package sandbox_test

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	sandbox "github.com/yaoapp/yao/sandbox/v2"
	"github.com/yaoapp/yao/tai"
	"github.com/yaoapp/yao/tai/volume"
	"github.com/yaoapp/yao/workspace"
)

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

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	if pool != "" {
		if err := m.EnsureImage(ctx, pool, co.Image, sandbox.ImagePullOptions{}); err != nil {
			t.Fatalf("EnsureImage(%s, %s): %v", pool, co.Image, err)
		}
	}

	box, err := m.Create(ctx, co)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() {
		m.Remove(context.Background(), box.ID())
	})
	return box
}
