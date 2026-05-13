package agent

import (
	"fmt"
	"path/filepath"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/config"
)

// DownloadHandler handles the agent_download tool.
// Args[0]: id (string, dot notation e.g. "yao.slides")
func DownloadHandler(proc *process.Process) interface{} {
	id := proc.ArgsString(0)
	if id == "" {
		return map[string]interface{}{"error": "id is required (e.g. 'yao.slides')"}
	}
	if err := validateID(id); err != nil {
		return map[string]interface{}{"error": err.Error()}
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
