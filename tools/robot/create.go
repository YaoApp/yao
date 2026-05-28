package robot

import (
	_ "embed"
	"encoding/json"

	"github.com/yaoapp/gou/process"
)

//go:embed create_schema.json
var CreateSchemaJSON []byte

func CreateHandler(proc *process.Process) interface{} {
	info := authFromProcess(proc)
	if info == nil {
		return map[string]any{"error": "unauthorized"}
	}
	if CreateRobotFn == nil {
		return map[string]any{"error": "robot API not initialized"}
	}

	raw := proc.ArgsMap(0)
	if raw == nil {
		return map[string]any{"error": "invalid arguments"}
	}

	buf, err := json.Marshal(raw)
	if err != nil {
		return map[string]any{"error": "invalid arguments"}
	}

	var req CreateRequest
	if err := json.Unmarshal(buf, &req); err != nil {
		return map[string]any{"error": "invalid arguments: " + err.Error()}
	}

	if req.DisplayName == "" {
		return map[string]any{"error": "display_name is required"}
	}

	resp, err := CreateRobotFn(proc.Context, info, &req)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}

	return resp.Data
}
