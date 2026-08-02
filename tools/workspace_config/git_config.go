package workspace_config

import (
	_ "embed"

	"github.com/yaoapp/gou/process"
	ws "github.com/yaoapp/yao/workspace"
)

//go:embed git_config_schema.json
var GitConfigSchemaJSON []byte

// GitConfigHandler handles workspace_git_config tool calls.
//
// Args[0]: workspace_id (string, required)
// Args[1]: action (string: "get" | "set")
// Args[2]: key (string)
// Args[3]: value (string, for "set")
func GitConfigHandler(proc *process.Process) interface{} {
	if proc.Authorized == nil {
		return map[string]any{"error": "unauthorized"}
	}

	wsID := proc.ArgsString(0)
	action := proc.ArgsString(1)
	key := proc.ArgsString(2)

	if err := resolveAndCheck(proc, wsID); err != nil {
		return map[string]any{"error": err.Error()}
	}

	m := ws.M()
	switch action {
	case "get":
		values, err := m.GitConfigGet(proc.Context, wsID, key)
		if err != nil {
			return map[string]any{"error": err.Error()}
		}
		return map[string]any{"values": values}

	case "set":
		value := proc.ArgsString(3)
		if err := m.GitConfigSet(proc.Context, wsID, key, value); err != nil {
			return map[string]any{"error": err.Error()}
		}
		return map[string]any{"success": true}

	default:
		return map[string]any{"error": "action must be 'get' or 'set'"}
	}
}
