package agent

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/config"
)

// DownloadHandler handles the agent_download tool.
// Restricted to the smith namespace — used for downloading agents to edit.
// For read-only reference of other namespaces, use agent_reference instead.
// Args[0]: id (string, dot notation e.g. "smith.weather")
func DownloadHandler(proc *process.Process) interface{} {
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
			"error": fmt.Sprintf("download restricted to '%s' namespace; use agent_reference for other agents", allowedDeployNamespace),
		}
	}

	wsFS, err := resolveWorkspaceFS(proc)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	relPath := idToPath(id)
	appRoot := config.Conf.Root
	srcURI := "local:///" + filepath.Join(appRoot, "assistants", relPath)
	dstPath := filepath.Join("agent-smith-dev", "assistants", relPath)

	result, copyErr := wsFS.Copy(srcURI, dstPath)
	if copyErr != nil {
		return map[string]interface{}{"error": fmt.Sprintf("download failed: %s", copyErr.Error())}
	}

	files := 0
	if result != nil {
		files = result.FilesSynced
	}

	return map[string]interface{}{
		"status": "ok",
		"path":   dstPath,
		"files":  files,
	}
}
