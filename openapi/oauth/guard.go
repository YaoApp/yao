package oauth

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/openapi/oauth/acl"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
)

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
				log.Error("[OAuth] Token refresh failed: %v", refreshErr)
				response.RespondWithError(c, http.StatusUnauthorized, types.ErrInvalidRefreshToken)
				c.Abort()
				return false
			}
			claims = newClaims
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
