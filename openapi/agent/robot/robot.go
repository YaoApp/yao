package robot

import (
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

// Attach attaches the robot API handlers to the router with OAuth protection
// This provides OAuth-protected endpoints for robot management
// Base path: /v1/agent/robots
func Attach(group *gin.RouterGroup, oauth types.OAuth) {

	// Apply OAuth guard to all routes
	group.Use(oauth.Guard)

	// Robot CRUD - Standard REST endpoints
	group.GET("", ListRobots)         // GET /robots - List robots with pagination and filtering
	group.POST("", CreateRobot)       // POST /robots - Create a new robot
	group.GET("/:id", GetRobot)       // GET /robots/:id - Get robot details
	group.PUT("/:id", UpdateRobot)    // PUT /robots/:id - Update robot
	group.DELETE("/:id", DeleteRobot) // DELETE /robots/:id - Delete robot

	// Robot Status
	group.GET("/:id/status", GetRobotStatus) // GET /robots/:id/status - Get robot runtime status
}
