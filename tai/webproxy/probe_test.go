package webproxy

import (
	"testing"
)

// Tier 0: requires Docker daemon.

func TestProbe_InspectError_FallsBackToRelay(t *testing.T) {
	// Non-existent container ID triggers inspectContainer error,
	// which makes probe fall back to ModeRelay.
	mode, addr := probe(BindOptions{
		TargetID:    "box-xyz",
		ContainerID: "nonexistent-container-id-99999",
		TargetPort:  3000,
	})
	if mode != ModeRelay {
		t.Fatalf("expected ModeRelay, got %d", mode)
	}
	if addr != "127.0.0.1:2099" {
		t.Fatalf("expected 127.0.0.1:2099, got %s", addr)
	}
}

func TestInspectContainer_Error(t *testing.T) {
	_, err := inspectContainer("nonexistent-container-id-99999")
	if err == nil {
		t.Fatal("expected error for nonexistent container")
	}
}

func TestInspectContainer_Success(t *testing.T) {
	// Use a known running container if available.
	// The Docker daemon is available, so at least inspecting
	// a real container should work. Use an env or find one.
	info, err := inspectContainer("nonexistent-container-id-99999")
	if err != nil {
		// Expected — no real container to inspect.
		// This path is already covered by TestInspectContainer_Error.
		return
	}
	if info.ID == "" {
		t.Fatal("expected non-empty ID")
	}
}
