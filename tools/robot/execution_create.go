package robot

import (
	_ "embed"
	"encoding/json"

	"github.com/yaoapp/gou/process"
)

//go:embed execution_create_schema.json
var ExecutionCreateSchemaJSON []byte

func ExecutionCreateHandler(proc *process.Process) interface{} {
	info := authFromProcess(proc)
	if info == nil {
		return map[string]any{"error": "unauthorized"}
	}
	if TriggerFn == nil {
		return map[string]any{"error": "robot API not initialized"}
	}

	raw := proc.ArgsMap(0)
	memberID, _ := raw["member_id"].(string)
	if memberID == "" {
		return map[string]any{"error": "member_id is required"}
	}

	if tt, ok := raw["trigger_type"].(string); ok && tt != "" {
		raw["type"] = tt
	}
	if _, hasType := raw["type"]; !hasType {
		raw["type"] = "human"
	}

	delete(raw, "member_id")
	delete(raw, "trigger_type")

	buf, err := json.Marshal(raw)
	if err != nil {
		return map[string]any{"error": "invalid arguments"}
	}

	var req TriggerRequest
	if err := json.Unmarshal(buf, &req); err != nil {
		return map[string]any{"error": "invalid arguments: " + err.Error()}
	}

	if req.Type != "human" && req.Type != "event" {
		return map[string]any{"error": "trigger_type must be 'human' or 'event'"}
	}

	if err := checkRobotWrite(proc, memberID); err != nil {
		return map[string]any{"error": err.Error()}
	}

	result, err := TriggerFn(proc.Context, info, memberID, &req)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}

	return map[string]any{
		"execution_id": result.ExecutionID,
		"accepted":     result.Accepted,
		"status":       "submitted",
		"message":      result.Message,
	}
}
