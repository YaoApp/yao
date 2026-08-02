package workspace_config

import (
	_ "embed"

	"github.com/yaoapp/gou/process"
	ws "github.com/yaoapp/yao/workspace"
)

//go:embed git_credential_schema.json
var GitCredentialSchemaJSON []byte

// GitCredentialHandler handles workspace_git_credential tool calls.
//
// Args[0]: workspace_id (string, required)
// Args[1]: action (string: "set" | "list" | "delete")
// Args[2]: host (string, for "set" and "delete")
// Args[3]: username (string, for "set", optional)
// Args[4]: token (string, for "set")
func GitCredentialHandler(proc *process.Process) interface{} {
	if proc.Authorized == nil {
		return map[string]any{"error": "unauthorized"}
	}

	wsID := proc.ArgsString(0)
	action := proc.ArgsString(1)

	if err := resolveAndCheck(proc, wsID); err != nil {
		return map[string]any{"error": err.Error()}
	}

	m := ws.M()
	switch action {
	case "set":
		host := proc.ArgsString(2)
		username := proc.ArgsString(3)
		token := proc.ArgsString(4)
		if err := m.GitCredentialSet(proc.Context, wsID, host, username, token); err != nil {
			return map[string]any{"error": err.Error()}
		}
		return map[string]any{"success": true}

	case "list":
		entries, err := m.GitCredentialList(proc.Context, wsID)
		if err != nil {
			return map[string]any{"error": err.Error()}
		}
		result := make([]map[string]any, 0, len(entries))
		for _, e := range entries {
			result = append(result, map[string]any{
				"host":     e.Host,
				"username": e.Username,
			})
		}
		return map[string]any{"credentials": result}

	case "delete":
		host := proc.ArgsString(2)
		if err := m.GitCredentialDelete(proc.Context, wsID, host); err != nil {
			return map[string]any{"error": err.Error()}
		}
		return map[string]any{"success": true}

	default:
		return map[string]any{"error": "action must be 'set', 'list', or 'delete'"}
	}
}
