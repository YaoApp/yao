package llm

import (
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/connector"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
)

// Provider represents an LLM provider option
type Provider struct {
	Label   string `json:"label"`
	Value   string `json:"value"`
	Type    string `json:"type"`    // "openai"
	Builtin bool   `json:"builtin"` // true for system built-in, false for user-defined
}

// Attach attaches the LLM management handlers to the router with OAuth protection
func Attach(group *gin.RouterGroup, oauth oauthTypes.OAuth) {

	// Create providers group with OAuth guard
	providers := group.Group("/providers")
	providers.Use(oauth.Guard)

	// LLM Providers endpoints
	providers.GET("/", listProviders) // GET /providers - List all LLM providers
}

// listProviders lists all available LLM providers (built-in + user-defined)
func listProviders(c *gin.Context) {
	allProviders := make([]Provider, 0)

	// Track which connectors we've already added (to avoid duplicates)
	added := make(map[string]bool)

	// 1. Get system built-in OpenAI-compatible LLM connectors
	for _, opt := range connector.AIConnectors {
		connType := getConnectorType(opt.Value)
		// Only include OpenAI-compatible LLM connectors
		if connType == "openai" {
			allProviders = append(allProviders, Provider{
				Label:   opt.Label,
				Value:   opt.Value,
				Type:    connType,
				Builtin: true,
			})
			added[opt.Value] = true
		}
	}

	// 2. Get user-defined OpenAI-compatible LLM connectors from the global connector registry
	// This includes all loaded connectors, both built-in and user-defined
	// Only include OpenAI-compatible connectors (standard openai format)
	for id, conn := range connector.Connectors {
		// Skip if already added
		if added[id] {
			continue
		}

		// Only include OpenAI-compatible LLM connectors
		connType := getConnectorType(id)
		if connType == "openai" {
			meta := conn.GetMetaInfo()
			label := meta.Label
			if label == "" {
				label = id
			}

			allProviders = append(allProviders, Provider{
				Label:   label,
				Value:   id,
				Type:    connType,
				Builtin: meta.Builtin,
			})
			added[id] = true
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
