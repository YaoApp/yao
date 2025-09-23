package oauth

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// Guard is the OAuth guard middleware
func (s *Service) Guard(c *gin.Context) {
	// Get the token from the request
	token := s.getAccessToken(c)

	// Validate the token
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		c.Abort()
		return
	}

	// Validate the token
	claims, err := s.VerifyToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		c.Abort()
		return
	}

	// Auto refresh the token
	if claims.ExpiresAt.Before(time.Now()) {
		s.tryAutoRefreshToken(c, claims)
	}

	// Set Authorized Info
	s.setAuthorizedInfo(c, claims)
}

// GetAuthorizedInfo Get Authorized Info from context
func GetAuthorizedInfo(c *gin.Context) *types.AuthorizedInfo {
	info := &types.AuthorizedInfo{}

	if subject, ok := c.Get("__subject"); ok {
		info.Subject = subject.(string)
	}

	if clientID, ok := c.Get("__client_id"); ok {
		info.ClientID = clientID.(string)
	}

	if userID, ok := c.Get("__user_id"); ok {
		info.UserID = userID.(string)
	}

	if scope, ok := c.Get("__scope"); ok {
		info.Scope = scope.(string)
	}

	return info
}

// Set Authorized Info in context
func (s *Service) setAuthorizedInfo(c *gin.Context, claims *types.TokenClaims) {
	sid := s.getSessionID(c)

	// Set __sid in context
	if sid != "" {
		c.Set("__sid", sid)
	}

	// Set __userID in context
	userID, err := s.UserID(claims.ClientID, claims.Subject)
	if err == nil && userID != "" {
		c.Set("__user_id", userID)
	}

	// Set subject scope, client_id, user_id in context
	c.Set("__subject", claims.Subject)
	c.Set("__scope", claims.Scope)
	c.Set("__client_id", claims.ClientID)
}

func (s *Service) tryAutoRefreshToken(c *gin.Context, _ *types.TokenClaims) {
	refreshToken := s.getRefreshToken(c)
	if refreshToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		c.Abort()
		return
	}

	// Verify the refresh token
	_, err := s.VerifyToken(refreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		c.Abort()
		return
	}

	// @Todo: Auto refresh the token
}

func (s *Service) getAccessToken(c *gin.Context) string {
	token := c.GetHeader("Authorization")
	if token == "" {
		cookie, err := c.Cookie("__Host-access_token")
		if err != nil {
			return ""
		}
		token = cookie
	}
	return strings.TrimPrefix(token, "Bearer ")
}

func (s *Service) getRefreshToken(c *gin.Context) string {
	token := c.GetHeader("Authorization")
	if token == "" {
		cookie, err := c.Cookie("__Host-refresh_token")
		if err != nil {
			return ""
		}
		token = cookie
	}
	return strings.TrimPrefix(token, "Bearer ")
}

// Get Session ID from cookies, headers, or query string
func (s *Service) getSessionID(c *gin.Context) string {

	// 0. If has __sid in context, return it
	sid, ok := c.Get("__sid")
	if ok {
		return sid.(string)
	}

	// 1. Try to get Session ID from cookies first
	if sid, err := c.Cookie("__Host-session_id"); err == nil && sid != "" {
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
