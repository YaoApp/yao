package agent

import (
	"fmt"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/llmprovider"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
)

// ConnectorsHandler handles the agent_connectors tool.
// No input args. Returns the current user's LLM connector matrix without keys.
func ConnectorsHandler(proc *process.Process) interface{} {
	authInfo := authorized.ProcessAuthInfo(proc)

	if llmprovider.Global == nil {
		return map[string]interface{}{"error": "llmprovider not initialized"}
	}

	var roles map[string]llmprovider.RoleTarget
	var err error

	if authInfo != nil {
		roles, err = llmprovider.Global.ListRolesBy(authInfo)
	} else {
		roles, err = llmprovider.Global.ListRoles()
	}
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("failed to list roles: %s", err.Error())}
	}

	result := make(map[string]interface{}, len(roles))
	for role, target := range roles {
		connID := target.Provider
		info := map[string]interface{}{
			"id":    connID,
			"model": target.Model,
		}

		conn, exists := connector.Connectors[connID]
		if exists {
			setting := conn.Setting()
			meta := conn.GetMetaInfo()
			if meta.Label != "" {
				info["name"] = meta.Label
			}
			if model, ok := setting["model"]; ok && info["model"] == "" {
				info["model"] = model
			}
			if caps, ok := setting["capabilities"]; ok {
				info["capabilities"] = sanitizeCapabilities(caps)
			}
			if t := settingStr(setting, "auth_mode"); t != "" {
				info["type"] = "openai"
			}
		}

		result[role] = info
	}

	return result
}
