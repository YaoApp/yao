package clip

import (
	_ "embed"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/openapi/audit"
)

//go:embed list_schema.json
var ListSchemaJSON []byte

// ListHandler is the tools.clip_list process handler.
// Lists all clips owned by the current user+team, returning metadata only.
func ListHandler(proc *process.Process) interface{} {
	auth := proc.Authorized
	if auth == nil {
		audit.Record(audit.Entry{
			Operation:    "clip_list",
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

	clips := listClips(userID, teamID)

	items := make([]map[string]any, 0, len(clips))
	for _, c := range clips {
		items = append(items, map[string]any{
			"id":          c.ID,
			"label":       c.Label,
			"description": c.Description,
		})
	}

	audit.Record(audit.Entry{
		Operation:    "clip_list",
		Category:     "data",
		Severity:     "low",
		UserID:       userID,
		TeamID:       teamID,
		ResourceType: "clip",
		Source:       "mcp",
		Success:      true,
	})

	return map[string]any{"clips": items}
}
