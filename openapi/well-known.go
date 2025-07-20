package openapi

import "github.com/gin-gonic/gin"

// attachWellKnown attaches the well-known handlers to the router
func (openapi *OpenAPI) attachWellKnown(router *gin.Engine) {

	// OAuth Discovery and Metadata Endpoints
	wellKnown := router.Group("/.well-known")

	// OAuth Authorization Server Metadata - RFC 8414 (Required by MCP)
	wellKnown.GET("/oauth-authorization-server", openapi.oauthServerMetadata)

	// OpenID Connect Discovery - OpenID Connect Discovery 1.0
	wellKnown.GET("/openid_configuration", openapi.oauthOpenIDConfiguration)

	// OAuth Protected Resource Metadata - RFC 9728 (Required by MCP)
	wellKnown.GET("/oauth-protected-resource", openapi.oauthProtectedResourceMetadata)
}

// oauthServerMetadata returns authorization server metadata - RFC 8414
func (openapi *OpenAPI) oauthServerMetadata(c *gin.Context) {}

// oauthOpenIDConfiguration returns OpenID Connect configuration
func (openapi *OpenAPI) oauthOpenIDConfiguration(c *gin.Context) {}

// oauthProtectedResourceMetadata returns protected resource metadata - RFC 9728
func (openapi *OpenAPI) oauthProtectedResourceMetadata(c *gin.Context) {}
