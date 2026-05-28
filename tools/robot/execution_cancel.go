package robot

import (
	_ "embed"

	"github.com/yaoapp/gou/process"
)

//go:embed execution_cancel_schema.json
var ExecutionCancelSchemaJSON []byte

func ExecutionCancelHandler(proc *process.Process) interface{} {
	memberID := proc.ArgsString(0)
	if memberID == "" {
		return map[string]any{"error": "member_id is required"}
	}

	execID := proc.ArgsString(1)
	if execID == "" {
		return map[string]any{"error": "execution_id is required"}
	}

	if err := checkRobotWrite(proc, memberID); err != nil {
		return map[string]any{"error": err.Error()}
	}

	if StopExecutionFn == nil {
		return map[string]any{"error": "robot API not initialized"}
	}

	info := authFromProcess(proc)
	if err := StopExecutionFn(proc.Context, info, execID); err != nil {
		return map[string]any{"error": err.Error()}
	}

	return map[string]any{
		"execution_id": execID,
		"status":       "cancelled",
	}
}
