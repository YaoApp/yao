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
	allArgs := proc.ArgsMap(5)

	background, _ := allArgs["background"].(string)
	outputFormat, _ := allArgs["output_format"].(string)
	quality, _ := allArgs["quality"].(string)
	compression := extractIntArg(allArgs, "output_compression", -1)
	extra := extractExtra(allArgs)

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

	options := map[string]interface{}{}
	for k, v := range extra {
		options[k] = v
	}
	options["size"] = size
	if model != "" {
		options["model"] = model
	}
	if background != "" {
		options["background"] = background
	}
	if outputFormat != "" {
		options["output_format"] = outputFormat
	}
	if compression >= 0 {
		options["output_compression"] = compression
	}
	if quality != "" {
		options["quality"] = quality
	}

	resp, err := agentLLM.GenerateImage(conn, prompt, options)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("image generation failed: %v", err)}
	}

	usedModel := resolveModelName(conn, model)
	result := map[string]interface{}{
		"format":     resp.Format,
		"dimensions": size,
		"model":      usedModel,
		"provider":   connectorID,
	}
	if background != "" {
		result["background"] = background
	}
	if outputFormat != "" {
		result["output_format"] = outputFormat
	}
	if quality != "" {
		result["quality"] = quality
	}

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
		result["path"] = "workspace://" + wsID + "/" + relPath
		return result
	}

	result["image"] = resp.Image
	return result
}

func parseWorkspaceURI(uri string) (string, string) {
	rest := strings.TrimPrefix(uri, "workspace://")
	idx := strings.Index(rest, "/")
	if idx < 0 {
		return rest, ""
	}
	return rest[:idx], rest[idx+1:]
}
