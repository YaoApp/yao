//go:build unit

package sandboxv2_test

import (
	"testing"

	sandboxv2 "github.com/yaoapp/yao/agent/sandbox/v2"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	"github.com/yaoapp/yao/share"
	taitypes "github.com/yaoapp/yao/tai/types"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
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
		Mode:         "tunnel",
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
		Mode:         "tunnel",
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
	if len(runners) != 3 {
		t.Errorf("k8s: expected 3 runners, got %v", runners)
	}
	for _, expected := range []string{"tai", "claude", "opencode"} {
		found := false
		for _, r := range runners {
			if r == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("k8s: missing %q in %v", expected, runners)
		}
	}
}

func TestInferRunners_HostExec(t *testing.T) {
	prev := share.Tools
	share.Tools = &share.ExtTools{
		Runners: map[string]*share.ExtToolInfo{
			"tai":      {Name: "tai", Available: true},
			"claude":   {Name: "claude", Available: true},
			"opencode": {Name: "opencode", Available: true},
		},
	}
	defer func() { share.Tools = prev }()

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
	node := taitypes.NodeMeta{Mode: "tunnel"}
	runners := sandboxv2.InferRunners(node, "")
	if len(runners) != 0 {
		t.Errorf("no capabilities: got %v, want empty", runners)
	}
}

// --- ResolveRunnerSet tests ---

func TestResolveRunnerSet_R1_AllAuto(t *testing.T) {
	testprepare.PrepareUnit(t)
	cfg := &types.RunnerConfig{Supports: []string{"claude", "opencode"}}
	preferred, allowed := sandboxv2.ResolveRunnerSet(nil, cfg, "")
	if preferred != "claude" {
		t.Errorf("preferred = %q, want claude", preferred)
	}
	if len(allowed) != 2 || allowed[0] != "claude" || allowed[1] != "opencode" {
		t.Errorf("allowed = %v, want [claude opencode]", allowed)
	}
}

func TestResolveRunnerSet_R2_UserPick(t *testing.T) {
	testprepare.PrepareUnit(t)
	cfg := &types.RunnerConfig{Supports: []string{"claude", "opencode"}}
	preferred, allowed := sandboxv2.ResolveRunnerSet([]string{"opencode"}, cfg, "")
	if preferred != "opencode" {
		t.Errorf("preferred = %q, want opencode", preferred)
	}
	if len(allowed) != 1 || allowed[0] != "opencode" {
		t.Errorf("allowed = %v, want [opencode]", allowed)
	}
}

func TestResolveRunnerSet_R3_DSLOverride(t *testing.T) {
	testprepare.PrepareUnit(t)
	cfg := &types.RunnerConfig{Name: "claude", Supports: []string{"claude", "opencode"}}
	preferred, allowed := sandboxv2.ResolveRunnerSet(nil, cfg, "")
	if preferred != "claude" {
		t.Errorf("preferred = %q, want claude", preferred)
	}
	if len(allowed) != 2 {
		t.Errorf("allowed = %v, want 2 items", allowed)
	}
}

func TestResolveRunnerSet_R4_GlobalFallback(t *testing.T) {
	testprepare.PrepareUnit(t)
	cfg := &types.RunnerConfig{Supports: []string{"claude", "opencode"}}
	preferred, allowed := sandboxv2.ResolveRunnerSet(nil, cfg, "opencode")
	if preferred != "opencode" {
		t.Errorf("preferred = %q, want opencode", preferred)
	}
	if len(allowed) != 2 {
		t.Errorf("allowed = %v, want 2 items", allowed)
	}
}

func TestResolveRunnerSet_R5_DSL_in_Allowed(t *testing.T) {
	testprepare.PrepareUnit(t)
	cfg := &types.RunnerConfig{Name: "claude", Supports: []string{"claude", "opencode"}}
	preferred, allowed := sandboxv2.ResolveRunnerSet([]string{"claude", "opencode"}, cfg, "opencode")
	if preferred != "claude" {
		t.Errorf("preferred = %q, want claude (DSL > global when both in allowed)", preferred)
	}
	if len(allowed) != 2 {
		t.Errorf("allowed = %v, want 2 items", allowed)
	}
}

func TestResolveRunnerSet_R5b_DSL_not_in_Allowed(t *testing.T) {
	testprepare.PrepareUnit(t)
	cfg := &types.RunnerConfig{Name: "claude", Supports: []string{"claude", "opencode"}}
	preferred, allowed := sandboxv2.ResolveRunnerSet([]string{"opencode"}, cfg, "opencode")
	if preferred != "opencode" {
		t.Errorf("preferred = %q, want opencode (DSL claude not in user-allowed set)", preferred)
	}
	if len(allowed) != 1 || allowed[0] != "opencode" {
		t.Errorf("allowed = %v, want [opencode]", allowed)
	}
}

func TestResolveRunnerSet_R6_EmptyIntersection(t *testing.T) {
	testprepare.PrepareUnit(t)
	cfg := &types.RunnerConfig{Supports: []string{"claude", "opencode"}}
	preferred, allowed := sandboxv2.ResolveRunnerSet([]string{"tai"}, cfg, "")
	if len(allowed) != 2 {
		t.Errorf("allowed = %v, want fallback to supports [claude opencode]", allowed)
	}
	if preferred != "claude" {
		t.Errorf("preferred = %q, want claude (first of supports)", preferred)
	}
}

func TestResolveRunnerSet_UseDefault_Ignored(t *testing.T) {
	testprepare.PrepareUnit(t)
	cfg := &types.RunnerConfig{Name: "use::default", Supports: []string{"claude", "opencode"}}
	preferred, allowed := sandboxv2.ResolveRunnerSet(nil, cfg, "use::default")
	if preferred != "claude" {
		t.Errorf("preferred = %q, want claude (use::default ignored)", preferred)
	}
	if len(allowed) != 2 {
		t.Errorf("allowed = %v, want [claude opencode]", allowed)
	}
}

func TestResolveRunnerSet_EmptySupports(t *testing.T) {
	testprepare.PrepareUnit(t)
	cfg := &types.RunnerConfig{}
	preferred, allowed := sandboxv2.ResolveRunnerSet(nil, cfg, "")
	if len(allowed) != len(sandboxv2.SupportedRunners) {
		t.Errorf("allowed = %v, want all SupportedRunners", allowed)
	}
	if preferred != sandboxv2.SupportedRunners[0] {
		t.Errorf("preferred = %q, want %q", preferred, sandboxv2.SupportedRunners[0])
	}
}
