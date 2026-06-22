package task

import (
	_ "embed"
	"fmt"

	"github.com/yaoapp/gou/process"
)

//go:embed stop_schema.json
var StopSchemaJSON []byte

// StopHandler handles the task.stop tool call from agents
func StopHandler(proc *process.Process) interface{} {
	if FnStop == nil {
		return map[string]interface{}{"error": "task.stop not available"}
	}

	auth := proc.Authorized
	args := proc.ArgsMap(0)

	chatID, _ := args["chat_id"].(string)
	if chatID == "" {
		return map[string]interface{}{"error": "task.stop: chat_id is required"}
	}

	force := false
	if v, ok := args["force"].(bool); ok {
		force = v
	}

	err := FnStop(proc.Context, auth, chatID, force)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("task.stop: %s", err.Error())}
	}
	return map[string]interface{}{"ok": true}
}
