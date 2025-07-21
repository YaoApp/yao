package oauth

import (
	"net/http"

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
}
