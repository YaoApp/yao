package image

import (
	_ "embed"
	"fmt"

	"github.com/yaoapp/gou/process"
	agentLLM "github.com/yaoapp/yao/agent/llm"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
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

	return map[string]interface{}{
		"image":  resp.Image,
		"format": resp.Format,
		"size":   size,
	}
}
