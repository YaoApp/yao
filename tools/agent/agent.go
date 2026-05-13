package agent

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yaoapp/gou/process"
	taiworkspace "github.com/yaoapp/yao/tai/workspace"
	ws "github.com/yaoapp/yao/workspace"
	"google.golang.org/grpc/metadata"
)

//go:embed list_schema.json
var ListSchemaJSON []byte

//go:embed download_schema.json
var DownloadSchemaJSON []byte

//go:embed deploy_schema.json
var DeploySchemaJSON []byte

//go:embed reference_schema.json
var ReferenceSchemaJSON []byte

//go:embed connectors_schema.json
var ConnectorsSchemaJSON []byte

const allowedDeployNamespace = "smith"

type agentInfo struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Capabilities string `json:"capabilities,omitempty"`
}

type packageDSL struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	Capabilities string `json:"capabilities"`
}

func resolveWorkspaceFS(proc *process.Process) (taiworkspace.FS, error) {
	workspaceID := extractWorkspaceID(proc)
	if workspaceID == "" {
		return nil, fmt.Errorf("workspace_id not available (container must set CTX_WORKSPACE_ID)")
	}

	fs, err := ws.M().FS(context.Background(), workspaceID)
	if err != nil {
		return nil, fmt.Errorf("workspace %s: %w", workspaceID, err)
	}
	return fs, nil
}

func extractWorkspaceID(proc *process.Process) string {
	if proc.Context == nil {
		return ""
	}
	md, ok := metadata.FromIncomingContext(proc.Context)
	if !ok {
		return ""
	}
	ids := md.Get("x-workspace-id")
	if len(ids) > 0 && ids[0] != "" {
		return ids[0]
	}
	return ""
}

func extractLocale(proc *process.Process) string {
	if proc.Context == nil {
		return "en-us"
	}
	md, ok := metadata.FromIncomingContext(proc.Context)
	if !ok {
		return "en-us"
	}
	vals := md.Get("x-locale")
	if len(vals) > 0 && vals[0] != "" {
		return strings.ToLower(vals[0])
	}
	return "en-us"
}

func validateID(id string) error {
	if strings.Contains(id, "..") {
		return fmt.Errorf("invalid id: path traversal not allowed")
	}
	if strings.ContainsAny(id, "/\\") {
		return fmt.Errorf("invalid id: use dot notation (e.g. 'yao.slides')")
	}
	parts := strings.SplitN(id, ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("invalid id format: expected 'namespace.name' (e.g. 'yao.slides')")
	}
	return nil
}

func idToPath(id string) string {
	return strings.Replace(id, ".", "/", 1)
}

func settingStr(setting map[string]interface{}, key string) string {
	if v, ok := setting[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func sanitizeCapabilities(caps interface{}) interface{} {
	data, err := json.Marshal(caps)
	if err != nil {
		return nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return caps
	}
	delete(m, "key")
	delete(m, "secret")
	delete(m, "token")
	return m
}
