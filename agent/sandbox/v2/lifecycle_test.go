package sandboxv2_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	agentContext "github.com/yaoapp/yao/agent/context"
	sandboxv2 "github.com/yaoapp/yao/agent/sandbox/v2"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
	infra "github.com/yaoapp/yao/sandbox/v2"
)

// ===========================================================================
// BuildIdentifier — pure-function tests (no infra needed)
// ===========================================================================

func TestBuildIdentifier_Oneshot(t *testing.T) {
	cfg := &types.SandboxConfig{Lifecycle: "oneshot"}
	id := sandboxv2.BuildIdentifier(cfg, "owner1", "chat1", "ast1", "ws1", nil)
	if id != "" {
		t.Errorf("oneshot should return empty, got %q", id)
	}
}

func TestBuildIdentifier_Session(t *testing.T) {
	cfg := &types.SandboxConfig{Lifecycle: "session"}
	id := sandboxv2.BuildIdentifier(cfg, "owner1", "chat42", "ast1", "ws1", nil)
	if id != "owner1-ast1-chat42" {
		t.Errorf("session: got %q, want %q", id, "owner1-ast1-chat42")
	}
}

func TestBuildIdentifier_Longrunning(t *testing.T) {
	cfg := &types.SandboxConfig{Lifecycle: "longrunning"}
	id := sandboxv2.BuildIdentifier(cfg, "owner1", "chat1", "ast99", "ws1", nil)
	if id != "owner1-ast99.ws1" {
		t.Errorf("longrunning: got %q, want %q", id, "owner1-ast99.ws1")
	}
}

func TestBuildIdentifier_Persistent(t *testing.T) {
	cfg := &types.SandboxConfig{Lifecycle: "persistent"}
	id := sandboxv2.BuildIdentifier(cfg, "owner1", "chat1", "ast99", "ws1", nil)
	if id != "owner1-ast99.ws1" {
		t.Errorf("persistent: got %q, want %q", id, "owner1-ast99.ws1")
	}
}

func TestBuildIdentifier_MetadataOverride(t *testing.T) {
	cfg := &types.SandboxConfig{Lifecycle: "session"}
	meta := map[string]any{"computer_id": "custom-box"}
	id := sandboxv2.BuildIdentifier(cfg, "owner1", "chat1", "ast1", "ws1", meta)
	// computer_id is used for routing only, not for identifier generation.
	if id != "owner1-ast1-chat1" {
		t.Errorf("metadata override: got %q, want %q", id, "owner1-ast1-chat1")
	}
}

func TestBuildIdentifier_MetadataEmptyIgnored(t *testing.T) {
	cfg := &types.SandboxConfig{Lifecycle: "session"}
	meta := map[string]any{"computer_id": ""}
	id := sandboxv2.BuildIdentifier(cfg, "owner1", "chat42", "ast1", "ws1", meta)
	if id != "owner1-ast1-chat42" {
		t.Errorf("empty metadata should fall through to session, got %q", id)
	}
}

func TestBuildIdentifier_UnknownLifecycle(t *testing.T) {
	cfg := &types.SandboxConfig{Lifecycle: "unknown"}
	id := sandboxv2.BuildIdentifier(cfg, "owner1", "chat1", "ast1", "ws1", nil)
	if id != "" {
		t.Errorf("unknown lifecycle should return empty, got %q", id)
	}
}

// ===========================================================================
// GetComputer — real container tests
// ===========================================================================

func makeAgentCtx(teamID, userID, chatID, assistantID string, metadata map[string]any) *agentContext.Context {
	var auth *oauthTypes.AuthorizedInfo
	if teamID != "" || userID != "" {
		auth = &oauthTypes.AuthorizedInfo{TeamID: teamID, UserID: userID}
	}
	return &agentContext.Context{
		Context:     context.Background(),
		Authorized:  auth,
		ChatID:      chatID,
		AssistantID: assistantID,
		Metadata:    metadata,
	}
}

func TestGetComputer_BoxCreate(t *testing.T) {
	skipIfNoDocker(t)

	for _, nc := range boxNodes() {
		nc := nc
		t.Run(nc.Name, func(t *testing.T) {
			m := setupManager(t, &nc)
			ensureImage(t, m, nc)

			wsID := fmt.Sprintf("lc-create-%d", time.Now().UnixNano())
			createTestWorkspace(t, nc.TaiID, wsID)

			cfg := &types.SandboxConfig{
				Version:   "2.0",
				Lifecycle: "oneshot",
				Computer:  types.ComputerConfig{Image: testImage()},
				NodeID:    nc.TaiID,
			}
			meta := map[string]any{"workspace_id": wsID}
			ctx := makeAgentCtx("team-t1", "", "chat-1", "ast-1", meta)

			computer, identifier, err := sandboxv2.GetComputer(ctx, cfg, m)
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
		})
	}
}

