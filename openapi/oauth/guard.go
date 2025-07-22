package oauth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Guard is the OAuth guard middleware
func (s *Service) Guard(c *gin.Context) {
	// Get the token from the request
	token := c.GetHeader("Authorization")

	// Validate the token
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		c.Abort()
		return
	}

	// Validate the token
	_, err := s.VerifyToken(strings.TrimPrefix(token, "Bearer "))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		c.Abort()
		return
	}

}
