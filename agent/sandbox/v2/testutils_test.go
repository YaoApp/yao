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
	tairuntime "github.com/yaoapp/yao/tai/runtime"
	"github.com/yaoapp/yao/workspace"
)

// ---------------------------------------------------------------------------
// Build-tag extension points (same pattern as sandbox/v2).
// ---------------------------------------------------------------------------
var (
	extraNodeProviders     []func() []nodeConfig
	extraHostExecProviders []func() []hostTarget
)

// ---------------------------------------------------------------------------
// Node / host configuration
// ---------------------------------------------------------------------------

type nodeConfig struct {
	Name    string
	Addr    string
	TaiID   string
	DialOps []tai.DialOption
}

type hostTarget struct {
	Name  string
	Addr  string
	TaiID string
}

// ---------------------------------------------------------------------------
// Environment helpers
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
// Node / host discovery
// ---------------------------------------------------------------------------

func boxNodes() []nodeConfig {
	nodes := []nodeConfig{
		{Name: "local", Addr: testLocalAddr()},
	}
	for _, fn := range extraNodeProviders {
		nodes = append(nodes, fn()...)
	}
	return nodes
}

func hostTargets() []hostTarget {
	var targets []hostTarget
	for _, fn := range extraHostExecProviders {
		targets = append(targets, fn()...)
	}
	return targets
}

// ---------------------------------------------------------------------------
// Dial + Register helper (replaces old tai.New)
// ---------------------------------------------------------------------------

func dialForTest(addr string, dialOps ...tai.DialOption) (*tai.ConnResources, error) {
	if addr == "local" || addr == "" {
		return tai.DialLocal("", "", nil)
	}
	host, grpcPort := parseHostPort(addr)
	ports := tai.Ports{GRPC: grpcPort}
	return tai.DialRemote(host, ports, dialOps...)
}

func registerForTest(t testing.TB, addr string, dialOps ...tai.DialOption) (string, *tai.ConnResources) {
	t.Helper()
	if registry.Global() == nil {
		registry.Init(nil)
	}
	res, err := dialForTest(addr, dialOps...)
	if err != nil {
		t.Fatalf("dialForTest(%s): %v", addr, err)
	}
	taiID := taiIDFromAddr(addr)
	reg := registry.Global()
	reg.Register(&registry.TaiNode{TaiID: taiID, Mode: modeForAddr(addr)})
	reg.SetResources(taiID, res)
	return taiID, res
}

func taiIDFromAddr(addr string) string {
	if addr == "local" || addr == "" {
		return "local"
	}
	addr = strings.TrimPrefix(addr, "tai://")
	parts := strings.SplitN(addr, ":", 2)
	return parts[0]
}

func modeForAddr(addr string) string {
	if addr == "local" || addr == "" {
		return "local"
	}
	return "direct"
}

func parseHostPort(addr string) (string, int) {
	addr = strings.TrimPrefix(addr, "tai://")
	parts := strings.SplitN(addr, ":", 2)
	h := parts[0]
	if len(parts) == 2 {
		if p, err := strconv.Atoi(parts[1]); err == nil {
			return h, p
		}
	}
	return h, 19100
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
		res, err := dialForTest(nc.Addr, nc.DialOps...)
		if err != nil {
			continue
		}
		sb := res.Runtime
		if sb == nil {
			res.Close()
			continue
		}
		containers, _ := sb.List(ctx, tairuntime.ListOptions{All: true})
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
		res.Close()
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
	taiID, res := registerForTest(t, nc.Addr, nc.DialOps...)
	nc.TaiID = taiID
	t.Cleanup(func() { res.Close() })

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
// Skip helpers
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
