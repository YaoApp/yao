package utils

import (
	"github.com/gin-gonic/gin"
)

// SessionGuard is a guard that checks if the session ID is valid
func SessionGuard(c *gin.Context) {
	sid := GetSessionID(c)
	if sid != "" {
		c.Set("__sid", sid)
	}
}

// GetSessionID retrieves the session ID from cookies
// Tries to get session ID from different possible cookie names (with and without security prefixes)
func GetSessionID(c *gin.Context) string {
	return getCookieWithPrefixes(c, "session_id")
}

// GetAccessToken retrieves the access token from cookies
func GetAccessToken(c *gin.Context) string {
	return getCookieWithPrefixes(c, "access_token")
}

// GetRefreshToken retrieves the refresh token from cookies (checks /auth path)
func GetRefreshToken(c *gin.Context) string {
	return getCookieWithPrefixes(c, "refresh_token")
}

// getCookieWithPrefixes tries to get a cookie value from different possible names with security prefixes
func getCookieWithPrefixes(c *gin.Context, baseName string) string {
	// Try to get cookie from different naming conventions (most secure first)
	cookieNames := []string{
		"__Host-" + baseName,   // Most secure with __Host- prefix
		"__Secure-" + baseName, // With __Secure- prefix
		baseName,               // Plain cookie name (fallback)
	}

	for _, cookieName := range cookieNames {
		if value, err := c.Cookie(cookieName); err == nil && value != "" {
			return value
		}
	}

	return ""
}

// HasValidSession checks if there's a valid session ID in the cookies
func HasValidSession(c *gin.Context) bool {
	return GetSessionID(c) != ""
}

// GetTokenFromCookie gets any named token from cookies with security prefix support
func GetTokenFromCookie(c *gin.Context, tokenName string) string {
	return getCookieWithPrefixes(c, tokenName)
}
