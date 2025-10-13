package user

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/kun/exception"
)

// Session Utilities

// GetUserIDFromSession gets the current user ID from session
// Returns the user ID string or throws an exception if not authenticated
func GetUserIDFromSession(process *process.Process) string {
	sessionData, err := session.Global().ID(process.Sid).Get("__user_id")
	if err != nil || sessionData == nil {
		exception.New("user not authenticated", 401).Throw()
	}

	userIDStr, ok := sessionData.(string)
	if !ok {
		exception.New("invalid user_id in session", 401).Throw()
	}

	return userIDStr
}

// Type Conversion Utilities

// toBool converts various types to boolean
// Supports: bool, int, int64, float64, string
// Returns false for nil or unsupported types
func toBool(v interface{}) bool {
	if v == nil {
		return false
	}

	switch val := v.(type) {
	case bool:
		return val
	case int:
		return val != 0
	case int64:
		return val != 0
	case float64:
		return val != 0
	case string:
		return val == "true" || val == "1"
	default:
		return false
	}
}

// toString converts various types to string
// Supports: string, int, int64, float64, bool
// Returns empty string for nil or unsupported types
func toString(v interface{}) string {
	if v == nil {
		return ""
	}

	switch val := v.(type) {
	case string:
		return val
	case int:
		return fmt.Sprintf("%d", val)
	case int64:
		return fmt.Sprintf("%d", val)
	case float64:
		return fmt.Sprintf("%.0f", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

// toInt64 converts various types to int64
// Supports: int, int64, float64, string
// Returns 0 for nil or unsupported types
func toInt64(v interface{}) int64 {
	if v == nil {
		return 0
	}

	switch val := v.(type) {
	case int64:
		return val
	case int:
		return int64(val)
	case float64:
		return int64(val)
	case string:
		if parsed, err := strconv.ParseInt(val, 10, 64); err == nil {
			return parsed
		}
		return 0
	default:
		return 0
	}
}

// toTimeString converts various time types to RFC3339 string
// Supports: time.Time, string, int64 (unix timestamp)
// Returns empty string for nil or unsupported types
func toTimeString(v interface{}) string {
	if v == nil {
		return ""
	}

	switch val := v.(type) {
	case time.Time:
		if val.IsZero() {
			return ""
		}
		return val.Format(time.RFC3339)
	case string:
		// Try to parse as RFC3339 first
		if t, err := time.Parse(time.RFC3339, val); err == nil {
			return t.Format(time.RFC3339)
		}
		// Try to parse as other common formats
		formats := []string{
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05Z",
			"2006-01-02T15:04:05.000Z",
		}
		for _, format := range formats {
			if t, err := time.Parse(format, val); err == nil {
				return t.Format(time.RFC3339)
			}
		}
		return val // Return as-is if can't parse
	case int64:
		// Assume unix timestamp
		if val > 0 {
			return time.Unix(val, 0).Format(time.RFC3339)
		}
		return ""
	default:
		return ""
	}
}

// Security Utilities

// maskEmail masks an email address for privacy protection
// Keeps the first and last character of the local part, masks the middle with ***
// Examples:
//   - "john.doe@example.com" -> "j***e@example.com"
//   - "a@example.com" -> "a***@example.com"
//   - "ab@example.com" -> "a***b@example.com"
//
// Returns empty string for invalid email or empty input
func maskEmail(email string) string {
	if email == "" {
		return ""
	}

	// Split email into local and domain parts
	parts := strings.Split(email, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "" // Invalid email format
	}

	local := parts[0]
	domain := parts[1]

	// Mask the local part
	var masked string
	localLen := len(local)
	switch localLen {
	case 1:
		// Single character: show it with ***
		masked = local + "***"
	case 2:
		// Two characters: show first + *** + last
		masked = string(local[0]) + "***" + string(local[1])
	default:
		// Three or more characters: show first + *** + last
		masked = string(local[0]) + "***" + string(local[localLen-1])
	}

	return masked + "@" + domain
}

// parseUserAgent extracts device and platform information from User-Agent string
// Returns device type ("mobile", "tablet", "desktop") and platform ("ios", "android", "web", etc.)
func parseUserAgent(userAgent string) (device string, platform string) {
	if userAgent == "" {
		return "unknown", "unknown"
	}

	ua := strings.ToLower(userAgent)

	// Detect platform
	switch {
	case strings.Contains(ua, "android"):
		platform = "android"
	case strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad") || strings.Contains(ua, "ipod"):
		platform = "ios"
	case strings.Contains(ua, "windows"):
		platform = "windows"
	case strings.Contains(ua, "mac os x") || strings.Contains(ua, "macintosh"):
		platform = "macos"
	case strings.Contains(ua, "linux"):
		platform = "linux"
	case strings.Contains(ua, "chrome os"):
		platform = "chromeos"
	default:
		platform = "web"
	}

	// Detect device type
	switch {
	case strings.Contains(ua, "mobile") || strings.Contains(ua, "iphone") || strings.Contains(ua, "ipod"):
		device = "mobile"
	case strings.Contains(ua, "tablet") || strings.Contains(ua, "ipad"):
		device = "tablet"
	default:
		device = "desktop"
	}

	return device, platform
}

// makeLoginContext creates a LoginContext from gin.Context with all fields populated
func makeLoginContext(c *gin.Context) *LoginContext {
	userAgent := c.GetHeader("User-Agent")
	device, platform := parseUserAgent(userAgent)

	return &LoginContext{
		IP:        userIPAddress(c),
		UserAgent: userAgent,
		Device:    device,
		Platform:  platform,
	}
}

// Network Utilities

// userIPAddress extracts the real client IP address from various HTTP headers
// Handles proxy headers, CDN headers, and direct connections
func userIPAddress(c *gin.Context) string {
	// Define HTTP headers to check, ordered by priority
	headers := []string{
		"X-Real-IP",                // Nginx proxy_set_header X-Real-IP
		"X-Forwarded-For",          // Standard proxy header
		"X-Client-IP",              // Apache mod_remoteip, Squid
		"X-Forwarded",              // Legacy proxy standard
		"X-Cluster-Client-IP",      // Cluster environment
		"Forwarded-For",            // Pre-RFC 7239 standard
		"Forwarded",                // RFC 7239 standard
		"CF-Connecting-IP",         // Cloudflare
		"True-Client-IP",           // Akamai, CloudFlare Enterprise
		"X-Original-Forwarded-For", // Original forwarded
	}

	// Check each header one by one
	for _, header := range headers {
		value := c.GetHeader(header)
		if value == "" {
			continue
		}

		// Handle cases that may contain multiple IPs (e.g., X-Forwarded-For: client, proxy1, proxy2)
		ips := parseIPList(value)
		for _, ip := range ips {
			if isValidPublicIP(ip) {
				return ip
			}
		}
	}

	// If none found, use the remote address of the connection
	remoteAddr := c.Request.RemoteAddr
	if ip := extractIPFromAddr(remoteAddr); ip != "" && isValidPublicIP(ip) {
		return ip
	}

	// Final fallback, return RemoteAddr (may include port)
	return extractIPFromAddr(remoteAddr)
}

// parseIPList parses IP list string, handles comma-separated multiple IPs
func parseIPList(value string) []string {
	var ips []string

	// Handle RFC 7239 Forwarded header format: for=192.0.2.60;proto=http;by=203.0.113.43
	if strings.Contains(value, "for=") {
		parts := strings.Split(value, ";")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "for=") {
				ip := strings.TrimPrefix(part, "for=")
				// Remove possible quotes and brackets
				ip = strings.Trim(ip, "\"[]")
				if ip != "" {
					ips = append(ips, ip)
				}
			}
		}
	} else {
		// Handle comma-separated IP list
		parts := strings.Split(value, ",")
		for _, part := range parts {
			ip := strings.TrimSpace(part)
			if ip != "" {
				ips = append(ips, ip)
			}
		}
	}

	return ips
}

// extractIPFromAddr extracts IP from address (which may include port)
func extractIPFromAddr(addr string) string {
	if addr == "" {
		return ""
	}

	// Handle IPv6 format [::1]:8080
	if strings.HasPrefix(addr, "[") {
		if idx := strings.Index(addr, "]:"); idx != -1 {
			return addr[1:idx]
		}
		return strings.Trim(addr, "[]")
	}

	// Handle IPv4 format 127.0.0.1:8080
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}

	return addr
}

