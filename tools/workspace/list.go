package workspace

import (
	_ "embed"

	"github.com/yaoapp/gou/process"
	ws "github.com/yaoapp/yao/workspace"
)

//go:embed list_schema.json
var ListSchemaJSON []byte

// ListHandler is the tools.workspace_list process handler.
//
// Args[0]: node (string, optional) — filter by Tai node
func ListHandler(proc *process.Process) interface{} {
	auth := proc.Authorized
	if auth == nil {
		return map[string]any{"error": "unauthorized"}
	}

	m := ws.M()
	if m == nil {
		return map[string]any{"workspaces": []any{}}
	}

	node := proc.ArgsString(0)
	list, err := m.List(proc.Context, ws.ListOptions{
		Owner: resolveOwner(auth),
		Node:  node,
	})
	if err != nil {
		return map[string]any{"error": err.Error()}
	}

	result := make([]map[string]any, 0, len(list))
	for _, w := range list {
		result = append(result, map[string]any{
			"id":         w.ID,
			"name":       w.Name,
			"node":       w.Node,
			"created_at": w.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}
	return map[string]any{"workspaces": result}
}
