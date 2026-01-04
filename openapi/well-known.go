package openapi

import (
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/share"
)

// attachWellKnown attaches the well-known handlers to the router
func (openapi *OpenAPI) attachWellKnown(router *gin.Engine) {

	// OAuth Discovery and Metadata Endpoints
	wellKnown := router.Group("/.well-known")

	// Yao Configuration Metadata - for client discovery
	wellKnown.GET("/yao", openapi.yaoMetadata)

	// OAuth Authorization Server Metadata - RFC 8414 (Required by MCP)
	wellKnown.GET("/oauth-authorization-server", openapi.oauthServerMetadata)

	// OpenID Connect Discovery - OpenID Connect Discovery 1.0
	wellKnown.GET("/openid_configuration", openapi.oauthOpenIDConfiguration)

	// OAuth Protected Resource Metadata - RFC 9728 (Required by MCP)
	wellKnown.GET("/oauth-protected-resource", openapi.oauthProtectedResourceMetadata)
}

// YaoMetadata represents the Yao server configuration metadata
type YaoMetadata struct {
	// Application information
	Name        string `json:"name,omitempty"`
	Version     string `json:"version,omitempty"`
	Description string `json:"description,omitempty"`

	// OpenAPI configuration
	OpenAPI   string `json:"openapi"`              // OpenAPI base URL (e.g., "/v1")
	IssuerURL string `json:"issuer_url,omitempty"` // OAuth issuer URL

	// Dashboard configuration
	Dashboard string                 `json:"dashboard,omitempty"` // Admin dashboard root path
	Optional  map[string]interface{} `json:"optional,omitempty"`  // Optional settings

	// Developer information
	Developer *share.Developer `json:"developer,omitempty"`
}

// yaoMetadata returns Yao server configuration metadata
func (openapi *OpenAPI) yaoMetadata(c *gin.Context) {
	// Get admin root path
	dashboard := share.App.AdminRoot
	if dashboard == "" {
		dashboard = "yao"
	}

	metadata := YaoMetadata{
		Name:        share.App.Name,
		Version:     share.App.Version,
		Description: share.App.Description,
		OpenAPI:     openapi.Config.BaseURL,
		IssuerURL:   openapi.Config.OAuth.IssuerURL,
		Dashboard:   "/" + dashboard,
		Optional:    share.App.Optional,
	}

	// Include developer info if available
	if share.App.Developer.ID != "" || share.App.Developer.Name != "" {
		metadata.Developer = &share.App.Developer
	}

	c.JSON(200, metadata)
}

// oauthServerMetadata returns authorization server metadata - RFC 8414
func (openapi *OpenAPI) oauthServerMetadata(c *gin.Context) {}

// oauthOpenIDConfiguration returns OpenID Connect configuration
func (openapi *OpenAPI) oauthOpenIDConfiguration(c *gin.Context) {}

// oauthProtectedResourceMetadata returns protected resource metadata - RFC 9728
func (openapi *OpenAPI) oauthProtectedResourceMetadata(c *gin.Context) {}
