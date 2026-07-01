package secret

import (
	_ "embed"
	"fmt"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/openapi/audit"
	"github.com/yaoapp/yao/setting"
)

//go:embed read_schema.json
var ReadSchemaJSON []byte

// ReadHandler is the tools.secret_read process handler.
// Reads a single secret value from the setting registry by name.
// The caller's identity is extracted from the process authorized info.
//
// Args[0]: name (string) — secret key name
func ReadHandler(proc *process.Process) interface{} {
	name := proc.ArgsString(0)
	if name == "" {
		return map[string]any{"error": "name is required"}
	}

	auth := proc.Authorized
	if auth == nil {
		audit.Record(audit.Entry{
			Operation:      "secret_read",
			Category:       "security",
			Severity:       "high",
			TargetResource: name,
			ResourceType:   "secret",
			Source:         "mcp",
			Success:        false,
			ErrorMessage:   "unauthorized: no auth context",
		})
		return map[string]any{"error": "unauthorized"}
	}

	userID := auth.UserID
	teamID := auth.TeamID

	assistantID, err := extractAssistantID(proc)
	if err != nil {
		audit.Record(audit.Entry{
			Operation:      "secret_read",
			Category:       "security",
			Severity:       "high",
			UserID:         userID,
			TeamID:         teamID,
			TargetResource: name,
			ResourceType:   "secret",
			Source:         "mcp",
			Success:        false,
			ErrorMessage:   err.Error(),
		})
		return map[string]any{"error": err.Error()}
	}

	chatID := extractChatID(proc)

	secretsMap, err := getMergedSecrets(userID, teamID, assistantID, chatID)
	if err != nil {
		audit.Record(audit.Entry{
			Operation:      "secret_read",
			Category:       "data",
			Severity:       "medium",
			UserID:         userID,
			TeamID:         teamID,
			Application:    assistantID,
			TargetResource: name,
			ResourceType:   "secret",
			Source:         "mcp",
			Success:        false,
			ErrorMessage:   err.Error(),
		})
		return map[string]any{"error": fmt.Sprintf("setting read failed: %v", err)}
	}

	if secretsMap == nil {
		audit.Record(audit.Entry{
			Operation:      "secret_read",
			Category:       "data",
			Severity:       "medium",
			UserID:         userID,
			TeamID:         teamID,
			Application:    assistantID,
			TargetResource: name,
			ResourceType:   "secret",
			Source:         "mcp",
			Success:        false,
			ErrorMessage:   fmt.Sprintf("secret %q not found", name),
		})
		return map[string]any{"error": fmt.Sprintf("secret %q not found", name)}
	}

	entryRaw, ok := secretsMap[name]
	if !ok {
		audit.Record(audit.Entry{
			Operation:      "secret_read",
			Category:       "data",
			Severity:       "medium",
			UserID:         userID,
			TeamID:         teamID,
			Application:    assistantID,
			TargetResource: name,
			ResourceType:   "secret",
			Source:         "mcp",
			Success:        false,
			ErrorMessage:   fmt.Sprintf("secret %q not found", name),
		})
		return map[string]any{"error": fmt.Sprintf("secret %q not found", name)}
	}

	entryMap, ok := entryRaw.(map[string]interface{})
	if !ok {
		audit.Record(audit.Entry{
			Operation:      "secret_read",
			Category:       "data",
			Severity:       "medium",
			UserID:         userID,
			TeamID:         teamID,
			Application:    assistantID,
			TargetResource: name,
			ResourceType:   "secret",
			Source:         "mcp",
			Success:        false,
			ErrorMessage:   "invalid secret entry format",
		})
		return map[string]any{"error": "invalid secret entry format"}
	}

	value, _ := entryMap["value"].(string)
	if value != "" {
		value = setting.Decrypt(value)
	}

	audit.Record(audit.Entry{
		Operation:      "secret_read",
		Category:       "data",
		Severity:       "medium",
		UserID:         userID,
		TeamID:         teamID,
		Application:    assistantID,
		TargetResource: name,
		ResourceType:   "secret",
		Source:         "mcp",
		Success:        true,
	})

	return map[string]any{
		"name":  name,
		"value": value,
	}
}
