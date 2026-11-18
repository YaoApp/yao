package openapi

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	openapiauth "github.com/yaoapp/yao/openapi/auth"
	oauthpkg "github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/openapi/user"
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
	oauth.POST("/device/authorize", openapi.oauthDeviceAuthorize)

	// Pushed Authorization Request - RFC 9126
	oauth.POST("/par", openapi.oauthPushedAuthorizationRequest)

	// OIDC RP-Initiated Logout
	oauth.GET("/logout", openapi.oauthEndSession)
	oauth.POST("/logout", openapi.oauthEndSession)

	// Token Exchange - RFC 8693
	oauth.POST("/token_exchange", openapi.oauthTokenExchange)

	// Node-token lifecycle endpoints are part of OAuth authentication.
	openapiauth.Attach(base, openapi.OAuth)

}

// OAuth Core Endpoints Implementation

// canUseDirectTaoLogin reports whether an unauthenticated authorization
// request can skip the deployment login page and go straight to Tao. This is
// limited to native (desktop/mobile) clients and only when Tao is configured.
func (openapi *OpenAPI) canUseDirectTaoLogin(c *gin.Context, clientID string) bool {
	if clientID == "" || !user.TaoConfigured() {
		return false
	}
	service, ok := openapi.OAuth.(*oauthpkg.Service)
	if !ok {
		return false
	}
	client, err := service.GetClientProvider().GetClientByID(c.Request.Context(), clientID)
	if err != nil || client == nil {
		return false
	}
	return client.ApplicationType == types.ApplicationTypeNative
}

// parsePromptValues parses OIDC prompt values. Unknown values are rejected;
// prompt=none cannot be combined with other values.
func parsePromptValues(raw string) ([]string, error) {
	values := strings.Fields(raw)
	if len(values) == 0 {
		return nil, nil
	}
	allowed := map[string]bool{"none": true, "login": true, "consent": true, "select_account": true}
	hasNone := false
	for _, value := range values {
		if !allowed[value] {
			return nil, fmt.Errorf("unsupported prompt value: %s", value)
		}
		if value == "none" {
			hasNone = true
		}
	}
	if hasNone && len(values) > 1 {
		return nil, fmt.Errorf("prompt=none cannot be combined with other values")
	}
	return values, nil
}

