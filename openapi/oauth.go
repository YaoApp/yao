package openapi

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
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
	authReq, parseErr := openapi.parseAuthorizationRequest(c)
	if parseErr != nil {
		response.RespondWithAuthorizationError(c, authReq.RedirectURI, parseErr, authReq.State)
		return
	}

	// Call OAuth service to process authorization request
	authResp, err := openapi.OAuth.Authorize(c, authReq)
	if err != nil {
		// OAuth service returned an error
		response.RespondWithAuthorizationError(c, authReq.RedirectURI, response.ErrServerError, authReq.State)
		return
	}

	// Check if authorization response contains an error
	if authResp.Error != "" {
		// Convert OAuth service error to ErrorResponse
		oauthError := &response.ErrorResponse{
			Code:             authResp.Error,
			ErrorDescription: authResp.ErrorDescription,
		}
		response.RespondWithAuthorizationError(c, authReq.RedirectURI, oauthError, authReq.State)
		return
	}

	// Success: redirect to client with authorization code
	redirectURL := authReq.RedirectURI
	if redirectURL != "" {
		separator := "?"
		if len(redirectURL) > 0 && redirectURL[len(redirectURL)-1:] == "?" {
			separator = "&"
		}

		redirectURL += separator + "code=" + authResp.Code
		if authResp.State != "" {
			redirectURL += "&state=" + authResp.State
		}

		c.Redirect(http.StatusFound, redirectURL)
		return
	}

	// Fallback: return JSON response if no redirect URI (should not happen with valid requests)
	response.RespondWithSuccess(c, response.StatusOK, authResp)
}

// oauthToken handles token requests - RFC 6749 Section 3.2
func (openapi *OpenAPI) oauthToken(c *gin.Context) {
	grantType := c.PostForm("grant_type")

	// Validate grant type
	if grantType == "" {
		response.RespondWithSecureError(c, response.StatusBadRequest, response.ErrInvalidRequest)
		return
	}

	switch grantType {
	case types.GrantTypeAuthorizationCode, types.GrantTypeClientCredentials, types.GrantTypeDeviceCode:
		// Handle standard grants through OAuth.Token()
		openapi.handleStandardTokenGrant(c, grantType)

	case types.GrantTypeRefreshToken:
		// Handle refresh token grant through OAuth.RefreshToken() - RFC 6749 Section 6
		openapi.handleRefreshTokenGrant(c)

	case types.GrantTypeTokenExchange:
		// Handle token exchange through OAuth.TokenExchange() - RFC 8693
		openapi.handleTokenExchangeGrant(c)

	default:
		response.RespondWithSecureError(c, response.StatusBadRequest, response.ErrUnsupportedGrantType)
	}
}

