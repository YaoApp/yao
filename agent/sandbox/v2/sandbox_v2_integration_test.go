//go:build integration

package sandboxv2_test

import (
	"testing"

	sandboxv2 "github.com/yaoapp/yao/agent/sandbox/v2"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
	"github.com/yaoapp/yao/unit-test/agent/testprepare/sandboxtest"
)

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
