package ocr

import (
	_ "embed"
	"sort"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/setting"
	"github.com/yaoapp/yao/tools/image"
)

//go:embed providers_schema.json
var ProvidersSchemaJSON []byte

type ocrProviderEntry struct {
	ID             string   `json:"id"`
	Type           string   `json:"type"` // "llm" or "api"
	Name           string   `json:"name"`
	Status         string   `json:"status"` // "connected" | "unconfigured" | "disconnected" (api only)
	Capabilities   []string `json:"capabilities,omitempty"`
	SupportedTypes []string `json:"supported_types,omitempty"`
}

// ProvidersHandler is the tools.ocr_providers process handler.
// Returns a unified list of both LLM (VLM-OCR) and traditional API OCR providers.
func ProvidersHandler(proc *process.Process) interface{} {
	authInfo := authorized.ProcessAuthInfo(proc)
	if authInfo == nil {
		return map[string]interface{}{"error": "unauthorized: no auth info in request"}
	}

	providers := listOCRProviders(authInfo)
	return map[string]interface{}{
		"providers": providers,
	}
}

// listOCRProviders collects both API and LLM OCR providers.
func listOCRProviders(authInfo *oauthTypes.AuthorizedInfo) []ocrProviderEntry {
	var result []ocrProviderEntry

	// 1. LLM connectors with OCR capability
	// VLM supports all types via prompt engineering
	vlmTypes := sortedKeys(vlmSupportedTypes)
	llmProviders, err := image.ListProvidersByCapability("ocr", authInfo)
	if err == nil {
		for _, p := range llmProviders {
			for _, m := range p.Models {
				result = append(result, ocrProviderEntry{
					ID:             m.ConnectorID,
					Type:           "llm",
					Name:           m.Name,
					Status:         "connected",
					Capabilities:   []string{"ocr", "vision"},
					SupportedTypes: vlmTypes,
				})
			}
		}
	}

	// 2. Traditional API providers from OCR settings
	apiHandlerTypes := map[string]map[string]bool{
		"paddleocr": paddleSupportedTypes,
		"baidu":     baiduSupportedTypes,
		"google":    googleSupportedTypes,
		"azure":     azureSupportedTypes,
	}
	apiPresets := []struct {
		key  string
		name string
	}{
		{"paddleocr", "PaddleOCR"},
		{"baidu", "Baidu OCR"},
		{"google", "Google Cloud Vision"},
		{"azure", "Azure Document Intelligence"},
	}

	for _, p := range apiPresets {
		entry := ocrProviderEntry{
			ID:             p.key,
			Type:           "api",
			Name:           p.name,
			Status:         "unconfigured",
			SupportedTypes: sortedKeys(apiHandlerTypes[p.key]),
		}

		if setting.Global != nil {
			merged, mergeErr := setting.Global.GetMerged(
				authInfo.GetUserID(), authInfo.GetTeamID(), "ocr.providers."+p.key,
			)
			if mergeErr == nil {
				if enabled, ok := merged["enabled"].(bool); ok && enabled {
					entry.Status = "connected"
					if status, ok := merged["status"].(string); ok && status != "" {
						entry.Status = status
					}
				}
			}
		}

		result = append(result, entry)
	}

	return result
}

// sortedKeys returns the keys of a bool map in sorted order.
func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
