package user

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/openapi/oauth"
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/openapi/utils"
)

// authbackPrepare receives the post data and forwards to the authback handler
func authbackPrepare(c *gin.Context) {
	code := c.PostForm("code")
	state := c.PostForm("state")
	user := c.PostForm("user") // form_post may include user info
	providerID := c.Param("provider")
	redirectURI, err := getRedirectURI(providerID, state)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Failed to get redirect URI",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Cache user info if provided (form_post mode)
	if user != "" {
		saveUserInfo(providerID, state, user)
	}

	params := url.Values{}
	params.Add("code", code)
	params.Add("state", state)
	c.Redirect(http.StatusFound, redirectURI+"?"+params.Encode())
}

// authback is the handler for OAuth callback
func authback(c *gin.Context) {
	sid := utils.GetSessionID(c)
	var params OAuthAuthbackRequest
	providerID := c.Param("provider")

	// Check if provider exists first
	provider, err := GetProvider(providerID)
	if err != nil || provider == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: fmt.Sprintf("OAuth provider '%s' not found", providerID),
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return
	}

	if err := c.ShouldBind(&params); err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid request",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	if params.State == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "State is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	if err := validateState(providerID, sid, params.State); err != nil {
		log.With(log.F{"sid": sid, "state": params.State}).Error("Invalid state")
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid state",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Get redirect URI
	redirectURI, err := getRedirectURI(providerID, params.State)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Failed to get redirect URI",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Get provider
	provider, err = GetProvider(providerID)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: fmt.Sprintf("Failed to get provider: %v", err),
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return
	}

	// if response mode is form_post
	if provider.ResponseMode == "form_post" {
		// Replace the redirectURI to
		pathname := strings.TrimSuffix(c.Request.URL.Path, "/callback") + "/authorize/prepare"
		newRedirectURI, err := reconstructRedirectURI(redirectURI, pathname, c)
		if err != nil {
			log.Error("Failed to reconstruct redirectURI: %v", err)
			response.RespondWithError(c, response.StatusBadRequest, &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: "Invalid redirect URI format",
			})
			return
		}
		redirectURI = newRedirectURI
	}

	// Get AccessToken
	tokenResponse, err := provider.AccessToken(params.Code, redirectURI)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: fmt.Sprintf("Failed to get user info: %v", err),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Read cached user info before cleaning up (for form_post mode)
	cachedUserInfo, _ := getUserInfo(providerID, params.State)

	// Remove the state from the session and cache (also cleans up user cache automatically)
	err = removeState(providerID, sid)
	if err != nil {
		log.With(log.F{"sid": sid, "providerID": providerID}).Error("Failed to remove state")
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Failed to remove state",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Get UserInfo - use different method based on user_info_source
	var userInfo *OAuthUserInfoResponse
	if provider.UserInfoSource == UserInfoSourceIDToken {
		// For OAuth providers that use id_token, pass cached user info for merging
		userInfo, err = provider.GetUserInfoFromTokenResponse(tokenResponse, cachedUserInfo)
	} else {
		// For standard OAuth providers that use userinfo endpoint
		userInfo, err = provider.GetUserInfo(tokenResponse.AccessToken, tokenResponse.TokenType)
	}

	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: fmt.Sprintf("Failed to get user info: %v", err),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// LoginThirdParty(providerID, userInfo)
	loginResponse, err := LoginThirdParty(providerID, userInfo, userIPAddress(c))
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Failed to login: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Send all login cookies (access token, refresh token, and session ID)
	SendLoginCookies(c, loginResponse, sid)

	// Send IDToken to the client
	response.RespondWithSuccess(c, response.StatusOK, LoginSuccessResponse{
		SessionID:             sid,
		IDToken:               loginResponse.IDToken,
		AccessToken:           loginResponse.AccessToken,
		RefreshToken:          loginResponse.RefreshToken,
		ExpiresIn:             loginResponse.ExpiresIn,
		RefreshTokenExpiresIn: loginResponse.RefreshTokenExpiresIn,
		MFAEnabled:            loginResponse.MFAEnabled,
	})
}

