package inbox

import (
	_ "embed"
	"fmt"

	"github.com/yaoapp/gou/process"
)

//go:embed view_schema.json
var ViewSchemaJSON []byte

// ViewHandler handles the inbox.view tool call from agents
func ViewHandler(proc *process.Process) interface{} {
	if FnView == nil {
		return map[string]interface{}{"error": "inbox.view not available"}
	}

	auth := proc.Authorized
	args := proc.ArgsMap(0)

	chatID, _ := args["chat_id"].(string)
	if chatID == "" {
		return map[string]interface{}{"error": "inbox.view: chat_id is required"}
	}

	err := FnView(proc.Context, auth, chatID)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("inbox.view: %s", err.Error())}
	}
	return map[string]interface{}{"status": "ok"}
}
