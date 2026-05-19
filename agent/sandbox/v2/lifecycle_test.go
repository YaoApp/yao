//go:build unit

package sandboxv2_test

import (
	"context"
	"testing"

	sandboxv2 "github.com/yaoapp/yao/agent/sandbox/v2"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestBuildIdentifier_Oneshot(t *testing.T) {
	testprepare.PrepareUnit(t)
	cfg := &types.SandboxConfig{Lifecycle: "oneshot"}
	id := sandboxv2.BuildIdentifier(cfg, "owner1", "chat1", "ast1", "ws1", nil)
	if id != "" {
		t.Errorf("oneshot should return empty, got %q", id)
	}
}

func TestBuildIdentifier_Session(t *testing.T) {
	testprepare.PrepareUnit(t)
	cfg := &types.SandboxConfig{Lifecycle: "session"}
	id := sandboxv2.BuildIdentifier(cfg, "owner1", "chat42", "ast1", "ws1", nil)
	if id != "owner1-ast1-chat42" {
		t.Errorf("session: got %q, want %q", id, "owner1-ast1-chat42")
	}
}

func TestBuildIdentifier_Longrunning(t *testing.T) {
	testprepare.PrepareUnit(t)
	cfg := &types.SandboxConfig{Lifecycle: "longrunning"}
	id := sandboxv2.BuildIdentifier(cfg, "owner1", "chat1", "ast99", "ws1", nil)
	if id != "owner1-ast99.ws1" {
		t.Errorf("longrunning: got %q, want %q", id, "owner1-ast99.ws1")
	}
}

func TestBuildIdentifier_Persistent(t *testing.T) {
	testprepare.PrepareUnit(t)
	cfg := &types.SandboxConfig{Lifecycle: "persistent"}
	id := sandboxv2.BuildIdentifier(cfg, "owner1", "chat1", "ast99", "ws1", nil)
	if id != "owner1-ast99.ws1" {
		t.Errorf("persistent: got %q, want %q", id, "owner1-ast99.ws1")
	}
}

func TestBuildIdentifier_MetadataOverride(t *testing.T) {
	testprepare.PrepareUnit(t)
	cfg := &types.SandboxConfig{Lifecycle: "session"}
	meta := map[string]any{"computer_id": "custom-box"}
	id := sandboxv2.BuildIdentifier(cfg, "owner1", "chat1", "ast1", "ws1", meta)
	if id != "owner1-ast1-chat1" {
		t.Errorf("metadata override: got %q, want %q", id, "owner1-ast1-chat1")
	}
}

func TestBuildIdentifier_MetadataEmptyIgnored(t *testing.T) {
	testprepare.PrepareUnit(t)
	cfg := &types.SandboxConfig{Lifecycle: "session"}
	meta := map[string]any{"computer_id": ""}
	id := sandboxv2.BuildIdentifier(cfg, "owner1", "chat42", "ast1", "ws1", meta)
	if id != "owner1-ast1-chat42" {
		t.Errorf("empty metadata should fall through to session, got %q", id)
	}
}

func TestBuildIdentifier_UnknownLifecycle(t *testing.T) {
	testprepare.PrepareUnit(t)
	cfg := &types.SandboxConfig{Lifecycle: "unknown"}
	id := sandboxv2.BuildIdentifier(cfg, "owner1", "chat1", "ast1", "ws1", nil)
	if id != "" {
		t.Errorf("unknown lifecycle should return empty, got %q", id)
	}
}

func TestLifecycleAction_NilSafe(t *testing.T) {
	testprepare.PrepareUnit(t)
	cfg := &types.SandboxConfig{Lifecycle: "oneshot"}
	sandboxv2.LifecycleAction(context.Background(), cfg, nil, nil)
	sandboxv2.LifecycleAction(context.Background(), nil, nil, nil)
}
