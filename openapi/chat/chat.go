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

	// ==========================================================================
	// Chat Completions (Streaming API)
	// ==========================================================================

	// List Chat Completions
	group.GET("/completions", placeholder)

	// Create Chat Completion
	group.POST("/completions", GinCreateCompletions)

	// Update Chat Completion Metadata
	group.PUT("/completions", GinUpdateCompletions)

	// Get Chat Completion Details
	group.GET("/completions/:completion_id", placeholder)

	// Get Chat Messages (by completion)
	group.GET("/completions/:completion_id/messages", placeholder)

	// Delete Chat Completion
	group.DELETE("/completions/:completion_id", placeholder)

	// Append messages to running completion
	group.POST("/completions/:context_id/append", GinAppendMessages)

	// ==========================================================================
	// Chat Sessions (History Management)
	// ==========================================================================

	// List chat sessions with pagination and filtering
	// Query params: page, pagesize, assistant_id, status, keywords,
	//               start_time, end_time, time_field, order_by, order, group_by
	group.GET("/sessions", ListChats)

	// Get a single chat session by ID
	group.GET("/sessions/:chat_id", GetChat)

	// Update chat session (title, status, metadata)
	group.PUT("/sessions/:chat_id", UpdateChat)

	// Delete chat session
	group.DELETE("/sessions/:chat_id", DeleteChat)

	// Get messages for a chat session
	// Query params: request_id, role, block_id, thread_id, type, limit, offset
	group.GET("/sessions/:chat_id/messages", GetMessages)

	// ==========================================================================
	// Search References (Citation Support)
	// ==========================================================================

	// Get all references for a request
	// Returns all search references for citation support
	group.GET("/references/:request_id", GetReferences)

	// Get a single reference by request ID and index
	// Returns a specific reference for citation click handling
	group.GET("/references/:request_id/:index", GetReference)

}

func placeholder(c *gin.Context) {
	response.RespondWithSuccess(c, response.StatusOK, gin.H{"message": "placeholder"})
}
