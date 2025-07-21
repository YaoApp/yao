package openapi

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// Type aliases for OAuth types to simplify usage
type (
	// Core response types
	ErrorResponse        = types.ErrorResponse
	Token                = types.Token
	RefreshTokenResponse = types.RefreshTokenResponse

	// Authorization flow types
	AuthorizationRequest  = types.AuthorizationRequest
	AuthorizationResponse = types.AuthorizationResponse

	// Client management types
	ClientInfo                        = types.ClientInfo
	DynamicClientRegistrationRequest  = types.DynamicClientRegistrationRequest
	DynamicClientRegistrationResponse = types.DynamicClientRegistrationResponse

	// Extended OAuth types
	DeviceAuthorizationResponse = types.DeviceAuthorizationResponse
	PushedAuthorizationRequest  = types.PushedAuthorizationRequest
	PushedAuthorizationResponse = types.PushedAuthorizationResponse
	TokenExchangeResponse       = types.TokenExchangeResponse
	TokenIntrospectionResponse  = types.TokenIntrospectionResponse

	// Discovery types
	AuthorizationServerMetadata = types.AuthorizationServerMetadata
	ProtectedResourceMetadata   = types.ProtectedResourceMetadata

	// Security types
	WWWAuthenticateChallenge = types.WWWAuthenticateChallenge
	JWKSResponse             = types.JWKSResponse
	JWK                      = types.JWK
)

// Standard OAuth 2.0/2.1 Error Codes - RFC 6749 Section 5.2
var (
	// Authorization endpoint errors - RFC 6749 Section 4.1.2.1
	ErrInvalidRequest          = &ErrorResponse{Code: types.ErrorInvalidRequest, ErrorDescription: "The request is missing a required parameter, includes an invalid parameter value, includes a parameter more than once, or is otherwise malformed."}
	ErrUnauthorizedClient      = &ErrorResponse{Code: types.ErrorUnauthorizedClient, ErrorDescription: "The client is not authorized to request an authorization code using this method."}
	ErrAccessDenied            = &ErrorResponse{Code: types.ErrorAccessDenied, ErrorDescription: "The resource owner or authorization server denied the request."}
	ErrUnsupportedResponseType = &ErrorResponse{Code: types.ErrorUnsupportedResponseType, ErrorDescription: "The authorization server does not support obtaining an authorization code using this method."}
	ErrInvalidScope            = &ErrorResponse{Code: types.ErrorInvalidScope, ErrorDescription: "The requested scope is invalid, unknown, or malformed."}
	ErrServerError             = &ErrorResponse{Code: types.ErrorServerError, ErrorDescription: "The authorization server encountered an unexpected condition that prevented it from fulfilling the request."}
	ErrTemporarilyUnavailable  = &ErrorResponse{Code: types.ErrorTemporarilyUnavailable, ErrorDescription: "The authorization server is currently unable to handle the request due to a temporary overloading or maintenance of the server."}

	// Token endpoint errors - RFC 6749 Section 5.2
	ErrInvalidClient        = &ErrorResponse{Code: types.ErrorInvalidClient, ErrorDescription: "Client authentication failed (e.g., unknown client, no client authentication included, or unsupported authentication method)."}
	ErrInvalidGrant         = &ErrorResponse{Code: types.ErrorInvalidGrant, ErrorDescription: "The provided authorization grant (e.g., authorization code, resource owner credentials) or refresh token is invalid, expired, revoked, does not match the redirection URI used in the authorization request, or was issued to another client."}
	ErrUnsupportedGrantType = &ErrorResponse{Code: types.ErrorUnsupportedGrantType, ErrorDescription: "The authorization grant type is not supported by the authorization server."}

	// Token introspection and validation errors - RFC 7662
	ErrInvalidToken      = &ErrorResponse{Code: types.ErrorInvalidToken, ErrorDescription: "The access token provided is expired, revoked, malformed, or invalid for other reasons."}
	ErrInsufficientScope = &ErrorResponse{Code: types.ErrorInsufficientScope, ErrorDescription: "The request requires higher privileges than provided by the access token."}

	// Device authorization flow errors - RFC 8628 Section 3.5
	ErrAuthorizationPending = &ErrorResponse{Code: types.ErrorAuthorizationPending, ErrorDescription: "The authorization request is still pending as the end user hasn't yet completed the user-interaction steps."}
	ErrSlowDown             = &ErrorResponse{Code: types.ErrorSlowDown, ErrorDescription: "The client should slow down the polling requests to the token endpoint."}
	ErrExpiredToken         = &ErrorResponse{Code: types.ErrorExpiredToken, ErrorDescription: "The device_code has expired, and the device authorization session has concluded."}

	// Extended error codes for better developer experience
	ErrMissingRedirectURI       = &ErrorResponse{Code: "missing_redirect_uri", ErrorDescription: "The redirect_uri parameter is required but was not provided."}
	ErrInvalidRedirectURI       = &ErrorResponse{Code: "invalid_redirect_uri", ErrorDescription: "The redirect_uri parameter value is invalid or not registered for this client."}
	ErrMismatchedRedirectURI    = &ErrorResponse{Code: "mismatched_redirect_uri", ErrorDescription: "The redirect_uri does not match the one used in the authorization request."}
	ErrInvalidCodeVerifier      = &ErrorResponse{Code: "invalid_code_verifier", ErrorDescription: "The code_verifier does not match the code_challenge from the authorization request."}
	ErrMissingCodeChallenge     = &ErrorResponse{Code: "missing_code_challenge", ErrorDescription: "PKCE code_challenge is required but was not provided."}
	ErrInvalidCodeChallenge     = &ErrorResponse{Code: "invalid_code_challenge", ErrorDescription: "The code_challenge parameter is invalid or uses an unsupported method."}
	ErrInvalidClientMetadata    = &ErrorResponse{Code: "invalid_client_metadata", ErrorDescription: "The client metadata is invalid or contains unsupported values."}
	ErrInvalidSoftwareStatement = &ErrorResponse{Code: "invalid_software_statement", ErrorDescription: "The software statement is invalid or cannot be verified."}
	ErrUnapprovedSoftware       = &ErrorResponse{Code: "unapproved_software", ErrorDescription: "The software statement represents software that has been replaced or is otherwise invalid."}

	// Configuration and service errors
	ErrInvalidConfiguration     = types.ErrInvalidConfiguration
	ErrStoreMissing             = types.ErrStoreMissing
	ErrIssuerURLMissing         = types.ErrIssuerURLMissing
	ErrCertificateMissing       = types.ErrCertificateMissing
	ErrInvalidTokenLifetime     = types.ErrInvalidTokenLifetime
	ErrPKCEConfigurationInvalid = types.ErrPKCEConfigurationInvalid
)

