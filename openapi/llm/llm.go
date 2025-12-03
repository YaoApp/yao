package llm

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/connector/openai"
	"github.com/yaoapp/yao/agent"
	agentllm "github.com/yaoapp/yao/agent/llm"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
)

// Provider represents an LLM provider option
type Provider struct {
	Label        string                 `json:"label"`
	Value        string                 `json:"value"`
	Type         string                 `json:"type"`         // "openai"
	Builtin      bool                   `json:"builtin"`      // true for system built-in, false for user-defined
	Capabilities map[string]interface{} `json:"capabilities"` // Model capabilities from connector settings
}

// getModelCapabilities returns user-defined model capabilities from agent DSL
// Returns nil if agent not initialized or no models configured
func getModelCapabilities() map[string]openai.Capabilities {
	agentDSL := agent.GetAgent()
	if agentDSL != nil && agentDSL.Models != nil {
		return agentDSL.Models
	}
	return nil
}

// Attach attaches the LLM management handlers to the router with OAuth protection
func Attach(group *gin.RouterGroup, oauth oauthTypes.OAuth) {

	// Create providers group with OAuth guard
	group.Use(oauth.Guard)

	// LLM Providers endpoints
	group.GET("/providers", listProviders) // GET /providers - List all LLM providers
}

// listProviders lists all available LLM providers (built-in + user-defined)
// Supports filtering by capabilities using query parameter: ?filters=vision,tool_calls,audio
func listProviders(c *gin.Context) {
	allProviders := make([]Provider, 0)

	// Parse filter parameters from query string
	filtersParam := c.Query("filters")
	var filters []string
	if filtersParam != "" {
		filters = strings.Split(filtersParam, ",")
		for i, filter := range filters {
			filters[i] = strings.TrimSpace(strings.ToLower(filter))
		}
	}

	// Get user-defined model capabilities once at the start of request
	modelCapabilities := getModelCapabilities()

	// Get all OpenAI-compatible LLM connectors from AIConnectors
	// Note: All openai type connectors are automatically added to AIConnectors during loading
	// See gou/connector/connector.go LoadSource() for details
	for _, opt := range connector.AIConnectors {
		connType := getConnectorType(opt.Value)
		// Only include OpenAI-compatible LLM connectors
		if connType == "openai" {
			conn, ok := connector.Connectors[opt.Value]
			if !ok {
				continue
			}

			// Get capabilities from connector settings
			capabilities := getCapabilitiesWithModels(conn, modelCapabilities)

			// Apply capability filters
			if len(filters) > 0 && !matchesFilters(capabilities, filters) {
				continue
			}

			allProviders = append(allProviders, Provider{
				Label:        opt.Label,
				Value:        opt.Value,
				Type:         connType,
				Builtin:      conn.GetMetaInfo().Builtin,
				Capabilities: capabilities,
			})
		}
	}

	response.RespondWithSuccess(c, response.StatusOK, allProviders)
}

// getConnectorType retrieves the connector type by checking the global connector map
func getConnectorType(id string) string {
	conn, ok := connector.Connectors[id]
	if !ok {
		return "unknown"
	}

	// Only return openai type (OpenAI-compatible format)
	if conn.Is(connector.OPENAI) {
		return "openai"
	}

	return "unknown"
}

// getCapabilitiesWithModels extracts capabilities from connector settings
// Uses the unified capability getter from agent/llm package
// Takes modelCapabilities as parameter to avoid repeated calls to getModelCapabilities()
func getCapabilitiesWithModels(conn connector.Connector, modelCapabilities map[string]openai.Capabilities) map[string]interface{} {
	if conn == nil {
		return nil
	}

	// Use unified capability getter with user-defined model capabilities
	caps := agentllm.GetCapabilitiesFromConn(conn, modelCapabilities)
	return agentllm.ToMap(caps)
}

// matchesFilters checks if capabilities match all requested filters
// Filters are matched case-insensitively and support the following capability keys:
// - vision: true or string value like "openai", "claude"
// - audio: bool
// - tool_calls: bool
// - reasoning: bool
// - streaming: bool
// - json: bool
// - multimodal: bool
// - temperature_adjustable: bool
func matchesFilters(capabilities map[string]interface{}, filters []string) bool {
	if capabilities == nil {
		return false
	}

	// All filters must match (AND logic)
	for _, filter := range filters {
		matched := false

		// Check each capability field
		for key, value := range capabilities {
			keyLower := strings.ToLower(key)

			// Match the filter against capability key
			if keyLower == filter {
				// For vision, check if it's true or a non-empty string
				if filter == "vision" {
					if boolVal, ok := value.(bool); ok && boolVal {
						matched = true
						break
					}
					if strVal, ok := value.(string); ok && strVal != "" {
						matched = true
						break
					}
				} else {
					// For other capabilities, check if it's true
					if boolVal, ok := value.(bool); ok && boolVal {
						matched = true
						break
					}
				}
			}
		}

		// If any filter doesn't match, return false
		if !matched {
			return false
		}
	}

	return true
}
