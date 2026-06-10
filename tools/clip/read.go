package clip

import (
	_ "embed"
	"fmt"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/openapi/audit"
)

//go:embed read_schema.json
var ReadSchemaJSON []byte

// ReadHandler is the tools.clip_read process handler.
// Reads a stored clip by ID and returns the full content.
//
// Args[0]: id (string) — the clip ID (UUID string)
func ReadHandler(proc *process.Process) interface{} {
	id := proc.ArgsString(0)
	if id == "" {
		return map[string]any{"error": "id is required"}
	}

	auth := proc.Authorized
	if auth == nil {
		audit.Record(audit.Entry{
			Operation:      "clip_read",
			Category:       "security",
			Severity:       "high",
			TargetResource: id,
			ResourceType:   "clip",
			Source:         "mcp",
			Success:        false,
			ErrorMessage:   "unauthorized: no auth context",
		})
		return map[string]any{"error": "unauthorized"}
	}

	userID := auth.UserID
	teamID := auth.TeamID

	c := readClip(userID, teamID, id)
	if c == nil {
		audit.Record(audit.Entry{
			Operation:      "clip_read",
			Category:       "data",
			Severity:       "medium",
			UserID:         userID,
			TeamID:         teamID,
			TargetResource: id,
			ResourceType:   "clip",
			Source:         "mcp",
			Success:        false,
			ErrorMessage:   fmt.Sprintf("clip %q not found", id),
		})
		return map[string]any{"error": fmt.Sprintf("clip %q not found", id)}
	}

	audit.Record(audit.Entry{
		Operation:      "clip_read",
		Category:       "data",
		Severity:       "low",
		UserID:         userID,
		TeamID:         teamID,
		TargetResource: id,
		ResourceType:   "clip",
		Source:         "mcp",
		Success:        true,
	})

	return map[string]any{
		"id":          c.ID,
		"label":       c.Label,
		"description": c.Description,
		"data":        c.Data,
	}
}