// handleStandardTokenGrant handles authorization_code, client_credentials, and device_code grants
func (openapi *OpenAPI) handleStandardTokenGrant(c *gin.Context, grantType string) {
	// Extract client credentials from Basic Auth header or form parameters
	clientID, clientSecret := openapi.extractClientCredentials(c)
	if clientID == "" {
		response.RespondWithSecureError(c, response.StatusBadRequest, response.ErrInvalidClient)
		return
	}

	// Validate client credentials using OAuth service
	oauthService, ok := openapi.OAuth.(*oauth.Service)
	if !ok {
		response.RespondWithSecureError(c, response.StatusBadRequest, response.ErrInvalidClient)
		return
	}

	clientInfo, err := oauthService.GetClientProvider().GetClientByCredentials(c, clientID, clientSecret)
	if err != nil {
		response.RespondWithSecureError(c, response.StatusBadRequest, response.ErrInvalidClient)
		return
	}

	// Extract PKCE parameter
	codeVerifier := c.PostForm("code_verifier")

	// Extract grant-specific "code" parameter
	var code string
	switch grantType {
	case types.GrantTypeAuthorizationCode:
		code = c.PostForm("code")
		redirectURI := c.PostForm("redirect_uri")

		// Basic validation for authorization code grant
		if code == "" || redirectURI == "" {
			response.RespondWithSecureError(c, response.StatusBadRequest, response.ErrInvalidRequest)
			return
		}

		// Validate that client supports authorization code grant
		if !openapi.clientSupportsGrantType(clientInfo, types.GrantTypeAuthorizationCode) {
			response.RespondWithSecureError(c, response.StatusUnauthorized, response.ErrUnauthorizedClient)
			return
		}

	case types.GrantTypeDeviceCode:
		code = c.PostForm("device_code")

		// Basic validation for device code grant
		if code == "" {
			response.RespondWithSecureError(c, response.StatusBadRequest, response.ErrInvalidRequest)
			return
		}

		// Validate that client supports device code grant
		if !openapi.clientSupportsGrantType(clientInfo, types.GrantTypeDeviceCode) {
			response.RespondWithSecureError(c, response.StatusUnauthorized, response.ErrUnauthorizedClient)
			return
		}

	case types.GrantTypeClientCredentials:
		// No code needed for client credentials
		code = ""

		// Validate that client supports client credentials grant
		if !openapi.clientSupportsGrantType(clientInfo, types.GrantTypeClientCredentials) {
			response.RespondWithSecureError(c, response.StatusUnauthorized, response.ErrUnauthorizedClient)
			return
		}
	}

	// Call OAuth service to handle the token request
	token, err := openapi.OAuth.Token(c, grantType, code, clientID, codeVerifier)
	if err != nil {
		// Convert OAuth service error to token error response with security headers
		if oauthErr, ok := err.(*response.ErrorResponse); ok {
			response.RespondWithSecureError(c, response.StatusBadRequest, oauthErr)
		} else {
			response.RespondWithSecureError(c, response.StatusBadRequest, response.ErrInvalidGrant)
		}
		return
	}

	// Return successful token response with OAuth security headers (RFC 6749 Section 5.1: MUST set Cache-Control: no-store)
	response.RespondWithSecureSuccess(c, response.StatusOK, token)
}

// handleRefreshTokenGrant handles refresh token requests - RFC 6749 Section 6
func (openapi *OpenAPI) handleRefreshTokenGrant(c *gin.Context) {
	// Extract client credentials from Basic Auth header or form parameters
	clientID, clientSecret := openapi.extractClientCredentials(c)
	if clientID == "" {
		response.RespondWithSecureError(c, response.StatusBadRequest, response.ErrInvalidClient)
		return
	}

	// Validate client credentials using OAuth service
	oauthService, ok := openapi.OAuth.(*oauth.Service)
	if !ok {
		response.RespondWithSecureError(c, response.StatusBadRequest, response.ErrInvalidClient)
		return
	}

	clientInfo, err := oauthService.GetClientProvider().GetClientByCredentials(c, clientID, clientSecret)
	if err != nil {
		response.RespondWithSecureError(c, response.StatusBadRequest, response.ErrInvalidClient)
		return
	}

	// Validate that client supports refresh token grant
	if !openapi.clientSupportsGrantType(clientInfo, types.GrantTypeRefreshToken) {
		response.RespondWithSecureError(c, response.StatusUnauthorized, response.ErrUnauthorizedClient)
		return
	}

	refreshToken := c.PostForm("refresh_token")
	scope := c.PostForm("scope")

	// Basic validation
	if refreshToken == "" {
		response.RespondWithSecureError(c, response.StatusBadRequest, response.ErrInvalidRequest)
		return
	}

	// Call OAuth service to handle refresh token grant
	var refreshResponse *types.RefreshTokenResponse
	if scope != "" {
		refreshResponse, err = openapi.OAuth.RefreshToken(c, refreshToken, scope)
	} else {
		refreshResponse, err = openapi.OAuth.RefreshToken(c, refreshToken)
	}
	if err != nil {
		// Convert OAuth service error to token error response with security headers
		if oauthErr, ok := err.(*response.ErrorResponse); ok {
			response.RespondWithSecureError(c, response.StatusBadRequest, oauthErr)
		} else {
			response.RespondWithSecureError(c, response.StatusBadRequest, response.ErrInvalidGrant)
		}
		return
	}

	// Return successful refresh token response with security headers
	response.RespondWithSecureSuccess(c, response.StatusOK, refreshResponse)
}

