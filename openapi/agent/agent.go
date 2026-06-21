package agent

import (
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/openapi/agent/board"
	"github.com/yaoapp/yao/openapi/agent/inbox"
	"github.com/yaoapp/yao/openapi/agent/robot"
	"github.com/yaoapp/yao/openapi/agent/task"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// Attach attaches the agent (assistant) API handlers to the router with OAuth protection
// This provides OAuth-protected endpoints for assistant management, mirroring the agent assistant API
func Attach(group *gin.RouterGroup, oauth types.OAuth) {

	// Get the Agent instance
	// n := agent.GetAgent()

	// Apply OAuth guard to all routes
	group.Use(oauth.Guard)

	// Assistant CRUD - Standard REST endpoints
	group.GET("/assistants", ListAssistants)            // GET /assistants - List assistants
	group.POST("/assistants", CreateAssistant)          // POST /assistants - Create assistant
	group.GET("/assistants/tags", ListAssistantTags)    // GET /assistants/tags - Get all assistant tags with permission filtering
	group.GET("/assistants/:id", GetAssistant)          // GET /assistants/:id - Get assistant details with permission verification
	group.GET("/assistants/:id/info", GetAssistantInfo) // GET /assistants/:id/messages - Get assistant Information
	group.PUT("/assistants/:id", UpdateAssistant)       // PUT /assistants/:id - Update assistant
	group.DELETE("/assistants/:id", DeleteAssistant)    // DELETE /assistants/:id - Delete assistant

	// Assistant Actions
	// group.POST("/assistants/:id/call", agent.HandleAssistantCall) // POST /assistants/:id/call - Execute assistant API

	// Runner & image queries
	group.GET("/runners", ListRunners)
	group.GET("/images", ListImages)

	// Robot routes - Attach as sub-router
	// Routes: GET/POST /robots, GET/PUT/DELETE /robots/:id, GET /robots/:id/status
	robot.Attach(group.Group("/robots"), oauth)

	// Task routes - Kanban task CRUD + Move
	task.Attach(group.Group("/tasks"), oauth)

	// Board routes - Kanban board/column CRUD + templates
	board.Attach(group.Group("/boards"), oauth)

	// Inbox routes - Mail notifications CRUD
	inbox.Attach(group.Group("/inbox"), oauth)
}