// Standard HTTP Status Codes for OAuth Responses
const (
	// Success responses
	StatusOK        = http.StatusOK        // 200 - Successful token response
	StatusCreated   = http.StatusCreated   // 201 - Successful client registration
	StatusNoContent = http.StatusNoContent // 204 - Successful token revocation

	// Client error responses
	StatusBadRequest          = http.StatusBadRequest          // 400 - Invalid request parameters
	StatusUnauthorized        = http.StatusUnauthorized        // 401 - Authentication required
	StatusForbidden           = http.StatusForbidden           // 403 - Access denied
	StatusNotFound            = http.StatusNotFound            // 404 - Client or resource not found
	StatusMethodNotAllowed    = http.StatusMethodNotAllowed    // 405 - HTTP method not supported
	StatusNotAcceptable       = http.StatusNotAcceptable       // 406 - Content type not acceptable
	StatusConflict            = http.StatusConflict            // 409 - Client already exists
	StatusUnprocessableEntity = http.StatusUnprocessableEntity // 422 - Invalid client metadata

	// Server error responses
	StatusInternalServerError = http.StatusInternalServerError // 500 - Internal server error
	StatusNotImplemented      = http.StatusNotImplemented      // 501 - Feature not implemented
	StatusBadGateway          = http.StatusBadGateway          // 502 - Bad gateway
	StatusServiceUnavailable  = http.StatusServiceUnavailable  // 503 - Service temporarily unavailable
)

