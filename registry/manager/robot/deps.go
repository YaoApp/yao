package robot

import (
	"encoding/json"
	"strings"
)

// RobotJSON represents the portable fields exported from a robot member record.
type RobotJSON struct {
	DisplayName   string          `json:"display_name,omitempty"`
	Bio           *string         `json:"bio,omitempty"`
	SystemPrompt  string          `json:"system_prompt,omitempty"`
	LanguageModel string          `json:"language_model,omitempty"`
	RobotConfig   json.RawMessage `json:"robot_config,omitempty"`
	Agents        []string        `json:"agents,omitempty"`
	MCPServers    []string        `json:"mcp_servers,omitempty"`
}

// robotConfig is a partial parse of robot_config for dependency extraction.
type robotConfig struct {
	Resources struct {
		Phases map[string]string `json:"phases,omitempty"`
	} `json:"resources,omitempty"`
}

// RobotDep represents a dependency extracted from a robot configuration.
type RobotDep struct {
	PackageID string // "@scope/name"
	Type      string // "assistant" or "mcp"
}

// AnalyzeDeps extracts dependencies from a RobotJSON following DESIGN-ROBOT.md rules:
//   - phases values: "yao.robot-host" → @yao/robot-host (assistant)
//   - agents values: "yao.keeper.fetch" → @yao/keeper (first-layer assistant, take first 2 segments)
//   - mcp_servers values: "ark.image.text2img" → @ark/image.text2img (mcp)
//   - Excludes: __yao.* prefixed built-in agents
func AnalyzeDeps(robot *RobotJSON) []RobotDep {
	seen := map[string]bool{}
	var deps []RobotDep

	addDep := func(pkgID, depType string) {
		if seen[pkgID] {
			return
		}
		seen[pkgID] = true
		deps = append(deps, RobotDep{PackageID: pkgID, Type: depType})
	}

	// Extract from phases (all are assistants)
	if len(robot.RobotConfig) > 0 {
		var cfg robotConfig
		if err := json.Unmarshal(robot.RobotConfig, &cfg); err == nil {
			for _, yaoID := range cfg.Resources.Phases {
				if isBuiltIn(yaoID) {
					continue
				}
				pkgID := yaoIDToPackageID(yaoID)
				if pkgID != "" {
					addDep(pkgID, "assistant")
				}
			}
		}
	}

	// Extract from agents (first-layer assistant)
	for _, yaoID := range robot.Agents {
		if isBuiltIn(yaoID) {
			continue
		}
		pkgID := agentYaoIDToPackageID(yaoID)
		if pkgID != "" {
			addDep(pkgID, "assistant")
		}
	}

	// Extract from mcp_servers
	for _, yaoID := range robot.MCPServers {
		if isBuiltIn(yaoID) {
			continue
		}
		pkgID := yaoIDToPackageID(yaoID)
		if pkgID != "" {
			addDep(pkgID, "mcp")
		}
	}

	return deps
}

// isBuiltIn returns true for __yao.* prefixed IDs.
func isBuiltIn(yaoID string) bool {
	return strings.HasPrefix(yaoID, "__yao.")
}

// yaoIDToPackageID converts "yao.robot-host" → "@yao/robot-host".
// First "." separates scope from name.
func yaoIDToPackageID(yaoID string) string {
	idx := strings.Index(yaoID, ".")
	if idx <= 0 || idx >= len(yaoID)-1 {
		return ""
	}
	scope := yaoID[:idx]
	name := yaoID[idx+1:]
	return "@" + scope + "/" + name
}

// agentYaoIDToPackageID converts "yao.keeper.fetch" → "@yao/keeper".
// Takes the first two segments only (first-layer assistant).
func agentYaoIDToPackageID(yaoID string) string {
	parts := strings.SplitN(yaoID, ".", 3)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return ""
	}
	return "@" + parts[0] + "/" + parts[1]
}
