package image

import (
	"context"
	_ "embed"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/yaoapp/gou/process"
	agentLLM "github.com/yaoapp/yao/agent/llm"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	ws "github.com/yaoapp/yao/workspace"
)

//go:embed generate_schema.json
var GenerateSchemaJSON []byte

// GenerateHandler is the tools.image_generate process handler.
func GenerateHandler(proc *process.Process) interface{} {
	prompt := proc.ArgsString(0)
	if prompt == "" {
		return map[string]interface{}{"error": "prompt is required"}
	}

	provider := proc.ArgsString(1)
	size := proc.ArgsString(2, "1024x1024")
	model := proc.ArgsString(3)
	output := proc.ArgsString(4)

	authInfo := authorized.ProcessAuthInfo(proc)
	if authInfo == nil {
		return map[string]interface{}{"error": "unauthorized: no auth info in request"}
	}

	connectorID := provider
	if connectorID == "" {
		connectorID = findFirstImageGenConnector(authInfo)
		if connectorID == "" {
			return map[string]interface{}{"error": "no image generation provider available; configure one or specify a provider"}
		}
	}

	conn, _, err := agentLLM.ResolveConnector(connectorID, authInfo)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("resolve connector: %v", err)}
	}

	options := map[string]interface{}{"size": size}
	if model != "" {
		options["model"] = model
	}
	resp, err := agentLLM.GenerateImage(conn, prompt, options)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("image generation failed: %v", err)}
	}

	usedModel := resolveModelName(conn, model)

	if strings.HasPrefix(output, "workspace://") {
		imgBytes, err := base64.StdEncoding.DecodeString(resp.Image)
		if err != nil {
			return map[string]interface{}{"error": fmt.Sprintf("decode generated image: %v", err)}
		}
		if len(imgBytes) == 0 {
			return map[string]interface{}{"error": "generated image is empty"}
		}

		wsID, relPath := parseWorkspaceURI(output)
		ext := filepath.Ext(relPath)
		if ext == "" {
			relPath += "." + resp.Format
		}
		if err := ws.M().WriteFile(context.Background(), wsID, relPath, imgBytes, 0644); err != nil {
			return map[string]interface{}{"error": fmt.Sprintf("write to workspace: %v", err)}
		}
		return map[string]interface{}{
			"path":       "workspace://" + wsID + "/" + relPath,
			"format":     resp.Format,
			"dimensions": size,
			"model":      usedModel,
			"provider":   connectorID,
		}
	}

	return map[string]interface{}{
		"image":      resp.Image,
		"format":     resp.Format,
		"dimensions": size,
		"model":      usedModel,
		"provider":   connectorID,
	}
}

func parseWorkspaceURI(uri string) (string, string) {
	rest := strings.TrimPrefix(uri, "workspace://")
	idx := strings.Index(rest, "/")
	if idx < 0 {
		return rest, ""
	}
	return rest[:idx], rest[idx+1:]
}
