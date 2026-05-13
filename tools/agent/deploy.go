package agent

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/agent/caller"
	"github.com/yaoapp/yao/config"
)

// DeployHandler handles the agent_deploy tool.
// Args[0]: id (string, dot notation e.g. "smith.weather")
// Args[1]: message (string, optional deploy message)
func DeployHandler(proc *process.Process) interface{} {
	id := proc.ArgsString(0)
	if id == "" {
		return map[string]interface{}{"error": "id is required (e.g. 'smith.weather')"}
	}
	if err := validateID(id); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	parts := strings.SplitN(id, ".", 2)
	if len(parts) != 2 || parts[0] != allowedDeployNamespace {
		return map[string]interface{}{
			"status":  "error",
			"message": fmt.Sprintf("deploy restricted to namespace '%s'", allowedDeployNamespace),
		}
	}

	wsFS, err := resolveWorkspaceFS(proc)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	relPath := idToPath(id)
	appRoot := config.Conf.Root
	srcPath := filepath.Join("agent-smith-dev", "assistants", relPath)
	dstURI := "local:///" + filepath.Join(appRoot, "assistants", relPath)

	result, copyErr := wsFS.Copy(srcPath, dstURI)
	if copyErr != nil {
		return map[string]interface{}{"error": fmt.Sprintf("deploy failed: %s", copyErr.Error())}
	}

	files := 0
	if result != nil {
		files = result.FilesSynced
	}

	msg := ""
	if len(proc.Args) > 1 {
		msg = proc.ArgsString(1)
	}
	if msg != "" {
		log.Info("[agent_deploy] %s: %s (%d files)", id, msg, files)
	}

	if caller.AssistantReloadFunc != nil {
		if err := caller.AssistantReloadFunc(id); err != nil {
			log.Warn("[agent_deploy] reload %s: %s (files deployed, restart to apply)", id, err.Error())
		}
	}

	return map[string]interface{}{
		"status":       "ok",
		"path":         filepath.Join("assistants", relPath),
		"synced_files": files,
	}
}