// handleTokenExchangeGrant handles token exchange requests - RFC 8693
func (openapi *OpenAPI) handleTokenExchangeGrant(c *gin.Context) {
	subjectToken := c.PostForm("subject_token")
	subjectTokenType := c.PostForm("subject_token_type")
	audience := c.PostForm("audience")
	scope := c.PostForm("scope")

	// Basic validation
	if subjectToken == "" {
		response.RespondWithSecureError(c, response.StatusBadRequest, response.ErrInvalidRequest)
		return
	}

	// Call OAuth service to handle token exchange
	exchangeResponse, err := openapi.OAuth.TokenExchange(c, subjectToken, subjectTokenType, audience, scope)
	if err != nil {
		// Convert OAuth service error to token error response with security headers
		if oauthErr, ok := err.(*response.ErrorResponse); ok {
			response.RespondWithSecureError(c, response.StatusBadRequest, oauthErr)
		} else {
			response.RespondWithSecureError(c, response.StatusBadRequest, response.ErrInvalidGrant)
		}
		return
	}

	// Return successful token exchange response with security headers
	response.RespondWithSecureSuccess(c, response.StatusOK, exchangeResponse)
}

// oauthRevoke handles token revocation - RFC 7009
func (openapi *OpenAPI) oauthRevoke(c *gin.Context) {
	token := c.PostForm("token")
	tokenTypeHint := c.PostForm("token_type_hint") // Optional hint about token type

	if token == "" {
		response.RespondWithSecureError(c, response.StatusBadRequest, response.ErrInvalidRequest)
		return
	}

	// Call OAuth service to revoke the token
	err := openapi.OAuth.Revoke(c, token, tokenTypeHint)
	if err != nil {
		// OAuth spec requires returning 200 even for invalid tokens to prevent information leakage
		// Only return error for server errors
		if oauthErr, ok := err.(*response.ErrorResponse); ok && oauthErr.Code == response.ErrServerError.Code {
			response.RespondWithError(c, response.StatusInternalServerError, response.ErrServerError)
			return
		}
	}

	// RFC 7009: Return 200 OK for successful revocation (or invalid tokens)
	c.Status(response.StatusOK)
}

// oauthIntrospect handles token introspection - RFC 7662
func (openapi *OpenAPI) oauthIntrospect(c *gin.Context) {
	token := c.PostForm("token")

	if token == "" {
		response.RespondWithSecureError(c, response.StatusBadRequest, response.ErrInvalidRequest)
		return
	}

	// Call OAuth service to introspect the token
	introspectionResult, err := openapi.OAuth.Introspect(c, token)
	if err != nil {
		// Return inactive token response on error (RFC 7662) with security headers
		tokenResponse := &response.TokenIntrospectionResponse{
			Active: false,
		}
		response.RespondWithSecureSuccess(c, response.StatusOK, tokenResponse)
		return
	}

	// Convert OAuth service response to API response format
	tokenResponse := &response.TokenIntrospectionResponse{
		Active:    introspectionResult.Active,
		Scope:     introspectionResult.Scope,
		ClientID:  introspectionResult.ClientID,
		Username:  introspectionResult.Username,
		TokenType: introspectionResult.TokenType,
		ExpiresAt: introspectionResult.ExpiresAt,
		IssuedAt:  introspectionResult.IssuedAt,
		Subject:   introspectionResult.Subject,
		Audience:  introspectionResult.Audience,
	}

	response.RespondWithSecureSuccess(c, response.StatusOK, tokenResponse)
}

