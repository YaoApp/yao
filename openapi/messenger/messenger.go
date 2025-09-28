package messenger

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/messenger"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// Attach attaches the messenger webhook handlers to the router
func Attach(group *gin.RouterGroup, oauth types.OAuth) {

	// Webhook endpoint with provider parameter - public interface
	group.GET("/webhook/:provider", webhookHandler)
	group.POST("/webhook/:provider", webhookHandler)
}

// webhookHandler is the handler for webhook endpoint
func webhookHandler(c *gin.Context) {
	// Get provider name from URL parameter
	providerName := c.Param("provider")
	if providerName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "provider parameter is required",
		})
		return
	}

	// Check if messenger service is available
	if messenger.Instance == nil {
		log.Warn("[OpenAPI Messenger] Messenger service not initialized")
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "messenger service not available",
		})
		return
	}

	// Directly pass gin.Context to messenger service for processing
	err := messenger.Instance.TriggerWebhook(providerName, c)
	if err != nil {
		log.Error("[OpenAPI Messenger] Failed to process webhook for provider %s: %v", providerName, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to process webhook",
			"details": err.Error(),
		})
		return
	}

	// Return success response
	c.JSON(http.StatusOK, gin.H{
		"status":   "received",
		"message":  "webhook processed successfully",
		"provider": providerName,
	})
}
