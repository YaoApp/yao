//go:build unit

package sandboxv2_test

import (
	"context"
	"testing"

	sandboxv2 "github.com/yaoapp/yao/agent/sandbox/v2"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func localNode(runners []string, canBox, canHost bool) sandboxv2.NodeCandidate {
	return sandboxv2.NodeCandidate{
		ID: "local", IsLocal: true, Runners: runners,
		CanBox: canBox, CanHost: canHost, OS: "linux", Arch: "amd64",
	}
}

func remoteNode(id string, runners []string, canBox, canHost bool) sandboxv2.NodeCandidate {
	return sandboxv2.NodeCandidate{
		ID: id, IsLocal: false, Runners: runners,
		CanBox: canBox, CanHost: canHost, OS: "linux", Arch: "amd64",
	}
}

// mockWSResolver implements sandboxv2.WorkspaceResolver for testing.
type mockWSResolver struct {
	nodeID string
	err    error
}

func (m *mockWSResolver) NodeForWorkspace(_ context.Context, _ string) (string, error) {
	return m.nodeID, m.err
}

// --- SelectNode decision table ---

func TestSelectNode_S1_WorkspaceBound_NoWSManager(t *testing.T) {
	testprepare.PrepareUnit(t)
	nodes := []sandboxv2.NodeCandidate{
		remoteNode("r1", []string{"claude"}, true, false),
	}
	sel, err := sandboxv2.SelectNode(nodes, &sandboxv2.SelectionCriteria{
		WorkspaceID: "ws-1",
		Preferred:   "claude",
		Allowed:     []string{"claude"},
		Image:       "ubuntu",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sel.Runner != "claude" || sel.Mode != "box" {
		t.Errorf("got runner=%q mode=%q, want claude/box (WSManager=nil -> skip STEP 0)", sel.Runner, sel.Mode)
	}
}

func TestSelectNode_S1_WorkspaceBound_HitsNode(t *testing.T) {
	testprepare.PrepareUnit(t)
	nodes := []sandboxv2.NodeCandidate{
		remoteNode("r1", []string{"claude"}, true, true),
		remoteNode("r2", []string{"claude", "opencode"}, true, true),
	}
	sel, err := sandboxv2.SelectNode(nodes, &sandboxv2.SelectionCriteria{
		WorkspaceID: "ws-1",
		Preferred:   "opencode",
		Allowed:     []string{"claude", "opencode"},
		Image:       "ubuntu",
		WSManager:   &mockWSResolver{nodeID: "r2"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sel.NodeID != "r2" || sel.Runner != "opencode" || sel.Mode != "box" {
		t.Errorf("got node=%q runner=%q mode=%q, want r2/opencode/box (workspace binding prefers preferred runner)", sel.NodeID, sel.Runner, sel.Mode)
	}
}

func TestSelectNode_S1_WorkspaceBound_NodeOffline(t *testing.T) {
	testprepare.PrepareUnit(t)
	nodes := []sandboxv2.NodeCandidate{
		remoteNode("r1", []string{"claude"}, true, true),
	}
	sel, err := sandboxv2.SelectNode(nodes, &sandboxv2.SelectionCriteria{
		WorkspaceID: "ws-1",
		Preferred:   "claude",
		Allowed:     []string{"claude"},
		Image:       "ubuntu",
		WSManager:   &mockWSResolver{nodeID: "r-offline"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sel.NodeID != "r1" {
		t.Errorf("got node=%q, want r1 (offline ws node -> fallthrough)", sel.NodeID)
	}
}

func TestSelectNode_S2_RemoteBox_Preferred(t *testing.T) {
	testprepare.PrepareUnit(t)
	nodes := []sandboxv2.NodeCandidate{
		remoteNode("r1", []string{"claude", "opencode"}, true, true),
	}
	sel, err := sandboxv2.SelectNode(nodes, &sandboxv2.SelectionCriteria{
		Preferred: "claude",
		Allowed:   []string{"claude", "opencode"},
		Image:     "ubuntu",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sel.NodeID != "r1" || sel.Runner != "claude" || sel.Mode != "box" {
		t.Errorf("got node=%q runner=%q mode=%q, want r1/claude/box", sel.NodeID, sel.Runner, sel.Mode)
	}
}

func TestSelectNode_S3_RemoteBox_Fallback(t *testing.T) {
	testprepare.PrepareUnit(t)
	nodes := []sandboxv2.NodeCandidate{
		remoteNode("r1", []string{"opencode"}, true, false),
	}
	sel, err := sandboxv2.SelectNode(nodes, &sandboxv2.SelectionCriteria{
		Preferred: "claude",
		Allowed:   []string{"claude", "opencode"},
		Image:     "ubuntu",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sel.Runner != "opencode" || sel.Mode != "box" {
		t.Errorf("got runner=%q mode=%q, want opencode/box", sel.Runner, sel.Mode)
	}
}

func TestSelectNode_S4_RemoteHost_ImageDowngrade(t *testing.T) {
	testprepare.PrepareUnit(t)
	nodes := []sandboxv2.NodeCandidate{
		remoteNode("r1", []string{"claude"}, false, true),
	}
	sel, err := sandboxv2.SelectNode(nodes, &sandboxv2.SelectionCriteria{
		Preferred: "claude",
		Allowed:   []string{"claude"},
		Image:     "ubuntu",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sel.Mode != "host" || sel.Runner != "claude" || sel.NodeID != "r1" {
		t.Errorf("got node=%q runner=%q mode=%q, want r1/claude/host (image degraded)", sel.NodeID, sel.Runner, sel.Mode)
	}
}

func TestSelectNode_S5_LocalHost_NoDocker(t *testing.T) {
	testprepare.PrepareUnit(t)
	nodes := []sandboxv2.NodeCandidate{
		localNode([]string{"claude", "yaocode"}, false, true),
	}
	sel, err := sandboxv2.SelectNode(nodes, &sandboxv2.SelectionCriteria{
		Preferred: "claude",
		Allowed:   []string{"claude"},
		Image:     "ubuntu",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sel.Mode != "host" || !sel.IsLocal || sel.Runner != "claude" {
		t.Errorf("got node=%q runner=%q mode=%q isLocal=%v, want local/claude/host/true", sel.NodeID, sel.Runner, sel.Mode, sel.IsLocal)
	}
}

func TestSelectNode_S6_LocalBox(t *testing.T) {
	testprepare.PrepareUnit(t)
	nodes := []sandboxv2.NodeCandidate{
		localNode([]string{"claude", "yaocode"}, true, true),
	}
	sel, err := sandboxv2.SelectNode(nodes, &sandboxv2.SelectionCriteria{
		Preferred: "claude",
		Allowed:   []string{"claude"},
		Image:     "ubuntu",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sel.Mode != "box" || !sel.IsLocal || sel.Runner != "claude" {
		t.Errorf("got node=%q runner=%q mode=%q isLocal=%v, want local/claude/box/true", sel.NodeID, sel.Runner, sel.Mode, sel.IsLocal)
	}
}

func TestSelectNode_S7_LocalDisabled(t *testing.T) {
	testprepare.PrepareUnit(t)
	nodes := []sandboxv2.NodeCandidate{
		localNode([]string{}, false, false),
	}
	_, err := sandboxv2.SelectNode(nodes, &sandboxv2.SelectionCriteria{
		Preferred: "claude",
		Allowed:   []string{"claude"},
		Image:     "ubuntu",
	})
	if err == nil {
		t.Fatal("expected error for disabled local node")
	}
}

func TestSelectNode_S8_RemoteHost_Preferred(t *testing.T) {
	testprepare.PrepareUnit(t)
	nodes := []sandboxv2.NodeCandidate{
		remoteNode("r1", []string{"claude", "opencode"}, false, true),
	}
	sel, err := sandboxv2.SelectNode(nodes, &sandboxv2.SelectionCriteria{
		Preferred: "claude",
		Allowed:   []string{"claude", "opencode"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sel.Runner != "claude" || sel.Mode != "host" {
		t.Errorf("got runner=%q mode=%q, want claude/host", sel.Runner, sel.Mode)
	}
}

func TestSelectNode_S9a_LocalYaocode(t *testing.T) {
	testprepare.PrepareUnit(t)
	nodes := []sandboxv2.NodeCandidate{
		localNode([]string{"yaocode", "claude"}, false, true),
	}
	sel, err := sandboxv2.SelectNode(nodes, &sandboxv2.SelectionCriteria{
		Preferred: "yaocode",
		Allowed:   []string{"yaocode", "claude"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sel.Mode != "local" || sel.Runner != "yaocode" {
		t.Errorf("got mode=%q runner=%q, want local/yaocode", sel.Mode, sel.Runner)
	}
}

func TestSelectNode_S9b_LocalClaude(t *testing.T) {
	testprepare.PrepareUnit(t)
	nodes := []sandboxv2.NodeCandidate{
		localNode([]string{"yaocode", "claude"}, false, true),
	}
	sel, err := sandboxv2.SelectNode(nodes, &sandboxv2.SelectionCriteria{
		Preferred: "claude",
		Allowed:   []string{"claude"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sel.Mode != "host" || sel.Runner != "claude" {
		t.Errorf("got mode=%q runner=%q, want host/claude", sel.Mode, sel.Runner)
	}
}

func TestSelectNode_S10_NoNodes(t *testing.T) {
	testprepare.PrepareUnit(t)
	_, err := sandboxv2.SelectNode(nil, &sandboxv2.SelectionCriteria{
		Preferred: "claude",
		Allowed:   []string{"claude"},
	})
	if err == nil {
		t.Fatal("expected error for empty nodes")
	}
}

func TestSelectNode_S11_UserPick_Available(t *testing.T) {
	testprepare.PrepareUnit(t)
	nodes := []sandboxv2.NodeCandidate{
		localNode([]string{"yaocode", "claude", "opencode"}, false, true),
	}
	sel, err := sandboxv2.SelectNode(nodes, &sandboxv2.SelectionCriteria{
		Preferred: "claude",
		Allowed:   []string{"claude"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sel.Runner != "claude" || sel.Mode != "host" || !sel.IsLocal {
		t.Errorf("got runner=%q mode=%q isLocal=%v, want claude/host/true", sel.Runner, sel.Mode, sel.IsLocal)
	}
}

func TestSelectNode_S12_UserPick_Unavailable(t *testing.T) {
	testprepare.PrepareUnit(t)
	nodes := []sandboxv2.NodeCandidate{
		localNode([]string{"yaocode"}, false, true),
	}
	_, err := sandboxv2.SelectNode(nodes, &sandboxv2.SelectionCriteria{
		Preferred: "claude",
		Allowed:   []string{"claude"},
	})
	if err == nil {
		t.Fatal("expected error when requested runner is unavailable")
	}
}

func TestSelectNode_S13_GroupPriority(t *testing.T) {
	testprepare.PrepareUnit(t)
	nodes := []sandboxv2.NodeCandidate{
		localNode([]string{"claude"}, false, true),
		remoteNode("r1", []string{"claude"}, true, false),
	}
	sel, err := sandboxv2.SelectNode(nodes, &sandboxv2.SelectionCriteria{
		Preferred: "claude",
		Allowed:   []string{"claude"},
		Image:     "ubuntu",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sel.NodeID != "r1" || sel.Mode != "box" || sel.Runner != "claude" {
		t.Errorf("got node=%q runner=%q mode=%q, want r1/claude/box (remote box > local host)", sel.NodeID, sel.Runner, sel.Mode)
	}
}

func TestSelectNode_Filter_Arch(t *testing.T) {
	testprepare.PrepareUnit(t)
	nodes := []sandboxv2.NodeCandidate{
		{ID: "arm", Runners: []string{"claude"}, CanHost: true, OS: "linux", Arch: "arm64"},
		{ID: "x86", Runners: []string{"claude"}, CanHost: true, OS: "linux", Arch: "amd64"},
	}
	sel, err := sandboxv2.SelectNode(nodes, &sandboxv2.SelectionCriteria{
		Preferred: "claude",
		Allowed:   []string{"claude"},
		Filter:    &types.ComputerFilter{Arch: "arm64"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sel.NodeID != "arm" {
		t.Errorf("got node=%q, want arm (filtered by Arch)", sel.NodeID)
	}
}

func TestSelectNode_Filter_OS(t *testing.T) {
	testprepare.PrepareUnit(t)
	nodes := []sandboxv2.NodeCandidate{
		{ID: "win", Runners: []string{"claude"}, CanHost: true, OS: "windows", Arch: "amd64"},
		remoteNode("lin", []string{"claude"}, false, true),
	}
	sel, err := sandboxv2.SelectNode(nodes, &sandboxv2.SelectionCriteria{
		Preferred: "claude",
		Allowed:   []string{"claude"},
		Filter:    &types.ComputerFilter{OS: "linux"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sel.NodeID != "lin" {
		t.Errorf("got node=%q, want lin (filtered by OS)", sel.NodeID)
	}
}

// --- H4 scenario: Docker-only local node ---

func TestResolveMode_LocalDockerOnly_NoImage(t *testing.T) {
	testprepare.PrepareUnit(t)
	n := localNode([]string{"claude"}, true, false)
	mode := sandboxv2.ExportResolveMode(&n, "claude", "")
	if mode != "box" {
		t.Errorf("got %q, want box (local Docker-only falls back to box)", mode)
	}
}

func TestResolveMode_RemoteDockerOnly_NoImage(t *testing.T) {
	testprepare.PrepareUnit(t)
	n := remoteNode("r1", []string{"claude"}, true, false)
	mode := sandboxv2.ExportResolveMode(&n, "claude", "")
	if mode != "box" {
		t.Errorf("got %q, want box (remote Docker-only falls back to box)", mode)
	}
}

func TestResolveMode_NoCapabilities(t *testing.T) {
	testprepare.PrepareUnit(t)
	n := localNode([]string{"claude"}, false, false)
	mode := sandboxv2.ExportResolveMode(&n, "claude", "")
	if mode != "" {
		t.Errorf("got %q, want empty (no feasible mode)", mode)
	}
}

// --- resolveMode ---

func TestResolveMode_LocalYaocode(t *testing.T) {
	testprepare.PrepareUnit(t)
	n := localNode([]string{"yaocode"}, true, true)
	mode := sandboxv2.ExportResolveMode(&n, "yaocode", "ubuntu")
	if mode != "local" {
		t.Errorf("got %q, want local", mode)
	}
}

func TestResolveMode_LocalClaude_NoImage(t *testing.T) {
	testprepare.PrepareUnit(t)
	n := localNode([]string{"claude"}, false, true)
	mode := sandboxv2.ExportResolveMode(&n, "claude", "")
	if mode != "host" {
		t.Errorf("got %q, want host", mode)
	}
}

func TestResolveMode_LocalClaude_WithImage(t *testing.T) {
	testprepare.PrepareUnit(t)
	n := localNode([]string{"claude"}, true, true)
	mode := sandboxv2.ExportResolveMode(&n, "claude", "ubuntu")
	if mode != "box" {
		t.Errorf("got %q, want box", mode)
	}
}

func TestResolveMode_Remote_WithImage(t *testing.T) {
	testprepare.PrepareUnit(t)
	n := remoteNode("r1", []string{"claude"}, true, false)
	mode := sandboxv2.ExportResolveMode(&n, "claude", "ubuntu")
	if mode != "box" {
		t.Errorf("got %q, want box", mode)
	}
}

func TestResolveMode_Remote_NoImage(t *testing.T) {
	testprepare.PrepareUnit(t)
	n := remoteNode("r1", []string{"claude"}, true, true)
	mode := sandboxv2.ExportResolveMode(&n, "claude", "")
	if mode != "host" {
		t.Errorf("got %q, want host", mode)
	}
}

// --- pickRunnerOnNode ---

func TestPickRunnerOnNode_FirstAllowedMatch(t *testing.T) {
	testprepare.PrepareUnit(t)
	n := remoteNode("r1", []string{"claude", "tai"}, true, false)
	runner := sandboxv2.ExportPickRunnerOnNode(&n, &sandboxv2.SelectionCriteria{
		Allowed: []string{"claude", "opencode"},
	})
	if runner != "claude" {
		t.Errorf("got %q, want claude (first in Allowed that node supports)", runner)
	}
}

func TestPickRunnerOnNode_AllowedOrderMatters(t *testing.T) {
	testprepare.PrepareUnit(t)
	n := remoteNode("r1", []string{"claude", "opencode"}, true, false)
	runner := sandboxv2.ExportPickRunnerOnNode(&n, &sandboxv2.SelectionCriteria{
		Preferred: "claude",
		Allowed:   []string{"opencode", "claude"},
	})
	if runner != "opencode" {
		t.Errorf("got %q, want opencode (Allowed order takes precedence, Preferred is ignored by pickRunnerOnNode)", runner)
	}
}

func TestPickRunnerOnNode_PreferredNotInNode(t *testing.T) {
	testprepare.PrepareUnit(t)
	n := remoteNode("r1", []string{"opencode", "tai"}, true, false)
	runner := sandboxv2.ExportPickRunnerOnNode(&n, &sandboxv2.SelectionCriteria{
		Preferred: "claude",
		Allowed:   []string{"claude", "opencode"},
	})
	if runner != "opencode" {
		t.Errorf("got %q, want opencode (fallback)", runner)
	}
}

func TestPickRunnerOnNode_NoOverlap(t *testing.T) {
	testprepare.PrepareUnit(t)
	n := remoteNode("r1", []string{"tai"}, true, false)
	runner := sandboxv2.ExportPickRunnerOnNode(&n, &sandboxv2.SelectionCriteria{
		Preferred: "claude",
		Allowed:   []string{"claude"},
	})
	if runner != "" {
		t.Errorf("got %q, want empty (no overlap)", runner)
	}
}