// oauthJWKS returns JSON Web Key Set - RFC 7517
func (openapi *OpenAPI) oauthJWKS(c *gin.Context) {
	jwks, err := openapi.OAuth.JWKS(c)
	if err != nil {
		response.RespondWithError(c, response.StatusInternalServerError, response.ErrServerError)
		return
	}

	// RFC 7517 compliance: Return JWKS directly with security headers
	response.RespondWithSecureSuccess(c, response.StatusOK, jwks)
}

// oauthUserInfo returns user information - OpenID Connect Core 1.0
func (openapi *OpenAPI) oauthUserInfo(c *gin.Context) {
	// Check for Bearer token in Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" || len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		response.RespondWithError(c, response.StatusUnauthorized, response.ErrInvalidToken)
		return
	}

	// TODO: Implement user info retrieval
	response.RespondWithError(c, response.StatusNotImplemented, response.ErrServerError)
}

// OAuth Extended Endpoints Implementation

// oauthRegister handles dynamic client registration - RFC 7591
func (openapi *OpenAPI) oauthRegister(c *gin.Context) {
	var req response.DynamicClientRegistrationRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.RespondWithSecureError(c, response.StatusBadRequest, response.ErrInvalidClientMetadata)
		return
	}

	// Basic validation
	if len(req.RedirectURIs) == 0 {
		response.RespondWithSecureError(c, response.StatusBadRequest, response.ErrMissingRedirectURI)
		return
	}

	res, err := openapi.OAuth.DynamicClientRegistration(c, &req)
	if err != nil {
		response.RespondWithSecureError(c, response.StatusBadRequest, response.ErrInvalidClientMetadata)
		return
	}

	// Return the authorization response with security headers (RFC 7591 compliant, contains client credentials)
	response.RespondWithSecureSuccess(c, response.StatusCreated, res)
}

// extractClientCredentials extracts client ID and secret from Basic Auth header or form parameters
func (openapi *OpenAPI) extractClientCredentials(c *gin.Context) (clientID, clientSecret string) {
	// First, try to get from HTTP Basic Auth header (RFC 6749 Section 3.2.1)
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Basic ") {
		// Decode Basic Auth
		encoded := strings.TrimPrefix(authHeader, "Basic ")
		decoded, err := base64Decode(encoded)
		if err == nil {
			parts := strings.SplitN(string(decoded), ":", 2)
			if len(parts) == 2 {
				return parts[0], parts[1]
			}
		}
	}

	// Fallback to form parameters (RFC 6749 Section 3.2.1)
	clientID = c.PostForm("client_id")
	clientSecret = c.PostForm("client_secret")

	return clientID, clientSecret
}

// clientSupportsGrantType checks if a client supports a specific grant type
func (openapi *OpenAPI) clientSupportsGrantType(clientInfo *types.ClientInfo, grantType string) bool {
	if clientInfo == nil || len(clientInfo.GrantTypes) == 0 {
		return false
	}

	for _, supportedGrantType := range clientInfo.GrantTypes {
		if supportedGrantType == grantType {
			return true
		}
	}

	return false
}

// base64Decode decodes a base64 string
func base64Decode(data string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(data)
}

// oauthGetClient retrieves client configuration - RFC 7592
func (openapi *OpenAPI) oauthGetClient(c *gin.Context) {
	clientID := c.Param("client_id")

	if clientID == "" {
		response.RespondWithError(c, response.StatusBadRequest, response.ErrInvalidRequest)
		return
	}

	// TODO: Implement client retrieval logic
	response.RespondWithError(c, response.StatusNotFound, response.ErrInvalidClient)
}

// oauthUpdateClient updates client configuration - RFC 7592
func (openapi *OpenAPI) oauthUpdateClient(c *gin.Context) {
	clientID := c.Param("client_id")

	if clientID == "" {
		response.RespondWithError(c, response.StatusBadRequest, response.ErrInvalidRequest)
		return
	}

	var req response.DynamicClientRegistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.RespondWithError(c, response.StatusBadRequest, response.ErrInvalidClientMetadata)
		return
	}

	// TODO: Implement client update logic
	response.RespondWithError(c, response.StatusNotImplemented, response.ErrServerError)
}

