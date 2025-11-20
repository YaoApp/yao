package trace

import (
	"github.com/gin-gonic/gin"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
)

// Attach attaches the trace API handlers to the router with OAuth protection
func Attach(group *gin.RouterGroup, oauth oauthtypes.OAuth) {
	// Apply OAuth guard to all routes
	group.Use(oauth.Guard)

	// Trace API endpoints
	group.GET("/traces/:traceID/events", GetEvents)         // GET /traces/:traceID/events?stream=true - Get trace events (support SSE streaming)
	group.GET("/traces/:traceID/info", GetInfo)             // GET /traces/:traceID/info - Get trace info
	group.GET("/traces/:traceID/nodes", GetNodes)           // GET /traces/:traceID/nodes - Get all nodes
	group.GET("/traces/:traceID/nodes/:nodeID", GetNode)    // GET /traces/:traceID/nodes/:nodeID - Get single node
	group.GET("/traces/:traceID/logs", GetLogs)             // GET /traces/:traceID/logs - Get all logs
	group.GET("/traces/:traceID/logs/:nodeID", GetLogs)     // GET /traces/:traceID/logs/:nodeID - Get logs for specific node
	group.GET("/traces/:traceID/spaces", GetSpaces)         // GET /traces/:traceID/spaces - Get all spaces (metadata only)
	group.GET("/traces/:traceID/spaces/:spaceID", GetSpace) // GET /traces/:traceID/spaces/:spaceID - Get single space with all data
}
