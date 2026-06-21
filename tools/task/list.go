package task

import (
	_ "embed"
	"fmt"

	"github.com/yaoapp/gou/process"
	tasksvc "github.com/yaoapp/yao/agent/task"
)

//go:embed list_schema.json
var ListSchemaJSON []byte

// ListHandler handles the task.list tool call from agents
func ListHandler(proc *process.Process) interface{} {
	if FnList == nil {
		return map[string]interface{}{"error": "task.list not available"}
	}

	auth := proc.Authorized
	args := proc.ArgsMap(0)

	q := &tasksvc.ListQuery{}
	if v, ok := args["run_status"].(string); ok {
		q.RunStatus = v
	}
	if v, ok := args["assistant_id"].(string); ok {
		q.AssistantID = v
	}
	if v, ok := args["board_id"].(string); ok {
		q.BoardID = v
	}
	if v, ok := args["page"].(float64); ok {
		q.Page = int(v)
	}
	if v, ok := args["page_size"].(float64); ok {
		q.PageSize = int(v)
	}

	result, err := FnList(proc.Context, auth, q)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("task.list: %s", err.Error())}
	}
	return result
}
