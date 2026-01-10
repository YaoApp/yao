package user

import (
	"net"
	"strings"

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

	// Get locale from Accept-Language header or X-Locale header
	locale := c.GetHeader("X-Locale")
	if locale == "" {
		locale = c.GetHeader("Accept-Language")
		// Parse Accept-Language to get primary language (e.g., "zh-CN,zh;q=0.9" -> "zh-CN")
		if idx := strings.Index(locale, ","); idx > 0 {
			locale = locale[:idx]
		}
	}

	return &LoginContext{
		IP:        userIPAddress(c),
		UserAgent: userAgent,
		Device:    device,
		Platform:  platform,
		Locale:    locale,
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
