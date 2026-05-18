package sandboxv2_test

import (
	"testing"

	sandboxv2 "github.com/yaoapp/yao/agent/sandbox/v2"
	taitypes "github.com/yaoapp/yao/tai/types"
)

func TestCanonicalRunner(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"yao", "yaocode"},
		{"yaocode", "yaocode"},
		{"Yao", "yaocode"},
		{"YAOCODE", "yaocode"},
		{"claude", "claude"},
		{"Claude", "claude"},
		{"claude/cli", "claude"},
		{"opencode", "opencode"},
		{"opencode/cli", "opencode"},
		{"tai", "tai"},
		{"TAI", "tai"},
		{"unknown", "unknown"},
		{"custom/sub", "custom"},
	}
	for _, tc := range cases {
		got := sandboxv2.ExportCanonicalRunner(tc.input)
		if got != tc.want {
			t.Errorf("canonicalRunner(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestContainsRunner(t *testing.T) {
	list := []string{"claude", "opencode"}

	if !sandboxv2.ExportContainsRunner(list, "claude") {
		t.Error("should contain claude")
	}
	if !sandboxv2.ExportContainsRunner(list, "Claude") {
		t.Error("should contain Claude (case insensitive)")
	}
	if !sandboxv2.ExportContainsRunner(list, "opencode/cli") {
		t.Error("should contain opencode/cli (slash variant)")
	}
	if sandboxv2.ExportContainsRunner(list, "tai") {
		t.Error("should not contain tai")
	}
	if sandboxv2.ExportContainsRunner(nil, "claude") {
		t.Error("nil list should not contain anything")
	}
}

func TestInferRunners_Local(t *testing.T) {
	node := taitypes.NodeMeta{Mode: "local"}
	runners := sandboxv2.InferRunners(node, "")
	if len(runners) != 1 || runners[0] != "yaocode" {
		t.Errorf("local: got %v, want [yaocode]", runners)
	}
}

func TestInferRunners_Docker_NoImage(t *testing.T) {
	node := taitypes.NodeMeta{
		Mode:         "direct",
		Capabilities: taitypes.Capabilities{Docker: true},
	}
	runners := sandboxv2.InferRunners(node, "")
	if len(runners) != 3 {
		t.Fatalf("docker no image: got %v, want 3 runners", runners)
	}
	expected := map[string]bool{"tai": true, "claude": true, "opencode": true}
	for _, r := range runners {
		if !expected[r] {
			t.Errorf("unexpected runner %q", r)
		}
	}
}

func TestInferRunners_Docker_ClaudeImage(t *testing.T) {
	node := taitypes.NodeMeta{
		Mode:         "direct",
		Capabilities: taitypes.Capabilities{Docker: true},
	}
	runners := sandboxv2.InferRunners(node, "my-registry/claude-sandbox:latest")
	has := func(name string) bool {
		for _, r := range runners {
			if r == name {
				return true
			}
		}
		return false
	}
	if !has("tai") {
		t.Error("should include tai")
	}
	if !has("claude") {
		t.Error("should include claude (from image name)")
	}
	if has("opencode") {
		t.Error("should not include opencode (not in image name)")
	}
}

func TestInferRunners_K8s(t *testing.T) {
	node := taitypes.NodeMeta{
		Mode:         "tunnel",
		Capabilities: taitypes.Capabilities{K8s: true},
	}
	runners := sandboxv2.InferRunners(node, "")
	if len(runners) < 2 {
		t.Errorf("k8s: got %v", runners)
	}
}

func TestInferRunners_HostExec(t *testing.T) {
	node := taitypes.NodeMeta{
		Mode:         "tunnel",
		Capabilities: taitypes.Capabilities{HostExec: true},
	}
	runners := sandboxv2.InferRunners(node, "")
	expected := map[string]bool{"tai": true, "claude": true, "opencode": true}
	for _, r := range runners {
		if !expected[r] {
			t.Errorf("unexpected runner %q", r)
		}
	}
	if len(runners) != 3 {
		t.Errorf("hostexec: got %v, want 3 runners", runners)
	}
}

func TestInferRunners_NoCapabilities(t *testing.T) {
	node := taitypes.NodeMeta{Mode: "direct"}
	runners := sandboxv2.InferRunners(node, "")
	if len(runners) != 0 {
		t.Errorf("no capabilities: got %v, want empty", runners)
	}
}
