package ocr

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yaoapp/gou/process"
	agentLLM "github.com/yaoapp/yao/agent/llm"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/setting"
	"github.com/yaoapp/yao/tools/image"
)

//go:embed recognize_schema.json
var RecognizeSchemaJSON []byte

// RecognizeHandler is the tools.ocr_recognize process handler.
func RecognizeHandler(proc *process.Process) interface{} {
	source := proc.ArgsString(0)
	if source == "" {
		return map[string]interface{}{"error": "source is required: provide a file path, URL, or URI"}
	}

	provider := proc.ArgsString(1)
	ocrType := proc.ArgsString(2, "general")
	outputFormat := proc.ArgsString(3, "text")
	mode := proc.ArgsString(4, "accurate")
	language := proc.ArgsString(5)
	prompt := proc.ArgsString(6)
	allArgs := proc.ArgsMap(7)

	pages, _ := allArgs["pages"].(string)
	extra := extractExtra(allArgs)

	authInfo := authorized.ProcessAuthInfo(proc)
	if authInfo == nil {
		return map[string]interface{}{"error": "unauthorized: no auth info in request"}
	}

	if ocrType != "" && !isValidType(ocrType) {
		return map[string]interface{}{"error": fmt.Sprintf("invalid type %q; valid values: general, table, handwriting, invoice, receipt, id_card, bank_card, license, vehicle_license, passport, license_plate, document", ocrType)}
	}
	ocrType = defaultType(ocrType)

	result, err := recognize(proc.Context, recognizeArgs{
		source:       source,
		provider:     provider,
		outputFormat: outputFormat,
		mode:         mode,
		language:     language,
		pages:        pages,
		prompt:       prompt,
		ocrType:      ocrType,
		extra:        extra,
		authInfo:     authInfo,
	})
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return result
}

type recognizeArgs struct {
	source       string
	provider     string
	outputFormat string
	mode         string
	language     string
	pages        string
	prompt       string
	ocrType      string
	extra        map[string]interface{}
	authInfo     *oauthTypes.AuthorizedInfo
}

func recognize(goCtx context.Context, args recognizeArgs) (interface{}, error) {
	// 1. Read source bytes + MIME type
	data, mimeType, err := image.ReadBytes(args.source)
	if err != nil {
		return nil, fmt.Errorf("read source: %w", err)
	}

	// 2. Resolve provider
	providerID, providerType, err := resolveProvider(args.provider, args.authInfo)
	if err != nil {
		return nil, err
	}

	// 3. Select handler and inject credentials
	handler, err := selectHandler(providerID, providerType, args.authInfo)
	if err != nil {
		return nil, err
	}

	// 4. Type degradation
	actual, degradedFrom := degradeType(args.ocrType, handler.SupportedTypes())

	// 5. Build unified request
	req := &OCRRequest{
		Source:       data,
		MimeType:     mimeType,
		SourcePath:   args.source,
		Type:         actual,
		OutputFormat: args.outputFormat,
		Mode:         args.mode,
		Language:     args.language,
		Pages:        args.pages,
		Prompt:       args.prompt,
		Extra:        args.extra,
	}

	// 6. Call handler
	resp, err := handler.Recognize(goCtx, req)
	if err != nil {
		return nil, fmt.Errorf("ocr recognize (%s): %w", providerID, err)
	}

	// 7. Attach metadata
	if resp.Metadata == nil {
		resp.Metadata = map[string]string{}
	}
	resp.Metadata["provider"] = providerID
	if degradedFrom != "" {
		resp.Metadata["degraded_from"] = degradedFrom
	}

	// 8. Format output
	return formatResponse(resp, args.outputFormat, actual), nil
}

// resolveProvider determines which provider to use and its type ("llm" or "api").
// Priority: explicit arg > tool_assignment setting > first enabled OCR provider.
func resolveProvider(explicit string, authInfo *oauthTypes.AuthorizedInfo) (id string, kind string, err error) {
	if explicit != "" {
		return classifyProvider(explicit, authInfo)
	}

	// Read tool_assignment from settings
	if setting.Global != nil {
		merged, mergeErr := setting.Global.GetMerged(
			authInfo.GetUserID(), authInfo.GetTeamID(), "ocr.tool_assignment",
		)
		if mergeErr == nil {
			if assigned, ok := merged["ocr_recognize"].(string); ok && assigned != "" {
				return classifyProvider(assigned, authInfo)
			}
		}
	}

	// Fallback: find first enabled OCR API provider
	if setting.Global != nil {
		for _, key := range []string{"paddleocr", "baidu", "google", "azure"} {
			merged, mergeErr := setting.Global.GetMerged(
				authInfo.GetUserID(), authInfo.GetTeamID(), "ocr.providers."+key,
			)
			if mergeErr != nil {
				continue
			}
			if enabled, ok := merged["enabled"].(bool); ok && enabled {
				return key, "api", nil
			}
		}
	}

	// Fallback: find first LLM connector with OCR capability
	providers, listErr := image.ListProvidersByCapability("ocr", authInfo)
	if listErr == nil && len(providers) > 0 {
		if cid := providers[0].FirstConnectorID(); cid != "" {
			return cid, "llm", nil
		}
	}

	return "", "", fmt.Errorf("no OCR provider configured; add one via Settings > OCR or specify the provider parameter")
}

