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
