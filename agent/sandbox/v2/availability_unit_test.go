//go:build unit

package sandboxv2_test

import (
	"testing"

	sandboxv2 "github.com/yaoapp/yao/agent/sandbox/v2"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestCheckAvailability_NoNodes(t *testing.T) {
	testprepare.PrepareUnit(t)
	res := sandboxv2.CheckAvailability(nil, []string{"claude"}, "claude", "", nil)
	if res.Runnable {
		t.Error("expected Runnable=false with nil nodes (no registry)")
	}
	if res.Reason != "no_nodes" {
		t.Errorf("expected reason=no_nodes, got %q", res.Reason)
	}
}

func TestCheckAvailability_EmptyNodes(t *testing.T) {
	testprepare.PrepareUnit(t)
	res := sandboxv2.CheckAvailability([]sandboxv2.NodeCandidate{}, []string{"claude"}, "claude", "", nil)
	if res.Runnable {
		t.Error("expected Runnable=false with empty nodes")
	}
}

func TestCheckAvailability_HostModeRunnable(t *testing.T) {
	testprepare.PrepareUnit(t)
	nodes := []sandboxv2.NodeCandidate{
		localNode([]string{"claude", "yaocode"}, false, true),
	}
	res := sandboxv2.CheckAvailability(nodes, []string{"claude", "yaocode"}, "claude", "", nil)
	if !res.Runnable {
		t.Errorf("expected Runnable=true for host-mode agent with claude installed, got reason=%q", res.Reason)
	}
}

func TestCheckAvailability_DockerRunnable(t *testing.T) {
	testprepare.PrepareUnit(t)
	nodes := []sandboxv2.NodeCandidate{
		localNode([]string{"claude", "opencode", "tai", "yaocode"}, true, true),
	}
	res := sandboxv2.CheckAvailability(nodes, []string{"claude"}, "claude", "ghcr.io/example/claude:latest", nil)
	if !res.Runnable {
		t.Errorf("expected Runnable=true for docker agent, got reason=%q", res.Reason)
	}
}

func TestCheckAvailability_NoMatchingRunner(t *testing.T) {
	testprepare.PrepareUnit(t)
	nodes := []sandboxv2.NodeCandidate{
		localNode([]string{"yaocode"}, false, true),
	}
	res := sandboxv2.CheckAvailability(nodes, []string{"claude"}, "claude", "", nil)
	if res.Runnable {
		t.Error("expected Runnable=false when no node supports the required runner")
	}
}

func TestCheckAvailability_FilterKindHost(t *testing.T) {
	testprepare.PrepareUnit(t)
	nodes := []sandboxv2.NodeCandidate{
		localNode([]string{"claude", "yaocode"}, false, true),
	}
	filter := &types.ComputerFilter{Kind: types.StringOrArray{"host"}}
	res := sandboxv2.CheckAvailability(nodes, []string{"claude", "yaocode"}, "claude", "", filter)
	if !res.Runnable {
		t.Errorf("expected Runnable=true with kind=host filter and host-capable node, got reason=%q", res.Reason)
	}
}

func TestCheckAvailability_FilterKindBoxNoDocker(t *testing.T) {
	testprepare.PrepareUnit(t)
	nodes := []sandboxv2.NodeCandidate{
		localNode([]string{"claude", "yaocode"}, false, true),
	}
	filter := &types.ComputerFilter{Kind: types.StringOrArray{"box"}}
	res := sandboxv2.CheckAvailability(nodes, []string{"claude"}, "claude", "img:latest", filter)
	if res.Runnable {
		t.Error("expected Runnable=false with kind=box filter but no Docker-capable node")
	}
}