// classifyProvider identifies whether a provider ID is an LLM connector or traditional API key.
func classifyProvider(id string, authInfo *oauthTypes.AuthorizedInfo) (string, string, error) {
	if strings.HasPrefix(id, "llm:") {
		return strings.TrimPrefix(id, "llm:"), "llm", nil
	}

	// Check if it's a known OCR preset key
	for _, key := range []string{"paddleocr", "baidu", "google", "azure"} {
		if id == key {
			return id, "api", nil
		}
	}

	// Try resolving as LLM connector
	conn, caps, err := agentLLM.ResolveConnector(id, authInfo)
	if err == nil && conn != nil && caps != nil && caps.HasOCR() {
		return id, "llm", nil
	}

	return "", "", fmt.Errorf("unknown provider %q: not a configured OCR API or OCR-capable LLM connector", id)
}

// selectHandler instantiates the appropriate ProviderHandler with credentials.
func selectHandler(providerID, providerType string, authInfo *oauthTypes.AuthorizedInfo) (ProviderHandler, error) {
	if providerType == "llm" {
		return &VLMHandler{ConnectorID: providerID, AuthInfo: authInfo}, nil
	}

	// Traditional API: read credentials from settings
	creds, err := readOCRCredentials(providerID, authInfo)
	if err != nil {
		return nil, fmt.Errorf("read %s credentials: %w", providerID, err)
	}

	switch providerID {
	case "paddleocr":
		return &PaddleOCRHandler{
			BaseURL: creds["base_url"],
			APIKey:  creds["api_key"],
		}, nil
	case "baidu":
		return &BaiduHandler{
			APIKey:    creds["api_key"],
			SecretKey: creds["secret_key"],
		}, nil
	case "google":
		return &GoogleHandler{
			APIKey: creds["api_key"],
		}, nil
	case "azure":
		return &AzureHandler{
			APIKey:   creds["api_key"],
			Endpoint: creds["endpoint"],
		}, nil
	default:
		return nil, fmt.Errorf("unsupported OCR provider: %s", providerID)
	}
}

// readOCRCredentials reads decrypted credential values for a traditional OCR provider.
func readOCRCredentials(providerKey string, authInfo *oauthTypes.AuthorizedInfo) (map[string]string, error) {
	if setting.Global == nil {
		return nil, fmt.Errorf("setting registry not initialized")
	}

	merged, err := setting.Global.GetMerged(
		authInfo.GetUserID(), authInfo.GetTeamID(), "ocr.providers."+providerKey,
	)
	if err != nil {
		return nil, fmt.Errorf("provider %s not configured", providerKey)
	}

	result := map[string]string{}
	if fv, ok := merged["field_values"].(map[string]interface{}); ok {
		for k, v := range fv {
			if s, ok := v.(string); ok {
				result[k] = setting.Decrypt(s)
			}
		}
	}
	return result, nil
}

// formatResponse converts an OCRResponse to the requested output format.
func formatResponse(resp *OCRResponse, outputFormat, ocrType string) interface{} {
	switch outputFormat {
	case "json":
		result := map[string]interface{}{
			"text":  resp.Text,
			"pages": resp.Pages,
		}
		if len(resp.Blocks) > 0 {
			result["blocks"] = resp.Blocks
		}
		if len(resp.Fields) > 0 {
			result["fields"] = resp.Fields
		}
		if len(resp.Metadata) > 0 {
			result["metadata"] = resp.Metadata
		}
		return result

	case "markdown":
		// For structured types with Fields, format as KV list regardless of Text
		if len(resp.Fields) > 0 {
			return formatMarkdown(resp, ocrType)
		}
		// For general/document types, resp.Text often already contains Markdown
		if resp.Text != "" {
			return resp.Text
		}
		return formatMarkdown(resp, ocrType)

	default: // "text"
		return resp.Text
	}
}

// formatMarkdown builds a Markdown representation from structured OCR data.
func formatMarkdown(resp *OCRResponse, ocrType string) string {
	var sb strings.Builder

	// Structured fields → key-value list
	if len(resp.Fields) > 0 {
		for k, v := range resp.Fields {
			sb.WriteString(fmt.Sprintf("- **%s**: %v\n", k, v))
		}
		return sb.String()
	}

	// Blocks → plain text fallback
	for _, b := range resp.Blocks {
		sb.WriteString(b.Text)
		sb.WriteString("\n")
	}
	if sb.Len() > 0 {
		return sb.String()
	}
	return ""
}

// extractExtra pulls recognized extra keys from the all-args map.
func extractExtra(allArgs map[string]interface{}) map[string]interface{} {
	extra := map[string]interface{}{}
	if e, ok := allArgs["extra"].(map[string]interface{}); ok {
		for k, v := range e {
			extra[k] = v
		}
	}
	return extra
}

// extractJSONStr is a helper to extract a string field from raw JSON bytes.
func extractJSONStr(data []byte, key string) string {
	var m map[string]interface{}
	if json.Unmarshal(data, &m) != nil {
		return ""
	}
	s, _ := m[key].(string)
	return s
}
