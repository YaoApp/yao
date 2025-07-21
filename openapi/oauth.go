package openapi

import (
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

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

}

// OAuth Core Endpoints Implementation

// oauthAuthorize handles authorization requests - RFC 6749 Section 3.1
func (openapi *OpenAPI) oauthAuthorize(c *gin.Context) {
	// Parse and validate authorization request
	authReq, err := openapi.parseAuthorizationRequest(c)
	if err != nil {
		openapi.respondWithAuthorizationError(c, authReq.RedirectURI, err, authReq.State)
		return
	}

	// TODO: Implement full authorization logic
	// For now, return server error to indicate not implemented
	openapi.respondWithAuthorizationError(c, authReq.RedirectURI, ErrServerError, authReq.State)
}

// oauthToken handles token requests - RFC 6749 Section 3.2
func (openapi *OpenAPI) oauthToken(c *gin.Context) {
	grantType := c.PostForm("grant_type")

	// Validate grant type
	if grantType == "" {
		openapi.respondWithTokenError(c, ErrInvalidRequest)
		return
	}

	switch grantType {
	case types.GrantTypeAuthorizationCode:
		openapi.handleAuthorizationCodeGrant(c)
	case types.GrantTypeRefreshToken:
		openapi.handleRefreshTokenGrant(c)
	case types.GrantTypeClientCredentials:
		openapi.handleClientCredentialsGrant(c)
	case types.GrantTypeDeviceCode:
		openapi.handleDeviceCodeGrant(c)
	default:
		openapi.respondWithTokenError(c, ErrUnsupportedGrantType)
	}
}

// oauthRevoke handles token revocation - RFC 7009
func (openapi *OpenAPI) oauthRevoke(c *gin.Context) {
	token := c.PostForm("token")

	if token == "" {
		openapi.respondWithError(c, StatusBadRequest, ErrInvalidRequest)
		return
	}

	// TODO: Implement token revocation logic
	c.Status(StatusNoContent)
}

// oauthIntrospect handles token introspection - RFC 7662
func (openapi *OpenAPI) oauthIntrospect(c *gin.Context) {
	token := c.PostForm("token")

	if token == "" {
		openapi.respondWithError(c, StatusBadRequest, ErrInvalidRequest)
		return
	}

	// TODO: Implement token introspection logic
	// Return inactive token for now
	response := &TokenIntrospectionResponse{
		Active: false,
	}

	openapi.respondWithSuccess(c, StatusOK, response)
}

// oauthJWKS returns JSON Web Key Set - RFC 7517
func (openapi *OpenAPI) oauthJWKS(c *gin.Context) {
	// TODO: Implement JWKS generation
	jwks := &JWKSResponse{
		Keys: []JWK{},
	}

	openapi.respondWithSuccess(c, StatusOK, jwks)
}

// oauthUserInfo returns user information - OpenID Connect Core 1.0
func (openapi *OpenAPI) oauthUserInfo(c *gin.Context) {
	// Check for Bearer token in Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" || len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		openapi.respondWithError(c, StatusUnauthorized, ErrInvalidToken)
		return
	}

	// TODO: Implement user info retrieval
	openapi.respondWithError(c, StatusNotImplemented, ErrServerError)
}

// OAuth Extended Endpoints Implementation

// oauthRegister handles dynamic client registration - RFC 7591
func (openapi *OpenAPI) oauthRegister(c *gin.Context) {
	var req DynamicClientRegistrationRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		openapi.respondWithError(c, StatusBadRequest, ErrInvalidClientMetadata)
		return
	}

	// Basic validation
	if len(req.RedirectURIs) == 0 {
		openapi.respondWithError(c, StatusBadRequest, ErrMissingRedirectURI)
		return
	}

	res, err := openapi.OAuth.DynamicClientRegistration(c, &req)
	if err != nil {
		openapi.respondWithError(c, StatusBadRequest, ErrInvalidClientMetadata)
		return
	}

	// Return the registration response directly (RFC 7591 compliant)
	openapi.respondWithOAuthDirect(c, StatusCreated, res)
}