func TestGetComputer_BoxReuse(t *testing.T) {
	skipIfNoDocker(t)

	for _, nc := range boxNodes() {
		nc := nc
		t.Run(nc.Name, func(t *testing.T) {
			m := setupManager(t, &nc)
			ensureImage(t, m, nc)

			wsID := fmt.Sprintf("lc-reuse-%d", time.Now().UnixNano())
			createTestWorkspace(t, nc.TaiID, wsID)

			cfg := &types.SandboxConfig{
				Version:   "2.0",
				Lifecycle: "session",
				Computer:  types.ComputerConfig{Image: testImage()},
				NodeID:    nc.TaiID,
			}
			meta := map[string]any{"workspace_id": wsID}
			ctx := makeAgentCtx("team-reuse", "", "chat-reuse", "ast-1", meta)

			computer1, id1, err := sandboxv2.GetComputer(ctx, cfg, m)
			if err != nil {
				t.Fatalf("first GetComputer: %v", err)
			}
			defer cleanupComputer(t, m, cfg)

			cfg2 := &types.SandboxConfig{
				Version:   "2.0",
				Lifecycle: "session",
				Computer:  types.ComputerConfig{Image: testImage()},
				NodeID:    nc.TaiID,
			}
			computer2, id2, err := sandboxv2.GetComputer(ctx, cfg2, m)
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
		})
	}
}

func TestGetComputer_WorkspaceBindAlways(t *testing.T) {
	skipIfNoDocker(t)

	for _, nc := range boxNodes() {
		nc := nc
		t.Run(nc.Name, func(t *testing.T) {
			m := setupManager(t, &nc)
			ensureImage(t, m, nc)

			wsID := fmt.Sprintf("lc-ws-%d", time.Now().UnixNano())
			createTestWorkspace(t, nc.TaiID, wsID)

			cfg := &types.SandboxConfig{
				Version:   "2.0",
				Lifecycle: "oneshot",
				Computer:  types.ComputerConfig{Image: testImage()},
				NodeID:    nc.TaiID,
			}
			meta := map[string]any{"workspace_id": wsID}
			ctx := makeAgentCtx("team-ws", "", "chat-ws", "ast-ws", meta)

			computer, _, err := sandboxv2.GetComputer(ctx, cfg, m)
			if err != nil {
				t.Fatalf("GetComputer: %v", err)
			}
			defer cleanupComputer(t, m, cfg)

			if cfg.WorkspaceID != wsID {
				t.Errorf("WorkspaceID = %q, want %q", cfg.WorkspaceID, wsID)
			}

			ws := computer.Workplace()
			if ws == nil {
				t.Fatal("Workplace() returned nil, workspace should always be bound")
			}
		})
	}
}

func TestGetComputer_WorkspaceFallbackOwner(t *testing.T) {
	skipIfNoDocker(t)

	for _, nc := range boxNodes() {
		nc := nc
		t.Run(nc.Name, func(t *testing.T) {
			m := setupManager(t, &nc)
			ensureImage(t, m, nc)

			ownerID := fmt.Sprintf("lc-owner-%d", time.Now().UnixNano())
			createTestWorkspace(t, nc.TaiID, ownerID)

			cfg := &types.SandboxConfig{
				Version:   "2.0",
				Lifecycle: "oneshot",
				Computer:  types.ComputerConfig{Image: testImage()},
				NodeID:    nc.TaiID,
			}
			ctx := makeAgentCtx(ownerID, "", "chat-fb", "ast-fb", nil)

			computer, _, err := sandboxv2.GetComputer(ctx, cfg, m)
			if err != nil {
				t.Fatalf("GetComputer: %v", err)
			}
			defer cleanupComputer(t, m, cfg)

			if cfg.WorkspaceID != ownerID {
				t.Errorf("WorkspaceID = %q, want %q (should fallback to ownerID)", cfg.WorkspaceID, ownerID)
			}

			ws := computer.Workplace()
			if ws == nil {
				t.Fatal("Workplace() returned nil")
			}
		})
	}
}

