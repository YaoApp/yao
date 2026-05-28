package robot

import (
	_ "embed"
	"encoding/json"

	"github.com/yaoapp/gou/process"
)

//go:embed update_schema.json
var UpdateSchemaJSON []byte

func UpdateHandler(proc *process.Process) interface{} {
	info := authFromProcess(proc)
	if info == nil {
		return map[string]any{"error": "unauthorized"}
	}
	if UpdateRobotFn == nil {
		return map[string]any{"error": "robot API not initialized"}
	}

	raw := proc.ArgsMap(0)
	memberID, _ := raw["member_id"].(string)
	if memberID == "" {
		return map[string]any{"error": "member_id is required"}
	}

	if err := checkRobotWrite(proc, memberID); err != nil {
		return map[string]any{"error": err.Error()}
	}

	delete(raw, "member_id")
	buf, err := json.Marshal(raw)
	if err != nil {
		return map[string]any{"error": "invalid arguments"}
	}

	var req UpdateRequest
	if err := json.Unmarshal(buf, &req); err != nil {
		return map[string]any{"error": "invalid arguments: " + err.Error()}
	}

	resp, err := UpdateRobotFn(proc.Context, info, memberID, &req)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}

	return resp.Data
}
