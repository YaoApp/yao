package robot

import (
	_ "embed"

	"github.com/yaoapp/gou/process"
)

//go:embed status_schema.json
var StatusSchemaJSON []byte

func StatusHandler(proc *process.Process) interface{} {
	memberID := proc.ArgsString(0)
	if memberID == "" {
		return map[string]any{"error": "member_id is required"}
	}

	info := authFromProcess(proc)
	if info == nil {
		return map[string]any{"error": "unauthorized"}
	}
	if GetRobotStatusFn == nil {
		return map[string]any{"error": "robot API not initialized"}
	}

	state, err := GetRobotStatusFn(proc.Context, info, memberID)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	if !canRead(info, state.YaoTeamID, state.YaoCreatedBy) {
		return map[string]any{"error": "permission denied"}
	}

	return map[string]any{
		"member_id":    state.MemberID,
		"team_id":      state.TeamID,
		"display_name": state.DisplayName,
		"bio":          state.Bio,
		"status":       state.Status,
		"running":      state.Running,
		"max_running":  state.MaxRunning,
		"running_ids":  state.RunningIDs,
		"last_run":     state.LastRun,
		"next_run":     state.NextRun,
	}
}
