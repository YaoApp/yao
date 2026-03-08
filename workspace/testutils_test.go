package workspace_test

import (
	"context"
	"net/url"
	"os"
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
	registerClient(tb, pc)
	return workspace.NewManager()
}

func registerClient(tb testing.TB, pc poolConfig) *tai.Client {
	tb.Helper()
	if pc.Addr == "local" {
		return localClient(tb, tb.TempDir())
	}
	client, err := tai.New(pc.Addr)
	if err != nil {
		tb.Fatalf("tai.New(%s): %v", pc.Addr, err)
	}
	tb.Cleanup(func() { client.Close() })
	return client
}

func localClient(tb testing.TB, dataDir string) *tai.Client {
	tb.Helper()
	vol := volume.NewLocal(dataDir)
	client, err := tai.New("local", tai.WithVolume(vol), tai.WithDataDir(dataDir))
	if err != nil {
		tb.Fatalf("tai.New local: %v", err)
	}
	tb.Cleanup(func() { client.Close() })
	return client
}

func setupManagerMultiNode(t *testing.T) (*workspace.Manager, string, string) {
	t.Helper()
	ensureRegistry(t)

	dir1 := t.TempDir()
	vol1 := volume.NewLocal(dir1)
	_, err := tai.New("docker://node-a", tai.WithVolume(vol1), tai.WithDataDir(dir1))
	if err != nil {
		t.Fatalf("tai.New node-a: %v", err)
	}

	dir2 := t.TempDir()
	vol2 := volume.NewLocal(dir2)
	_, err = tai.New("docker://node-b", tai.WithVolume(vol2), tai.WithDataDir(dir2))
	if err != nil {
		t.Fatalf("tai.New node-b: %v", err)
	}

	return workspace.NewManager(), "docker://node-a", "docker://node-b"
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
