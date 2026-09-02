package image

import (
	"context"
	_ "embed"
	"encoding/base64"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/yaoapp/gou/process"
	agentLLM "github.com/yaoapp/yao/agent/llm"
	"github.com/yaoapp/yao/llmprovider"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
	ws "github.com/yaoapp/yao/workspace"
)

//go:embed edit_schema.json
var EditSchemaJSON []byte

// EditHandler is the tools.image_edit process handler.
func EditHandler(proc *process.Process) interface{} {
	imageInput := proc.ArgsString(0)
	if imageInput == "" {
		return map[string]interface{}{"error": "image_path is required: provide a URL, workspace://, or attach:// URI"}
	}

	prompt := proc.ArgsString(1)
	if prompt == "" {
		return map[string]interface{}{"error": "prompt is required"}
	}

	provider := proc.ArgsString(2)
	size := proc.ArgsString(3, "1024x1024")
	model := proc.ArgsString(4)
	output := proc.ArgsString(5)
	allArgs := proc.ArgsMap(6)

	mask, _ := allArgs["mask"].(string)
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
		connectorID = findFirstImageEditConnector(authInfo)
		if connectorID == "" {
			connectorID = findFirstImageGenConnector(authInfo)
		}
		if connectorID == "" {
			return map[string]interface{}{"error": "no image editing provider available; configure one or specify a provider"}
		}
	}

	imageInput, err := resolveEditInput(imageInput)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("resolve image: %v", err)}
	}

	if mask != "" {
		mask, err = resolveEditInput(mask)
		if err != nil {
			return map[string]interface{}{"error": fmt.Sprintf("resolve mask: %v", err)}
		}
	}

	conn, caps, err := agentLLM.ResolveConnector(connectorID, authInfo)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("resolve connector: %v", err)}
	}

	editFormat := ""
	if caps != nil {
		editFormat = caps.GetImageEditingFormat()
	}

	options := map[string]interface{}{}
	for k, v := range extra {
		options[k] = v
	}
	options["size"] = size
	if model != "" {
		options["model"] = model
	}
	if mask != "" {
		options["mask"] = mask
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

	resp, err := agentLLM.EditImage(conn, imageInput, prompt, options, editFormat)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("image editing failed: %v", err)}
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
			return map[string]interface{}{"error": fmt.Sprintf("decode edited image: %v", err)}
		}
		if len(imgBytes) == 0 {
			return map[string]interface{}{"error": "edited image is empty"}
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

// resolveEditInput converts workspace://, attach://, and yao:// URIs to data URIs
// so that the downstream EditImage (which only handles data:, http(s):, and raw base64)
// can process them. Other inputs are passed through unchanged.
func resolveEditInput(input string) (string, error) {
	if !strings.HasPrefix(input, "workspace://") &&
		!strings.HasPrefix(input, "attach://") &&
		!strings.HasPrefix(input, "yao://") {
		return input, nil
	}
	raw, err := readBytes(input)
	if err != nil {
		return "", err
	}
	mime := http.DetectContentType(raw)
	if !strings.HasPrefix(mime, "image/") {
		mime = "image/png"
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(raw), nil
}

// findFirstImageEditConnector returns the connector ID of the first available
// image editing provider, or empty string if none found.
func findFirstImageEditConnector(authInfo *oauthTypes.AuthorizedInfo) string {
	if llmprovider.Global == nil {
		return ""
	}
	providers, err := listProvidersByCapability("image_editing", authInfo)
	if err != nil || len(providers) == 0 {
		return ""
	}
	if len(providers[0].Models) == 0 {
		return ""
	}
	return providers[0].Models[0].ConnectorID
}