// isValidPublicIP checks if the IP is a valid public IP
func isValidPublicIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	// Filter out private IPs, local IPs, etc.
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return false
	}

	// Check if it's a private IP range
	if ip.To4() != nil {
		// IPv4 private address ranges
		return !isPrivateIPv4(ip)
	}
	// IPv6 private address ranges
	return !isPrivateIPv6(ip)
}

// isPrivateIPv4 checks if it's an IPv4 private address
func isPrivateIPv4(ip net.IP) bool {
	// 10.0.0.0/8
	if ip[12] == 10 {
		return true
	}
	// 172.16.0.0/12
	if ip[12] == 172 && ip[13] >= 16 && ip[13] <= 31 {
		return true
	}
	// 192.168.0.0/16
	if ip[12] == 192 && ip[13] == 168 {
		return true
	}
	// 169.254.0.0/16 (Link-Local)
	if ip[12] == 169 && ip[13] == 254 {
		return true
	}
	return false
}

// isPrivateIPv6 checks if it's an IPv6 private address
func isPrivateIPv6(ip net.IP) bool {
	// fc00::/7 (Unique Local)
	if ip[0] >= 0xfc && ip[0] <= 0xfd {
		return true
	}
	// fe80::/10 (Link-Local)
	if ip[0] == 0xfe && (ip[1]&0xc0) == 0x80 {
		return true
	}
	return false
}
