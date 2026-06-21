package inbox

import (
	_ "embed"
	"fmt"

	"github.com/yaoapp/gou/process"
)

//go:embed read_schema.json
var ReadSchemaJSON []byte

// ReadHandler handles the inbox.read tool call from agents
func ReadHandler(proc *process.Process) interface{} {
	if FnRead == nil {
		return map[string]interface{}{"error": "inbox.read not available"}
	}

	auth := proc.Authorized
	args := proc.ArgsMap(0)

	mailID, _ := args["mail_id"].(string)
	if mailID == "" {
		return map[string]interface{}{"error": "inbox.read: mail_id is required"}
	}

	err := FnRead(proc.Context, auth, mailID)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("inbox.read: %s", err.Error())}
	}
	return map[string]interface{}{"status": "ok"}
}
