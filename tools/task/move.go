package task

import (
	_ "embed"
	"fmt"

	"github.com/yaoapp/gou/process"
	tasksvc "github.com/yaoapp/yao/agent/task"
)

//go:embed move_schema.json
var MoveSchemaJSON []byte

// MoveHandler handles the task.move tool call from agents
func MoveHandler(proc *process.Process) interface{} {
	if FnMove == nil {
		return map[string]interface{}{"error": "task.move not available"}
	}

	auth := proc.Authorized
	args := proc.ArgsMap(0)

	chatID, _ := args["chat_id"].(string)
	if chatID == "" {
		return map[string]interface{}{"error": "task.move: chat_id is required"}
	}

	req := &tasksvc.MoveReq{}
	if v, ok := args["column_id"].(string); ok {
		req.ColumnID = v
	}
	if v, ok := args["position"].(float64); ok {
		req.Position = int(v)
	}

	err := FnMove(proc.Context, auth, chatID, req)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("task.move: %s", err.Error())}
	}
	return map[string]interface{}{"status": "ok"}
}
