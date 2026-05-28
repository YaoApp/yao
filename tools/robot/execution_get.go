package robot

import (
	_ "embed"

	"github.com/yaoapp/gou/process"
)

//go:embed execution_get_schema.json
var ExecutionGetSchemaJSON []byte

func ExecutionGetHandler(proc *process.Process) interface{} {
	memberID := proc.ArgsString(0)
	if memberID == "" {
		return map[string]any{"error": "member_id is required"}
	}

	execID := proc.ArgsString(1)
	if execID == "" {
		return map[string]any{"error": "execution_id is required"}
	}

	if err := checkRobotRead(proc, memberID); err != nil {
		return map[string]any{"error": err.Error()}
	}

	if GetExecutionFn == nil {
		return map[string]any{"error": "robot API not initialized"}
	}

	info := authFromProcess(proc)
	result, err := GetExecutionFn(proc.Context, info, execID)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}

	return result.Data
}
