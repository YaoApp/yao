package robot

import (
	_ "embed"

	"github.com/yaoapp/gou/process"
)

//go:embed list_schema.json
var ListSchemaJSON []byte

func ListHandler(proc *process.Process) interface{} {
	info := authFromProcess(proc)
	if info == nil {
		return map[string]any{"error": "unauthorized"}
	}
	if ListAllRobotsFn == nil {
		return map[string]any{"error": "robot API not initialized"}
	}

	args := proc.ArgsMap(0)
	var requestedTeamID string
	if v, ok := args["team_id"].(string); ok {
		requestedTeamID = v
	}

	query := &ListQuery{
		TeamID: buildListFilter(info, requestedTeamID),
	}
	if v, ok := args["status"].(string); ok && v != "" {
		query.Status = v
	}
	if v, ok := args["keywords"].(string); ok {
		query.Keywords = v
	}
	if v, ok := args["page"].(float64); ok {
		query.Page = int(v)
	}
	if v, ok := args["pagesize"].(float64); ok {
		query.PageSize = int(v)
	}

	result, err := ListAllRobotsFn(proc.Context, info, query)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}

	return map[string]any{
		"data":     result.Data,
		"total":    result.Total,
		"page":     result.Page,
		"pagesize": result.PageSize,
	}
}
