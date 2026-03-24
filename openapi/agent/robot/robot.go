package robot

import (
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/openapi/oauth/types"

	_ "github.com/yaoapp/yao/agent/robot" // register robot.* process handlers
)

// Attach attaches the robot API handlers to the router with OAuth protection
// This provides OAuth-protected endpoints for robot management
// Base path: /v1/agent/robots
func Attach(group *gin.RouterGroup, oauth types.OAuth) {

	// Apply OAuth guard to all routes
	group.Use(oauth.Guard)

	// Robot CRUD - Standard REST endpoints
	group.GET("", ListAllRobots) // GET /robots - List robots with pagination and filtering
	group.POST("", CreateRobot)  // POST /robots - Create a new robot

	// Activities - Cross-robot activity feed for team (must be before /:id to avoid conflict)
	group.GET("/activities", ListActivities) // GET /robots/activities - List team activities

	// Integration credential verification (must be before /:id to avoid conflict)
	group.POST("/integrations/verify", VerifyIntegration) // POST /robots/integrations/verify - Verify integration credentials

	// WeChat iLink Bot QR code login
	group.POST("/integrations/weixin/qrcode", CreateWeixinQRCode)           // POST /robots/integrations/weixin/qrcode - Create QR session
	group.GET("/integrations/weixin/qrcode/:session_key", PollWeixinQRCode) // GET  /robots/integrations/weixin/qrcode/:session_key - Poll QR status

	group.GET("/:id", GetRobot)       // GET /robots/:id - Get robot details
	group.PUT("/:id", UpdateRobot)    // PUT /robots/:id - Update robot
	group.DELETE("/:id", DeleteRobot) // DELETE /robots/:id - Delete robot

	// Robot Status
	group.GET("/:id/status", GetRobotStatus) // GET /robots/:id/status - Get robot runtime status

	// Execution Management
	group.GET("/:id/executions", ListExecutions)                   // GET /robots/:id/executions - List robot executions
	group.GET("/:id/executions/:exec_id", GetExecution)            // GET /robots/:id/executions/:exec_id - Get execution details
	group.POST("/:id/executions/:exec_id/pause", PauseExecution)   // POST /robots/:id/executions/:exec_id/pause - Pause execution
	group.POST("/:id/executions/:exec_id/resume", ResumeExecution) // POST /robots/:id/executions/:exec_id/resume - Resume execution
	group.POST("/:id/executions/:exec_id/cancel", CancelExecution) // POST /robots/:id/executions/:exec_id/cancel - Cancel execution

	// Results (Deliveries) - Completed executions with delivery content
	group.GET("/:id/results", ListResults)          // GET /robots/:id/results - List robot results
	group.GET("/:id/results/:result_id", GetResult) // GET /robots/:id/results/:result_id - Get result details

	// Trigger & Intervene
	group.POST("/:id/trigger", TriggerRobot)     // POST /robots/:id/trigger - Trigger robot execution
	group.POST("/:id/intervene", InterveneRobot) // POST /robots/:id/intervene - Human intervention

	// Host Agent Chat (mirror of standard Chat Completion API)
	group.GET("/:id/host", RobotHostID)                                    // GET /robots/:id/host - Get host assistant ID
	group.POST("/:id/completions", RobotCompletions)                       // POST /robots/:id/completions - Chat with host agent
	group.POST("/:id/completions/:context_id/append", RobotAppendMessages) // POST /robots/:id/completions/:context_id/append - Append messages

	// Execute - Direct execution trigger (called by CUI after Host confirms goals)
	group.POST("/:id/execute", ExecuteRobot) // POST /robots/:id/execute - Execute with confirmed goals

	// V2: Unified Interact API (suspend-resume, human-in-the-loop)
	group.POST("/:id/interact", InteractRobot)                               // POST /robots/:id/interact - Unified interaction
	group.POST("/:id/executions/:exec_id/tasks/:task_id/reply", ReplyToTask) // POST /robots/:id/executions/:exec_id/tasks/:task_id/reply - Reply to waiting task
	group.POST("/:id/executions/:exec_id/confirm", ConfirmExecution)         // POST /robots/:id/executions/:exec_id/confirm - Confirm execution
}
