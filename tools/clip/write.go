package clip

import (
	_ "embed"
	"fmt"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/openapi/audit"
)

//go:embed write_schema.json
var WriteSchemaJSON []byte

// WriteHandler is the tools.clip_write process handler.
// Stores a content clip and returns its ID for later retrieval.
//
// Args[0]: label (string)
// Args[1]: description (string)
// Args[2]: data (map[string]string)
func WriteHandler(proc *process.Process) interface{} {
	label := proc.ArgsString(0)
	description := proc.ArgsString(1)
	dataRaw := proc.ArgsMap(2)

	if label == "" {
		return map[string]any{"error": "label is required"}
	}
	if description == "" {
		return map[string]any{"error": "description is required"}
	}

	auth := proc.Authorized
	if auth == nil {
		audit.Record(audit.Entry{
			Operation:    "clip_write",
			Category:     "security",
			Severity:     "high",
			ResourceType: "clip",
			Source:       "mcp",
			Success:      false,
			ErrorMessage: "unauthorized: no auth context",
		})
		return map[string]any{"error": "unauthorized"}
	}

	userID := auth.UserID
	teamID := auth.TeamID

	data := make(map[string]string)
	for k, v := range dataRaw {
		data[k] = fmt.Sprintf("%v", v)
	}

	if dataSize(data) > maxDataSize {
		audit.Record(audit.Entry{
			Operation:    "clip_write",
			Category:     "data",
			Severity:     "medium",
			UserID:       userID,
			TeamID:       teamID,
			ResourceType: "clip",
			Source:       "mcp",
			Success:      false,
			ErrorMessage: "data exceeds 5MB limit",
		})
		return map[string]any{"error": "data exceeds 5MB limit"}
	}

	c := writeClip(userID, teamID, label, description, data)

	audit.Record(audit.Entry{
		Operation:      "clip_write",
		Category:       "data",
		Severity:       "low",
		UserID:         userID,
		TeamID:         teamID,
		TargetResource: c.ID,
		ResourceType:   "clip",
		Source:         "mcp",
		Success:        true,
	})

	return map[string]any{
		"id":          c.ID,
		"label":       c.Label,
		"description": c.Description,
	}
}
