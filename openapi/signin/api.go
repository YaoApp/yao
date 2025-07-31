package signin

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/openapi/utils"
)

// OAuthAuthorizationURLResponse represents the response for OAuth authorization URL
type OAuthAuthorizationURLResponse struct {
	AuthorizationURL string `json:"authorization_url"`
	State            string `json:"state"`
}

// Attach attaches the signin handlers to the router
func Attach(group *gin.RouterGroup, oauth types.OAuth) {
	group.GET("/signin", getConfig)
	group.POST("/signin", signin)
	group.POST("/signin/authback/:provider", authback)
	group.GET("/signin/oauth/:provider/authorize", getOAuthAuthorizationURL)
	group.POST("/signin/oauth/:provider/authorize/prepare", authbackPrepare) // Receive the post data and forward to the authback handler
}

// getConfig is the handler for get signin configuration
func getConfig(c *gin.Context) {
	// Get locale from query parameter (optional)
	locale := c.Query("locale")

	// Get public configuration for the specified locale
	config := GetPublicConfig(locale)

	// Set session id if not exists
	sid := utils.GetSessionID(c)
	if sid == "" {
		sid = generateSessionID()
		response.SendSessionCookie(c, sid)
	}

	// If no configuration found, return error
	if config == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "No signin configuration found for the requested locale",
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return
	}

	// Return the public configuration
	response.RespondWithSuccess(c, response.StatusOK, config)
}

// signin is the handler for signin (password login)
func signin(c *gin.Context) {}

// authback is the handler for authback
func authbackPrepare(c *gin.Context) {
	code := c.PostForm("code")
	state := c.PostForm("state")
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

	// Remove the redirect URI from the session
	err = removeRedirectURI(providerID, state)
	if err != nil {
		log.Warn("Failed to remove redirect URI: %v", err)
	}

	params := url.Values{}
	params.Add("code", code)
	params.Add("state", state)
	c.Redirect(http.StatusFound, redirectURI+"?"+params.Encode())
}

// authback is the handler for authback
func authback(c *gin.Context) {
	sid := utils.GetSessionID(c)
	providerID := c.Param("provider")
	state := c.PostForm("state")

	if state == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "State is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	if err := validateState(providerID, sid, state); err != nil {
		log.With(log.F{"sid": sid, "state": state}).Error("Invalid state")
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid state",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Remove the state from the session
	err := removeState(providerID, sid)
	if err != nil {
		log.With(log.F{"sid": sid, "providerID": providerID}).Error("Failed to remove state")
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Failed to remove state",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Respond with success
	response.RespondWithSuccess(c, response.StatusOK, maps.Map{
		"state": state,
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
	locale := c.Query("locale")

	// Get full configuration
	config := GetFullConfig(locale)
	if config == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "No signin configuration found",
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return
	}

	// Find the provider
	var provider *Provider
	if config.ThirdParty != nil && config.ThirdParty.Providers != nil {
		for _, p := range config.ThirdParty.Providers {
			if p.ID == providerID {
				provider = p
				break
			}
		}
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
	} else {
		params.Add("scope", "openid profile email")
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
	err := saveState(providerID, sid, state)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Failed to save OAuth state",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// if response mode is form_post, save the redirect URI to the session
	if provider.ResponseMode == "form_post" {
		err := saveRedirectURI(providerID, state, redirectURI)
		if err != nil {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: "Failed to save OAuth redirect URI",
			}
			response.RespondWithError(c, response.StatusInternalServerError, errorResp)
			return
		}

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

	// Build the authorization URL
	authorizationURL := fmt.Sprintf("%s?%s", provider.Endpoints.Authorization, params.Encode())

	// Return the authorization URL and state
	response.RespondWithSuccess(c, response.StatusOK, &OAuthAuthorizationURLResponse{
		AuthorizationURL: authorizationURL,
		State:            state,
	})
}

// generateRandomState generates a cryptographically secure random state parameter
func generateRandomState() (string, error) {
	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
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

// generateSessionID generates a session ID
func generateSessionID() string {
	return session.ID()
}

// saveState saves the state to the session
func saveState(providerID, sid, state string) error {
	return session.Global().ID(sid).SetWithEx(fmt.Sprintf("oauth_state_%s", providerID), state, 20*time.Minute)
}

// saveRedirectURI saves the redirect URI to the session
func saveRedirectURI(providerID, state, redirectURI string) error {
	key := fmt.Sprintf("oauth_redirect_uri_%s_%s", providerID, state)
	store, err := store.Get("__yao.oauth.cache")
	if err != nil {
		return err
	}
	store.Set(key, redirectURI, 20*time.Minute)
	return nil
}

// getRedirectURI gets the redirect URI from the session
func getRedirectURI(providerID, state string) (string, error) {
	key := fmt.Sprintf("oauth_redirect_uri_%s_%s", providerID, state)
	store, err := store.Get("__yao.oauth.cache")
	if err != nil {
		return "", err
	}

	value, ok := store.Get(key)
	if !ok || value == nil {
		return "", fmt.Errorf("redirect URI not found")
	}
	return value.(string), nil
}

func removeRedirectURI(providerID, state string) error {
	key := fmt.Sprintf("oauth_redirect_uri_%s_%s", providerID, state)
	store, err := store.Get("__yao.oauth.cache")
	if err != nil {
		return err
	}
	store.Del(key)
	return nil
}

func removeState(providerID, sid string) error {
	return session.Global().ID(sid).Del(fmt.Sprintf("oauth_state_%s", providerID))
}

// validateState validates the state from the session
func validateState(providerID, sid, state string) error {
	value, err := session.Global().ID(sid).Get(fmt.Sprintf("oauth_state_%s", providerID))
	if err != nil {
		return err
	}

	if value != state {
		return fmt.Errorf("invalid state")
	}

	return nil
}
