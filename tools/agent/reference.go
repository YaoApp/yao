package agent

import (
	"fmt"
	"path/filepath"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/config"
)

// ReferenceHandler handles the agent_reference tool.
// Downloads agent source code to .references/ for read-only study.
// Args[0]: id (string, dot notation e.g. "yao.slides")
func ReferenceHandler(proc *process.Process) interface{} {
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
	dstPath := filepath.Join("agent-smith-dev", ".references", relPath)

	result, copyErr := wsFS.Copy(srcURI, dstPath)
	if copyErr != nil {
		return map[string]interface{}{"error": fmt.Sprintf("reference download failed: %s", copyErr.Error())}
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