func promptHas(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

// authorizeReturnToWithPrompt rebuilds the current authorize return URL with a
// replacement prompt value, without mutating the incoming request.
func authorizeReturnToWithPrompt(c *gin.Context, prompt string) string {
	if c == nil || c.Request == nil {
		return ""
	}
	original := c.Request.URL.RawQuery
	q := c.Request.URL.Query()
	if prompt == "" {
		q.Del("prompt")
	} else {
		q.Set("prompt", prompt)
	}
	c.Request.URL.RawQuery = q.Encode()
	result := oauthpkg.BuildAuthorizeReturnTo(c)
	c.Request.URL.RawQuery = original
	return result
}

func (openapi *OpenAPI) redirectAuthorizeLogin(c *gin.Context, authReq *types.AuthorizationRequest, prompt string, forceTao bool) {
	returnTo := authorizeReturnToWithPrompt(c, prompt)
	direct := (forceTao && user.TaoConfigured()) || openapi.canUseDirectTaoLogin(c, authReq.ClientID)
	if direct {
		q := url.Values{}
		q.Set("redirect", "1")
		q.Set("return_to", returnTo)
		if forceTao && strings.TrimSpace(authReq.Prompt) != "" {
			q.Set("prompt", authReq.Prompt)
		}
		if strings.TrimSpace(authReq.LoginHint) != "" {
			q.Set("login_hint", authReq.LoginHint)
		}
		if strings.TrimSpace(authReq.MaxAge) != "" {
			q.Set("max_age", authReq.MaxAge)
		}
		c.Redirect(http.StatusFound, "/v1/user/oauth/tao/authorize?"+q.Encode())
		return
	}
	c.Redirect(http.StatusFound, oauthpkg.BuildLoginRedirectURL(oauthpkg.DefaultLoginEntryPath, returnTo))
}

func (openapi *OpenAPI) buildConsentURL(c *gin.Context, token string) string {
	raw := user.GetConsentURL(strings.TrimSpace(c.Query("locale")))
	parsed, err := url.Parse(raw)
	if err != nil || raw == "" {
		parsed = &url.URL{Path: "/app/consent"}
	}
	q := c.Request.URL.Query()
	q.Set("consent_token", token)
	parsed.RawQuery = q.Encode()
	return parsed.String()
}

func validateAuthorizeTeam(ctx context.Context, service *oauthpkg.Service, authReq *types.AuthorizationRequest, userID string) error {
	if authReq == nil || authReq.TeamID == "" {
		return nil
	}
	provider, err := service.GetUserProvider()
	if err != nil || provider == nil {
		return fmt.Errorf("user provider is not available")
	}
	team, err := provider.GetTeamByMember(ctx, authReq.TeamID, userID)
	if err != nil {
		return fmt.Errorf("user is not a member of team %s", authReq.TeamID)
	}
	if tenantID, ok := team["tenant_id"].(string); ok {
		authReq.TenantID = tenantID
	}
	return nil
}

func consentRequestMatches(submitted, stored *types.AuthorizationRequest) bool {
	if submitted == nil || stored == nil {
		return false
	}
	checks := []struct{ a, b string }{
		{submitted.ClientID, stored.ClientID},
		{submitted.RedirectURI, stored.RedirectURI},
		{submitted.State, stored.State},
		{submitted.CodeChallenge, stored.CodeChallenge},
		{submitted.CodeChallengeMethod, stored.CodeChallengeMethod},
	}
	for _, check := range checks {
		if check.a != "" && check.a != check.b {
			return false
		}
	}
	return true
}

// oauthAuthorize handles authorization requests - RFC 6749 Section 3.1
func (openapi *OpenAPI) oauthAuthorize(c *gin.Context) {
	// Parse and validate authorization request
	authReq, parseErr := openapi.parseAuthorizationRequest(c)
	if parseErr != nil {
		if !openapi.authorizationErrorRedirectSafe(c, authReq, parseErr) {
			response.RespondWithError(c, http.StatusBadRequest, parseErr)
			return
		}
		response.RespondWithAuthorizationError(c, authReq.RedirectURI, parseErr, authReq.State)
		return
	}

	promptValues, promptErr := parsePromptValues(authReq.Prompt)
	if promptErr != nil {
		response.RespondWithError(c, http.StatusBadRequest, &response.ErrorResponse{
			Code:             types.ErrorInvalidRequest,
			ErrorDescription: promptErr.Error(),
		})
		return
	}

	// Resolve the Yao login session before issuing an authorization code.
	// A valid session is required: without it the browser is sent to the Yao
	// login page with the current authorize URL as return_to.
	if oauthService, ok := openapi.OAuth.(*oauthpkg.Service); ok {
		// Validate the request first so invalid clients/redirect URIs/response
		// types/scopes never reach the login page.
		// Yao OAuth 2.1 client authorization always requires PKCE S256.
		if authReq.CodeChallenge == "" || authReq.CodeChallengeMethod != types.CodeChallengeMethodS256 {
			response.RespondWithError(c, http.StatusBadRequest, &response.ErrorResponse{
				Code:             types.ErrorInvalidRequest,
				ErrorDescription: "PKCE code_challenge with S256 is required",
			})
			return
		}

		if validationErr := oauthService.ValidateAuthorizationRequest(c, authReq); validationErr != nil {
			oauthError := &response.ErrorResponse{
				Code:             validationErr.Error,
				ErrorDescription: validationErr.ErrorDescription,
			}
			if !authorizationErrorRedirectSafe(validationErr.Error, validationErr.ErrorDescription) {
				response.RespondWithError(c, http.StatusBadRequest, oauthError)
				return
			}
			response.RespondWithAuthorizationError(c, authReq.RedirectURI, oauthError, authReq.State)
			return
		}

		// The Tao callback may have arrived from a cross-site redirect where the
		// browser dropped the SameSite=Strict session cookies. Consume the
		// one-time handoff and use its access token for this request only.
		isPOST := c.Request.Method == http.MethodPost
		var consentRecord *oauthpkg.ConsentRecord
		postedTeamID := ""
		if isPOST {
			postedTeamID = strings.TrimSpace(openapi.getParam(c, "team_id"))
			consentToken := openapi.getParam(c, "consent_token")
			if consentToken == "" {
				response.RespondWithError(c, http.StatusBadRequest, &response.ErrorResponse{
					Code:             types.ErrorInvalidRequest,
					ErrorDescription: "consent_token is required",
				})
				return
			}
			record, err := oauthService.ConsumeConsentToken(c, consentToken)
			if err != nil {
				response.RespondWithError(c, http.StatusBadRequest, &response.ErrorResponse{
					Code:             types.ErrorInvalidRequest,
					ErrorDescription: "Invalid or expired consent token",
				})
				return
			}
			if !consentRequestMatches(authReq, record.Request) {
				response.RespondWithError(c, http.StatusBadRequest, &response.ErrorResponse{
					Code:             types.ErrorInvalidRequest,
					ErrorDescription: "Authorization request does not match consent token",
				})
				return
			}
			authReq = record.Request
			// The team is the user's choice on the consent page. It is not part
			// of consentRequestMatches (and need not be present on the original
			// GET), so merge the POSTed value before membership validation.
			if postedTeamID != "" {
				authReq.TeamID = postedTeamID
			}
			consentRecord = record
		}

		handoffUsed := false
		if !isPOST {
			if handoffID := openapi.getParam(c, "login_handoff"); handoffID != "" {
				handoff, ok := user.ConsumeTaoLoginHandoff(handoffID)
				if !ok || handoff.AccessToken == "" {
					response.RespondWithError(c, http.StatusBadRequest, &response.ErrorResponse{
						Code:             types.ErrorInvalidRequest,
						ErrorDescription: "Invalid or expired login handoff",
					})
					return
				}
				c.Request.Header.Set("Authorization", "Bearer "+handoff.AccessToken)
				handoffUsed = true
			}
		}

		var session *oauthpkg.AuthorizeSession
		if consentRecord != nil {
			session = &oauthpkg.AuthorizeSession{
				UserID:         consentRecord.UserID,
				Subject:        consentRecord.Subject,
				AuthSource:     consentRecord.AuthSource,
				IdentitySource: consentRecord.IdentitySource,
			}
		} else {
			var err error
			session, err = oauthService.ResolveAuthorizeSession(c, authReq.ClientID)
			if err != nil {
				response.RespondWithAuthorizationError(c, authReq.RedirectURI, response.ErrServerError, authReq.State)
				return
			}
		}

		if session == nil {
			if isPOST {
				response.RespondWithAuthorizationError(c, authReq.RedirectURI, &response.ErrorResponse{
					Code:             types.ErrorLoginRequired,
					ErrorDescription: "Login required",
				}, authReq.State)
				return
			}
			if handoffUsed {
				response.RespondWithAuthorizationError(c, authReq.RedirectURI, &response.ErrorResponse{
					Code:             types.ErrorLoginRequired,
					ErrorDescription: "Login handoff could not establish a session",
				}, authReq.State)
				return
			}
			if promptHas(promptValues, "none") {
				response.RespondWithAuthorizationError(c, authReq.RedirectURI, &response.ErrorResponse{
					Code:             types.ErrorLoginRequired,
					ErrorDescription: "Login required",
				}, authReq.State)
				return
			}
			// Interactive login: preserve prompt=consent so the next request
			// reaches the consent page rather than looping through login again.
			// login/select_account/max_age=0 must force the Tao OP to reauthenticate.
			forceReauth := promptHas(promptValues, "login") || promptHas(promptValues, "select_account") || strings.TrimSpace(authReq.MaxAge) == "0"
			openapi.redirectAuthorizeLogin(c, authReq, "consent", forceReauth)
			return
		}

		// The consent record is authoritative for identity after the page POSTs.
		authReq.Subject = session.Subject
		authReq.UserID = session.UserID
		authReq.AuthSource = session.AuthSource
		authReq.IdentitySource = session.IdentitySource

		if err := validateAuthorizeTeam(c.Request.Context(), oauthService, authReq, session.UserID); err != nil {
			response.RespondWithAuthorizationError(c, authReq.RedirectURI, &response.ErrorResponse{
				Code:             types.ErrorAccessDenied,
				ErrorDescription: err.Error(),
			}, authReq.State)
			return
		}

		if isPOST {
			// The consent POST is the confirmation step; issue the code.
			authReq.Prompt = ""
		} else {
			if promptHas(promptValues, "none") {
				// Already authenticated: no page and no code? OIDC prompt=none
				// issues the code directly when the session is valid.
				authReq.Prompt = ""
			} else if promptHas(promptValues, "login") || promptHas(promptValues, "select_account") || strings.TrimSpace(authReq.MaxAge) == "0" {
				response.DeleteAllAuthCookies(c)
				openapi.redirectAuthorizeLogin(c, authReq, "consent", true)
				return
			} else {
				record := &oauthpkg.ConsentRecord{
					Request:        authReq,
					UserID:         session.UserID,
					Subject:        session.Subject,
					AuthSource:     session.AuthSource,
					IdentitySource: session.IdentitySource,
				}
				token, err := oauthService.CreateConsentToken(c.Request.Context(), record)
				if err != nil {
					response.RespondWithAuthorizationError(c, authReq.RedirectURI, response.ErrServerError, authReq.State)
					return
				}
				c.Redirect(http.StatusFound, openapi.buildConsentURL(c, token))
				return
			}
		}
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

	// The request_uri is one-time after successful authorization. It is kept
	// through the login redirect and consumed only when a code is issued.
	if authReq.RequestURI != "" {
		if service, ok := openapi.OAuth.(*oauthpkg.Service); ok {
			_, _ = service.ConsumePushedAuthorizationRequest(c, authReq.RequestURI)
		}
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

	case types.GrantTypeJWTBearer:
		// Handle JWT Bearer assertion grant - RFC 7523
		openapi.handleJWTBearerGrant(c)

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
	oauthService, ok := openapi.OAuth.(*oauthpkg.Service)
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
		code = ""

		// RFC 6749 §4.4: client_credentials requires confidential client
		if clientInfo.ClientType == types.ClientTypePublic {
			response.RespondWithSecureError(c, response.StatusUnauthorized, response.ErrUnauthorizedClient)
			return
		}

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
	oauthService, ok := openapi.OAuth.(*oauthpkg.Service)
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

// handleJWTBearerGrant exchanges a signed JWT access token (possibly expired) for a fresh one.
// The JWT signature must be valid; expiration is allowed up to RefreshTokenLifetime.
// This reuses the same verification and issuance logic as Guard.TryRefreshToken
// but operates as an explicit token endpoint grant without cookies or refresh tokens.
func (openapi *OpenAPI) handleJWTBearerGrant(c *gin.Context) {
	oauthService, ok := openapi.OAuth.(*oauthpkg.Service)
	if !ok {
		response.RespondWithSecureError(c, response.StatusBadRequest, response.ErrInvalidClient)
		return
	}

	// Check feature flag
	if !oauthService.GetConfig().Features.JWTBearerEnabled {
		response.RespondWithSecureError(c, response.StatusBadRequest, response.ErrUnsupportedGrantType)
		return
	}

	assertion := c.PostForm("assertion")
	if assertion == "" {
		response.RespondWithSecureError(c, response.StatusBadRequest, response.ErrInvalidRequest)
		return
	}

	// Verify JWT signature (allow expired tokens)
	claims, err := oauthService.VerifyTokenAllowExpired(assertion)
	if err != nil || claims == nil {
		response.RespondWithSecureError(c, response.StatusBadRequest, &response.ErrorResponse{
			Code:             "invalid_grant",
			ErrorDescription: "Invalid JWT assertion",
		})
		return
	}

	// Reject JWTs expired beyond RefreshTokenLifetime to limit the replay window.
	// This aligns with Guard's implicit upper bound: once the refresh token expires,
	// Guard also stops refreshing.
	if !claims.ExpiresAt.IsZero() && claims.ExpiresAt.Before(time.Now()) {
		maxAge := oauthService.GetConfig().Token.RefreshTokenLifetime
		if maxAge <= 0 {
			maxAge = 24 * time.Hour
		}
		deadline := claims.ExpiresAt.Add(maxAge)
		if time.Now().After(deadline) {
			response.RespondWithSecureError(c, response.StatusBadRequest, &response.ErrorResponse{
				Code:             "invalid_grant",
				ErrorDescription: "JWT assertion expired beyond maximum allowed age",
			})
			return
		}
	}

	// Derive accessTTL from the original token's iat/exp (same logic as Guard.TryRefreshToken)
	var accessTTL time.Duration
	if !claims.IssuedAt.IsZero() && !claims.ExpiresAt.IsZero() {
		accessTTL = claims.ExpiresAt.Sub(claims.IssuedAt)
	}
	if accessTTL <= 0 {
		accessTTL = oauthService.GetConfig().Token.AccessTokenLifetime
	}
	if accessTTL <= 0 {
		accessTTL = time.Hour
	}

	// Propagate extra claims (team_id, tenant_id, etc.) — same as Guard.TryRefreshToken
	extraClaims := claims.Extra
	if extraClaims == nil {
		extraClaims = make(map[string]interface{})
	}
	if claims.TeamID != "" {
		extraClaims["team_id"] = claims.TeamID
	}
	if claims.TenantID != "" {
		extraClaims["tenant_id"] = claims.TenantID
	}

	expiresIn := int(accessTTL.Seconds())
	newToken, err := oauthService.MakeAccessToken(
		claims.ClientID,
		claims.Scope,
		claims.Subject,
		expiresIn,
		extraClaims,
	)
	if err != nil {
		response.RespondWithSecureError(c, response.StatusInternalServerError, &response.ErrorResponse{
			Code:             "server_error",
			ErrorDescription: "Failed to issue access token",
		})
		return
	}

	tokenResponse := &types.RefreshTokenResponse{
		AccessToken: newToken,
		TokenType:   types.TokenTypeBearer,
		ExpiresIn:   expiresIn,
	}

	response.RespondWithSecureSuccess(c, response.StatusOK, tokenResponse)
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
// The response carries the standard subject plus the Yao identity extensions
// auth_source / identity_source. Team/Tenant claims are never returned.
func (openapi *OpenAPI) oauthUserInfo(c *gin.Context) {
	// Check for Bearer token in Authorization header or access token cookie
	tokenStr := extractBearerToken(c)
	if tokenStr == "" {
		response.RespondWithError(c, response.StatusUnauthorized, response.ErrInvalidToken)
		return
	}

	oauthService, ok := openapi.OAuth.(*oauthpkg.Service)
	if !ok {
		response.RespondWithError(c, response.StatusInternalServerError, response.ErrServerError)
		return
	}

	claims, err := oauthService.VerifyToken(tokenStr)
	if err != nil || claims == nil || claims.Subject == "" {
		response.RespondWithError(c, response.StatusUnauthorized, response.ErrInvalidToken)
		return
	}

	userInfo := oauthService.BuildUserInfo(c.Request.Context(), claims)
	response.RespondWithSecureSuccess(c, response.StatusOK, userInfo)
}

// OAuth Extended Endpoints Implementation

// oauthRegister handles dynamic client registration - RFC 7591
func (openapi *OpenAPI) oauthRegister(c *gin.Context) {
	var req response.DynamicClientRegistrationRequest

	remoteIP, userAgent := c.ClientIP(), c.Request.UserAgent()

	if err := c.ShouldBindJSON(&req); err != nil {
		oauthpkg.RecordDynamicClientRegistrationAudit(nil, nil, response.ErrInvalidClientMetadata.Code, remoteIP, userAgent)
		response.RespondWithSecureError(c, response.StatusBadRequest, response.ErrInvalidClientMetadata)
		return
	}

	// Basic validation
	if len(req.RedirectURIs) == 0 {
		oauthpkg.RecordDynamicClientRegistrationAudit(&req, nil, response.ErrMissingRedirectURI.Code, remoteIP, userAgent)
		response.RespondWithSecureError(c, response.StatusBadRequest, response.ErrMissingRedirectURI)
		return
	}

	// Filter out client_credentials grant from HTTP DCR requests.
	// In-process DCR (server self-registration) is not affected.
	if len(req.GrantTypes) > 0 {
		filteredGrants := make([]string, 0, len(req.GrantTypes))
		for _, g := range req.GrantTypes {
			if g != "client_credentials" {
				filteredGrants = append(filteredGrants, g)
			}
		}
		req.GrantTypes = filteredGrants
	}

	res, err := openapi.OAuth.DynamicClientRegistration(c, &req)
	if err != nil {
		errorCode := response.ErrInvalidClientMetadata.Code
		if oauthErr, ok := err.(*response.ErrorResponse); ok && oauthErr.Code != "" {
			errorCode = oauthErr.Code
		}
		oauthpkg.RecordDynamicClientRegistrationAudit(&req, nil, errorCode, remoteIP, userAgent)
		if oauthErr, ok := err.(*response.ErrorResponse); ok {
			response.RespondWithSecureError(c, response.StatusBadRequest, oauthErr)
			return
		}
		response.RespondWithSecureError(c, response.StatusBadRequest, response.ErrInvalidClientMetadata)
		return
	}

	// DCR stays enabled; only an audit entry is recorded.
	oauthpkg.RecordDynamicClientRegistrationAudit(&req, &response.ClientInfo{ClientID: res.ClientID, ClientName: req.ClientName}, "", remoteIP, userAgent)

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
		response.RespondWithSecureError(c, response.StatusBadRequest, response.ErrInvalidRequest)
		return
	}

	scope := c.PostForm("scope")
	oauthService := openapi.OAuth

	res, err := oauthService.DeviceAuthorization(c, clientID, scope)
	if err != nil {
		if oauthErr, ok := err.(*response.ErrorResponse); ok {
			response.RespondWithSecureError(c, response.StatusBadRequest, oauthErr)
		} else {
			response.RespondWithSecureError(c, response.StatusBadRequest, response.ErrInvalidRequest)
		}
		return
	}

	response.RespondWithSecureSuccess(c, response.StatusOK, res)
}

// oauthDeviceAuthorize allows an authenticated user to authorize a pending device code.
func (openapi *OpenAPI) oauthDeviceAuthorize(c *gin.Context) {
	tokenStr := extractBearerToken(c)
	if tokenStr == "" {
		response.RespondWithSecureError(c, response.StatusUnauthorized, &response.ErrorResponse{
			Code:             types.ErrorInvalidGrant,
			ErrorDescription: "Bearer token required",
		})
		return
	}
	svc, ok := openapi.OAuth.(*oauthpkg.Service)
	if !ok {
		response.RespondWithSecureError(c, response.StatusInternalServerError, &response.ErrorResponse{
			Code:             types.ErrorServerError,
			ErrorDescription: "OAuth service unavailable",
		})
		return
	}

	tokenClaims, err := svc.VerifyToken(tokenStr)
	if err != nil || tokenClaims == nil {
		response.RespondWithSecureError(c, response.StatusUnauthorized, &response.ErrorResponse{
			Code:             types.ErrorInvalidGrant,
			ErrorDescription: "Invalid or expired token",
		})
		return
	}

	if tokenClaims.Subject == "" || tokenClaims.ClientID == "" {
		response.RespondWithSecureError(c, response.StatusUnauthorized, &response.ErrorResponse{
			Code:             types.ErrorInvalidGrant,
			ErrorDescription: "Token has no subject",
		})
		return
	}
	userID, err := svc.UserID(tokenClaims.ClientID, tokenClaims.Subject)
	if err != nil || userID == "" {
		response.RespondWithSecureError(c, response.StatusUnauthorized, &response.ErrorResponse{
			Code:             types.ErrorInvalidGrant,
			ErrorDescription: "Token subject is unknown",
		})
		return
	}

	extraClaims := make(map[string]interface{})
	if tokenClaims.ClientID != "" {
		extraClaims["authorizer_client_id"] = tokenClaims.ClientID
	}

	userCode := c.PostForm("user_code")
	if userCode == "" {
		userCode = c.Query("user_code")
	}
	if userCode == "" {
		var body struct {
			UserCode string `json:"user_code"`
		}
		if c.ShouldBindJSON(&body) == nil {
			userCode = body.UserCode
		}
	}
	if userCode == "" {
		response.RespondWithSecureError(c, response.StatusBadRequest, response.ErrInvalidRequest)
		return
	}

	if err := svc.AuthorizeDeviceForUser(c.Request.Context(), userCode, userID, extraClaims); err != nil {
		if oauthErr, ok := err.(*response.ErrorResponse); ok {
			response.RespondWithSecureError(c, response.StatusBadRequest, oauthErr)
		} else {
			response.RespondWithSecureError(c, response.StatusBadRequest, response.ErrInvalidGrant)
		}
		return
	}

	response.RespondWithSecureSuccess(c, response.StatusOK, map[string]string{"status": "authorized"})
}

// oauthPushedAuthorizationRequest handles PAR - RFC 9126
func (openapi *OpenAPI) oauthPushedAuthorizationRequest(c *gin.Context) {
	var req response.PushedAuthorizationRequest
	if strings.HasPrefix(c.ContentType(), "application/json") {
		if err := c.ShouldBindJSON(&req); err != nil {
			response.RespondWithError(c, response.StatusBadRequest, response.ErrInvalidRequest)
			return
		}
	} else {
		req = response.PushedAuthorizationRequest{
			ClientID:            c.PostForm("client_id"),
			ResponseType:        c.PostForm("response_type"),
			RedirectURI:         c.PostForm("redirect_uri"),
			Scope:               c.PostForm("scope"),
			State:               c.PostForm("state"),
			CodeChallenge:       c.PostForm("code_challenge"),
			CodeChallengeMethod: c.PostForm("code_challenge_method"),
			Resource:            c.PostForm("resource"),
			Prompt:              c.PostForm("prompt"),
			LoginHint:           c.PostForm("login_hint"),
			MaxAge:              c.PostForm("max_age"),
			TeamID:              c.PostForm("team_id"),
		}
	}
	if req.ClientID == "" {
		response.RespondWithError(c, response.StatusBadRequest, response.ErrInvalidRequest)
		return
	}
	service, ok := openapi.OAuth.(*oauthpkg.Service)
	if !ok {
		response.RespondWithError(c, response.StatusBadRequest, response.ErrInvalidRequest)
		return
	}
	parResponse, err := service.PushAuthorizationRequest(c, &req)
	if err != nil {
		if oauthErr, ok := err.(*response.ErrorResponse); ok {
			response.RespondWithSecureError(c, response.StatusBadRequest, oauthErr)
			return
		}
		response.RespondWithSecureError(c, response.StatusBadRequest, response.ErrInvalidRequest)
		return
	}
	response.RespondWithSecureSuccess(c, response.StatusCreated, parResponse)
}

// oauthEndSession handles OIDC RP-Initiated Logout. The minimum supported
// form is client_id + post_logout_redirect_uri + state; id_token_hint is
// accepted but not required.
func (openapi *OpenAPI) oauthEndSession(c *gin.Context) {
	clientID := openapi.getParam(c, "client_id")
	redirectURI := openapi.getParam(c, "post_logout_redirect_uri")
	state := openapi.getParam(c, "state")
	service, ok := openapi.OAuth.(*oauthpkg.Service)
	if !ok {
		response.RespondWithError(c, http.StatusInternalServerError, response.ErrServerError)
		return
	}

	if redirectURI != "" {
		if clientID == "" {
			response.RespondWithError(c, http.StatusBadRequest, &response.ErrorResponse{
				Code:             types.ErrorInvalidRequest,
				ErrorDescription: "client_id is required with post_logout_redirect_uri",
			})
			return
		}
		client, err := service.GetClientProvider().GetClientByID(c.Request.Context(), clientID)
		if err != nil || client == nil {
			response.RespondWithError(c, http.StatusBadRequest, response.ErrInvalidClient)
			return
		}
		valid := false
		for _, registered := range client.RedirectURIs {
			if registered == redirectURI {
				valid = true
				break
			}
		}
		if !valid {
			response.RespondWithError(c, http.StatusBadRequest, &response.ErrorResponse{
				Code:             types.ErrorInvalidRequest,
				ErrorDescription: "post_logout_redirect_uri is not registered for this client",
			})
			return
		}
	}

	ctx := c.Request.Context()
	if accessToken := service.GetAccessToken(c); accessToken != "" {
		_ = service.Revoke(ctx, accessToken, "access_token")
	}
	if refreshToken := service.GetRefreshToken(c); refreshToken != "" {
		_ = service.Revoke(ctx, refreshToken, "refresh_token")
	}
	response.DeleteAllAuthCookies(c)

	if redirectURI != "" {
		target, err := url.Parse(redirectURI)
		if err != nil {
			response.RespondWithError(c, http.StatusBadRequest, response.ErrInvalidRequest)
			return
		}
		if state != "" {
			q := target.Query()
			q.Set("state", state)
			target.RawQuery = q.Encode()
		}
		c.Redirect(http.StatusFound, target.String())
		return
	}
	response.RespondWithSuccess(c, http.StatusOK, map[string]interface{}{"message": "Logout successful"})
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
// authorizationErrorRedirectSafe reports whether an OAuth error may be sent to
// the client redirect_uri. Client and redirect_uri validation must complete
// before any redirect; otherwise the endpoint returns a JSON 400 and never
// becomes an open redirect.
func authorizationErrorRedirectSafe(code, description string) bool {
	switch code {
	case types.ErrorInvalidClient:
		return false
	case types.ErrorInvalidRequest:
		if description == "" {
			return false
		}
		desc := strings.ToLower(description)
		return !strings.Contains(desc, "redirect") && !strings.Contains(desc, "client")
	default:
		return true
	}
}

func (openapi *OpenAPI) authorizationErrorRedirectSafe(c *gin.Context, authReq *types.AuthorizationRequest, parseErr *response.ErrorResponse) bool {
	if authReq == nil || authReq.ClientID == "" || authReq.RedirectURI == "" {
		return false
	}
	service, ok := openapi.OAuth.(*oauthpkg.Service)
	if !ok {
		return false
	}
	validation := service.ValidateAuthorizationRequest(c, authReq)
	if validation == nil {
		return true
	}
	return authorizationErrorRedirectSafe(validation.Error, validation.ErrorDescription)
}

func (openapi *OpenAPI) parseAuthorizationRequest(c *gin.Context) (*types.AuthorizationRequest, *response.ErrorResponse) {
	// RFC 9126: when request_uri is present, resolve the stored PAR and use its
	// parameters. It is retrieved here and consumed only after authorization
	// succeeds, because the browser may pass through the Yao login page first.
	if requestURI := openapi.getParam(c, "request_uri"); requestURI != "" {
		service, ok := openapi.OAuth.(*oauthpkg.Service)
		if !ok {
			return nil, response.ErrInvalidRequest
		}
		pushed, err := service.GetPushedAuthorizationRequest(c, requestURI)
		if err != nil {
			return nil, response.ErrInvalidRequest
		}
		if clientID := openapi.getParam(c, "client_id"); clientID != "" && clientID != pushed.ClientID {
			return nil, response.ErrInvalidClient
		}
		return &types.AuthorizationRequest{
			ClientID:            pushed.ClientID,
			ResponseType:        pushed.ResponseType,
			RedirectURI:         pushed.RedirectURI,
			Scope:               pushed.Scope,
			State:               pushed.State,
			CodeChallenge:       pushed.CodeChallenge,
			CodeChallengeMethod: pushed.CodeChallengeMethod,
			Resource:            pushed.Resource,
			Prompt:              pushed.Prompt,
			LoginHint:           pushed.LoginHint,
			MaxAge:              pushed.MaxAge,
			TeamID:              pushed.TeamID,
			RequestURI:          requestURI,
		}, nil
	}
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
		Prompt:              openapi.getParam(c, "prompt"),
		LoginHint:           openapi.getParam(c, "login_hint"),
		MaxAge:              openapi.getParam(c, "max_age"),
		TeamID:              openapi.getParam(c, "team_id"),
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

// extractBearerToken reads the access token from Authorization header or cookie,
// matching the same logic as guard.getAccessToken.
func extractBearerToken(c *gin.Context) string {
	if auth := c.GetHeader("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	cookieName := response.GetCookieName("access_token")
	if cookie, err := c.Cookie(cookieName); err == nil && cookie != "" {
		return strings.TrimPrefix(cookie, "Bearer ")
	}
	return ""
}
