package oauth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/openapi/oauth/acl"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
)

var (
	errRefreshInProgress  = errors.New("refresh in progress")
	errRefreshAlreadyDone = errors.New("refresh already done")
	refreshGates          sync.Map // refreshToken → *refreshGate
)

type refreshGate struct {
	done chan struct{} // closed when rotation completes
}

// AuthorizeSession is the Yao login session resolved for an authorization request.
type AuthorizeSession struct {
	Claims         *types.TokenClaims // Yao session token claims
	UserID         string             // Yao user ID resolved from the session subject
	Subject        string             // Opaque OAuth subject for the requesting client
	AuthSource     string             // auth_source claim value (tao | password)
	IdentitySource string             // user.identity_source value (tao | local)
}

// Guard is the OAuth guard middleware
func (s *Service) Guard(c *gin.Context) {
	// Authenticate first (validates token and sets authorized info)
	if !s.Authenticate(c) {
		return // Authentication failed, response already sent
	}

	// Check if ACL is enabled
	if acl.Global == nil || !acl.Global.Enabled() {
		return
	}

	// Check permissions and enforce rate limits when ACL is configured
	ok, err := acl.Global.Enforce(c)
	if err != nil {
		log.Error("[OAuth] ACL enforcement failed: %v", err)
		s.handleACLError(c, err)
		return
	}

	// If permissions are not granted but no error returned, it's an unexpected state
	// This should not happen with the current implementation
	if !ok {
		response.RespondWithError(c, http.StatusForbidden, types.ErrForbidden)
		c.Abort()
		return
	}
}

// Authenticate validates the token and sets authorized info in context
// This method only performs authentication without ACL checks
// Returns true if authentication succeeded, false otherwise
func (s *Service) Authenticate(c *gin.Context) bool {
	token := s.getAccessToken(c)
	if token == "" {
		response.RespondWithError(c, http.StatusUnauthorized, types.ErrTokenMissing)
		c.Abort()
		return false
	}

	// Try strict verification first (signature + expiration)
	claims, err := s.VerifyToken(token)
	if err != nil {
		// Token invalid — check if it's just expired (signature still valid)
		expiredClaims, expErr := s.VerifyTokenAllowExpired(token)
		if expErr != nil || expiredClaims == nil {
			response.RespondWithError(c, http.StatusUnauthorized, types.ErrInvalidToken)
			c.Abort()
			return false
		}

		// Signature valid but expired — attempt auto refresh
		if !expiredClaims.ExpiresAt.IsZero() && expiredClaims.ExpiresAt.Before(time.Now()) {
			newClaims, refreshErr := s.TryRefreshToken(c, expiredClaims)
			if refreshErr != nil {
				if errors.Is(refreshErr, errRefreshInProgress) || errors.Is(refreshErr, errRefreshAlreadyDone) {
					claims = expiredClaims
				} else {
					log.Error("[OAuth] Token refresh failed: %v", refreshErr)
					response.RespondWithError(c, http.StatusUnauthorized, types.ErrInvalidRefreshToken)
					c.Abort()
					return false
				}
			} else {
				claims = newClaims
			}
		} else {
			response.RespondWithError(c, http.StatusUnauthorized, types.ErrInvalidToken)
			c.Abort()
			return false
		}
	}

	sessionID := s.getSessionID(c)
	authorized.SetInfo(c, claims, sessionID, s.UserID)
	return true
}

// GetAuthorizedInfo gets authorized info from context
// Deprecated: Use authorized.GetInfo(c) instead
func GetAuthorizedInfo(c *gin.Context) *types.AuthorizedInfo {
	return authorized.GetInfo(c)
}