// oauthGetClient retrieves client configuration - RFC 7592
func (openapi *OpenAPI) oauthGetClient(c *gin.Context) {
	clientID := c.Param("client_id")

	if clientID == "" {
		openapi.respondWithError(c, StatusBadRequest, ErrInvalidRequest)
		return
	}

	// TODO: Implement client retrieval logic
	openapi.respondWithError(c, StatusNotFound, ErrInvalidClient)
}

// oauthUpdateClient updates client configuration - RFC 7592
func (openapi *OpenAPI) oauthUpdateClient(c *gin.Context) {
	clientID := c.Param("client_id")

	if clientID == "" {
		openapi.respondWithError(c, StatusBadRequest, ErrInvalidRequest)
		return
	}

	var req DynamicClientRegistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		openapi.respondWithError(c, StatusBadRequest, ErrInvalidClientMetadata)
		return
	}

	// TODO: Implement client update logic
	openapi.respondWithError(c, StatusNotImplemented, ErrServerError)
}

// oauthDeleteClient deletes client configuration - RFC 7592
func (openapi *OpenAPI) oauthDeleteClient(c *gin.Context) {
	clientID := c.Param("client_id")

	if clientID == "" {
		openapi.respondWithError(c, StatusBadRequest, ErrInvalidRequest)
		return
	}

	// TODO: Implement client deletion logic
	c.Status(StatusNoContent)
}

// oauthDeviceAuthorization handles device authorization - RFC 8628
func (openapi *OpenAPI) oauthDeviceAuthorization(c *gin.Context) {
	clientID := c.PostForm("client_id")

	if clientID == "" {
		openapi.respondWithTokenError(c, ErrInvalidRequest)
		return
	}

	// TODO: Implement device authorization logic
	response := &DeviceAuthorizationResponse{
		DeviceCode:      "generated-device-code",
		UserCode:        "USER-CODE",
		VerificationURI: "https://example.com/device",
		ExpiresIn:       900, // 15 minutes
		Interval:        5,   // 5 seconds
	}

	openapi.respondWithTokenSuccess(c, response)
}

// oauthPushedAuthorizationRequest handles PAR - RFC 9126
func (openapi *OpenAPI) oauthPushedAuthorizationRequest(c *gin.Context) {
	var req PushedAuthorizationRequest

	if err := c.ShouldBind(&req); err != nil {
		openapi.respondWithError(c, StatusBadRequest, ErrInvalidRequest)
		return
	}

	// Basic validation
	if req.ClientID == "" {
		openapi.respondWithError(c, StatusBadRequest, ErrInvalidRequest)
		return
	}

	// TODO: Implement PAR logic
	response := &PushedAuthorizationResponse{
		RequestURI: "urn:example:bwc4JK-ESC0w8acc191e-Y1LTC2",
		ExpiresIn:  60, // 60 seconds
	}

	openapi.respondWithSuccess(c, StatusCreated, response)
}

// oauthTokenExchange handles token exchange - RFC 8693
func (openapi *OpenAPI) oauthTokenExchange(c *gin.Context) {
	grantType := c.PostForm("grant_type")

	if grantType != types.GrantTypeTokenExchange {
		openapi.respondWithTokenError(c, ErrUnsupportedGrantType)
		return
	}

	subjectToken := c.PostForm("subject_token")
	if subjectToken == "" {
		openapi.respondWithTokenError(c, ErrInvalidRequest)
		return
	}

	// TODO: Implement token exchange logic
	response := &TokenExchangeResponse{
		AccessToken:     "exchanged-access-token",
		IssuedTokenType: "urn:ietf:params:oauth:token-type:access_token",
		TokenType:       types.TokenTypeBearer,
		ExpiresIn:       3600, // 1 hour
	}

	openapi.respondWithTokenSuccess(c, response)
}

// Helper functions for token grant handling