// oauthDeleteClient deletes client configuration - RFC 7592
func (openapi *OpenAPI) oauthDeleteClient(c *gin.Context) {
	clientID := c.Param("client_id")

	if clientID == "" {
		response.RespondWithError(c, response.StatusBadRequest, response.ErrInvalidRequest)
		return
	}

	// TODO: Implement client deletion logic
	c.Status(response.StatusNoContent)
}

// oauthDeviceAuthorization handles device authorization - RFC 8628
func (openapi *OpenAPI) oauthDeviceAuthorization(c *gin.Context) {
	clientID := c.PostForm("client_id")

	if clientID == "" {
		response.RespondWithError(c, response.StatusBadRequest, response.ErrInvalidRequest)
		return
	}

	// TODO: Implement device authorization logic
	deviceResponse := &response.DeviceAuthorizationResponse{
		DeviceCode:      "generated-device-code",
		UserCode:        "USER-CODE",
		VerificationURI: "https://example.com/device",
		ExpiresIn:       900, // 15 minutes
		Interval:        5,   // 5 seconds
	}

	response.RespondWithSuccess(c, response.StatusOK, deviceResponse)
}

// oauthPushedAuthorizationRequest handles PAR - RFC 9126
func (openapi *OpenAPI) oauthPushedAuthorizationRequest(c *gin.Context) {
	var req response.PushedAuthorizationRequest

	if err := c.ShouldBind(&req); err != nil {
		response.RespondWithError(c, response.StatusBadRequest, response.ErrInvalidRequest)
		return
	}

	// Basic validation
	if req.ClientID == "" {
		response.RespondWithError(c, response.StatusBadRequest, response.ErrInvalidRequest)
		return
	}

	// TODO: Implement PAR logic
	parResponse := &response.PushedAuthorizationResponse{
		RequestURI: "urn:example:bwc4JK-ESC0w8acc191e-Y1LTC2",
		ExpiresIn:  60, // 60 seconds
	}

	response.RespondWithSuccess(c, response.StatusCreated, parResponse)
}

// oauthTokenExchange handles token exchange - RFC 8693
func (openapi *OpenAPI) oauthTokenExchange(c *gin.Context) {
	grantType := c.PostForm("grant_type")

	if grantType != types.GrantTypeTokenExchange {
		response.RespondWithError(c, response.StatusBadRequest, response.ErrUnsupportedGrantType)
		return
	}

	subjectToken := c.PostForm("subject_token")
	if subjectToken == "" {
		response.RespondWithError(c, response.StatusBadRequest, response.ErrInvalidRequest)
		return
	}

	// TODO: Implement token exchange logic
	exchangeResponse := &response.TokenExchangeResponse{
		AccessToken:     "exchanged-access-token",
		IssuedTokenType: "urn:ietf:params:oauth:token-type:access_token",
		TokenType:       types.TokenTypeBearer,
		ExpiresIn:       3600, // 1 hour
	}

	response.RespondWithSuccess(c, response.StatusOK, exchangeResponse)
}

// parseAuthorizationRequest parses and validates authorization request parameters
func (openapi *OpenAPI) parseAuthorizationRequest(c *gin.Context) (*types.AuthorizationRequest, *response.ErrorResponse) {
	// Parse authorization request parameters from both GET (query) and POST (form) methods
	authReq := &types.AuthorizationRequest{
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
		return authReq, response.ErrInvalidRequest
	}

	// Validate response_type parameter - RFC 6749 Section 3.1.1
	if authReq.ResponseType == "" {
		return authReq, response.ErrInvalidRequest
	}

	// Check supported response types
	switch authReq.ResponseType {
	case types.ResponseTypeCode:
		// Authorization code flow - supported
	case types.ResponseTypeToken:
		// Implicit flow - deprecated in OAuth 2.1, return error
		return authReq, response.ErrUnsupportedResponseType
	default:
		return authReq, response.ErrUnsupportedResponseType
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