func TestGetComputer_OwnerPriority(t *testing.T) {
	skipIfNoDocker(t)

	nc := boxNodes()[0]
	m := setupManager(t, &nc)
	ensureImage(t, m, nc)

	t.Run("teamID", func(t *testing.T) {
		wsID := fmt.Sprintf("lc-ownp-team-%d", time.Now().UnixNano())
		createTestWorkspace(t, nc.TaiID, wsID)

		cfg := &types.SandboxConfig{
			Version: "2.0", Lifecycle: "oneshot",
			Computer: types.ComputerConfig{Image: testImage()},
			NodeID:   nc.TaiID,
		}
		ctx := makeAgentCtx("my-team", "my-user", "c", "a", map[string]any{"workspace_id": wsID})
		_, _, err := sandboxv2.GetComputer(ctx, cfg, m)
		if err != nil {
			t.Fatalf("GetComputer: %v", err)
		}
		defer cleanupComputer(t, m, cfg)
		if cfg.Owner != "my-team" {
			t.Errorf("Owner = %q, want %q (teamID takes precedence)", cfg.Owner, "my-team")
		}
	})

	t.Run("userID", func(t *testing.T) {
		wsID := fmt.Sprintf("lc-ownp-user-%d", time.Now().UnixNano())
		createTestWorkspace(t, nc.TaiID, wsID)

		cfg := &types.SandboxConfig{
			Version: "2.0", Lifecycle: "oneshot",
			Computer: types.ComputerConfig{Image: testImage()},
			NodeID:   nc.TaiID,
		}
		ctx := makeAgentCtx("", "my-user", "c", "a", map[string]any{"workspace_id": wsID})
		_, _, err := sandboxv2.GetComputer(ctx, cfg, m)
		if err != nil {
			t.Fatalf("GetComputer: %v", err)
		}
		defer cleanupComputer(t, m, cfg)
		if cfg.Owner != "my-user" {
			t.Errorf("Owner = %q, want %q", cfg.Owner, "my-user")
		}
	})

	t.Run("anonymous", func(t *testing.T) {
		wsID := fmt.Sprintf("lc-ownp-anon-%d", time.Now().UnixNano())
		createTestWorkspace(t, nc.TaiID, wsID)

		cfg := &types.SandboxConfig{
			Version: "2.0", Lifecycle: "oneshot",
			Computer: types.ComputerConfig{Image: testImage()},
			NodeID:   nc.TaiID,
		}
		ctx := makeAgentCtx("", "", "c", "a", map[string]any{"workspace_id": wsID})
		_, _, err := sandboxv2.GetComputer(ctx, cfg, m)
		if err != nil {
			t.Fatalf("GetComputer: %v", err)
		}
		defer cleanupComputer(t, m, cfg)
		if cfg.Owner != "anonymous" {
			t.Errorf("Owner = %q, want %q", cfg.Owner, "anonymous")
		}
	})
}

func TestGetComputer_HostMode(t *testing.T) {
	skipIfNoHostExec(t)

	for _, tgt := range hostTargets() {
		tgt := tgt
		t.Run(tgt.Name, func(t *testing.T) {
			m := setupHostManager(t, &tgt)

			cfg := &types.SandboxConfig{
				Version:   "2.0",
				Lifecycle: "session",
				Computer:  types.ComputerConfig{},
				NodeID:    tgt.TaiID,
			}
			ctx := makeAgentCtx("team-host", "", "chat-host", "ast-host", nil)

			computer, _, err := sandboxv2.GetComputer(ctx, cfg, m)
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

			ws := computer.Workplace()
			if ws == nil {
				t.Fatal("Workplace() returned nil on host mode")
			}
		})
	}
}

func TestGetComputer_HostMissingNodeID(t *testing.T) {
	skipIfNoDocker(t)

	nc := boxNodes()[0]
	m := setupManager(t, &nc)

	cfg := &types.SandboxConfig{
		Version:   "2.0",
		Lifecycle: "session",
		Computer:  types.ComputerConfig{},
		NodeID:    "",
	}
	ctx := makeAgentCtx("team-err", "", "c", "a", nil)

	_, _, err := sandboxv2.GetComputer(ctx, cfg, m)
	if err == nil {
		t.Fatal("expected error for host mode without nodeID")
	}
	if !strings.Contains(err.Error(), "nodeID") {
		t.Errorf("error should mention nodeID, got: %v", err)
	}
}

