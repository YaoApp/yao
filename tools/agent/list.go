package agent

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	goufs "github.com/yaoapp/gou/fs"
	"github.com/yaoapp/gou/process"
)

// ListHandler handles the agent_list tool.
// Args[0]: namespace (string, optional)
func ListHandler(proc *process.Process) interface{} {
	namespace := ""
	if len(proc.Args) > 0 {
		namespace = proc.ArgsString(0)
	}

	app, err := goufs.Get("app")
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("app filesystem: %s", err.Error())}
	}

	root := "/assistants"
	exists, _ := app.Exists(root)
	if !exists {
		return map[string]interface{}{"agents": []agentInfo{}}
	}

	nsDirs, err := app.ReadDir(root, false)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("read assistants dir: %s", err.Error())}
	}

	agents := make([]agentInfo, 0)
	for _, nsDir := range nsDirs {
		nsName := filepath.Base(nsDir)
		if namespace != "" && nsName != namespace {
			continue
		}

		agentDirs, err := app.ReadDir(nsDir, false)
		if err != nil {
			continue
		}

		for _, agentDir := range agentDirs {
			pkgFile := filepath.Join(agentDir, "package.yao")
			pkgExists, _ := app.Exists(pkgFile)
			if !pkgExists {
				continue
			}

			data, err := app.ReadFile(pkgFile)
			if err != nil {
				continue
			}

			var pkg packageDSL
			if err := json.Unmarshal(data, &pkg); err != nil {
				continue
			}

			agentName := filepath.Base(agentDir)
			id := nsName + "." + agentName

			if strings.HasPrefix(id, "__yao.") {
				continue
			}

			agents = append(agents, agentInfo{
				ID:           id,
				Name:         pkg.Name,
				Description:  pkg.Description,
				Capabilities: pkg.Capabilities,
			})
		}
	}

	return map[string]interface{}{"agents": agents}
}