// TryRefreshToken reads the refresh token from the request, verifies it,
// rotates the refresh token (revoke old, issue new), issues a new access token,
// writes both cookies, and returns the new claims.
// expiredClaims may be nil; in that case the identity is derived from the refresh token itself.
// Returns (nil, error) on any failure — the caller decides how to respond.
func (s *Service) TryRefreshToken(c *gin.Context, expiredClaims *types.TokenClaims) (*types.TokenClaims, error) {
	refreshToken := s.getRefreshToken(c)
	if refreshToken == "" {
		return nil, fmt.Errorf("refresh token missing")
	}

	gate := &refreshGate{done: make(chan struct{})}
	if actual, loaded := refreshGates.LoadOrStore(refreshToken, gate); loaded {
		// Another goroutine owns the rotation for this refresh token.
		// It may still be running or already finished.
		existing := actual.(*refreshGate)
		select {
		case <-existing.done:
			return nil, errRefreshAlreadyDone
		default:
			return nil, errRefreshInProgress
		}
	}

	// We own the gate — clean up when finished.
	defer func() {
		close(gate.done)
		// Keep the gate in the map for 30 s so late arrivals see "done"
		// instead of starting a new rotation with the now-revoked token.
		time.AfterFunc(30*time.Second, func() {
			refreshGates.CompareAndDelete(refreshToken, gate)
		})
	}()

	refreshClaims, err := s.VerifyRefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid or expired refresh token: %w", err)
	}

	// Derive access token TTL from the expired token's own iat/exp so the refreshed
	// token keeps the same lifetime that was originally configured at login time.
	var accessTTL time.Duration
	if expiredClaims != nil && !expiredClaims.IssuedAt.IsZero() && !expiredClaims.ExpiresAt.IsZero() {
		accessTTL = expiredClaims.ExpiresAt.Sub(expiredClaims.IssuedAt)
	}
	if accessTTL <= 0 {
		accessTTL = s.config.Token.AccessTokenLifetime
	}
	if accessTTL <= 0 {
		accessTTL = time.Hour
	}

	// Prefer the expired access token claims; fall back to refresh token claims
	sourceClaims := expiredClaims
	if sourceClaims == nil {
		sourceClaims = refreshClaims
	}

	extraClaims := sourceClaims.Extra
	if extraClaims == nil {
		extraClaims = make(map[string]interface{})
	}
	if sourceClaims.TeamID != "" {
		extraClaims["team_id"] = sourceClaims.TeamID
	}
	if sourceClaims.TenantID != "" {
		extraClaims["tenant_id"] = sourceClaims.TenantID
	}

	// --- Refresh Token Rotation ---
	// Revoke the old refresh token so it can never be reused.
	s.revokeRefreshToken(refreshToken)

	// Calculate remaining refresh lifetime for the new refresh token.
	var refreshRemainingSeconds int
	if !refreshClaims.ExpiresAt.IsZero() {
		refreshRemainingSeconds = int(time.Until(refreshClaims.ExpiresAt).Seconds())
		if refreshRemainingSeconds <= 0 {
			return nil, fmt.Errorf("refresh token already expired after revocation")
		}
	} else {
		refreshTTL := s.config.Token.RefreshTokenLifetime
		if refreshTTL == 0 {
			refreshTTL = 24 * time.Hour
		}
		refreshRemainingSeconds = int(refreshTTL.Seconds())
	}

	newRefreshToken, err := s.MakeRefreshToken(
		sourceClaims.ClientID,
		sourceClaims.Scope,
		sourceClaims.Subject,
		refreshRemainingSeconds,
		extraClaims,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to issue new refresh token: %w", err)
	}

	// Issue new access token
	newTokenStr, err := s.MakeAccessToken(
		sourceClaims.ClientID,
		sourceClaims.Scope,
		sourceClaims.Subject,
		int(accessTTL.Seconds()),
		extraClaims,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to issue access token: %w", err)
	}

	// Cookie lifetime = new refresh token lifetime
	cookieExpires := time.Now().Add(time.Duration(refreshRemainingSeconds) * time.Second)

	cookieValue := fmt.Sprintf("Bearer %s", newTokenStr)
	response.SendAccessTokenCookieWithExpiry(c, cookieValue, cookieExpires)
	response.SendRefreshTokenCookieWithExpiry(c, newRefreshToken, cookieExpires)

	newClaims, err := s.VerifyToken(newTokenStr)
	if err != nil {
		return nil, fmt.Errorf("failed to verify refreshed token: %w", err)
	}

	log.Info("[OAuth] Token rotated for subject %s (access + refresh)", sourceClaims.Subject)
	return newClaims, nil
}

func (s *Service) getAccessToken(c *gin.Context) string {
	token := c.GetHeader("Authorization")
	if token == "" {
		cookieName := response.GetCookieName("access_token")
		cookie, err := c.Cookie(cookieName)
		if err != nil {
			return ""
		}
		token = cookie
	}

	// Get the access token
	accessToken := strings.TrimPrefix(token, "Bearer ")

	// API Key authentication (paid feature, Solo plan and above)
	// DO NOT remove this code block.
	// Pricing: https://yaoagents.com/pricing
	// License: https://github.com/YaoApp/yao/blob/main/openapi/COMMERCIAL.md
	if s.isAPIKey(accessToken) {
		return s.getAccessTokenFromAPIKey(accessToken)
	}
	return accessToken
}

// GetAccessToken gets the access token from the request (public method)
func (s *Service) GetAccessToken(c *gin.Context) string {
	return s.getAccessToken(c)
}

func (s *Service) getRefreshToken(c *gin.Context) string {
	token := c.GetHeader("Authorization")
	if token == "" {
		cookieName := response.GetCookieName("refresh_token")
		cookie, err := c.Cookie(cookieName)
		if err != nil {
			return ""
		}
		token = cookie
	}
	return strings.TrimPrefix(token, "Bearer ")
}