// ===========================================================================
// LifecycleAction — behavior tests
// ===========================================================================

func TestLifecycleAction_Oneshot(t *testing.T) {
	skipIfNoDocker(t)

	for _, nc := range boxNodes() {
		nc := nc
		t.Run(nc.Name, func(t *testing.T) {
			m := setupManager(t, &nc)
			ensureImage(t, m, nc)

			wsID := fmt.Sprintf("lc-oneshot-%d", time.Now().UnixNano())
			createTestWorkspace(t, nc.TaiID, wsID)

			cfg := &types.SandboxConfig{
				Version:   "2.0",
				Lifecycle: "oneshot",
				Computer:  types.ComputerConfig{Image: testImage()},
				NodeID:    nc.TaiID,
			}
			ctx := makeAgentCtx("team-oneshot", "", "c", "a", map[string]any{"workspace_id": wsID})

			computer, _, err := sandboxv2.GetComputer(ctx, cfg, m)
			if err != nil {
				t.Fatalf("GetComputer: %v", err)
			}

			boxID := cfg.ID

			sandboxv2.LifecycleAction(context.Background(), cfg, computer, m)

			_, getErr := m.Get(context.Background(), boxID)
			if getErr == nil {
				t.Error("box should be removed after oneshot LifecycleAction")
			}
		})
	}
}

func TestLifecycleAction_Session(t *testing.T) {
	skipIfNoDocker(t)

	for _, nc := range boxNodes() {
		nc := nc
		t.Run(nc.Name, func(t *testing.T) {
			m := setupManager(t, &nc)
			ensureImage(t, m, nc)

			wsID := fmt.Sprintf("lc-sess-%d", time.Now().UnixNano())
			createTestWorkspace(t, nc.TaiID, wsID)

			cfg := &types.SandboxConfig{
				Version:   "2.0",
				Lifecycle: "session",
				Computer:  types.ComputerConfig{Image: testImage()},
				NodeID:    nc.TaiID,
			}
			ctx := makeAgentCtx("team-sess", "", "chat-sess", "ast-sess", map[string]any{"workspace_id": wsID})

			computer, _, err := sandboxv2.GetComputer(ctx, cfg, m)
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
		})
	}
}

func TestLifecycleAction_Persistent(t *testing.T) {
	skipIfNoDocker(t)

	for _, nc := range boxNodes() {
		nc := nc
		t.Run(nc.Name, func(t *testing.T) {
			m := setupManager(t, &nc)
			ensureImage(t, m, nc)

			wsID := fmt.Sprintf("lc-pers-%d", time.Now().UnixNano())
			createTestWorkspace(t, nc.TaiID, wsID)

			cfg := &types.SandboxConfig{
				Version:   "2.0",
				Lifecycle: "persistent",
				Computer:  types.ComputerConfig{Image: testImage()},
				NodeID:    nc.TaiID,
			}
			ctx := makeAgentCtx("team-pers", "", "chat-pers", "ast-pers", map[string]any{"workspace_id": wsID})

			computer, _, err := sandboxv2.GetComputer(ctx, cfg, m)
			if err != nil {
				t.Fatalf("GetComputer: %v", err)
			}
			defer cleanupComputer(t, m, cfg)

			sandboxv2.LifecycleAction(context.Background(), cfg, computer, m)

			box, err := m.Get(context.Background(), cfg.ID)
			if err != nil {
				t.Fatalf("box should still exist after persistent LifecycleAction: %v", err)
			}
			if box == nil {
				t.Fatal("box is nil after persistent LifecycleAction")
			}
		})
	}
}

func TestLifecycleAction_NilSafe(t *testing.T) {
	cfg := &types.SandboxConfig{Lifecycle: "oneshot"}
	sandboxv2.LifecycleAction(context.Background(), cfg, nil, nil)
	sandboxv2.LifecycleAction(context.Background(), nil, nil, nil)
}

// ===========================================================================
// helpers
// ===========================================================================

func ensureImage(t *testing.T, m *infra.Manager, nc nodeConfig) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := m.EnsureImage(ctx, nc.TaiID, testImage(), infra.ImagePullOptions{}); err != nil {
		t.Fatalf("EnsureImage: %v", err)
	}
}

func cleanupComputer(t *testing.T, m *infra.Manager, cfg *types.SandboxConfig) {
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
