package openapi

import "github.com/gin-gonic/gin"

// OAuth handlers
// NOTE: If using versioned paths like /v1/oauth, ensure that:
// 1. Discovery endpoints (.well-known) are at the root level, not versioned
// 2. Server metadata correctly returns versioned OAuth endpoint URLs
// 3. MCP clients are configured with the correct base URL for discovery
//
// Example setup:
//   - OAuth endpoints: /v1/oauth/authorize, /v1/oauth/token, etc.
//   - Discovery endpoints: /.well-known/oauth-authorization-server (root level)
//   - MCP client URL: https://server.com/v1/mcp (for MCP protocol)
//   - Authorization discovery: https://server.com/.well-known/oauth-authorization-server
func (openapi *OpenAPI) attachOAuth(base *gin.RouterGroup) {

	// OAuth Core Endpoints (RFC 6749, OAuth 2.1)
	oauth := base.Group("/oauth")

	// Authorization endpoint - RFC 6749 Section 3.1
	oauth.GET("/authorize", openapi.oauthAuthorize)
	oauth.POST("/authorize", openapi.oauthAuthorize) // Support both GET and POST

	// Token endpoint - RFC 6749 Section 3.2
	oauth.POST("/token", openapi.oauthToken)

	// Token revocation endpoint - RFC 7009
	oauth.POST("/revoke", openapi.oauthRevoke)

	// Token introspection endpoint - RFC 7662
	oauth.POST("/introspect", openapi.oauthIntrospect)

	// JSON Web Key Set endpoint - RFC 7517
	oauth.GET("/jwks", openapi.oauthJWKS)

	// UserInfo endpoint - OpenID Connect Core 1.0
	oauth.GET("/userinfo", openapi.oauthUserInfo)
	oauth.POST("/userinfo", openapi.oauthUserInfo) // Support both GET and POST

	// OAuth Extended Endpoints
	// Dynamic Client Registration - RFC 7591 (Required by MCP)
	oauth.POST("/register", openapi.oauthRegister)

	// Client Configuration - RFC 7592
	oauth.GET("/register/:client_id", openapi.oauthGetClient)
	oauth.PUT("/register/:client_id", openapi.oauthUpdateClient)
	oauth.DELETE("/register/:client_id", openapi.oauthDeleteClient)

	// Device Authorization Flow - RFC 8628
	oauth.POST("/device_authorization", openapi.oauthDeviceAuthorization)

	// Pushed Authorization Request - RFC 9126
	oauth.POST("/par", openapi.oauthPushedAuthorizationRequest)

	// Token Exchange - RFC 8693
	oauth.POST("/token_exchange", openapi.oauthTokenExchange)

	// OAuth Discovery and Metadata Endpoints
	// IMPORTANT: These should be at the root level for proper MCP discovery
	// If base is /v1, consider mounting these at the application root instead
	wellKnown := base.Group("/.well-known")

	// OAuth Authorization Server Metadata - RFC 8414 (Required by MCP)
	wellKnown.GET("/oauth-authorization-server", openapi.oauthServerMetadata)

	// OpenID Connect Discovery - OpenID Connect Discovery 1.0
	wellKnown.GET("/openid_configuration", openapi.oauthOpenIDConfiguration)

	// OAuth Protected Resource Metadata - RFC 9728 (Required by MCP)
	wellKnown.GET("/oauth-protected-resource", openapi.oauthProtectedResourceMetadata)

}

// OAuth Core Endpoints Implementation

// oauthAuthorize handles authorization requests - RFC 6749 Section 3.1
func (openapi *OpenAPI) oauthAuthorize(c *gin.Context) {}

// oauthToken handles token requests - RFC 6749 Section 3.2
func (openapi *OpenAPI) oauthToken(c *gin.Context) {}

// oauthRevoke handles token revocation - RFC 7009
func (openapi *OpenAPI) oauthRevoke(c *gin.Context) {}

// oauthIntrospect handles token introspection - RFC 7662
func (openapi *OpenAPI) oauthIntrospect(c *gin.Context) {}

// oauthJWKS returns JSON Web Key Set - RFC 7517
func (openapi *OpenAPI) oauthJWKS(c *gin.Context) {}

// oauthUserInfo returns user information - OpenID Connect Core 1.0
func (openapi *OpenAPI) oauthUserInfo(c *gin.Context) {}

// OAuth Extended Endpoints Implementation

// oauthRegister handles dynamic client registration - RFC 7591
func (openapi *OpenAPI) oauthRegister(c *gin.Context) {}

// oauthGetClient retrieves client configuration - RFC 7592
func (openapi *OpenAPI) oauthGetClient(c *gin.Context) {}

// oauthUpdateClient updates client configuration - RFC 7592
func (openapi *OpenAPI) oauthUpdateClient(c *gin.Context) {}

// oauthDeleteClient deletes client configuration - RFC 7592
func (openapi *OpenAPI) oauthDeleteClient(c *gin.Context) {}

// oauthDeviceAuthorization handles device authorization - RFC 8628
func (openapi *OpenAPI) oauthDeviceAuthorization(c *gin.Context) {}

// oauthPushedAuthorizationRequest handles PAR - RFC 9126
func (openapi *OpenAPI) oauthPushedAuthorizationRequest(c *gin.Context) {}

// oauthTokenExchange handles token exchange - RFC 8693
func (openapi *OpenAPI) oauthTokenExchange(c *gin.Context) {}

// OAuth Discovery and Metadata Endpoints Implementation

// oauthServerMetadata returns authorization server metadata - RFC 8414
func (openapi *OpenAPI) oauthServerMetadata(c *gin.Context) {}

// oauthOpenIDConfiguration returns OpenID Connect configuration
func (openapi *OpenAPI) oauthOpenIDConfiguration(c *gin.Context) {}

// oauthProtectedResourceMetadata returns protected resource metadata - RFC 9728
func (openapi *OpenAPI) oauthProtectedResourceMetadata(c *gin.Context) {}
