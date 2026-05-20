package sandboxv2

import (
	"strings"

	taitypes "github.com/yaoapp/yao/tai/types"
)

// SupportedRunners lists all product-level runner names.
var SupportedRunners = []string{"yaocode", "tai", "claude", "opencode"}

// GlobalRunnerFunc is set by agent.Load() to provide the global default runner
// from agent.yml. This avoids an import cycle between agent and agent/sandbox/v2.
var GlobalRunnerFunc func() string

// canonicalRunner normalises a runner name to its canonical form.
// Handles backward-compat aliases and slash-qualified variants.
func canonicalRunner(name string) string {
	base := name
	if i := strings.Index(name, "/"); i >= 0 {
		base = name[:i]
	}
	switch strings.ToLower(base) {
	case "yao":
		return "yaocode"
	case "yaocode":
		return "yaocode"
	case "claude":
		return "claude"
	case "opencode":
		return "opencode"
	case "tai":
		return "tai"
	default:
		return strings.ToLower(base)
	}
}

// containsRunner checks whether list contains the given runner (canonicalised).
func containsRunner(list []string, name string) bool {
	canon := canonicalRunner(name)
	for _, r := range list {
		if canonicalRunner(r) == canon {
			return true
		}
	}
	return false
}

// InferRunners guesses the supported runners for a legacy node that has no
// Runners field (pre-upgrade Tai nodes). The heuristic:
//   - local node → ["yaocode"]
//   - Docker/K8s capable → image-based guessing + "tai" fallback
//   - HostExec only → CLI presence guessing + "tai" fallback
func InferRunners(node taitypes.NodeMeta, image string) []string {
	if node.Mode == "local" {
		return []string{"yaocode"}
	}

	var runners []string

	hasContainer := node.Capabilities.Docker || node.Capabilities.K8s
	if hasContainer {
		runners = append(runners, "tai")
		if image != "" {
			// Infer from image name keywords
			img := strings.ToLower(image)
			if strings.Contains(img, "claude") {
				runners = append(runners, "claude")
			}
			if strings.Contains(img, "opencode") {
				runners = append(runners, "opencode")
			}
		} else {
			// No image specified: Docker available means all box-mode runners can work
			runners = append(runners, "claude", "opencode")
		}
	} else if node.Capabilities.HostExec {
		runners = append(runners, "tai", "claude", "opencode")
	}

	return runners
}
