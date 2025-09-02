package job

import (
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/openapi/oauth/types"
)

func init() {
	// Register job process handlers
	process.RegisterGroup("job", map[string]process.Handler{
		"jobs.list":        ProcessListJobs,
		"jobs.get":         ProcessGetJob,
		"jobs.count":       ProcessCountJobs,
		"jobs.stop":        ProcessStopJob,
		"executions.list":  ProcessListExecutions,
		"executions.get":   ProcessGetExecution,
		"executions.count": ProcessCountExecutions,
		"executions.stop":  ProcessStopExecution,
		"logs.list":        ProcessListLogs,
		"categories.list":  ProcessListCategories,
		"categories.get":   ProcessGetCategory,
		"categories.count": ProcessCountCategories,
	})
}

// Attach attaches the Job API to the router
func Attach(group *gin.RouterGroup, oauth types.OAuth) {
	// Protect all endpoints with OAuth
	group.Use(oauth.Guard)

	// Job Management (Read-only operations)
	group.GET("/jobs", ListJobs)
	group.GET("/jobs/:jobID", GetJob)
	group.POST("/jobs/:jobID/stop", StopJob)

	// Execution Management
	group.GET("/jobs/:jobID/executions", ListExecutions)
	group.GET("/executions/:executionID", GetExecution)
	group.POST("/executions/:executionID/stop", StopExecution)

	// Log Management
	group.GET("/jobs/:jobID/logs", ListLogs)
	group.GET("/executions/:executionID/logs", ListExecutionLogs)

	// Category Management (Read-only)
	group.GET("/categories", ListCategories)
	group.GET("/categories/:categoryID", GetCategory)

	// Progress and Status
	group.GET("/jobs/:jobID/progress", GetJobProgress)
	group.GET("/executions/:executionID/progress", GetExecutionProgress)

	// Statistics
	group.GET("/stats", GetStats)
}
