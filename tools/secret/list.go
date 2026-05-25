package secret

import (
	_ "embed"
	"fmt"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/openapi/audit"
	"github.com/yaoapp/yao/setting"
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

	if setting.Global == nil {
		return map[string]any{"error": "setting registry not initialized"}
	}

	ns := resolveNamespace(assistantID)
	merged, err := setting.Global.GetMerged(userID, teamID, ns)
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

	// 2) Merge user-configured secrets from setting.Registry
	if secretsRaw, ok := merged["secrets"]; ok {
		if secretsMap, ok := secretsRaw.(map[string]interface{}); ok {
			for name, entryRaw := range secretsMap {
				entryMap, ok := entryRaw.(map[string]interface{})
				if !ok {
					continue
				}
				label, _ := entryMap["label"].(string)
				desc, _ := entryMap["description"].(string)

				if seen[name] {
					// Update predefined entry with user-provided label/desc if non-empty
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
					items = append(items, secretMeta{Name: name, Label: label, Description: desc})
				}
			}
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