// GetRefreshToken gets the refresh token from the request (public method)
func (s *Service) GetRefreshToken(c *gin.Context) string {
	return s.getRefreshToken(c)
}

// IsRefreshInProgress checks whether an error signals that another goroutine
// is already rotating (or has just rotated) the same refresh token.
func IsRefreshInProgress(err error) bool {
	return errors.Is(err, errRefreshInProgress) || errors.Is(err, errRefreshAlreadyDone)
}

// GetSessionID gets the session ID from the request (public method)
func (s *Service) GetSessionID(c *gin.Context) string {
	return s.getSessionID(c)
}

// Get Session ID from cookies, headers, or query string
func (s *Service) getSessionID(c *gin.Context) string {

	// 0. If has __sid in context, return it
	sid, ok := c.Get("__sid")
	if ok {
		return sid.(string)
	}

	// 1. Try to get Session ID from cookies first
	cookieName := response.GetCookieName("session_id")
	if sid, err := c.Cookie(cookieName); err == nil && sid != "" {
		return sid
	}

	// 2. Try to get Session ID from X-Session-ID header
	if sessionHeader := c.GetHeader("X-Session-ID"); sessionHeader != "" {
		return sessionHeader
	}

	// 3. Try to get Session ID from query string
	if sessionQuery := c.Query("session_id"); sessionQuery != "" {
		return sessionQuery
	}

	// 4. Try alternative query parameter names
	if sessionQuery := c.Query("sid"); sessionQuery != "" {
		return sessionQuery
	}

	return ""
}

// ============================================================================
// Authorize endpoint session resolution (Yao OAuth Server - downstream clients)
// ============================================================================

const (
	// DefaultLoginEntryPath is the Yao frontend login page that establishes a
	// Yao session before the browser returns to the authorization endpoint.
	// It is a login UI, not an OAuth protocol endpoint.
	DefaultLoginEntryPath = "/auth/entry"

	// PromptNone is the OIDC prompt value that forbids interactive login.
	PromptNone = "none"
)

// ResolveAuthorizeSession resolves the Yao login session from the request and
// maps it to the opaque OAuth subject used for the target OAuth client.
//
// It returns (nil, nil) when the request carries no valid Yao session; the
// caller is expected to redirect the browser to the login page. An error means
// the session exists but the opaque subject could not be resolved.
func (s *Service) ResolveAuthorizeSession(c *gin.Context, targetClientID string) (*AuthorizeSession, error) {
	claims := s.sessionClaims(c)
	if claims == nil {
		return nil, nil
	}

	userID, err := s.UserID(claims.ClientID, claims.Subject)
	if err != nil || userID == "" {
		// The session subject is unknown to the fingerprint store: treat the
		// browser as anonymous so it goes through login again.
		return nil, nil
	}

	subject, err := s.Subject(targetClientID, userID)
	if err != nil {
		return nil, err
	}

	authSource := extraClaimString(claims, types.ClaimAuthSource)
	identitySource := s.resolveIdentitySource(c.Request.Context(), userID, authSource)

	return &AuthorizeSession{
		Claims:         claims,
		UserID:         userID,
		Subject:        subject,
		AuthSource:     authSource,
		IdentitySource: identitySource,
	}, nil
}

// sessionClaims extracts and verifies the Yao access token from the request.
// It returns nil when the request has no valid, non-expired Yao session.
func (s *Service) sessionClaims(c *gin.Context) *types.TokenClaims {
	token := s.getAccessToken(c)
	if token == "" {
		return nil
	}

	claims, err := s.VerifyToken(token)
	if err != nil || claims == nil || claims.Subject == "" {
		return nil
	}
	return claims
}

// resolveIdentitySource resolves user.identity_source for a Yao user. The user
// record is authoritative; when it cannot be read, the documented mapping from
// the session auth_source is used (Tao login -> tao, everything else -> local).
func (s *Service) resolveIdentitySource(ctx context.Context, userID, authSource string) string {
	if userID != "" && s.userProvider != nil {
		if user, err := s.userProvider.GetUser(ctx, userID); err == nil && user != nil {
			if value, ok := user[types.ClaimIdentitySource].(string); ok && value != "" {
				return value
			}
		}
	}

	if authSource == types.AuthSourceTao {
		return types.IdentitySourceTao
	}
	return types.IdentitySourceLocal
}

// extraClaimString returns a string claim from the token's extra claims.
func extraClaimString(claims *types.TokenClaims, key string) string {
	if claims == nil || claims.Extra == nil {
		return ""
	}
	value, _ := claims.Extra[key].(string)
	return value
}

