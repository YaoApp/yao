package agent

import (
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/neo"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// Attach attaches the agent (assistant) API handlers to the router with OAuth protection
// This provides OAuth-protected endpoints for assistant management, mirroring the neo assistant API
func Attach(group *gin.RouterGroup, oauth types.OAuth) {

	// Get the Neo instance
	n := neo.GetNeo()

	// Create agents group with OAuth guard
	agents := group.Group("/agents")
	agents.Use(oauth.Guard)

	// Agent CRUD - Standard REST endpoints
	agents.GET("/", n.HandleAssistantList)         // GET /agents - List agents
	agents.POST("/", n.HandleAssistantSave)        // POST /agents - Create/Update agent
	agents.GET("/tags", n.HandleAssistantTags)     // GET /agents/tags - Get all agent tags
	agents.GET("/:id", n.HandleAssistantDetail)    // GET /agents/:id - Get agent details
	agents.DELETE("/:id", n.HandleAssistantDelete) // DELETE /agents/:id - Delete agent

	// Agent Actions
	agents.POST("/:id/call", n.HandleAssistantCall) // POST /agents/:id/call - Execute agent API
}