// getOAuthAuthorizationURL generates OAuth authorization URL for a provider
func getOAuthAuthorizationURL(c *gin.Context) {
	providerID := c.Param("provider")
	if providerID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Provider ID is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Get optional parameters
	redirectURI := c.Query("redirect_uri")
	state := c.Query("state")

	provider, err := GetProvider(providerID)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Failed to get provider",
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return
	}

	if provider == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: fmt.Sprintf("OAuth provider '%s' not found", providerID),
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return
	}

	// Validate required provider configuration
	if provider.ClientID == "" || provider.Endpoints == nil || provider.Endpoints.Authorization == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Provider configuration is incomplete",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Check if state is provided by user and validate format
	var warnings []string

	// Generate state if not provided
	if state == "" {
		var err error
		state, err = generateRandomState()
		if err != nil {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: "Failed to generate OAuth state",
			}
			response.RespondWithError(c, response.StatusInternalServerError, errorResp)
			return
		}
	} else {
		// User provided state - check if it's in UUID format
		if !isValidUUID(state) {
			warnings = append(warnings, "State parameter is not in UUID format. For better uniqueness and security, consider using UUID format.")
		}
	}

	// Set default redirect URI if not provided
	if redirectURI == "" {
		redirectURI = fmt.Sprintf("%s://%s/auth/callback", getScheme(c), c.Request.Host)
	}

	// Build authorization URL
	params := url.Values{}
	params.Add("client_id", provider.ClientID)
	params.Add("response_type", "code")
	params.Add("redirect_uri", redirectURI)
	params.Add("state", state)

	// Add scopes
	if len(provider.Scopes) > 0 {
		params.Add("scope", strings.Join(provider.Scopes, " "))
	}

	// Add response_mode if specified (required for Apple with name/email scopes)
	if provider.ResponseMode != "" {
		params.Add("response_mode", provider.ResponseMode)
	}

	// Set session id if not exists
	sid := utils.GetSessionID(c)
	if sid == "" {
		sid = generateSessionID()
		response.SendSessionCookie(c, sid)
	}

	// Save the state to the session for 20 minutes
	err = saveState(providerID, sid, state)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Failed to save OAuth state",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// if response mode is form_post
	if provider.ResponseMode == "form_post" {
		// Replace the redirectURI to
		pathname := c.Request.URL.Path + "/prepare"
		newRedirectURI, err := reconstructRedirectURI(redirectURI, pathname, c)
		if err != nil {
			log.Error("Failed to reconstruct redirectURI: %v", err)
			response.RespondWithError(c, response.StatusBadRequest, &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: "Invalid redirect URI format",
			})
			return
		}

		params.Set("redirect_uri", newRedirectURI)
	}

	// Save the redirect URI to the cache
	err = saveRedirectURI(providerID, state, redirectURI)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Failed to save OAuth redirect URI",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Build the authorization URL
	authorizationURL := fmt.Sprintf("%s?%s", provider.Endpoints.Authorization, params.Encode())

	// Return the authorization URL and state
	response.RespondWithSuccess(c, response.StatusOK, &OAuthAuthorizationURLResponse{
		AuthorizationURL: authorizationURL,
		State:            state,
		Warnings:         warnings,
	})
}

// Helper functions for OAuth state management

// generateRandomState generates a UUID-based state parameter for better uniqueness
func generateRandomState() (string, error) {
	u := uuid.New()
	return u.String(), nil
}

// isValidUUID checks if a string is a valid UUID format
func isValidUUID(s string) bool {
	// UUID v4 format: 8-4-4-4-12 hexadecimal characters
	uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	return uuidRegex.MatchString(strings.ToLower(s))
}

// getScheme returns the request scheme (http or https)
func getScheme(c *gin.Context) string {
	if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
		return "https"
	}
	return "http"
}

