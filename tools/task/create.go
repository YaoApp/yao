package task

import (
	_ "embed"
	"fmt"

	"github.com/yaoapp/gou/process"
	tasksvc "github.com/yaoapp/yao/agent/task"
)

//go:embed create_schema.json
var CreateSchemaJSON []byte

// CreateHandler handles the task.create tool call from agents
func CreateHandler(proc *process.Process) interface{} {
	if FnCreate == nil {
		return map[string]interface{}{"error": "task.create not available"}
	}

	auth := proc.Authorized
	args := proc.ArgsMap(0)

	req := &tasksvc.CreateReq{}
	if v, ok := args["chat_id"].(string); ok {
		req.ChatID = v
	}
	if v, ok := args["title"].(string); ok {
		req.Title = v
	}
	if v, ok := args["assistant_id"].(string); ok {
		req.AssistantID = v
	}
	if v, ok := args["column_id"].(string); ok {
		req.ColumnID = v
	}

	result, err := FnCreate(proc.Context, auth, req)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("task.create: %s", err.Error())}
	}
	return result
}
