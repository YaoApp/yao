package image

import (
	_ "embed"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/yaoapp/gou/process"
	agentLLM "github.com/yaoapp/yao/agent/llm"
	"github.com/yaoapp/yao/llmprovider"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
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

	conn, caps, err := agentLLM.ResolveConnector(connectorID, authInfo)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("resolve connector: %v", err)}
	}

	editFormat := ""
	if caps != nil {
		editFormat = caps.GetImageEditingFormat()
	}

	options := map[string]interface{}{"size": size}
	if model != "" {
		options["model"] = model
	}

	resp, err := agentLLM.EditImage(conn, imageInput, prompt, options, editFormat)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("image editing failed: %v", err)}
	}

	return map[string]interface{}{
		"image":  resp.Image,
		"format": resp.Format,
		"size":   size,
	}
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
