package robot

import (
	_ "embed"

	"github.com/yaoapp/gou/process"
)

//go:embed get_schema.json
var GetSchemaJSON []byte

func GetHandler(proc *process.Process) interface{} {
	memberID := proc.ArgsString(0)
	if memberID == "" {
		return map[string]any{"error": "member_id is required"}
	}

	info := authFromProcess(proc)
	if info == nil {
		return map[string]any{"error": "unauthorized"}
	}
	if GetRobotResponseFn == nil {
		return map[string]any{"error": "robot API not initialized"}
	}

	resp, err := GetRobotResponseFn(proc.Context, info, memberID)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	if !canRead(info, resp.YaoTeamID, resp.YaoCreatedBy) {
		return map[string]any{"error": "permission denied"}
	}

	return resp.Data
}
