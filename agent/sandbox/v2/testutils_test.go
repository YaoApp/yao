package sandboxv2_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	sandbox "github.com/yaoapp/yao/sandbox/v2"
	"github.com/yaoapp/yao/tai"
	"github.com/yaoapp/yao/tai/registry"
	taisandbox "github.com/yaoapp/yao/tai/sandbox"
	"github.com/yaoapp/yao/workspace"
)

// ---------------------------------------------------------------------------
// node configuration — mirrors sandbox/v2 testutils but scoped to prepare tests
// ---------------------------------------------------------------------------

type nodeConfig struct {
	Name    string
	Addr    string
	TaiID   string
	Options []tai.Option
}

type hostTarget struct {
	Name  string
	Addr  string
	TaiID string
}

// ---------------------------------------------------------------------------
// environment helpers (same conventions as sandbox/v2 + env.local.sh)
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// node discovery
// ---------------------------------------------------------------------------

func boxNodes() []nodeConfig {
	nodes := []nodeConfig{
		{Name: "local", Addr: testLocalAddr()},
	}
	if addr := os.Getenv("SANDBOX_TEST_REMOTE_ADDR"); addr != "" {
		nodes = append(nodes, nodeConfig{Name: "remote", Addr: addr})
	}
	return nodes
}

func hostTargets() []hostTarget {
	var targets []hostTarget
	if addr := os.Getenv("TAI_TEST_WIN_HOSTEXEC_LINUX"); addr != "" {
		targets = append(targets, hostTarget{Name: "win-linux", Addr: addr})
	}
	if addr := os.Getenv("TAI_TEST_WIN_HOSTEXEC_NATIVE"); addr != "" {
		targets = append(targets, hostTarget{Name: "win-native", Addr: addr})
	}
	return targets
}

// ---------------------------------------------------------------------------
// TestMain — purge stale containers from previous runs
// ---------------------------------------------------------------------------

func TestMain(m *testing.M) {
	purgeStale()
	os.Exit(m.Run())
}

func purgeStale() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, nc := range boxNodes() {
		client, err := tai.New(nc.Addr, nc.Options...)
		if err != nil {
			continue
		}
		sb := client.Sandbox()
		if sb == nil {
			client.Close()
			continue
		}
		containers, _ := sb.List(ctx, taisandbox.ListOptions{All: true})
		for _, c := range containers {
			id := c.Name
			if id == "" {
				id = c.ID
			}
			if strings.HasPrefix(id, "sb-prep-") || strings.HasPrefix(id, "sb-lc-") {
				sb.Remove(ctx, id, true)
				log.Printf("[purge] %s: removed %s", nc.Name, id)
			}
		}
		client.Close()
	}
}

// ---------------------------------------------------------------------------
// Manager + Box helpers
// ---------------------------------------------------------------------------

func setupManager(t *testing.T, nc *nodeConfig) *sandbox.Manager {
	t.Helper()
	if registry.Global() == nil {
		registry.Init(nil)
	}
	client, err := tai.New(nc.Addr, nc.Options...)
	if err != nil {
		t.Fatalf("tai.New(%s): %v", nc.Addr, err)
	}
	nc.TaiID = client.TaiID()

	sandbox.Init()
	m := sandbox.M()
	t.Cleanup(func() { m.Close() })
	return m
}

func createBox(t *testing.T, m *sandbox.Manager, nc nodeConfig) *sandbox.Box {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := m.EnsureImage(ctx, nc.TaiID, testImage(), sandbox.ImagePullOptions{}); err != nil {
		t.Fatalf("EnsureImage: %v", err)
	}

	box, err := m.Create(ctx, sandbox.CreateOptions{
		ID:     fmt.Sprintf("sb-prep-%d", time.Now().UnixNano()),
		Image:  testImage(),
		Owner:  "test-prepare",
		NodeID: nc.TaiID,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() {
		cCtx, cCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cCancel()
		if err := m.Remove(cCtx, box.ID()); err != nil {
			t.Logf("cleanup Remove(%s): %v", box.ID(), err)
		}
	})
	return box
}

func createHost(t *testing.T, m *sandbox.Manager, tgt hostTarget) *sandbox.Host {
	t.Helper()
	host, err := m.Host(context.Background(), tgt.TaiID)
	if err != nil {
		t.Skipf("Host(%s): %v", tgt.Name, err)
	}
	return host
}

func setupHostManager(t *testing.T, tgt *hostTarget) *sandbox.Manager {
	t.Helper()
	nc := nodeConfig{Name: tgt.Name, Addr: fmt.Sprintf("tai://%s", tgt.Addr)}
	m := setupManager(t, &nc)
	tgt.TaiID = nc.TaiID
	return m
}

// ---------------------------------------------------------------------------
// skip helpers
// ---------------------------------------------------------------------------

func skipIfNoDocker(t *testing.T) {
	t.Helper()
	if testLocalAddr() == "" {
		t.Skip("SANDBOX_TEST_LOCAL_ADDR not set")
	}
}

func skipIfNoHostExec(t *testing.T) {
	t.Helper()
	if len(hostTargets()) == 0 {
		t.Skip("no HostExec targets configured")
	}
}

func createTestWorkspace(t *testing.T, taiID, wsID string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, err := workspace.M().Create(ctx, workspace.CreateOptions{
		ID:    wsID,
		Owner: "test",
		Node:  taiID,
	})
	if err != nil && !strings.Contains(err.Error(), "exists") {
		t.Fatalf("create workspace %q: %v", wsID, err)
	}
	t.Cleanup(func() {
		cCtx, cCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cCancel()
		workspace.M().Delete(cCtx, wsID, true)
	})
}
