package job

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/job"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
)

// ListJobs lists jobs with pagination
func ListJobs(c *gin.Context) {
	// Get authorized information
	authInfo := authorized.GetInfo(c)

	// Parse pagination parameters
	page := 1
	pagesize := 20

	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	if ps := c.Query("pagesize"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 1000 {
			pagesize = parsed
		}
	}

	// Build query parameters from URL query
	param := model.QueryParam{
		Orders: []model.QueryOrder{
			{Column: "created_at", Option: "desc"},
		},
	}

	// Add filters
	var wheres []model.QueryWhere

	// Apply permission-based filtering
	wheres = append(wheres, AuthFilter(c, authInfo)...)

	// Add status filter if provided
	if status := c.Query("status"); status != "" {
		wheres = append(wheres, model.QueryWhere{
			Column: "status",
			Value:  status,
		})
	}

	// Add category filter if provided
	if categoryID := c.Query("category_id"); categoryID != "" {
		wheres = append(wheres, model.QueryWhere{
			Column: "category_id",
			Value:  categoryID,
		})
	}

	// Add keywords filter if provided (search in name and description)
	if keywords := c.Query("keywords"); keywords != "" {
		wheres = append(wheres, model.QueryWhere{
			Wheres: []model.QueryWhere{
				{
					Column: "name",
					Value:  "%" + keywords + "%",
					OP:     "like",
				},
				{
					Column: "description",
					Value:  "%" + keywords + "%",
					OP:     "like",
					Method: "orwhere",
				},
			},
		})
	}

	// Add enabled filter (default to show all for debugging)
	switch enabled := c.Query("enabled"); enabled {
	case "true", "1", "yes", "on":
		wheres = append(wheres, model.QueryWhere{
			Column: "enabled",
			Value:  true,
		})
	case "false", "0", "no", "off":
		wheres = append(wheres, model.QueryWhere{
			Column: "enabled",
			Value:  false,
		})
	default:
		// Default: show all records regardless of enabled status
	}

	// Apply all filters to param
	param.Wheres = wheres

	// Call job.ListJobs function
	result, err := job.ListJobs(param, page, pagesize)
	if err != nil {
		log.Error("Failed to list jobs: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetJob gets a specific job by ID
func GetJob(c *gin.Context) {
	// Get authorized information
	authInfo := authorized.GetInfo(c)

	jobID := c.Param("jobID")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "job_id is required"})
		return
	}

	// Call job.GetJob function
	jobInstance, err := job.GetJob(jobID)
	if err != nil {
		log.Error("Failed to get job %s: %v", jobID, err)
		if err.Error() == "job not found: "+jobID {
			c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// Check if user has access to this job
	if !HasJobAccess(c, authInfo, jobInstance) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	c.JSON(http.StatusOK, jobInstance)
}

// StopJob stops a running job
func StopJob(c *gin.Context) {
	// Get authorized information
	authInfo := authorized.GetInfo(c)

	jobID := c.Param("jobID")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "job_id is required"})
		return
	}

	// Get the job first
	jobInstance, err := job.GetJob(jobID)
	if err != nil {
		log.Error("Failed to get job %s: %v", jobID, err)
		if err.Error() == "job not found: "+jobID {
			c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// Check if user has access to this job
	if !HasJobAccess(c, authInfo, jobInstance) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	// Stop the job
	err = jobInstance.Stop()
	if err != nil {
		log.Error("Failed to stop job %s: %v", jobID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Job stopped successfully",
		"job_id":  jobID,
		"status":  "stopped",
	})
}

// GetJobProgress gets job progress information
func GetJobProgress(c *gin.Context) {
	// Get authorized information
	authInfo := authorized.GetInfo(c)

	jobID := c.Param("jobID")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "job_id is required"})
		return
	}

	// Get the job first
	jobInstance, err := job.GetJob(jobID)
	if err != nil {
		log.Error("Failed to get job %s: %v", jobID, err)
		if err.Error() == "job not found: "+jobID {
			c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// Check if user has access to this job
	if !HasJobAccess(c, authInfo, jobInstance) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	// Get executions for progress calculation
	executions, err := job.GetExecutions(jobID)
	if err != nil {
		log.Error("Failed to get executions for job %s: %v", jobID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Calculate progress
	totalExecutions := len(executions)
	completedCount := 0
	runningCount := 0
	failedCount := 0
	totalProgress := 0

	for _, execution := range executions {
		totalProgress += execution.Progress
		switch execution.Status {
		case "completed":
			completedCount++
		case "running":
			runningCount++
		case "failed":
			failedCount++
		}
	}

	averageProgress := 0
	if totalExecutions > 0 {
		averageProgress = totalProgress / totalExecutions
	}

	response := gin.H{
		"job_id":           jobID,
		"status":           jobInstance.Status,
		"progress":         averageProgress,
		"total_executions": totalExecutions,
		"completed_count":  completedCount,
		"running_count":    runningCount,
		"failed_count":     failedCount,
		"last_run_at":      jobInstance.LastRunAt,
		"next_run_at":      jobInstance.NextRunAt,
	}

	c.JSON(http.StatusOK, response)
}

// GetStats gets overall job statistics
func GetStats(c *gin.Context) {
	// Get authorized information
	authInfo := authorized.GetInfo(c)

	// Build base auth filter
	baseAuthFilter := AuthFilter(c, authInfo)

	// Count total jobs
	totalJobs, err := job.CountJobs(model.QueryParam{
		Wheres: baseAuthFilter,
	})
	if err != nil {
		log.Error("Failed to count total jobs: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Count running jobs
	runningJobs, err := job.CountJobs(model.QueryParam{
		Wheres: append(baseAuthFilter, model.QueryWhere{
			Column: "status", Value: "running",
		}),
	})
	if err != nil {
		log.Error("Failed to count running jobs: %v", err)
		runningJobs = 0
	}

	// Count completed jobs
	completedJobs, err := job.CountJobs(model.QueryParam{
		Wheres: append(baseAuthFilter, model.QueryWhere{
			Column: "status", Value: "completed",
		}),
	})
	if err != nil {
		log.Error("Failed to count completed jobs: %v", err)
		completedJobs = 0
	}

	// Count failed jobs
	failedJobs, err := job.CountJobs(model.QueryParam{
		Wheres: append(baseAuthFilter, model.QueryWhere{
			Column: "status", Value: "failed",
		}),
	})
	if err != nil {
		log.Error("Failed to count failed jobs: %v", err)
		failedJobs = 0
	}

	// Get categories for category stats
	categories, err := job.GetCategories(model.QueryParam{})
	if err != nil {
		log.Error("Failed to get categories: %v", err)
		categories = []*job.Category{}
	}

	categoryStats := make(map[string]int)
	for _, category := range categories {
		count, err := job.CountJobs(model.QueryParam{
			Wheres: append(baseAuthFilter, model.QueryWhere{
				Column: "category_id", Value: category.CategoryID,
			}),
		})
		if err != nil {
			count = 0
		}
		categoryStats[category.Name] = count
	}

	response := gin.H{
		"total_jobs":       totalJobs,
		"running_jobs":     runningJobs,
		"completed_jobs":   completedJobs,
		"failed_jobs":      failedJobs,
		"category_stats":   categoryStats,
		"total_categories": len(categories),
	}

	c.JSON(http.StatusOK, response)
}

// ========================
// Process Handlers
// ========================

// ProcessListJobs process handler for listing jobs
func ProcessListJobs(process *process.Process) interface{} {
	// TODO: Implement process handler for listing jobs
	args := process.Args
	log.Info("ProcessListJobs called with args: %v", args)

	// Default pagination values
	page := 1
	pagesize := 20

	// Parse arguments if provided
	if len(args) > 0 {
		if p, ok := args[0].(int); ok {
			page = p
		}
	}
	if len(args) > 1 {
		if ps, ok := args[1].(int); ok {
			pagesize = ps
		}
	}

	// Build query parameters
	param := model.QueryParam{}
	if len(args) > 2 {
		if queryParam, ok := args[2].(model.QueryParam); ok {
			param = queryParam
		}
	}

	// Call job.ListJobs function
	result, err := job.ListJobs(param, page, pagesize)
	if err != nil {
		log.Error("Failed to list jobs: %v", err)
		return map[string]interface{}{"error": err.Error()}
	}

	return result
}

// ProcessGetJob process handler for getting a job
func ProcessGetJob(process *process.Process) interface{} {
	// TODO: Implement process handler for getting a job
	args := process.Args
	if len(args) == 0 {
		return map[string]interface{}{"error": "job_id is required"}
	}

	jobID, ok := args[0].(string)
	if !ok {
		return map[string]interface{}{"error": "job_id must be a string"}
	}

	log.Info("ProcessGetJob called for job: %s", jobID)

	// Call job.GetJob function
	result, err := job.GetJob(jobID)
	if err != nil {
		log.Error("Failed to get job %s: %v", jobID, err)
		return map[string]interface{}{"error": err.Error()}
	}

	return result
}

// ProcessCountJobs process handler for counting jobs
func ProcessCountJobs(process *process.Process) interface{} {
	// TODO: Implement process handler for counting jobs
	args := process.Args
	log.Info("ProcessCountJobs called with args: %v", args)

	// Build query parameters
	param := model.QueryParam{}
	if len(args) > 0 {
		if queryParam, ok := args[0].(model.QueryParam); ok {
			param = queryParam
		}
	}

	// Call job.CountJobs function
	count, err := job.CountJobs(param)
	if err != nil {
		log.Error("Failed to count jobs: %v", err)
		return map[string]interface{}{"error": err.Error()}
	}

	return map[string]interface{}{"count": count}
}

// ProcessStopJob process handler for stopping a job
func ProcessStopJob(process *process.Process) interface{} {
	// TODO: Implement process handler for stopping a job
	args := process.Args
	if len(args) == 0 {
		return map[string]interface{}{"error": "job_id is required"}
	}

	jobID, ok := args[0].(string)
	if !ok {
		return map[string]interface{}{"error": "job_id must be a string"}
	}

	log.Info("ProcessStopJob called for job: %s", jobID)

	// Get the job first
	jobInstance, err := job.GetJob(jobID)
	if err != nil {
		log.Error("Failed to get job %s: %v", jobID, err)
		return map[string]interface{}{"error": err.Error()}
	}

	// Stop the job
	err = jobInstance.Stop()
	if err != nil {
		log.Error("Failed to stop job %s: %v", jobID, err)
		return map[string]interface{}{"error": err.Error()}
	}

	return map[string]interface{}{"message": "Job stopped successfully", "job_id": jobID}
}
