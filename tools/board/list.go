package board

import (
	_ "embed"
	"fmt"

	"github.com/yaoapp/gou/process"
	boardsvc "github.com/yaoapp/yao/agent/board"
)

//go:embed list_schema.json
var ListSchemaJSON []byte

// ListHandler handles the board.list tool call from agents
func ListHandler(proc *process.Process) interface{} {
	if FnList == nil {
		return map[string]interface{}{"error": "board.list not available"}
	}

	auth := proc.Authorized
	result, err := FnList(proc.Context, auth, &boardsvc.ListQuery{})
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("board.list: %s", err.Error())}
	}
	return result
}