// reconstructRedirectURI reconstructs redirectURI with new path while preserving the original host
func reconstructRedirectURI(originalRedirectURI, newPath string, c *gin.Context) (string, error) {
	// Parse the original redirectURI to extract host
	parsedURL, err := url.Parse(originalRedirectURI)
	if err != nil {
		return "", fmt.Errorf("failed to parse redirectURI: %v", err)
	}

	// Reconstruct with the original host and new path
	newRedirectURI := fmt.Sprintf("%s://%s%s", getScheme(c), parsedURL.Host, newPath)
	return newRedirectURI, nil
}

// Cache management functions

// userInfoKey returns the key for the user info
func userInfoKey(providerID, state string) string {
	return fmt.Sprintf("signin:user_info:%s:%s", providerID, state)
}

// stateKey returns the key for the state
func stateKey(providerID string) string {
	return fmt.Sprintf("signin:state:%s", providerID)
}

// redirectURIKey returns the key for the redirect URI
func redirectURIKey(providerID, state string) string {
	return fmt.Sprintf("signin:redirect_uri:%s:%s", providerID, state)
}

// saveState saves the state to the session
func saveState(providerID, sid, state string) error {
	return session.Global().ID(sid).SetWithEx(stateKey(providerID), state, 20*time.Minute)
}

// saveRedirectURI saves the redirect URI to the session
func saveRedirectURI(providerID, state, redirectURI string) error {
	key := redirectURIKey(providerID, state)
	store := oauth.OAuth.GetCache()
	return store.Set(key, redirectURI, 20*time.Minute)
}

// getRedirectURI gets the redirect URI from the session
func getRedirectURI(providerID, state string) (string, error) {
	key := redirectURIKey(providerID, state)
	store := oauth.OAuth.GetCache()
	value, ok := store.Get(key)
	if !ok || value == nil {
		return "", fmt.Errorf("redirect URI not found")
	}
	return value.(string), nil
}

func removeRedirectURI(providerID, state string) error {
	key := redirectURIKey(providerID, state)
	store := oauth.OAuth.GetCache()
	return store.Del(key)
}

// saveUserInfo saves the user info to cache (for form_post mode)
func saveUserInfo(providerID, state, userInfo string) error {
	key := userInfoKey(providerID, state)
	store := oauth.OAuth.GetCache()
	return store.Set(key, userInfo, 20*time.Minute)
}

// getUserInfo gets the user info from cache
func getUserInfo(providerID, state string) (string, error) {
	key := userInfoKey(providerID, state)
	store := oauth.OAuth.GetCache()
	value, ok := store.Get(key)
	if !ok || value == nil {
		return "", fmt.Errorf("user info not found")
	}
	return value.(string), nil
}

// removeUserInfo removes the user info from cache
func removeUserInfo(providerID, state string) error {
	key := userInfoKey(providerID, state)
	store := oauth.OAuth.GetCache()
	return store.Del(key)
}

// removeState removes the state from the session
func removeState(providerID, sid string) error {
	// Get the state from the session
	state, err := session.Global().ID(sid).Get(stateKey(providerID))
	if err != nil {
		return err
	}

	// Safely convert state to string
	stateStr, ok := state.(string)
	if !ok {
		return fmt.Errorf("invalid state type: expected string, got %T", state)
	}

	// Remove all related cached data
	removeRedirectURI(providerID, stateStr)
	removeUserInfo(providerID, stateStr)

	return session.Global().ID(sid).Del(stateKey(providerID))
}

// validateState validates the state from the session
func validateState(providerID, sid, state string) error {
	value, err := session.Global().ID(sid).Get(stateKey(providerID))
	if err != nil {
		return err
	}

	// Safely convert value to string
	stateStr, ok := value.(string)
	if !ok {
		return fmt.Errorf("invalid state type: expected string, got %T", value)
	}

	if stateStr != state {
		return fmt.Errorf("invalid state")
	}

	return nil
}

// getUserRealIP is the function to get the real IP address of the user
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
