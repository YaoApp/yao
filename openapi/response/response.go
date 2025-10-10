package response

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// Type aliases for OAuth types to simplify usage
type (
	// Core response types

	// ErrorResponse represents an OAuth 2.0 error response as defined in RFC 6749
	ErrorResponse = types.ErrorResponse

	// Token represents an OAuth 2.0 access token response as defined in RFC 6749
	Token = types.Token

	// RefreshTokenResponse represents an OAuth 2.0 refresh token response
	RefreshTokenResponse = types.RefreshTokenResponse

	// Authorization flow types

	// AuthorizationRequest represents an OAuth 2.0 authorization request parameters
	AuthorizationRequest = types.AuthorizationRequest

	// AuthorizationResponse represents an OAuth 2.0 authorization response
	AuthorizationResponse = types.AuthorizationResponse

	// Client management types

	// ClientInfo represents OAuth 2.0 client registration information
	ClientInfo = types.ClientInfo

	// DynamicClientRegistrationRequest represents a dynamic client registration request as defined in RFC 7591
	DynamicClientRegistrationRequest = types.DynamicClientRegistrationRequest

	// DynamicClientRegistrationResponse represents a dynamic client registration response as defined in RFC 7591
	DynamicClientRegistrationResponse = types.DynamicClientRegistrationResponse

	// Extended OAuth types

	// DeviceAuthorizationResponse represents a device authorization response as defined in RFC 8628
	DeviceAuthorizationResponse = types.DeviceAuthorizationResponse

	// PushedAuthorizationRequest represents a pushed authorization request as defined in RFC 9126
	PushedAuthorizationRequest = types.PushedAuthorizationRequest

	// PushedAuthorizationResponse represents a pushed authorization response as defined in RFC 9126
	PushedAuthorizationResponse = types.PushedAuthorizationResponse

	// TokenExchangeResponse represents a token exchange response as defined in RFC 8693
	TokenExchangeResponse = types.TokenExchangeResponse

	// TokenIntrospectionResponse represents a token introspection response as defined in RFC 7662
	TokenIntrospectionResponse = types.TokenIntrospectionResponse

	// Discovery types

	// AuthorizationServerMetadata represents OAuth 2.0 authorization server metadata as defined in RFC 8414
	AuthorizationServerMetadata = types.AuthorizationServerMetadata

	// ProtectedResourceMetadata represents OAuth 2.0 protected resource metadata as defined in RFC 9728
	ProtectedResourceMetadata = types.ProtectedResourceMetadata

	// Security types

	// WWWAuthenticateChallenge represents a WWW-Authenticate challenge header structure
	WWWAuthenticateChallenge = types.WWWAuthenticateChallenge

	// JWKSResponse represents a JSON Web Key Set response as defined in RFC 7517
	JWKSResponse = types.JWKSResponse

	// JWK represents a JSON Web Key as defined in RFC 7517
	JWK = types.JWK
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
	ErrMFARequired              = &ErrorResponse{Code: "mfa_required", ErrorDescription: "Multi-factor authentication is required to access this resource."}
	ErrTeamSelectionRequired    = &ErrorResponse{Code: "team_selection_required", ErrorDescription: "Team selection is required to access this resource."}

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

// setOAuthSecurityHeaders sets standard OAuth 2.0/2.1 security headers
// These headers are required by OAuth 2.1 specification for enhanced security
func SetOAuthSecurityHeaders(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("X-Frame-Options", "DENY")
	c.Header("Referrer-Policy", "no-referrer")
}

// setJSONContentType sets JSON content type header for OAuth responses
func SetJSONContentType(c *gin.Context) {
	c.Header("Content-Type", "application/json")
}

// RespondWithSuccess sends a successful response (no wrapper, direct data)
func RespondWithSuccess(c *gin.Context, statusCode int, data interface{}) {
	SetJSONContentType(c)
	c.JSON(statusCode, data)
}

// RespondWithError sends an error response (no wrapper, direct error)
func RespondWithError(c *gin.Context, statusCode int, err *ErrorResponse) {
	SetJSONContentType(c)

	// Add WWW-Authenticate header for 401 responses
	if statusCode == StatusUnauthorized {
		AddWWWAuthenticateHeader(c, err)
	}

	c.JSON(statusCode, err)
}

// SecureCookieOptions defines options for secure cookie configuration
type SecureCookieOptions struct {
	// MaxAge specifies the max age for the cookie in seconds (0 = session cookie, negative = delete cookie)
	// MaxAge takes precedence over Expires if both are set
	MaxAge int
	// Expires specifies the absolute expiration time for the cookie
	// If MaxAge is 0 and Expires is set, Expires will be used
	Expires *time.Time
	// Path specifies the cookie path (default: "/")
	Path string
	// Domain specifies the cookie domain (empty for current domain)
	Domain string
	// SameSite specifies the SameSite attribute ("Strict", "Lax", or "None")
	SameSite string
	// UseHostPrefix determines if __Host- prefix should be used (most secure)
	UseHostPrefix bool
	// UseSecurePrefix determines if __Secure- prefix should be used
	UseSecurePrefix bool
}

// NewSecureCookieOptions creates a new SecureCookieOptions with secure defaults
func NewSecureCookieOptions() *SecureCookieOptions {
	return &SecureCookieOptions{
		MaxAge:        0,     // Session cookie by default
		Path:          "/",   // Root path
		Domain:        "",    // Current domain
		SameSite:      "Lax", // Default SameSite policy
		UseHostPrefix: true,  // Use most secure __Host- prefix
	}
}

// WithMaxAge sets the MaxAge in seconds
func (o *SecureCookieOptions) WithMaxAge(maxAge int) *SecureCookieOptions {
	o.MaxAge = maxAge
	return o
}

// WithExpires sets the absolute expiration time
func (o *SecureCookieOptions) WithExpires(expires time.Time) *SecureCookieOptions {
	o.Expires = &expires
	return o
}

// WithDuration sets expiration based on duration from now
func (o *SecureCookieOptions) WithDuration(duration time.Duration) *SecureCookieOptions {
	expires := time.Now().Add(duration)
	o.Expires = &expires
	o.MaxAge = int(duration.Seconds())
	return o
}

// WithPath sets the cookie path
func (o *SecureCookieOptions) WithPath(path string) *SecureCookieOptions {
	o.Path = path
	return o
}

// WithDomain sets the cookie domain
func (o *SecureCookieOptions) WithDomain(domain string) *SecureCookieOptions {
	o.Domain = domain
	return o
}

// WithSameSite sets the SameSite attribute
func (o *SecureCookieOptions) WithSameSite(sameSite string) *SecureCookieOptions {
	o.SameSite = sameSite
	return o
}

// WithSecurePrefix uses __Secure- prefix instead of __Host-
func (o *SecureCookieOptions) WithSecurePrefix() *SecureCookieOptions {
	o.UseHostPrefix = false
	o.UseSecurePrefix = true
	return o
}

// WithoutPrefix disables security prefixes
func (o *SecureCookieOptions) WithoutPrefix() *SecureCookieOptions {
	o.UseHostPrefix = false
	o.UseSecurePrefix = false
	return o
}

// SendSecretCookie sends a secure cookie to the client with RFC 6265bis compliance
// For sensitive data like session_id, access_token, etc.
func SendSecretCookie(c *gin.Context, key string, value string) {
	options := &SecureCookieOptions{
		MaxAge:        0,     // Session cookie by default
		Path:          "/",   // Root path
		Domain:        "",    // Current domain
		SameSite:      "Lax", // Default SameSite policy
		UseHostPrefix: true,  // Use most secure __Host- prefix
	}
	SendSecureCookieWithOptions(c, key, value, options)
}

// SendSecureCookieWithOptions sends a secure cookie with custom options
func SendSecureCookieWithOptions(c *gin.Context, key string, value string, options *SecureCookieOptions) {
	// Apply RFC 6265bis prefix requirements
	cookieName := key
	cookiePath := options.Path
	cookieDomain := options.Domain

	if options.UseHostPrefix {
		// __Host- prefix: Requires Secure flag, no Domain attribute, Path=/
		cookieName = "__Host-" + key
		cookiePath = "/"  // Must be "/" for __Host- prefix
		cookieDomain = "" // Must be empty for __Host- prefix
	} else if options.UseSecurePrefix {
		// __Secure- prefix: Requires Secure flag, allows Domain and Path
		cookieName = "__Secure-" + key
	}

	// Ensure secure defaults
	if cookiePath == "" {
		cookiePath = "/"
	}

	// Determine effective MaxAge
	effectiveMaxAge := options.MaxAge
	if effectiveMaxAge == 0 && options.Expires != nil {
		// If MaxAge is 0 but Expires is set, calculate MaxAge from Expires
		duration := time.Until(*options.Expires)
		if duration > 0 {
			effectiveMaxAge = int(duration.Seconds())
		} else {
			effectiveMaxAge = -1 // Expired cookie
		}
	}

	// Set the cookie with secure flags
	// Gin's SetCookie: (name, value, maxAge, path, domain, secure, httpOnly)
	c.SetCookie(
		cookieName,      // name (with security prefix if specified)
		value,           // value
		effectiveMaxAge, // maxAge (calculated from Expires if needed)
		cookiePath,      // path
		cookieDomain,    // domain
		true,            // secure (HTTPS only) - required for security prefixes
		true,            // httpOnly (prevent XSS access)
	)

	// Get existing Set-Cookie headers for additional attributes
	cookies := c.Writer.Header()["Set-Cookie"]
	if len(cookies) > 0 {
		lastCookie := cookies[len(cookies)-1]

		// Add SameSite attribute if specified
		if options.SameSite != "" {
			lastCookie += "; SameSite=" + options.SameSite
		}

		// Add Expires attribute if specified and MaxAge is not used
		if options.Expires != nil && options.MaxAge == 0 {
			lastCookie += "; Expires=" + options.Expires.UTC().Format(time.RFC1123)
		}

		// Replace the last Set-Cookie header with enhanced version
		cookies[len(cookies)-1] = lastCookie
		c.Writer.Header()["Set-Cookie"] = cookies
	}
}

// SendSessionCookie sends a session cookie with __Host- prefix for maximum security
func SendSessionCookie(c *gin.Context, sessionID string) {
	options := NewSecureCookieOptions().WithSameSite("Lax")
	SendSecureCookieWithOptions(c, "session_id", sessionID, options)
}

// SendAccessTokenCookie sends an access token cookie with appropriate security settings
func SendAccessTokenCookie(c *gin.Context, accessToken string, maxAge int) {
	options := NewSecureCookieOptions().
		WithMaxAge(maxAge).
		WithSameSite("Strict")
	SendSecureCookieWithOptions(c, "access_token", accessToken, options)
}

// SendAccessTokenCookieWithExpiry sends an access token cookie with absolute expiration time
func SendAccessTokenCookieWithExpiry(c *gin.Context, accessToken string, expires time.Time) {
	options := NewSecureCookieOptions().
		WithExpires(expires).
		WithSameSite("Strict")
	SendSecureCookieWithOptions(c, "access_token", accessToken, options)
}

// SendAccessTokenCookieWithDuration sends an access token cookie with duration-based expiration
func SendAccessTokenCookieWithDuration(c *gin.Context, accessToken string, duration time.Duration) {
	options := NewSecureCookieOptions().
		WithDuration(duration).
		WithSameSite("Strict")
	SendSecureCookieWithOptions(c, "access_token", accessToken, options)
}

// SendRefreshTokenCookie sends a refresh token cookie with strict security settings
func SendRefreshTokenCookie(c *gin.Context, refreshToken string, maxAge int) {
	options := NewSecureCookieOptions().
		WithMaxAge(maxAge).
		WithPath("/auth").
		WithSameSite("Strict")
	SendSecureCookieWithOptions(c, "refresh_token", refreshToken, options)
}

// SendRefreshTokenCookieWithExpiry sends a refresh token cookie with absolute expiration time
func SendRefreshTokenCookieWithExpiry(c *gin.Context, refreshToken string, expires time.Time) {
	options := NewSecureCookieOptions().
		WithExpires(expires).
		WithPath("/auth").
		WithSameSite("Strict")
	SendSecureCookieWithOptions(c, "refresh_token", refreshToken, options)
}

// SendRefreshTokenCookieWithDuration sends a refresh token cookie with duration-based expiration
func SendRefreshTokenCookieWithDuration(c *gin.Context, refreshToken string, duration time.Duration) {
	options := NewSecureCookieOptions().
		WithDuration(duration).
		WithPath("/auth").
		WithSameSite("Strict")
	SendSecureCookieWithOptions(c, "refresh_token", refreshToken, options)
}

// DeleteSecureCookie deletes a secure cookie by setting it to expire immediately
func DeleteSecureCookie(c *gin.Context, key string) {
	options := NewSecureCookieOptions().WithMaxAge(-1) // Negative MaxAge deletes the cookie
	SendSecureCookieWithOptions(c, key, "", options)
}

// DeleteAllAuthCookies deletes all authentication-related cookies
func DeleteAllAuthCookies(c *gin.Context) {
	DeleteSecureCookie(c, "session_id")
	DeleteSecureCookie(c, "access_token")

	// Also delete refresh token with its specific path
	options := NewSecureCookieOptions().
		WithMaxAge(-1).
		WithPath("/auth")
	SendSecureCookieWithOptions(c, "refresh_token", "", options)
}

// Common duration constants for cookie expiration
const (
	// Session cookies (expires when browser closes)
	SessionCookie = 0
	// Short-lived tokens (typically for access tokens)
	OneHour     = 1 * time.Hour
	TwoHours    = 2 * time.Hour
	SixHours    = 6 * time.Hour
	TwelveHours = 12 * time.Hour
	// Medium-lived tokens
	OneDay   = 24 * time.Hour
	OneWeek  = 7 * 24 * time.Hour
	TwoWeeks = 14 * 24 * time.Hour
	// Long-lived tokens (typically for refresh tokens)
	OneMonth    = 30 * 24 * time.Hour
	ThreeMonths = 90 * 24 * time.Hour
	SixMonths   = 180 * 24 * time.Hour
	OneYear     = 365 * 24 * time.Hour
)

// RespondWithAuthorizationError sends an authorization endpoint error via redirect
func RespondWithAuthorizationError(c *gin.Context, redirectURI string, err *ErrorResponse, state string) {
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
	RespondWithError(c, StatusBadRequest, err)
}

// AddWWWAuthenticateHeader adds appropriate WWW-Authenticate header
func AddWWWAuthenticateHeader(c *gin.Context, err *ErrorResponse) {
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

// RespondWithSecureSuccess sends a successful response with OAuth security headers (for sensitive endpoints)
func RespondWithSecureSuccess(c *gin.Context, statusCode int, data interface{}) {
	SetOAuthSecurityHeaders(c)
	SetJSONContentType(c)
	c.JSON(statusCode, data)
}

// RespondWithSecureError sends an error response with OAuth security headers (for sensitive endpoints)
func RespondWithSecureError(c *gin.Context, statusCode int, err *ErrorResponse) {
	SetOAuthSecurityHeaders(c)
	SetJSONContentType(c)

	// Add WWW-Authenticate header for 401 responses
	if statusCode == StatusUnauthorized {
		AddWWWAuthenticateHeader(c, err)
	}

	c.JSON(statusCode, err)
}
