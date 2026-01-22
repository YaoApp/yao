package robot

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// GetLocale extracts locale from request
// Priority: query param > Accept-Language header > default
func GetLocale(c *gin.Context) string {
	// Check query param first
	if locale := c.Query("locale"); locale != "" {
		return strings.ToLower(strings.TrimSpace(locale))
	}

	// Check Accept-Language header
	if acceptLang := c.GetHeader("Accept-Language"); acceptLang != "" {
		// Parse first language from header (e.g., "en-US,en;q=0.9" -> "en-us")
		parts := strings.Split(acceptLang, ",")
		if len(parts) > 0 {
			lang := strings.Split(parts[0], ";")[0]
			return strings.ToLower(strings.TrimSpace(lang))
		}
	}

	// Default locale
	return "en-us"
}

// ParseBoolValue parses various string formats into a boolean pointer
func ParseBoolValue(value string) *bool {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "1", "true", "yes", "on":
		v := true
		return &v
	case "0", "false", "no", "off":
		v := false
		return &v
	}
	return nil
}
