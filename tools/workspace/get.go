package workspace

import (
	_ "embed"

	"github.com/yaoapp/gou/process"
)

//go:embed get_schema.json
var GetSchemaJSON []byte

// GetHandler is the tools.workspace_get process handler.
//
// Args[0]: id (string, required) — workspace ID
func GetHandler(proc *process.Process) interface{} {
	auth := proc.Authorized
	if auth == nil {
		return map[string]any{"error": "unauthorized"}
	}

	id := proc.ArgsString(0)
	if id == "" {
		return map[string]any{"error": "id is required"}
	}

	w, err := resolveAndCheck(proc, id)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}

	return map[string]any{
		"id":         w.ID,
		"name":       w.Name,
		"owner":      w.Owner,
		"node":       w.Node,
		"labels":     w.Labels,
		"created_at": w.CreatedAt.Format("2006-01-02T15:04:05Z"),
		"updated_at": w.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
