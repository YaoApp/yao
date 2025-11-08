package agent

import (
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/agent"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// Attach attaches the agent (assistant) API handlers to the router with OAuth protection
// This provides OAuth-protected endpoints for assistant management, mirroring the agent assistant API
func Attach(group *gin.RouterGroup, oauth types.OAuth) {

	// Get the Agent instance
	n := agent.GetAgent()

	// Create assistants group with OAuth guard
	assistants := group.Group("/assistants")
	assistants.Use(oauth.Guard)

	// Assistant CRUD - Standard REST endpoints
	assistants.GET("/", ListAssistants)                // GET /assistants - List assistants
	assistants.POST("/", n.HandleAssistantSave)        // POST /assistants - Create/Update assistant
	assistants.GET("/tags", n.HandleAssistantTags)     // GET /assistants/tags - Get all assistant tags
	assistants.GET("/:id", n.HandleAssistantDetail)    // GET /assistants/:id - Get assistant details
	assistants.DELETE("/:id", n.HandleAssistantDelete) // DELETE /assistants/:id - Delete assistant

	// Assistant Actions
	assistants.POST("/:id/call", n.HandleAssistantCall) // POST /assistants/:id/call - Execute assistant API
}
