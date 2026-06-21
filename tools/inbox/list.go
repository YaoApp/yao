package inbox

import (
	_ "embed"
	"fmt"

	"github.com/yaoapp/gou/process"
	inboxsvc "github.com/yaoapp/yao/agent/inbox"
)

//go:embed list_schema.json
var ListSchemaJSON []byte

// ListHandler handles the inbox.list tool call from agents
func ListHandler(proc *process.Process) interface{} {
	if FnList == nil {
		return map[string]interface{}{"error": "inbox.list not available"}
	}

	auth := proc.Authorized
	args := proc.ArgsMap(0)

	q := &inboxsvc.ListQuery{}
	if v, ok := args["filter"].(string); ok {
		q.Filter = v
	}
	if v, ok := args["keyword"].(string); ok {
		q.Keyword = v
	}

	result, err := FnList(proc.Context, auth, q)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("inbox.list: %s", err.Error())}
	}
	return result
}