func (openapi *OpenAPI) handleAuthorizationCodeGrant(c *gin.Context) {
	code := c.PostForm("code")
	redirectURI := c.PostForm("redirect_uri")
	clientID := c.PostForm("client_id")

	// Basic validation
	if code == "" || redirectURI == "" || clientID == "" {
		openapi.respondWithTokenError(c, ErrInvalidRequest)
		return
	}

	// TODO: Validate authorization code and PKCE
	// TODO: Generate tokens

	token := &Token{
		AccessToken:  "generated-access-token",
		TokenType:    types.TokenTypeBearer,
		ExpiresIn:    3600, // 1 hour
		RefreshToken: "generated-refresh-token",
		Scope:        "openid profile email",
	}

	// Use OAuth 2.1 compliant response
	openapi.respondWithTokenSuccess(c, token)
}

func (openapi *OpenAPI) handleRefreshTokenGrant(c *gin.Context) {
	refreshToken := c.PostForm("refresh_token")

	if refreshToken == "" {
		openapi.respondWithTokenError(c, ErrInvalidRequest)
		return
	}

	// TODO: Validate refresh token
	// TODO: Generate new tokens

	response := &RefreshTokenResponse{
		AccessToken:  "new-access-token",
		TokenType:    types.TokenTypeBearer,
		ExpiresIn:    3600,                // 1 hour
		RefreshToken: "new-refresh-token", // OAuth 2.1 requires refresh token rotation
		Scope:        "openid profile email",
	}

	openapi.respondWithTokenSuccess(c, response)
}

func (openapi *OpenAPI) handleClientCredentialsGrant(c *gin.Context) {
	// Client authentication is handled by middleware
	scope := c.PostForm("scope")

	// TODO: Validate client credentials
	// TODO: Generate access token

	token := &Token{
		AccessToken: "client-credentials-token",
		TokenType:   types.TokenTypeBearer,
		ExpiresIn:   3600, // 1 hour
		Scope:       scope,
	}

	openapi.respondWithTokenSuccess(c, token)
}

func (openapi *OpenAPI) handleDeviceCodeGrant(c *gin.Context) {
	deviceCode := c.PostForm("device_code")

	if deviceCode == "" {
		openapi.respondWithTokenError(c, ErrInvalidRequest)
		return
	}

	// TODO: Check device code status
	// For now, return authorization pending
	openapi.respondWithTokenError(c, ErrAuthorizationPending)
}

// parseAuthorizationRequest parses and validates authorization request parameters
func (openapi *OpenAPI) parseAuthorizationRequest(c *gin.Context) (*AuthorizationRequest, *ErrorResponse) {
	// Parse authorization request parameters from both GET (query) and POST (form) methods
	authReq := &AuthorizationRequest{
		ClientID:            openapi.getParam(c, "client_id"),
		ResponseType:        openapi.getParam(c, "response_type"),
		RedirectURI:         openapi.getParam(c, "redirect_uri"),
		Scope:               openapi.getParam(c, "scope"),
		State:               openapi.getParam(c, "state"),
		CodeChallenge:       openapi.getParam(c, "code_challenge"),
		CodeChallengeMethod: openapi.getParam(c, "code_challenge_method"),
		Resource:            openapi.getParam(c, "resource"),
		Nonce:               openapi.getParam(c, "nonce"),
	}

	// Basic validation
	if authReq.ClientID == "" {
		return authReq, ErrInvalidRequest
	}

	// Validate response_type parameter - RFC 6749 Section 3.1.1
	if authReq.ResponseType == "" {
		return authReq, ErrInvalidRequest
	}

	// Check supported response types
	switch authReq.ResponseType {
	case types.ResponseTypeCode:
		// Authorization code flow - supported
	case types.ResponseTypeToken:
		// Implicit flow - deprecated in OAuth 2.1, return error
		return authReq, ErrUnsupportedResponseType
	default:
		return authReq, ErrUnsupportedResponseType
	}

	return authReq, nil
}

// getParam gets parameter from both query string (GET) and form data (POST)
// This supports OAuth 2.0 authorization endpoint which can accept both GET and POST requests
func (openapi *OpenAPI) getParam(c *gin.Context, key string) string {
	// First try to get from query parameters (GET request)
	if value := c.Query(key); value != "" {
		return value
	}
	// Then try to get from POST form data (POST request)
	return c.PostForm(key)
}
