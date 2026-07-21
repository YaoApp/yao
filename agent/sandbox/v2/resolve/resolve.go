package resolve

import "fmt"

// NodeCandidate describes a single node's capabilities for selection.
type NodeCandidate struct {
	ID      string
	IsLocal bool
	Runners []string
	CanBox  bool
	CanHost bool
	OS      string
	Arch    string
}

// SelectionResult carries the final node selection decision.
type SelectionResult struct {
	NodeID  string
	Runner  string
	Mode    string // "box" | "host" | "local"
	IsLocal bool
}

// ResolveMode determines the execution mode from node capabilities, runner,
// and image. It returns an empty string when no feasible mode exists.
func ResolveMode(node *NodeCandidate, runner, image string) string {
	if node.IsLocal && runner == "yaocode" {
		return "local"
	}
	if node.CanBox && image != "" {
		return "box"
	}
	if node.CanHost {
		return "host"
	}
	if node.CanBox {
		return "box"
	}
	return ""
}

// BuildIdentifier determines the Computer identifier based on lifecycle policy.
// Returns "" for oneshot (always new) or unknown lifecycle values.
func BuildIdentifier(lifecycle, ownerID, chatID, assistantID, workspaceID string) string {
	switch lifecycle {
	case "oneshot":
		return ""
	case "session":
		return fmt.Sprintf("%s-%s-%s", ownerID, assistantID, chatID)
	case "longrunning", "persistent":
		return fmt.Sprintf("%s-%s.%s", ownerID, assistantID, workspaceID)
	default:
		return ""
	}
}
