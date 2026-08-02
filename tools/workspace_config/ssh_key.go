package workspace_config

import (
	_ "embed"

	"github.com/yaoapp/gou/process"
	ws "github.com/yaoapp/yao/workspace"
)

//go:embed ssh_key_schema.json
var SSHKeySchemaJSON []byte

// SSHKeyHandler handles workspace_ssh_key tool calls.
//
// Args[0]: workspace_id (string, required)
// Args[1]: action (string: "import" | "list" | "delete")
// Args[2]: name (string, for "import" and "delete")
// Args[3]: private_key (string, for "import")
// Args[4]: public_key (string, for "import", optional)
// Args[5]: host (string, for "import", optional)
func SSHKeyHandler(proc *process.Process) interface{} {
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
	case "import":
		name := proc.ArgsString(2)
		privateKey := proc.ArgsString(3)
		publicKey := proc.ArgsString(4)
		host := proc.ArgsString(5)
		if err := m.GitSSHKeyImport(proc.Context, wsID, name, privateKey, publicKey, host); err != nil {
			return map[string]any{"error": err.Error()}
		}
		return map[string]any{"success": true}

	case "list":
		keys, err := m.GitSSHKeyList(proc.Context, wsID)
		if err != nil {
			return map[string]any{"error": err.Error()}
		}
		result := make([]map[string]any, 0, len(keys))
		for _, k := range keys {
			result = append(result, map[string]any{
				"name":        k.Name,
				"public_key":  k.PublicKey,
				"fingerprint": k.Fingerprint,
			})
		}
		return map[string]any{"keys": result}

	case "delete":
		name := proc.ArgsString(2)
		if err := m.GitSSHKeyDelete(proc.Context, wsID, name); err != nil {
			return map[string]any{"error": err.Error()}
		}
		return map[string]any{"success": true}

	default:
		return map[string]any{"error": "action must be 'import', 'list', or 'delete'"}
	}
}
