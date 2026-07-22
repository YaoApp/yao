package sandboxv2

import (
	"context"
	"fmt"
	"strings"

	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/agent/sandbox/v2/resolve"
	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	"github.com/yaoapp/yao/tai/registry"
)

// NodeCandidate is an alias for resolve.NodeCandidate, kept here for backward
// compatibility with 30+ test references and callers.
type NodeCandidate = resolve.NodeCandidate

// WorkspaceResolver looks up which node owns a given workspace.
type WorkspaceResolver interface {
	NodeForWorkspace(ctx context.Context, id string) (string, error)
}

// SelectionCriteria describes what the current session requires.
type SelectionCriteria struct {
	WorkspaceID string
	Preferred   string
	Allowed     []string
	Image       string
	Filter      *types.ComputerFilter
	WSManager   WorkspaceResolver
}

// SelectionResult is an alias for resolve.SelectionResult, kept here for
// backward compatibility with callers and tests.
type SelectionResult = resolve.SelectionResult

// BuildNodeSnapshot reads all online nodes from the Tai registry and returns
// them as a flat slice of NodeCandidate. The local node is already included
// because tai.RegisterLocal() registers it at startup.
func BuildNodeSnapshot() []NodeCandidate {
	reg := registry.Global()
	if reg == nil {
		return nil
	}

	nodes := reg.List()
	out := make([]NodeCandidate, 0, len(nodes))
	for _, n := range nodes {
		if n.Status != "online" && n.Status != "" {
			continue
		}
		runners := n.Capabilities.Runners
		if len(runners) == 0 {
			runners = InferRunners(n, "")
		}
		out = append(out, NodeCandidate{
			ID:      n.TaiID,
			IsLocal: n.TaiID == "local" && n.Mode == "local",
			Runners: runners,
			CanBox:  n.Capabilities.Docker || n.Capabilities.K8s,
			CanHost: n.Capabilities.HostExec,
			OS:      n.System.OS,
			Arch:    n.System.Arch,
		})
	}
	return out
}

