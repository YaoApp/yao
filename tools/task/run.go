package task

import (
	_ "embed"
	"fmt"

	"github.com/yaoapp/gou/process"
	tasksvc "github.com/yaoapp/yao/agent/task"
)

//go:embed run_schema.json
var RunSchemaJSON []byte

// RunHandler handles the task.run tool call from agents
func RunHandler(proc *process.Process) interface{} {
	if FnRun == nil {
		return map[string]interface{}{"error": "task.run not available"}
	}

	auth := proc.Authorized
	args := proc.ArgsMap(0)

	chatID, _ := args["chat_id"].(string)
	if chatID == "" {
		return map[string]interface{}{"error": "task.run: chat_id is required"}
	}

	req := &tasksvc.RunReq{}
	if msg, ok := args["message"].(string); ok && msg != "" {
		req.Messages = []tasksvc.InputMessage{{Role: "user", Content: msg}}
	}

	result, err := FnRun(proc.Context, auth, chatID, req)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("task.run: %s", err.Error())}
	}
	return result
}
