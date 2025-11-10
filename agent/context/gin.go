package context

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/plan"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
)

// NewGin create a new context from gin context
func NewGin(c *gin.Context) Context {
	// Get authorized information
	authInfo := authorized.GetInfo(c)

	// Extract parameters from query and route
	chatID := c.Query("chat_id")
	assistantID := c.Param("assistant_id") // Get from route parameter
	locale := c.Query("locale")
	theme := c.Query("theme")
	referer := c.Query("referer")
	accept := c.Query("accept")

	// Parse client information from User-Agent header
	userAgent := c.GetHeader("User-Agent")
	clientType := parseClientType(userAgent)
	clientIP := c.ClientIP()

	// Create base context
	ctx := Context{
		Context:     c.Request.Context(),
		Space:       plan.NewMemorySharedSpace(),
		Authorized:  authInfo,
		ChatID:      chatID,
		AssistantID: assistantID,
		Locale:      locale,
		Theme:       theme,
		Client: Client{
			Type:      clientType,
			UserAgent: userAgent,
			IP:        clientIP,
		},
	}

	// Get Referer from query parameter, header, or default
	ctx.Referer = getValidatedValue(referer, c.GetHeader("X-Yao-Referer"), RefererAPI, validateReferer)

	// Get Accept from query parameter, header, or default
	ctx.Accept = getValidatedAccept(accept, c.GetHeader("X-Yao-Accept"), clientType)

	return ctx
}

// parseClientType parses the client type from User-Agent header
func parseClientType(userAgent string) string {
	if userAgent == "" {
		return "web" // Default to web
	}

	ua := strings.ToLower(userAgent)

	// Check for specific client types
	switch {
	case strings.Contains(ua, "yao-agent") || strings.Contains(ua, "agent"):
		return "agent"
	case strings.Contains(ua, "yao-jssdk") || strings.Contains(ua, "jssdk"):
		return "jssdk"
	case strings.Contains(ua, "android"):
		return "android"
	case strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad") || strings.Contains(ua, "ipod"):
		return "ios"
	case strings.Contains(ua, "windows"):
		return "windows"
	case strings.Contains(ua, "mac os x") || strings.Contains(ua, "macintosh"):
		return "macos"
	case strings.Contains(ua, "linux"):
		return "linux"
	default:
		return "web"
	}
}
