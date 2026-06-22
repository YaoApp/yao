package board

import (
	_ "embed"
	"fmt"

	"github.com/yaoapp/gou/process"
	boardsvc "github.com/yaoapp/yao/agent/board"
)

//go:embed create_schema.json
var CreateSchemaJSON []byte

// CreateHandler handles the board.create tool call from agents
func CreateHandler(proc *process.Process) interface{} {
	if FnCreate == nil {
		return map[string]interface{}{"error": "board.create not available"}
	}

	auth := proc.Authorized
	args := proc.ArgsMap(0)

	req := &boardsvc.CreateReq{}
	if v, ok := args["name"].(string); ok {
		req.Name = v
	}
	if v, ok := args["icon"].(string); ok {
		req.Icon = v
	}
	if v, ok := args["color"].(string); ok {
		req.Color = v
	}

	result, err := FnCreate(proc.Context, auth, req)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("board.create: %s", err.Error())}
	}
	return result
}
