package chat

import (
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
)

// Attach attaches the agent handlers to the router
func Attach(group *gin.RouterGroup, oauth types.OAuth) {

	// Protect all endpoints with OAuth
	group.Use(oauth.Guard)

	// List Chat Completions
	group.GET("/completions", placeholder)

	// Create Chat Completion
	group.POST("/completions", GinCreateCompletions)

	// Update Chat Completion Metadata
	group.PUT("/completions", GinUpdateCompletions)

	// Get Chat Completion Details
	group.GET("/completions/:completion_id", placeholder)

	// Get Chat Messages
	group.GET("/completions/:completion_id/messages", placeholder)

	// Delete Chat Completion
	group.DELETE("/completions/:completion_id", placeholder)

	// Append messages to running completion
	group.POST("/completions/:context_id/append", GinAppendMessages)

}

func placeholder(c *gin.Context) {
	response.RespondWithSuccess(c, response.StatusOK, gin.H{"message": "placeholder"})
}
