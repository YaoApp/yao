package sandboxv2

import "github.com/yaoapp/yao/agent/sandbox/v2/types"

// AvailabilityResult carries the runnable verdict for an agent.
type AvailabilityResult struct {
	Runnable bool   `json:"runnable"`
	Reason   string `json:"reason,omitempty"`
}

// CheckAvailability determines whether a sandbox agent can be executed with
// the current system state. When nodes is nil, a fresh snapshot is built
// internally; callers iterating over many agents should pre-build the
// snapshot once and pass it in.
func CheckAvailability(nodes []NodeCandidate, allowed []string, preferred string, image string, filter *types.ComputerFilter) *AvailabilityResult {
	if nodes == nil {
		nodes = BuildNodeSnapshot()
	}
	if len(nodes) == 0 {
		return &AvailabilityResult{Runnable: false, Reason: "no_nodes"}
	}
	_, err := SelectNode(nodes, &SelectionCriteria{
		Preferred: preferred,
		Allowed:   allowed,
		Image:     image,
		Filter:    filter,
	})
	if err != nil {
		return &AvailabilityResult{Runnable: false, Reason: err.Error()}
	}
	return &AvailabilityResult{Runnable: true}
}
