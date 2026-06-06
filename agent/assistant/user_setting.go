package assistant

import (
	"encoding/json"

	"github.com/yaoapp/yao/agent/sandbox/v2/types"
	"github.com/yaoapp/yao/setting"
)

// UserAgentSetting stores per-user agent preferences (runners, image, secrets, services).
// Persisted in setting.Registry under namespace "agent.<assistant_id>".
type UserAgentSetting struct {
	Runners  []string                      `json:"runners,omitempty"`
	Image    string                        `json:"image,omitempty"`
	Secrets  map[string]*types.SecretEntry `json:"secrets,omitempty"`
	Options  map[string]any                `json:"options,omitempty"`
	Services []types.ServiceConfig         `json:"services,omitempty"`
}

// ResolveServicesBatch resolves services for multiple assistants in one pass,
// using a single batch read for user settings instead of N individual lookups.
func ResolveServicesBatch(assistantIDs []string, userID, teamID string) map[string][]types.ServiceConfig {
	namespaces := make([]string, len(assistantIDs))
	for i, id := range assistantIDs {
		namespaces[i] = "agent." + id
	}

	allSettings := make(map[string]UserAgentSetting, len(assistantIDs))
	if setting.Global != nil {
		raw := setting.Global.GetMergedBatch(userID, teamID, namespaces)
		for _, id := range assistantIDs {
			if data, ok := raw["agent."+id]; ok {
				var us UserAgentSetting
				if b, err := json.Marshal(data); err == nil {
					json.Unmarshal(b, &us)
				}
				allSettings[id] = us
			}
		}
	}

	result := make(map[string][]types.ServiceConfig, len(assistantIDs))
	for _, id := range assistantIDs {
		us := allSettings[id]
		result[id] = ResolveServices(id, us.Services)
	}
	return result
}

// ResolveServices merges sandbox.yao default services with user overrides.
// If user has explicitly configured services, they are the final result.
// Otherwise, derive defaults from sandbox.yao computer.ports (label + port).
func ResolveServices(assistantID string, userServices []types.ServiceConfig) []types.ServiceConfig {
	if len(userServices) > 0 {
		return userServices
	}

	ast, err := LoadStore(assistantID)
	if err != nil || ast == nil || ast.SandboxV2 == nil {
		return []types.ServiceConfig{}
	}

	cfg := ast.SandboxV2
	if len(cfg.Computer.Ports) > 0 {
		result := make([]types.ServiceConfig, 0, len(cfg.Computer.Ports))
		for _, p := range cfg.Computer.Ports {
			result = append(result, types.ServiceConfig{
				Label: p.Label,
				Port:  p.Port,
			})
		}
		return result
	}

	return []types.ServiceConfig{}
}
