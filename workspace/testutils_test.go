package workspace_test

import (
	"context"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/yaoapp/yao/tai"
	"github.com/yaoapp/yao/tai/registry"
	"github.com/yaoapp/yao/tai/volume"
	"github.com/yaoapp/yao/workspace"
)

type poolConfig struct {
	Name string
	Addr string
}

func testPools() []poolConfig {
	pools := []poolConfig{
		{Name: "local", Addr: "local"},
	}
	if addr := os.Getenv("SANDBOX_TEST_REMOTE_ADDR"); addr != "" {
		name := taiIDFromAddr(addr)
		pools = append(pools, poolConfig{Name: name, Addr: addr})
	}
	return pools
}

func taiIDFromAddr(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "local" || addr == "" {
		return "local"
	}
	if !strings.Contains(addr, "://") {
		addr = "tai://" + addr
	}
	u, err := url.Parse(addr)
	if err != nil {
		return addr
	}
	h := u.Hostname()
	if h == "" {
		return addr
	}
	if p := u.Port(); p != "" {
		return h + "-" + p
	}
	return h
}

func ensureRegistry(tb testing.TB) {
	tb.Helper()
	registry.Init(nil)
}

func setupManagerForPool(tb testing.TB, pc poolConfig) *workspace.Manager {
	tb.Helper()
	ensureRegistry(tb)
	registerForTest(tb, pc)
	return workspace.NewManager()
}

func registerForTest(tb testing.TB, pc poolConfig) {
	tb.Helper()
	if pc.Addr == "local" {
		registerLocalForTest(tb, tb.TempDir())
		return
	}
	host, grpcPort := parseHostPort(pc.Addr)
	ports := tai.Ports{GRPC: grpcPort}
	res, err := tai.DialRemote(host, ports)
	if err != nil {
		tb.Fatalf("DialRemote(%s): %v", pc.Addr, err)
	}
	taiID := taiIDFromAddr(pc.Addr)
	reg := registry.Global()
	reg.Register(&registry.TaiNode{TaiID: taiID, Mode: "direct"})
	reg.SetResources(taiID, res)
	tb.Cleanup(func() { res.Close() })
}

func registerLocalForTest(tb testing.TB, dataDir string) {
	tb.Helper()
	vol := volume.NewLocal(dataDir)
	res, err := tai.DialLocal("", dataDir, vol)
	if err != nil {
		tb.Fatalf("DialLocal: %v", err)
	}
	reg := registry.Global()
	reg.Register(&registry.TaiNode{TaiID: "local", Mode: "local"})
	reg.SetResources("local", res)
	tb.Cleanup(func() { res.Close() })
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

func setupManagerMultiNode(t *testing.T) (*workspace.Manager, string, string) {
	t.Helper()
	ensureRegistry(t)

	dir1 := t.TempDir()
	vol1 := volume.NewLocal(dir1)
	res1, err := tai.DialLocal("", dir1, vol1)
	if err != nil {
		t.Fatalf("DialLocal node-a: %v", err)
	}
	reg := registry.Global()
	reg.Register(&registry.TaiNode{TaiID: "node-a", Mode: "local"})
	reg.SetResources("node-a", res1)
	t.Cleanup(func() { res1.Close() })

	dir2 := t.TempDir()
	vol2 := volume.NewLocal(dir2)
	res2, err := tai.DialLocal("", dir2, vol2)
	if err != nil {
		t.Fatalf("DialLocal node-b: %v", err)
	}
	reg.Register(&registry.TaiNode{TaiID: "node-b", Mode: "local"})
	reg.SetResources("node-b", res2)
	t.Cleanup(func() { res2.Close() })

	return workspace.NewManager(), "node-a", "node-b"
}

func createWorkspace(tb testing.TB, m *workspace.Manager, node string, opts ...func(*workspace.CreateOptions)) *workspace.Workspace {
	tb.Helper()
	co := workspace.CreateOptions{
		Name:  "test-workspace",
		Owner: "test-user",
		Node:  node,
	}
	for _, fn := range opts {
		fn(&co)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ws, err := m.Create(ctx, co)
	if err != nil {
		tb.Fatalf("Create workspace: %v", err)
	}
	tb.Cleanup(func() {
		m.Delete(context.Background(), ws.ID, true)
	})
	return ws
}
