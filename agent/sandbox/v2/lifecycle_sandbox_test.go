package sandboxv2_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	sandboxv2 "github.com/yaoapp/yao/agent/sandbox/v2"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	sandbox "github.com/yaoapp/yao/sandbox/v2"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
	"github.com/yaoapp/yao/unit-test/agent/testprepare/sandboxtest"
	"github.com/yaoapp/yao/workspace"
)

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

func cleanupComputer(t *testing.T, m *sandbox.Manager, cfg *types.SandboxConfig) {
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

func TestGetComputer_BoxCreate(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	m := sandbox.M()
	nodeID := sandboxtest.TaiIDFromAddr(sandboxtest.TestLocalAddr())
	sandboxtest.EnsureImage(t, m, nodeID)

	wsID := fmt.Sprintf("lc-create-%d", time.Now().UnixNano())
	createTestWorkspace(t, nodeID, wsID)

	cfg := &types.SandboxConfig{
		Version:   "2.0",
		Lifecycle: "oneshot",
		Computer:  types.ComputerConfig{Image: sandboxtest.TestImage()},
		NodeID:    nodeID,
	}
	meta := map[string]any{"workspace_id": wsID}
	ctx := makeAgentCtx("team-t1", "", "chat-1", "ast-1", meta)

	computer, identifier, err := sandboxv2.GetComputer(ctx, cfg, m, "claude", "box")
	if err != nil {
		t.Fatalf("GetComputer: %v", err)
	}
	defer cleanupComputer(t, m, cfg)

	if identifier == "" {
		t.Fatal("oneshot should get a random identifier, got empty")
	}

	info := computer.ComputerInfo()
	if info.Kind != "box" {
		t.Errorf("kind = %q, want %q", info.Kind, "box")
	}
	if cfg.Owner != "team-t1" {
		t.Errorf("cfg.Owner = %q, want %q", cfg.Owner, "team-t1")
	}
	if cfg.Kind != "box" {
		t.Errorf("cfg.Kind = %q, want %q", cfg.Kind, "box")
	}
}

func TestGetComputer_BoxReuse(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	m := sandbox.M()
	nodeID := sandboxtest.TaiIDFromAddr(sandboxtest.TestLocalAddr())
	sandboxtest.EnsureImage(t, m, nodeID)

	wsID := fmt.Sprintf("lc-reuse-%d", time.Now().UnixNano())
	createTestWorkspace(t, nodeID, wsID)

	cfg := &types.SandboxConfig{
		Version:   "2.0",
		Lifecycle: "session",
		Computer:  types.ComputerConfig{Image: sandboxtest.TestImage()},
		NodeID:    nodeID,
	}
	meta := map[string]any{"workspace_id": wsID}
	ctx := makeAgentCtx("team-reuse", "", "chat-reuse", "ast-1", meta)

	computer1, id1, err := sandboxv2.GetComputer(ctx, cfg, m, "claude", "box")
	if err != nil {
		t.Fatalf("first GetComputer: %v", err)
	}
	defer cleanupComputer(t, m, cfg)

	cfg2 := &types.SandboxConfig{
		Version:   "2.0",
		Lifecycle: "session",
		Computer:  types.ComputerConfig{Image: sandboxtest.TestImage()},
		NodeID:    nodeID,
	}
	computer2, id2, err := sandboxv2.GetComputer(ctx, cfg2, m, "claude", "box")
	if err != nil {
		t.Fatalf("second GetComputer: %v", err)
	}

	if id1 != id2 {
		t.Errorf("identifiers differ: %q vs %q", id1, id2)
	}

	info1 := computer1.ComputerInfo()
	info2 := computer2.ComputerInfo()
	if info1.ContainerID != info2.ContainerID {
		t.Errorf("container IDs differ: %q vs %q (should reuse)", info1.ContainerID, info2.ContainerID)
	}
}

func TestGetComputer_HostMode(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireHostExec(t)

	m := sandbox.M()
	hostTaiID := sandboxtest.TaiIDFromAddr(sandboxtest.HostExecAddr())

	cfg := &types.SandboxConfig{
		Version:   "2.0",
		Lifecycle: "session",
		Computer:  types.ComputerConfig{},
		NodeID:    hostTaiID,
	}
	ctx := makeAgentCtx("team-host", "", "chat-host", "ast-host", nil)

	computer, _, err := sandboxv2.GetComputer(ctx, cfg, m, "claude", "box")
	if err != nil {
		t.Fatalf("GetComputer host: %v", err)
	}

	if cfg.Kind != "host" {
		t.Errorf("Kind = %q, want %q", cfg.Kind, "host")
	}

	info := computer.ComputerInfo()
	if info.Kind != "host" {
		t.Errorf("ComputerInfo.Kind = %q, want %q", info.Kind, "host")
	}
}

// TestGetComputer_AutoPickNode verifies that when NodeID is empty and no
// computer_id is in metadata, GetComputer auto-selects a node matching the
// requested runner/mode from the registry. In a fully-initialized environment
// with tunneled Tai nodes (Docker + HostExec), this should succeed and
// populate cfg.NodeID with the picked node.
func TestGetComputer_AutoPickNode(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	m := sandbox.M()
	cfg := &types.SandboxConfig{
		Version:   "2.0",
		Lifecycle: "session",
		Computer:  types.ComputerConfig{Image: sandboxtest.TestImage()},
		NodeID:    "",
	}
	ctx := makeAgentCtx("team-auto", "", "c", "a", nil)

	computer, _, err := sandboxv2.GetComputer(ctx, cfg, m, "claude", "box")
	if err != nil {
		t.Fatalf("GetComputer auto-pick: %v", err)
	}
	defer cleanupComputer(t, m, cfg)

	if cfg.NodeID == "" {
		t.Fatal("expected cfg.NodeID to be populated by auto-pick")
	}
	if computer == nil {
		t.Fatal("expected non-nil computer from auto-pick")
	}
}

func TestLifecycleAction_Oneshot(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	m := sandbox.M()
	nodeID := sandboxtest.TaiIDFromAddr(sandboxtest.TestLocalAddr())
	sandboxtest.EnsureImage(t, m, nodeID)

	wsID := fmt.Sprintf("lc-oneshot-%d", time.Now().UnixNano())
	createTestWorkspace(t, nodeID, wsID)

	cfg := &types.SandboxConfig{
		Version:   "2.0",
		Lifecycle: "oneshot",
		Computer:  types.ComputerConfig{Image: sandboxtest.TestImage()},
		NodeID:    nodeID,
	}
	ctx := makeAgentCtx("team-oneshot", "", "c", "a", map[string]any{"workspace_id": wsID})

	computer, _, err := sandboxv2.GetComputer(ctx, cfg, m, "claude", "box")
	if err != nil {
		t.Fatalf("GetComputer: %v", err)
	}

	boxID := cfg.ID
	sandboxv2.LifecycleAction(context.Background(), cfg, computer, m)

	_, getErr := m.Get(context.Background(), boxID)
	if getErr == nil {
		t.Error("box should be removed after oneshot LifecycleAction")
	}
}

func TestLifecycleAction_Session(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	m := sandbox.M()
	nodeID := sandboxtest.TaiIDFromAddr(sandboxtest.TestLocalAddr())
	sandboxtest.EnsureImage(t, m, nodeID)

	wsID := fmt.Sprintf("lc-sess-%d", time.Now().UnixNano())
	createTestWorkspace(t, nodeID, wsID)

	cfg := &types.SandboxConfig{
		Version:   "2.0",
		Lifecycle: "session",
		Computer:  types.ComputerConfig{Image: sandboxtest.TestImage()},
		NodeID:    nodeID,
	}
	ctx := makeAgentCtx("team-sess", "", "chat-sess", "ast-sess", map[string]any{"workspace_id": wsID})

	computer, _, err := sandboxv2.GetComputer(ctx, cfg, m, "claude", "box")
	if err != nil {
		t.Fatalf("GetComputer: %v", err)
	}
	defer cleanupComputer(t, m, cfg)

	sandboxv2.LifecycleAction(context.Background(), cfg, computer, m)

	box, err := m.Get(context.Background(), cfg.ID)
	if err != nil {
		t.Fatalf("box should still exist after session LifecycleAction: %v", err)
	}
	if box == nil {
		t.Fatal("box is nil after session LifecycleAction")
	}
}

func TestLifecycleAction_StopAndResume(t *testing.T) {
	testprepare.PrepareSandbox(t)
	sandboxtest.RequireDocker(t)

	m := sandbox.M()
	nodeID := sandboxtest.TaiIDFromAddr(sandboxtest.TestLocalAddr())
	sandboxtest.EnsureImage(t, m, nodeID)

	wsID := fmt.Sprintf("lc-resume-%d", time.Now().UnixNano())
	createTestWorkspace(t, nodeID, wsID)

	cfg := &types.SandboxConfig{
		Version:   "2.0",
		Lifecycle: "persistent",
		Computer:  types.ComputerConfig{Image: sandboxtest.TestImage()},
		NodeID:    nodeID,
	}
	ctx := makeAgentCtx("team-resume", "", "chat-resume", "ast-resume", map[string]any{"workspace_id": wsID})

	computer, _, err := sandboxv2.GetComputer(ctx, cfg, m, "claude", "box")
	if err != nil {
		t.Fatalf("GetComputer: %v", err)
	}
	defer cleanupComputer(t, m, cfg)

	sandboxv2.LifecycleAction(context.Background(), cfg, computer, m)

	box, err := m.Get(context.Background(), cfg.ID)
	if err != nil {
		t.Fatalf("box should exist after persistent action: %v", err)
	}

	stopCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := box.Stop(stopCtx); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	if err := box.Start(stopCtx); err != nil {
		t.Fatalf("Start (resume): %v", err)
	}

	res, err := box.Exec(stopCtx, []string{"echo", "resumed"})
	if err != nil {
		t.Fatalf("Exec after resume: %v", err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("exit code %d after resume", res.ExitCode)
	}
}
