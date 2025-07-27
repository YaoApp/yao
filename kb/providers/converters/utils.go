package converters

import (
	"fmt"

	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/yao/kb/providers/factory"
	kbtypes "github.com/yaoapp/yao/kb/types"
)

// parseNestedConverter parses nested converter configuration
func parseNestedConverter(config interface{}) (types.Converter, error) {
	configMap, ok := config.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("converter config must be a map")
	}

	converterID, ok := configMap["converter"].(string)
	if !ok {
		return nil, fmt.Errorf("converter ID is required")
	}

	// Get converter factory
	converterFactory, exists := factory.Converters[converterID]
	if !exists {
		return nil, fmt.Errorf("converter %s not found", converterID)
	}

	// Parse properties
	var providerOption *kbtypes.ProviderOption
	if properties, ok := configMap["properties"]; ok {
		if propertiesStr, ok := properties.(string); ok {
			// Handle preset value - we'd need to look up the preset
			// For now, create a basic option with the preset as ID
			providerOption = &kbtypes.ProviderOption{
				Value: propertiesStr,
			}
		} else if propertiesMap, ok := properties.(map[string]interface{}); ok {
			// Handle direct properties map
			providerOption = &kbtypes.ProviderOption{
				Properties: propertiesMap,
			}
		}
	}

	return converterFactory.Make(providerOption)
}
