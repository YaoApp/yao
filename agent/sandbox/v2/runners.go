package sandboxv2

import (
	"strings"

	"github.com/yaoapp/yao/share"
	taitypes "github.com/yaoapp/yao/tai/types"

	"github.com/yaoapp/yao/agent/sandbox/v2/types"
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

// ResolveRunnerSet returns the preferred runner and the full set of allowed
// runners for a sandbox session. It replaces ResolveRunner in initSandboxV2
// to support multi-runner node selection with fallback.
//
// Priority for preferred: runnerCfg.Name > globalRunner > allowed[0].
// "use::default" at any level is treated as unset.
func ResolveRunnerSet(userRunners []string, runnerCfg *types.RunnerConfig, globalRunner string) (preferred string, allowed []string) {
	// 1. Determine the supports range from the DSL
	var supports []string
	if runnerCfg != nil && len(runnerCfg.Supports) > 0 {
		supports = runnerCfg.Supports
	} else {
		supports = SupportedRunners
	}

	// 2. Compute allowed = intersection(supports, userRunners)
	if len(userRunners) == 0 {
		allowed = make([]string, len(supports))
		copy(allowed, supports)
	} else {
		for _, u := range userRunners {
			cu := canonicalRunner(u)
			if containsRunner(supports, cu) {
				allowed = append(allowed, cu)
			}
		}
		if len(allowed) == 0 {
			allowed = make([]string, len(supports))
			copy(allowed, supports)
		}
	}

	// 3. Determine preferred
	if runnerCfg != nil && runnerCfg.Name != "" && runnerCfg.Name != "use::default" {
		c := canonicalRunner(runnerCfg.Name)
		if containsRunner(allowed, c) {
			preferred = c
		}
	}
	if preferred == "" && globalRunner != "" && globalRunner != "use::default" {
		c := canonicalRunner(globalRunner)
		if containsRunner(allowed, c) {
			preferred = c
		}
	}
	if preferred == "" && len(allowed) > 0 {
		preferred = allowed[0]
	}

	return preferred, allowed
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
		for _, name := range []string{"tai", "claude", "opencode"} {
			if runnerDetected(name) {
				runners = append(runners, name)
			}
		}
	}

	return runners
}

// runnerDetected checks share.Tools.Runners for actual CLI installation.
func runnerDetected(name string) bool {
	if share.Tools == nil || share.Tools.Runners == nil {
		return false
	}
	info := share.Tools.Runners[name]
	return info != nil && info.Available
}
