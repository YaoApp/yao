package messenger

import (
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/messenger"
	"github.com/yaoapp/yao/messenger/types"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
)

// Attach attaches the messenger webhook handlers to the router
func Attach(group *gin.RouterGroup, oauth oauthTypes.OAuth) {

	// Webhook endpoint with provider parameter - public interface
	group.GET("/webhook/:provider", webhookHandler)
	group.POST("/webhook/:provider", webhookHandler)

	// Private API endpoints for provider and channel information
	group.GET("/providers", oauth.Guard, getProvidersHandler)
	group.GET("/providers/:name", oauth.Guard, getProviderHandler)
	group.GET("/channels", oauth.Guard, getChannelsHandler)
}

// webhookHandler is the handler for webhook endpoint
func webhookHandler(c *gin.Context) {
	// Get provider name from URL parameter
	providerName := c.Param("provider")
	if providerName == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Provider parameter is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Check if messenger service is available
	if messenger.Instance == nil {
		log.Warn("[OpenAPI Messenger] Messenger service not initialized")
		errorResp := &response.ErrorResponse{
			Code:             response.ErrTemporarilyUnavailable.Code,
			ErrorDescription: "Messenger service not available",
		}
		response.RespondWithError(c, response.StatusServiceUnavailable, errorResp)
		return
	}

	// Directly pass gin.Context to messenger service for processing
	err := messenger.Instance.TriggerWebhook(providerName, c)
	if err != nil {
		log.Error("[OpenAPI Messenger] Failed to process webhook for provider %s: %v", providerName, err)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to process webhook: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Return success response
	successResp := gin.H{
		"status":   "received",
		"message":  "webhook processed successfully",
		"provider": providerName,
	}
	response.RespondWithSuccess(c, response.StatusOK, successResp)
}

// getProviderHandler returns public information about a specific provider
func getProviderHandler(c *gin.Context) {
	providerName := c.Param("name")
	if providerName == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Provider name is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Check if messenger service is available
	if messenger.Instance == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrTemporarilyUnavailable.Code,
			ErrorDescription: "Messenger service not available",
		}
		response.RespondWithError(c, response.StatusServiceUnavailable, errorResp)
		return
	}

	// Get provider
	provider, err := messenger.Instance.GetProvider(providerName)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             "provider_not_found",
			ErrorDescription: "Provider not found: " + providerName,
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return
	}

	// Return public information directly
	response.RespondWithSuccess(c, response.StatusOK, provider.GetPublicInfo())
}

// getProvidersHandler returns public information about all providers, with optional channel type filter
func getProvidersHandler(c *gin.Context) {
	// Check if messenger service is available
	if messenger.Instance == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrTemporarilyUnavailable.Code,
			ErrorDescription: "Messenger service not available",
		}
		response.RespondWithError(c, response.StatusServiceUnavailable, errorResp)
		return
	}

	// Get optional channel type filter from query parameter
	channelType := c.Query("channel_type")

	var providers []types.Provider
	if channelType != "" {
		// Filter by channel type
		providers = messenger.Instance.GetProviders(channelType)
	} else {
		// Get all providers
		providers = messenger.Instance.GetAllProviders()
	}

	// Convert to public information
	publicProviders := make([]interface{}, 0, len(providers))
	for _, provider := range providers {
		publicProviders = append(publicProviders, provider.GetPublicInfo())
	}

	successResp := gin.H{
		"providers": publicProviders,
		"count":     len(publicProviders),
	}

	if channelType != "" {
		successResp["channel_type"] = channelType
	}

	response.RespondWithSuccess(c, response.StatusOK, successResp)
}

// getChannelsHandler returns all available channels
func getChannelsHandler(c *gin.Context) {
	// Check if messenger service is available
	if messenger.Instance == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrTemporarilyUnavailable.Code,
			ErrorDescription: "Messenger service not available",
		}
		response.RespondWithError(c, response.StatusServiceUnavailable, errorResp)
		return
	}

	// Get all channels
	channels := messenger.Instance.GetChannels()

	successResp := gin.H{
		"channels": channels,
		"count":    len(channels),
	}
	response.RespondWithSuccess(c, response.StatusOK, successResp)
}
