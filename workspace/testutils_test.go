package workspace_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/yaoapp/yao/tai"
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
		pools = append(pools, poolConfig{Name: "remote", Addr: addr})
	}
	return pools
}

func setupManagerForPool(tb testing.TB, pc poolConfig) *workspace.Manager {
	tb.Helper()
	client := clientForPool(tb, pc)
	pools := map[string]*tai.Client{pc.Name: client}
	return workspace.NewManager(pools)
}

func clientForPool(tb testing.TB, pc poolConfig) *tai.Client {
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

func setupManagerMultiNode(t *testing.T) *workspace.Manager {
	t.Helper()
	pools := map[string]*tai.Client{
		"node-a": localClient(t, t.TempDir()),
		"node-b": localClient(t, t.TempDir()),
	}
	return workspace.NewManager(pools)
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
