package signin

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
)

// Attach attaches the signin handlers to the router
func Attach(group *gin.RouterGroup, oauth types.OAuth) {
	group.GET("/signin", getConfig)
	group.POST("/signin", signin)
	group.GET("/signin/authback/:id", authback)
	group.GET("/signin/oauth/:provider/authorize", getOAuthAuthorizationURL)
}

// getConfig is the handler for get signin configuration
func getConfig(c *gin.Context) {
	// Get locale from query parameter (optional)
	locale := c.Query("locale")

	// Get public configuration for the specified locale
	config := GetPublicConfig(locale)

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
func authback(c *gin.Context) {}

// OAuthAuthorizationURLResponse represents the response for OAuth authorization URL
type OAuthAuthorizationURLResponse struct {
	AuthorizationURL string `json:"authorization_url"`
	State            string `json:"state"`
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

	fmt.Println("redirect_uri", redirectURI)

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

	fmt.Println("client_id", provider.ClientID)
	fmt.Println("redirectURI", redirectURI)

	// Add scopes
	if len(provider.Scopes) > 0 {
		params.Add("scope", strings.Join(provider.Scopes, " "))
	} else {
		params.Add("scope", "openid profile email")
	}

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
