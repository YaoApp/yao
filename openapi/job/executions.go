package job

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/job"
)

// ListExecutions lists executions for a specific job
func ListExecutions(c *gin.Context) {
	jobID := c.Param("jobID")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "job_id is required"})
		return
	}

	// Get executions for the job
	executions, err := job.GetExecutions(jobID)
	if err != nil {
		log.Error("Failed to list executions for job %s: %v", jobID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Add optional status filter
	if status := c.Query("status"); status != "" {
		filtered := make([]*job.Execution, 0)
		for _, execution := range executions {
			if execution.Status == status {
				filtered = append(filtered, execution)
			}
		}
		executions = filtered
	}

	// Simple pagination (client-side)
	page := 1
	pagesize := 50
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

	total := len(executions)
	start := (page - 1) * pagesize
	end := start + pagesize

	if start >= total {
		executions = []*job.Execution{}
	} else {
		if end > total {
			end = total
		}
		executions = executions[start:end]
	}

	response := gin.H{
		"data":     executions,
		"page":     page,
		"pagesize": pagesize,
		"total":    total,
		"job_id":   jobID,
	}

	c.JSON(http.StatusOK, response)
}

// GetExecution gets a specific execution by ID
func GetExecution(c *gin.Context) {
	executionID := c.Param("executionID")
	if executionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "execution_id is required"})
		return
	}

	// Get the execution
	execution, err := job.GetExecution(executionID, model.QueryParam{})
	if err != nil {
		log.Error("Failed to get execution %s: %v", executionID, err)
		if err.Error() == "execution not found: "+executionID {
			c.JSON(http.StatusNotFound, gin.H{"error": "Execution not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, execution)
}

// StopExecution stops a running execution
func StopExecution(c *gin.Context) {
	executionID := c.Param("executionID")
	if executionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "execution_id is required"})
		return
	}

	// Get the execution first to find the job
	execution, err := job.GetExecution(executionID, model.QueryParam{})
	if err != nil {
		log.Error("Failed to get execution %s: %v", executionID, err)
		if err.Error() == "execution not found: "+executionID {
			c.JSON(http.StatusNotFound, gin.H{"error": "Execution not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// Get the job to stop the specific execution
	jobInstance, err := job.GetJob(execution.JobID)
	if err != nil {
		log.Error("Failed to get job %s for execution %s: %v", execution.JobID, executionID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// For now, we stop the entire job since individual execution stopping
	// would require more complex implementation in the job package
	err = jobInstance.Stop()
	if err != nil {
		log.Error("Failed to stop job %s (execution %s): %v", execution.JobID, executionID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Execution stopped successfully (job stopped)",
		"execution_id": executionID,
		"job_id":       execution.JobID,
		"status":       "stopped",
	})
}

// GetExecutionProgress gets execution progress information
func GetExecutionProgress(c *gin.Context) {
	executionID := c.Param("executionID")
	if executionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "execution_id is required"})
		return
	}

	// Get the execution
	execution, err := job.GetExecution(executionID, model.QueryParam{})
	if err != nil {
		log.Error("Failed to get execution %s: %v", executionID, err)
		if err.Error() == "execution not found: "+executionID {
			c.JSON(http.StatusNotFound, gin.H{"error": "Execution not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	response := gin.H{
		"execution_id":  executionID,
		"job_id":        execution.JobID,
		"status":        execution.Status,
		"progress":      execution.Progress,
		"started_at":    execution.StartedAt,
		"ended_at":      execution.EndedAt,
		"duration":      execution.Duration,
		"worker_id":     execution.WorkerID,
		"process_id":    execution.ProcessID,
		"retry_attempt": execution.RetryAttempt,
	}

	// Add error info if available
	if execution.ErrorInfo != nil {
		response["error_info"] = execution.ErrorInfo
	}

	// Add result if available
	if execution.Result != nil {
		response["result"] = execution.Result
	}

	c.JSON(http.StatusOK, response)
}

// ========================
// Process Handlers
// ========================

// ProcessListExecutions process handler for listing executions
func ProcessListExecutions(process *process.Process) interface{} {
	// TODO: Implement process handler for listing executions
	args := process.Args
	if len(args) == 0 {
		return map[string]interface{}{"error": "job_id is required"}
	}

	jobID, ok := args[0].(string)
	if !ok {
		return map[string]interface{}{"error": "job_id must be a string"}
	}

	log.Info("ProcessListExecutions called for job: %s", jobID)

	// Call job.GetExecutions function
	executions, err := job.GetExecutions(jobID)
	if err != nil {
		log.Error("Failed to list executions for job %s: %v", jobID, err)
		return map[string]interface{}{"error": err.Error()}
	}

	return map[string]interface{}{
		"executions": executions,
		"count":      len(executions),
	}
}

// ProcessGetExecution process handler for getting an execution
func ProcessGetExecution(process *process.Process) interface{} {
	// TODO: Implement process handler for getting an execution
	args := process.Args
	if len(args) == 0 {
		return map[string]interface{}{"error": "execution_id is required"}
	}

	executionID, ok := args[0].(string)
	if !ok {
		return map[string]interface{}{"error": "execution_id must be a string"}
	}

	log.Info("ProcessGetExecution called for execution: %s", executionID)

	// Build query parameters
	param := model.QueryParam{}
	if len(args) > 1 {
		if queryParam, ok := args[1].(model.QueryParam); ok {
			param = queryParam
		}
	}

	// Call job.GetExecution function
	execution, err := job.GetExecution(executionID, param)
	if err != nil {
		log.Error("Failed to get execution %s: %v", executionID, err)
		return map[string]interface{}{"error": err.Error()}
	}

	return execution
}

// ProcessCountExecutions process handler for counting executions
func ProcessCountExecutions(process *process.Process) interface{} {
	// TODO: Implement process handler for counting executions
	args := process.Args
	if len(args) == 0 {
		return map[string]interface{}{"error": "job_id is required"}
	}

	jobID, ok := args[0].(string)
	if !ok {
		return map[string]interface{}{"error": "job_id must be a string"}
	}

	log.Info("ProcessCountExecutions called for job: %s", jobID)

	// Build query parameters
	param := model.QueryParam{}
	if len(args) > 1 {
		if queryParam, ok := args[1].(model.QueryParam); ok {
			param = queryParam
		}
	}

	// Call job.CountExecutions function
	count, err := job.CountExecutions(jobID, param)
	if err != nil {
		log.Error("Failed to count executions for job %s: %v", jobID, err)
		return map[string]interface{}{"error": err.Error()}
	}

	return map[string]interface{}{"count": count}
}

// ProcessStopExecution process handler for stopping an execution
func ProcessStopExecution(process *process.Process) interface{} {
	// TODO: Implement process handler for stopping an execution
	args := process.Args
	if len(args) == 0 {
		return map[string]interface{}{"error": "execution_id is required"}
	}

	executionID, ok := args[0].(string)
	if !ok {
		return map[string]interface{}{"error": "execution_id must be a string"}
	}

	log.Info("ProcessStopExecution called for execution: %s", executionID)

	// Get the execution first to find the job
	execution, err := job.GetExecution(executionID, model.QueryParam{})
	if err != nil {
		log.Error("Failed to get execution %s: %v", executionID, err)
		return map[string]interface{}{"error": err.Error()}
	}

	// Get the job to stop the specific execution
	jobInstance, err := job.GetJob(execution.JobID)
	if err != nil {
		log.Error("Failed to get job %s for execution %s: %v", execution.JobID, executionID, err)
		return map[string]interface{}{"error": err.Error()}
	}

	// For now, we stop the entire job since individual execution stopping
	// would require more complex implementation in the job package
	err = jobInstance.Stop()
	if err != nil {
		log.Error("Failed to stop job %s (execution %s): %v", execution.JobID, executionID, err)
		return map[string]interface{}{"error": err.Error()}
	}

	return map[string]interface{}{
		"message":      "Execution stopped successfully",
		"execution_id": executionID,
		"job_id":       execution.JobID,
	}
}
