package providers

import (
	"time"

	"github.com/yaoapp/gou/graphrag/fetcher"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/yao/kb/providers/factory"
	kbtypes "github.com/yaoapp/yao/kb/types"
)

// FetcherHTTP is an HTTP fetcher provider
type FetcherHTTP struct{}

// FetcherMCP is an MCP fetcher provider
type FetcherMCP struct{}

// AutoRegister registers the fetcher providers
func init() {
	factory.Fetchers["__yao.http"] = &FetcherHTTP{}
	factory.Fetchers["__yao.mcp"] = &FetcherMCP{}
}

// === FetcherHTTP ===

// Make creates a new HTTP fetcher
func (f *FetcherHTTP) Make(option *kbtypes.ProviderOption) (types.Fetcher, error) {
	// Start with default values
	httpOptions := &fetcher.HTTPOptions{
		Headers:   make(map[string]string), // Custom headers
		UserAgent: "GraphRAG-Fetcher/1.0",  // Default user agent
		Timeout:   300 * time.Second,       // Default 5 minutes
	}

	// Extract values from Properties map
	if option != nil && option.Properties != nil {
		// Set headers
		if headers, ok := option.Properties["headers"]; ok {
			if headersMap, ok := headers.(map[string]interface{}); ok {
				for key, value := range headersMap {
					if valueStr, ok := value.(string); ok {
						httpOptions.Headers[key] = valueStr
					}
				}
			}
		}

		// Set user agent
		if userAgent, ok := option.Properties["user_agent"]; ok {
			if userAgentStr, ok := userAgent.(string); ok {
				httpOptions.UserAgent = userAgentStr
			}
		}

		// Set timeout (in seconds)
		if timeout, ok := option.Properties["timeout"]; ok {
			if timeoutInt, ok := timeout.(int); ok {
				httpOptions.Timeout = time.Duration(timeoutInt) * time.Second
			} else if timeoutFloat, ok := timeout.(float64); ok {
				httpOptions.Timeout = time.Duration(timeoutFloat) * time.Second
			}
		}
	}

	return fetcher.NewHTTPFetcher(httpOptions), nil
}

// Schema returns the schema for the HTTP fetcher provider
func (f *FetcherHTTP) Schema(provider *kbtypes.Provider, locale string) (*kbtypes.ProviderSchema, error) {
	return factory.GetSchemaFromBindata(factory.ProviderTypeFetcher, "http", locale)
}

// === FetcherMCP ===

// Make creates a new MCP fetcher
func (f *FetcherMCP) Make(option *kbtypes.ProviderOption) (types.Fetcher, error) {
	// Start with default values
	mcpOptions := &fetcher.MCPOptions{
		ID:                  "",      // Required - will be set from option
		Tool:                "fetch", // Default tool name
		ArgumentsMapping:    nil,     // Optional arguments mapping
		ResultMapping:       nil,     // Optional result mapping
		NotificationMapping: nil,     // Optional notification mapping
	}

	// Extract values from Properties map
	if option != nil && option.Properties != nil {
		// Set MCP ID (required)
		if id, ok := option.Properties["id"]; ok {
			if idStr, ok := id.(string); ok {
				mcpOptions.ID = idStr
			}
		}

		// Set tool name
		if tool, ok := option.Properties["tool"]; ok {
			if toolStr, ok := tool.(string); ok {
				mcpOptions.Tool = toolStr
			}
		}

		// Set arguments mapping
		if argumentsMapping, ok := option.Properties["arguments_mapping"]; ok {
			if argumentsMappingMap, ok := argumentsMapping.(map[string]interface{}); ok {
				argMap := make(map[string]string)
				for key, value := range argumentsMappingMap {
					if valueStr, ok := value.(string); ok {
						argMap[key] = valueStr
					}
				}
				if len(argMap) > 0 {
					mcpOptions.ArgumentsMapping = argMap
				}
			}
		}

		// Set result mapping (handle both "result_mapping" and "output_mapping" for compatibility)
		var resultMapping map[string]interface{}
		if rm, ok := option.Properties["result_mapping"]; ok {
			resultMapping, _ = rm.(map[string]interface{})
		} else if om, ok := option.Properties["output_mapping"]; ok {
			// Support kb.yao's "output_mapping" as alias for "result_mapping"
			resultMapping, _ = om.(map[string]interface{})
		}

		if resultMapping != nil {
			resMap := make(map[string]string)
			for key, value := range resultMapping {
				if valueStr, ok := value.(string); ok {
					resMap[key] = valueStr
				}
			}
			if len(resMap) > 0 {
				mcpOptions.ResultMapping = resMap
			}
		}

		// Set notification mapping
		if notificationMapping, ok := option.Properties["notification_mapping"]; ok {
			if notificationMappingMap, ok := notificationMapping.(map[string]interface{}); ok {
				notMap := make(map[string]string)
				for key, value := range notificationMappingMap {
					if valueStr, ok := value.(string); ok {
						notMap[key] = valueStr
					}
				}
				if len(notMap) > 0 {
					mcpOptions.NotificationMapping = notMap
				}
			}
		}
	}

	return fetcher.NewMCP(mcpOptions)
}

// Schema returns the schema for the MCP fetcher provider
func (f *FetcherMCP) Schema(provider *kbtypes.Provider, locale string) (*kbtypes.ProviderSchema, error) {
	return factory.GetSchemaFromBindata(factory.ProviderTypeFetcher, "mcp", locale)
}
