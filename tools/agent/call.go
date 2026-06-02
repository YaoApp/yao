package agent

import (
	_ "embed"

	"github.com/yaoapp/gou/process"
)

//go:embed call_schema.json
var CallSchemaJSON []byte

// CallHandler handles the agent_call MCP tool.
// Delegates to the agent.Call Process which manages context, auth, and streaming.
//
// Args[0]: assistant_id (string, required)
// Args[1]: message (string, required)
// Args[2]: workspace_id (string, optional)
func CallHandler(proc *process.Process) interface{} {
	if len(proc.Args) < 2 {
		return map[string]interface{}{"error": "assistant_id and message are required"}
	}

	assistantID := proc.ArgsString(0)
	message := proc.ArgsString(1)
	if assistantID == "" || message == "" {
		return map[string]interface{}{"error": "assistant_id and message must not be empty"}
	}

	// Build the request payload for agent.Call
	req := map[string]interface{}{
		"assistant_id": assistantID,
		"messages": []map[string]interface{}{
			{"role": "user", "content": message},
		},
	}

	// If workspace_id is provided explicitly, pass it as metadata
	if len(proc.Args) > 2 {
		wsID := proc.ArgsString(2)
		if wsID != "" {
			req["metadata"] = map[string]interface{}{
				"workspace_id": wsID,
			}
		}
	}

	// Delegate to the existing agent.Call process which handles auth,
	// context construction, agent loading, and streaming.
	p := process.New("agent.Call", req)
	p.Context = proc.Context
	result := p.Run()

	return result
}
