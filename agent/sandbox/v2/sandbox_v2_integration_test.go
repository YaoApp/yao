//go:build integration

package sandboxv2_test

import (
	"context"
	"testing"
	"time"

	sandboxv2 "github.com/yaoapp/yao/agent/sandbox/v2"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	sandbox "github.com/yaoapp/yao/sandbox/v2"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
	"github.com/yaoapp/yao/unit-test/agent/testprepare/sandboxtest"
)

func intCleanupComputer(t *testing.T, m *sandbox.Manager, cfg *types.SandboxConfig) {
	t.Helper()
	if cfg.ID == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := m.Remove(ctx, cfg.ID); err != nil {
		t.Logf("cleanup Remove(%s): %v", cfg.ID, err)
	}
}

func TestIntegration_SelectNode_BuildSnapshot(t *testing.T) {
	testprepare.PrepareSandbox(t)

	nodes := sandboxv2.BuildNodeSnapshot()
	if len(nodes) == 0 {
		t.Fatal("BuildNodeSnapshot returned no nodes")
	}

	hasLocal := false
	for _, n := range nodes {
		if n.IsLocal {
			hasLocal = true
			break
		}
	}
	if !hasLocal {
		t.Error("expected at least one local node in snapshot")
	}
}

func TestIntegration_SelectNode_BoxMode(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	nodes := sandboxv2.BuildNodeSnapshot()
	sel, err := sandboxv2.SelectNode(nodes, &sandboxv2.SelectionCriteria{
		Preferred: "claude",
		Allowed:   []string{"claude", "opencode"},
		Image:     sandboxtest.TestImage(),
	})
	if err != nil {
		t.Fatalf("SelectNode: %v", err)
	}
	if sel.Mode != "box" {
		t.Errorf("mode = %q, want box", sel.Mode)
	}
	if sel.Runner == "" {
		t.Error("runner should not be empty")
	}
}

func TestIntegration_SelectNode_HostMode(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireHostExec(t)

	nodes := sandboxv2.BuildNodeSnapshot()
	sel, err := sandboxv2.SelectNode(nodes, &sandboxv2.SelectionCriteria{
		Preferred: "claude",
		Allowed:   []string{"claude"},
	})
	if err != nil {
		t.Fatalf("SelectNode: %v", err)
	}
	if sel.Mode != "host" && sel.Mode != "local" {
		t.Errorf("mode = %q, want host or local", sel.Mode)
	}
}

func TestIntegration_E2E_SelectAndGetComputer(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	m := sandbox.M()
	nodeID := sandboxtest.TaiIDFromAddr(sandboxtest.TestLocalAddr())
	sandboxtest.EnsureImage(t, m, nodeID)

	nodes := sandboxv2.BuildNodeSnapshot()
	sel, err := sandboxv2.SelectNode(nodes, &sandboxv2.SelectionCriteria{
		Preferred: "claude",
		Allowed:   []string{"claude", "opencode"},
		Image:     sandboxtest.TestImage(),
	})
	if err != nil {
		t.Fatalf("SelectNode: %v", err)
	}

	cfg := &types.SandboxConfig{
		Version:   "2.0",
		Lifecycle: "oneshot",
		Computer:  types.ComputerConfig{Image: sandboxtest.TestImage()},
	}
	ctx := makeAgentCtx("team-int", "", "c-int", "a-int", nil)

	computer, _, err := sandboxv2.GetComputer(ctx, cfg, m, sel)
	if err != nil {
		t.Fatalf("GetComputer: %v", err)
	}
	defer func() {
		if cfg.ID != "" {
			intCleanupComputer(t, m, cfg)
		}
	}()

	if computer == nil {
		t.Fatal("computer should not be nil")
	}
	info := computer.ComputerInfo()
	if info.Kind == "" {
		t.Error("ComputerInfo.Kind should not be empty")
	}
}

func TestIntegration_E2E_FilterByOS(t *testing.T) {
	testprepare.PrepareSandbox(t)

	nodes := sandboxv2.BuildNodeSnapshot()
	_, err := sandboxv2.SelectNode(nodes, &sandboxv2.SelectionCriteria{
		Preferred: "claude",
		Allowed:   []string{"claude"},
		Filter:    &types.ComputerFilter{OS: "totally-fake-os"},
	})
	if err == nil {
		t.Error("expected error for impossible OS filter")
	}
}
