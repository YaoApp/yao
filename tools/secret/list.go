package secret

import (
	_ "embed"
	"fmt"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/openapi/audit"
)

//go:embed list_schema.json
var ListSchemaJSON []byte

// ListHandler is the tools.secret_list process handler.
// Returns secret metadata (name + description) without values.
//
// Args: none (identity from authorized info)
func ListHandler(proc *process.Process) interface{} {
	auth := proc.Authorized
	if auth == nil {
		audit.Record(audit.Entry{
			Operation:    "secret_list",
			Category:     "security",
			Severity:     "high",
			ResourceType: "secret",
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
			Operation:    "secret_list",
			Category:     "data",
			Severity:     "medium",
			UserID:       userID,
			TeamID:       teamID,
			ResourceType: "secret",
			Source:       "mcp",
			Success:      false,
			ErrorMessage: err.Error(),
		})
		return map[string]any{"error": err.Error()}
	}

	chatID := extractChatID(proc)

	secretsMap, err := getMergedSecrets(userID, teamID, assistantID, chatID)
	if err != nil {
		audit.Record(audit.Entry{
			Operation:    "secret_list",
			Category:     "data",
			Severity:     "low",
			UserID:       userID,
			TeamID:       teamID,
			Application:  assistantID,
			ResourceType: "secret",
			Source:       "mcp",
			Success:      false,
			ErrorMessage: err.Error(),
		})
		return map[string]any{"error": fmt.Sprintf("setting read failed: %v", err)}
	}

	type secretMeta struct {
		Name        string `json:"name"`
		Label       string `json:"label,omitempty"`
		Description string `json:"description,omitempty"`
	}

	seen := make(map[string]bool)
	var items []secretMeta

	// 1) Start with predefined secrets from sandbox.yao (label + description)
	for name, meta := range loadPredefinedSecrets(assistantID) {
		seen[name] = true
		items = append(items, secretMeta{
			Name:        name,
			Label:       meta.Label,
			Description: meta.Description,
		})
	}

	// 2) Merge user-configured secrets from L2+L3 merged map
	for name, entryRaw := range secretsMap {
		entryMap, ok := entryRaw.(map[string]interface{})
		if !ok {
			continue
		}
		label, _ := entryMap["label"].(string)
		desc, _ := entryMap["description"].(string)

		if seen[name] {
			for i := range items {
				if items[i].Name == name {
					if label != "" {
						items[i].Label = label
					}
					if desc != "" {
						items[i].Description = desc
					}
					break
				}
			}
		} else {
			seen[name] = true
			items = append(items, secretMeta{Name: name, Label: label, Description: desc})
		}
	}

	audit.Record(audit.Entry{
		Operation:    "secret_list",
		Category:     "data",
		Severity:     "low",
		UserID:       userID,
		TeamID:       teamID,
		Application:  assistantID,
		ResourceType: "secret",
		Source:       "mcp",
		Success:      true,
		Details:      map[string]any{"count": len(items)},
	})

	return map[string]any{"secrets": items}
}
