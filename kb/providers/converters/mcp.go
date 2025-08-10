package converters

import (
	"strings"

	"github.com/yaoapp/gou/graphrag/converter"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/yao/kb/providers/factory"
	kbtypes "github.com/yaoapp/yao/kb/types"
)

// MCP is a converter provider for mcp files
type MCP struct {
	Autodetect    []string `json:"autodetect" yaml:"autodetect"`         // Optional, default is empty, if not set, will not use autodetect
	MatchPriority int      `json:"match_priority" yaml:"match_priority"` // Optional, default is 0, the higher the number, the higher the priority
}

// Make creates a new MCP converter
func (mcp *MCP) Make(option *kbtypes.ProviderOption) (types.Converter, error) {
	// Start with default values
	mcpOptions := &converter.MCPOptions{
		ID:                  "",  // Will be set from option
		Tool:                "",  // Will be set from option
		ArgumentsMapping:    nil, // Optional
		ResultMapping:       nil, // Optional
		NotificationMapping: nil, // Optional
	}

	// Extract values from Properties map
	if option != nil && option.Properties != nil {
		if id, ok := option.Properties["id"]; ok {
			if idStr, ok := id.(string); ok {
				mcpOptions.ID = idStr
			}
		}

		if tool, ok := option.Properties["tool"]; ok {
			if toolStr, ok := tool.(string); ok {
				mcpOptions.Tool = toolStr
			}
		}

		if argsMapping, ok := option.Properties["arguments_mapping"]; ok {
			if argsMappingMap, ok := argsMapping.(map[string]interface{}); ok {
				// Convert map[string]interface{} to map[string]string
				stringMap := make(map[string]string)
				for k, v := range argsMappingMap {
					if vStr, ok := v.(string); ok {
						stringMap[k] = vStr
					}
				}
				if len(stringMap) > 0 {
					mcpOptions.ArgumentsMapping = stringMap
				}
			}
		}

		if resultMapping, ok := option.Properties["result_mapping"]; ok {
			if resultMappingMap, ok := resultMapping.(map[string]interface{}); ok {
				// Convert map[string]interface{} to map[string]string
				stringMap := make(map[string]string)
				for k, v := range resultMappingMap {
					if vStr, ok := v.(string); ok {
						stringMap[k] = vStr
					}
				}
				if len(stringMap) > 0 {
					mcpOptions.ResultMapping = stringMap
				}
			}
		}

		// Support both "result_mapping" and "output_mapping" for backward compatibility
		if outputMapping, ok := option.Properties["output_mapping"]; ok {
			if outputMappingMap, ok := outputMapping.(map[string]interface{}); ok {
				// Convert map[string]interface{} to map[string]string
				stringMap := make(map[string]string)
				for k, v := range outputMappingMap {
					if vStr, ok := v.(string); ok {
						stringMap[k] = vStr
					}
				}
				if len(stringMap) > 0 {
					mcpOptions.ResultMapping = stringMap
				}
			}
		}

		if notificationMapping, ok := option.Properties["notification_mapping"]; ok {
			if notificationMappingMap, ok := notificationMapping.(map[string]interface{}); ok {
				// Convert map[string]interface{} to map[string]string
				stringMap := make(map[string]string)
				for k, v := range notificationMappingMap {
					if vStr, ok := v.(string); ok {
						stringMap[k] = vStr
					}
				}
				if len(stringMap) > 0 {
					mcpOptions.NotificationMapping = stringMap
				}
			}
		}
	}

	return converter.NewMCP(mcpOptions)
}

// AutoDetect detects the converter based on the filename and content types
func (mcp *MCP) AutoDetect(filename, contentTypes string) (bool, int, error) {
	// If autodetect is empty, return false
	if mcp.Autodetect == nil {
		return false, 0, nil
	}

	// Check if the filename matches the autodetect
	for _, autodetect := range mcp.Autodetect {
		if strings.HasSuffix(filename, autodetect) {
			return true, mcp.MatchPriority, nil
		}

		// Check if the content types matches the autodetect
		if strings.Contains(contentTypes, autodetect) {
			return true, mcp.MatchPriority, nil
		}
	}

	return false, 0, nil
}

// Schema returns the schema for the MCP converter
func (mcp *MCP) Schema(provider *kbtypes.Provider, locale string) (*kbtypes.ProviderSchema, error) {
	return factory.GetSchemaFromBindata(factory.ProviderTypeConverter, "mcp", locale)
}