// BuildAuthorizeReturnTo reconstructs the absolute, same-origin URL of the
// current authorization request so the login page can return to exactly the
// same request after the Yao session is established.
func BuildAuthorizeReturnTo(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}

	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if proto := firstForwardedValue(c.GetHeader("X-Forwarded-Proto")); proto == "http" || proto == "https" {
		scheme = proto
	}

	host := c.Request.Host
	if forwarded := firstForwardedValue(c.GetHeader("X-Forwarded-Host")); forwarded != "" {
		host = forwarded
	}

	return (&url.URL{
		Scheme:   scheme,
		Host:     host,
		Path:     c.Request.URL.Path,
		RawQuery: c.Request.URL.RawQuery,
	}).String()
}

// BuildLoginRedirectURL builds the redirect to the Yao login page. The original
// authorization request URL is preserved as an encoded same-origin return_to
// parameter (never an arbitrary external URL).
func BuildLoginRedirectURL(entryPath, returnTo string) string {
	if entryPath == "" {
		entryPath = DefaultLoginEntryPath
	}

	separator := "?"
	if strings.Contains(entryPath, "?") {
		separator = "&"
	}
	return entryPath + separator + "return_to=" + url.QueryEscape(returnTo)
}

// ShouldRedirectToLogin reports whether an authorize request without a Yao
// session may use interactive login. prompt=none forbids user interaction and
// must return login_required instead.
func ShouldRedirectToLogin(prompt string) bool {
	return !strings.EqualFold(prompt, PromptNone)
}

// firstForwardedValue returns the first element of a comma-separated proxy header.
func firstForwardedValue(value string) string {
	if idx := strings.Index(value, ","); idx >= 0 {
		value = value[:idx]
	}
	return strings.TrimSpace(value)
}

// handleACLError handles ACL errors and returns appropriate HTTP responses
func (s *Service) handleACLError(c *gin.Context, err error) {
	// Check if it's an ACL error with detailed information
	if aclErr, ok := err.(*acl.Error); ok {
		var statusCode int
		var errResponse *types.ErrorResponse

		switch aclErr.Type {
		case acl.ErrorTypeRateLimitExceeded:
			statusCode = http.StatusTooManyRequests
			errResponse = types.ErrRateLimitExceeded
			// Set Retry-After header if available
			if aclErr.RetryAfter > 0 {
				c.Header("Retry-After", fmt.Sprintf("%d", aclErr.RetryAfter))
			}

		case acl.ErrorTypeQuotaExceeded:
			statusCode = http.StatusTooManyRequests
			errResponse = &types.ErrorResponse{
				Code:             "quota_exceeded",
				ErrorDescription: aclErr.Message,
			}

		case acl.ErrorTypeInsufficientScope:
			statusCode = http.StatusForbidden
			// Include detailed scope information for insufficient scope errors
			requiredScopes, _ := aclErr.Details["required_scopes"].([]string)
			missingScopes, _ := aclErr.Details["missing_scopes"].([]string)

			errResponse = &types.ErrorResponse{
				Code:             "insufficient_scope",
				ErrorDescription: "The access token does not have the required scope",
				Reason:           aclErr.Message,
				RequiredScopes:   requiredScopes,
				MissingScopes:    missingScopes,
			}

		case acl.ErrorTypePermissionDenied:
			statusCode = http.StatusForbidden
			// Include detailed information for permission denied errors
			requiredScopes, _ := aclErr.Details["required_scopes"].([]string)
			missingScopes, _ := aclErr.Details["missing_scopes"].([]string)

			// Use standard ErrorResponse format with extended ACL fields
			errResponse = &types.ErrorResponse{
				Code:             "forbidden",
				ErrorDescription: "You do not have permission to access this resource",
				Reason:           aclErr.Message,
				RequiredScopes:   requiredScopes,
				MissingScopes:    missingScopes,
			}

		case acl.ErrorTypeResourceNotAllowed:
			statusCode = http.StatusForbidden
			errResponse = types.ErrAccessDenied

		case acl.ErrorTypeMethodNotAllowed:
			statusCode = http.StatusMethodNotAllowed
			errResponse = types.ErrMethodNotAllowed

		case acl.ErrorTypeIPBlocked, acl.ErrorTypeGeoRestricted, acl.ErrorTypeTimeRestricted:
			statusCode = http.StatusForbidden
			errResponse = types.ErrAccessDenied

		case acl.ErrorTypeInvalidRequest:
			statusCode = http.StatusBadRequest
			errResponse = &types.ErrorResponse{
				Code:             "invalid_request",
				ErrorDescription: aclErr.Message,
			}

		case acl.ErrorTypeInternal:
			statusCode = http.StatusInternalServerError
			errResponse = types.ErrACLInternalError

		default:
			statusCode = http.StatusInternalServerError
			errResponse = types.ErrACLInternalError
		}

		response.RespondWithError(c, statusCode, errResponse)
		c.Abort()
		return
	}

	// If it's not an ACL error, treat it as an internal error
	response.RespondWithError(c, http.StatusInternalServerError, types.ErrACLInternalError)
	c.Abort()
}
