package authorized

import (
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// GetInfo extracts authorized information from the gin context
// This function reads authorization data that was set by the OAuth guard middleware
func GetInfo(c *gin.Context) *types.AuthorizedInfo {
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

	if teamID, ok := c.Get("__team_id"); ok {
		info.TeamID = teamID.(string)
	}

	if tenantID, ok := c.Get("__tenant_id"); ok {
		info.TenantID = tenantID.(string)
	}

	if sessionID, ok := c.Get("__sid"); ok {
		info.SessionID = sessionID.(string)
	}

	if rememberMe, ok := c.Get("__remember_me"); ok {
		if rmBool, ok := rememberMe.(bool); ok {
			info.RememberMe = rmBool
		}
	}

	return info
}

// SetInfo sets authorized information in the gin context
// This function should be called by the OAuth guard middleware after token validation
// userIDGetter is a function that resolves the user_id from clientID and subject
func SetInfo(c *gin.Context, claims *types.TokenClaims, sessionID string, userIDGetter func(clientID, subject string) (string, error)) {
	// Set session ID in context
	if sessionID != "" {
		c.Set("__sid", sessionID)
	}

	// Set user_id in context (resolve from claims)
	if userIDGetter != nil {
		userID, err := userIDGetter(claims.ClientID, claims.Subject)
		if err == nil && userID != "" {
			c.Set("__user_id", userID)
		}
	}

	// Set subject, scope, client_id in context
	c.Set("__subject", claims.Subject)
	c.Set("__scope", claims.Scope)
	c.Set("__client_id", claims.ClientID)

	// Set team_id and tenant_id in context if available
	if claims.TeamID != "" {
		c.Set("__team_id", claims.TeamID)
	}
	if claims.TenantID != "" {
		c.Set("__tenant_id", claims.TenantID)
	}

	// Set custom claims from Extra field into context
	if claims.Extra != nil {
		for key, value := range claims.Extra {
			c.Set("__"+key, value)
		}
	}
}