// StandardResponse represents a standard OAuth API response
type StandardResponse struct {
	Success   bool           `json:"success"`
	Data      interface{}    `json:"data,omitempty"`
	Error     *ErrorResponse `json:"error,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
	RequestID string         `json:"request_id,omitempty"`
}

// Response helper functions for consistent OAuth responses

// respondWithSuccess sends a successful OAuth response
func (openapi *OpenAPI) respondWithSuccess(c *gin.Context, statusCode int, data interface{}) {
	response := StandardResponse{
		Success:   true,
		Data:      data,
		Timestamp: time.Now().UTC(),
		RequestID: c.GetString("request_id"),
	}

	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.JSON(statusCode, response)
}

// respondWithError sends an OAuth error response
func (openapi *OpenAPI) respondWithError(c *gin.Context, statusCode int, err *ErrorResponse) {
	response := StandardResponse{
		Success:   false,
		Error:     err,
		Timestamp: time.Now().UTC(),
		RequestID: c.GetString("request_id"),
	}

	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")

	// Add WWW-Authenticate header for 401 responses
	if statusCode == StatusUnauthorized {
		openapi.addWWWAuthenticateHeader(c, err)
	}

	c.JSON(statusCode, response)
}

// respondWithTokenSuccess sends a successful token response (without wrapper)
func (openapi *OpenAPI) respondWithTokenSuccess(c *gin.Context, token interface{}) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.Header("Content-Type", "application/json;charset=UTF-8")
	c.JSON(StatusOK, token)
}

// respondWithTokenError sends a token endpoint error response (without wrapper)
func (openapi *OpenAPI) respondWithTokenError(c *gin.Context, err *ErrorResponse) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.Header("Content-Type", "application/json;charset=UTF-8")
	c.JSON(StatusBadRequest, err)
}

// respondWithAuthorizationError sends an authorization endpoint error via redirect
func (openapi *OpenAPI) respondWithAuthorizationError(c *gin.Context, redirectURI string, err *ErrorResponse, state string) {
	// Build error redirect URL
	redirectURL := redirectURI
	if redirectURL != "" {
		separator := "?"
		if len(redirectURL) > 0 && redirectURL[len(redirectURL)-1:] == "?" {
			separator = "&"
		}

		redirectURL += separator + "error=" + err.Code
		if err.ErrorDescription != "" {
			redirectURL += "&error_description=" + err.ErrorDescription
		}
		if err.ErrorURI != "" {
			redirectURL += "&error_uri=" + err.ErrorURI
		}
		if state != "" {
			redirectURL += "&state=" + state
		}

		c.Redirect(http.StatusFound, redirectURL)
		return
	}

	// Fallback to JSON error response if no redirect URI
	openapi.respondWithError(c, StatusBadRequest, err)
}

// addWWWAuthenticateHeader adds appropriate WWW-Authenticate header
func (openapi *OpenAPI) addWWWAuthenticateHeader(c *gin.Context, err *ErrorResponse) {
	challenge := &WWWAuthenticateChallenge{
		Scheme: types.WWWAuthenticateSchemeBearer,
		Realm:  "OAuth",
	}

	if err != nil {
		challenge.Error = err.Code
		challenge.ErrorDesc = err.ErrorDescription
		challenge.ErrorURI = err.ErrorURI
	}

	// Build WWW-Authenticate header value
	headerValue := challenge.Scheme
	if challenge.Realm != "" {
		headerValue += ` realm="` + challenge.Realm + `"`
	}
	if challenge.Error != "" {
		headerValue += `, error="` + challenge.Error + `"`
	}
	if challenge.ErrorDesc != "" {
		headerValue += `, error_description="` + challenge.ErrorDesc + `"`
	}
	if challenge.ErrorURI != "" {
		headerValue += `, error_uri="` + challenge.ErrorURI + `"`
	}

	c.Header("WWW-Authenticate", headerValue)
}

// Validation helper functions

// validateRedirectURI validates redirect URI according to RFC 6749
func (openapi *OpenAPI) validateRedirectURI(redirectURI string, client *ClientInfo) error {
	if redirectURI == "" {
		return ErrMissingRedirectURI
	}

	// Check if redirect URI is registered for the client
	for _, registeredURI := range client.RedirectURIs {
		if registeredURI == redirectURI {
			return nil
		}
	}

	return ErrInvalidRedirectURI
}

// validatePKCE validates PKCE parameters according to RFC 7636
func (openapi *OpenAPI) validatePKCE(codeChallenge, codeChallengeMethod, codeVerifier string) error {
	if codeChallenge == "" {
		return ErrMissingCodeChallenge
	}

	if codeChallengeMethod != types.CodeChallengeMethodS256 && codeChallengeMethod != types.CodeChallengeMethodPlain {
		return ErrInvalidCodeChallenge
	}

	// Additional PKCE validation logic would go here
	// This is a simplified example

	return nil
}

// createErrorWithState creates an error response with state parameter
func createErrorWithState(baseError *ErrorResponse, state string) *ErrorResponse {
	errorWithState := &ErrorResponse{
		Code:             baseError.Code,
		ErrorDescription: baseError.ErrorDescription,
		ErrorURI:         baseError.ErrorURI,
		State:            state,
	}
	return errorWithState
}

// OAuth 2.1 specific response helpers

// respondWithOAuth21Error ensures OAuth 2.1 compliance for error responses
func (openapi *OpenAPI) respondWithOAuth21Error(c *gin.Context, statusCode int, err *ErrorResponse) {
	// OAuth 2.1 requires additional security headers
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("X-Frame-Options", "DENY")
	c.Header("Referrer-Policy", "no-referrer")

	openapi.respondWithError(c, statusCode, err)
}

// respondWithOAuth21TokenSuccess ensures OAuth 2.1 compliance for token responses
func (openapi *OpenAPI) respondWithOAuth21TokenSuccess(c *gin.Context, token interface{}) {
	// OAuth 2.1 requires additional security headers
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("X-Frame-Options", "DENY")
	c.Header("Referrer-Policy", "no-referrer")
	c.Header("Content-Type", "application/json;charset=UTF-8")

	c.JSON(StatusOK, token)
}
