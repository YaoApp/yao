package robot

import (
	_ "embed"

	"github.com/yaoapp/gou/process"
)

//go:embed execution_list_schema.json
var ExecutionListSchemaJSON []byte

func ExecutionListHandler(proc *process.Process) interface{} {
	memberID := proc.ArgsString(0)
	if memberID == "" {
		return map[string]any{"error": "member_id is required"}
	}

	if err := checkRobotRead(proc, memberID); err != nil {
		return map[string]any{"error": err.Error()}
	}

	if ListExecutionsFn == nil {
		return map[string]any{"error": "robot API not initialized"}
	}

	info := authFromProcess(proc)
	query := &ExecutionQuery{
		Status:   proc.ArgsString(1),
		Page:     proc.ArgsInt(2),
		PageSize: proc.ArgsInt(3),
	}

	result, err := ListExecutionsFn(proc.Context, info, memberID, query)
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
