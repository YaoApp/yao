package robot

import (
	_ "embed"

	"github.com/yaoapp/gou/process"
)

//go:embed result_list_schema.json
var ResultListSchemaJSON []byte

func ResultListHandler(proc *process.Process) interface{} {
	memberID := proc.ArgsString(0)
	if memberID == "" {
		return map[string]any{"error": "member_id is required"}
	}

	if err := checkRobotRead(proc, memberID); err != nil {
		return map[string]any{"error": err.Error()}
	}

	if ListResultsFn == nil {
		return map[string]any{"error": "robot API not initialized"}
	}

	info := authFromProcess(proc)
	query := &ResultQuery{
		Page:     proc.ArgsInt(1),
		PageSize: proc.ArgsInt(2),
	}

	result, err := ListResultsFn(proc.Context, info, memberID, query)
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