// SelectNode is the centralised node selection entry point. It implements the
// full decision flow: workspace binding -> filter -> group -> pick.
func SelectNode(nodes []NodeCandidate, criteria *SelectionCriteria) (*SelectionResult, error) {
	if criteria == nil {
		return nil, fmt.Errorf("selection criteria is nil")
	}

	// STEP 0: workspace binding (highest priority)
	if criteria.WorkspaceID != "" && criteria.WSManager != nil {
		nodeID, err := criteria.WSManager.NodeForWorkspace(context.Background(), criteria.WorkspaceID)
		if err == nil && nodeID != "" {
			found := false
			for i := range nodes {
				if nodes[i].ID == nodeID {
					found = true
					runner := ""
					if criteria.Preferred != "" && nodeHasRunner(&nodes[i], criteria.Preferred) && containsRunner(criteria.Allowed, criteria.Preferred) {
						runner = criteria.Preferred
					}
					if runner == "" {
						runner = pickRunnerOnNode(&nodes[i], criteria)
					}
					if runner == "" {
						return nil, fmt.Errorf("workspace-bound node %q does not support any of the allowed runners %v", nodeID, criteria.Allowed)
					}
					mode := resolve.ResolveMode(&nodes[i], runner, criteria.Image)
					if mode == "" {
						return nil, fmt.Errorf("workspace-bound node %q has no feasible execution mode for runner %q (CanBox=%v CanHost=%v image=%q)", nodeID, runner, nodes[i].CanBox, nodes[i].CanHost, criteria.Image)
					}
					return &SelectionResult{
						NodeID:  nodeID,
						Runner:  runner,
						Mode:    mode,
						IsLocal: nodes[i].IsLocal,
					}, nil
				}
			}
			if !found {
				log.Warn("[node_selector] workspace %q bound to node %q, but node is offline or removed; falling through to normal selection", criteria.WorkspaceID, nodeID)
			}
		}
	}

	// STEP 1: filter — keep only nodes that support at least one allowed runner
	// and pass the ComputerFilter check.
	var filtered []NodeCandidate
	for i := range nodes {
		if !nodeSupportsAny(&nodes[i], criteria.Allowed) {
			continue
		}
		if !passFilter(&nodes[i], criteria.Filter) {
			continue
		}
		filtered = append(filtered, nodes[i])
	}

	if len(filtered) == 0 {
		return nil, fmt.Errorf("no online node supports runners=%v (image=%q)", criteria.Allowed, criteria.Image)
	}

	// STEP 2: group
	var groupBox, groupRHost, groupLHost []NodeCandidate
	for i := range filtered {
		n := &filtered[i]
		if n.CanBox && criteria.Image != "" {
			groupBox = append(groupBox, *n)
		}
		if n.CanHost && !n.IsLocal {
			groupRHost = append(groupRHost, *n)
		}
		if n.IsLocal && (n.CanHost || n.CanBox) {
			groupLHost = append(groupLHost, *n)
		}
	}

	// STEP 3: try groups in priority order
	groups := [][]NodeCandidate{groupBox, groupRHost, groupLHost}
	for _, group := range groups {
		if len(group) == 0 {
			continue
		}
		// First pass: look for a node supporting the preferred runner
		if criteria.Preferred != "" {
			for i := range group {
				if nodeHasRunner(&group[i], criteria.Preferred) && containsRunner(criteria.Allowed, criteria.Preferred) {
					mode := resolve.ResolveMode(&group[i], criteria.Preferred, criteria.Image)
					if mode == "" {
						continue
					}
					return &SelectionResult{
						NodeID:  group[i].ID,
						Runner:  criteria.Preferred,
						Mode:    mode,
						IsLocal: group[i].IsLocal,
					}, nil
				}
			}
		}
		// Second pass: any allowed runner
		for i := range group {
			runner := pickRunnerOnNode(&group[i], criteria)
			if runner != "" {
				mode := resolve.ResolveMode(&group[i], runner, criteria.Image)
				if mode == "" {
					continue
				}
				return &SelectionResult{
					NodeID:  group[i].ID,
					Runner:  runner,
					Mode:    mode,
					IsLocal: group[i].IsLocal,
				}, nil
			}
		}
	}

	// STEP 4: no match
	return nil, fmt.Errorf("no node matched: runners=%v preferred=%s image=%q", criteria.Allowed, criteria.Preferred, criteria.Image)
}

// pickRunnerOnNode selects the best runner from the intersection of
// criteria.Allowed and node.Runners, preserving the allowed ordering.
func pickRunnerOnNode(node *NodeCandidate, criteria *SelectionCriteria) string {
	for _, r := range criteria.Allowed {
		if nodeHasRunner(node, r) {
			return r
		}
	}
	return ""
}

// nodeSupportsAny returns true if the node supports at least one runner from
// the allowed list.
func nodeSupportsAny(node *NodeCandidate, allowed []string) bool {
	for _, r := range allowed {
		if nodeHasRunner(node, r) {
			return true
		}
	}
	return false
}

// nodeHasRunner checks if a node supports a specific runner (canonicalised).
func nodeHasRunner(node *NodeCandidate, runner string) bool {
	return containsRunner(node.Runners, runner)
}

// passFilter checks a node against the optional ComputerFilter.
func passFilter(node *NodeCandidate, filter *types.ComputerFilter) bool {
	if filter == nil {
		return true
	}
	if filter.OS != "" && !strings.EqualFold(node.OS, filter.OS) {
		return false
	}
	if filter.Arch != "" && !strings.EqualFold(node.Arch, filter.Arch) {
		return false
	}
	if len(filter.Kind) > 0 {
		matched := false
		for _, k := range filter.Kind {
			switch strings.ToLower(k) {
			case "host":
				if node.CanHost {
					matched = true
				}
			case "box":
				if node.CanBox {
					matched = true
				}
			}
		}
		if !matched {
			return false
		}
	}
	return true
}
