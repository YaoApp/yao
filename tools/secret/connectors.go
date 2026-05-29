package secret

import (
	_ "embed"
	"fmt"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/llmprovider"
	"github.com/yaoapp/yao/openapi/audit"
)

//go:embed connectors_schema.json
var ConnectorsSchemaJSON []byte

// ConnectorsHandler returns the full LLM connector role matrix including
// credentials. Designed for env-setup scripts that provision sandbox dev
// environments. Follows the same auth + audit pattern as secret_read.
func ConnectorsHandler(proc *process.Process) interface{} {
	auth := proc.Authorized
	if auth == nil {
		audit.Record(audit.Entry{
			Operation:    "secret_connectors",
			Category:     "security",
			Severity:     "high",
			ResourceType: "connector",
			Source:       "mcp",
			Success:      false,
			ErrorMessage: "unauthorized: no auth context",
		})
		return map[string]any{"error": "unauthorized"}
	}

	userID := auth.UserID
	teamID := auth.TeamID

	assistantID, err := extractAssistantID(proc)
	if err != nil {
		audit.Record(audit.Entry{
			Operation:    "secret_connectors",
			Category:     "security",
			Severity:     "high",
			UserID:       userID,
			TeamID:       teamID,
			ResourceType: "connector",
			Source:       "mcp",
			Success:      false,
			ErrorMessage: err.Error(),
		})
		return map[string]any{"error": err.Error()}
	}

	if llmprovider.Global == nil {
		audit.Record(audit.Entry{
			Operation:    "secret_connectors",
			Category:     "data",
			Severity:     "medium",
			UserID:       userID,
			TeamID:       teamID,
			Application:  assistantID,
			ResourceType: "connector",
			Source:       "mcp",
			Success:      false,
			ErrorMessage: "llmprovider not initialized",
		})
		return map[string]any{"error": "llmprovider not initialized"}
	}

	roles, err := llmprovider.Global.ListRolesBy(auth)
	if err != nil {
		audit.Record(audit.Entry{
			Operation:    "secret_connectors",
			Category:     "data",
			Severity:     "medium",
			UserID:       userID,
			TeamID:       teamID,
			Application:  assistantID,
			ResourceType: "connector",
			Source:       "mcp",
			Success:      false,
			ErrorMessage: fmt.Sprintf("list roles: %v", err),
		})
		return map[string]any{"error": fmt.Sprintf("failed to list roles: %s", err.Error())}
	}

	result := make(map[string]any, len(roles))
	for role, target := range roles {
		setting, err := llmprovider.Global.GetSetting(target.Provider)
		if err != nil {
			continue
		}

		entry := map[string]any{
			"model":     target.Model,
			"key":       settingStr(setting, "key"),
			"host":      settingStr(setting, "host"),
			"auth_mode": settingStrDefault(setting, "auth_mode", "bearer"),
		}

		providerType := "openai"
		if p, pErr := llmprovider.Global.Get(target.Provider, false); pErr == nil && p.Type != "" {
			providerType = p.Type
		}
		entry["type"] = providerType

		if model := settingStr(setting, "model"); model != "" && entry["model"] == "" {
			entry["model"] = model
		}
		if v, ok := setting["capabilities"]; ok {
			entry["capabilities"] = v
		}
		if v, ok := setting["thinking"]; ok {
			entry["thinking"] = v
		}
		if v, ok := setting["protocols"]; ok {
			entry["protocols"] = v
		}
		if v, ok := setting["max_tokens"]; ok {
			entry["max_tokens"] = v
		}
		if v, ok := setting["temperature"]; ok {
			entry["temperature"] = v
		}

		result[role] = entry
	}

	audit.Record(audit.Entry{
		Operation:    "secret_connectors",
		Category:     "data",
		Severity:     "medium",
		UserID:       userID,
		TeamID:       teamID,
		Application:  assistantID,
		ResourceType: "connector",
		Source:       "mcp",
		Success:      true,
		Details:      map[string]any{"count": len(result)},
	})

	return map[string]any{"roles": result}
}

func settingStr(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func settingStrDefault(m map[string]interface{}, key, fallback string) string {
	if s := settingStr(m, key); s != "" {
		return s
	}
	return fallback
}
