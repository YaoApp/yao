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

// ListLogs lists logs for a specific job
func ListLogs(c *gin.Context) {
	jobID := c.Param("jobID")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "job_id is required"})
		return
	}

	// Parse pagination parameters
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

	// Build query parameters
	param := model.QueryParam{
		Orders: []model.QueryOrder{
			{Column: "timestamp", Option: "desc"}, // 日志按时间戳倒序
		},
	}

	// Add level filter if provided
	if level := c.Query("level"); level != "" {
		param.Wheres = append(param.Wheres, model.QueryWhere{
			Column: "level",
			Value:  level,
		})
	}

	// Add execution_id filter if provided
	if executionID := c.Query("execution_id"); executionID != "" {
		param.Wheres = append(param.Wheres, model.QueryWhere{
			Column: "execution_id",
			Value:  executionID,
		})
	}

	// Call job.ListLogs function
	result, err := job.ListLogs(jobID, param, page, pagesize)
	if err != nil {
		log.Error("Failed to list logs for job %s: %v", jobID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ListExecutionLogs lists logs for a specific execution
func ListExecutionLogs(c *gin.Context) {
	executionID := c.Param("executionID")
	if executionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "execution_id is required"})
		return
	}

	// First get the execution to find the job_id
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

	// Parse pagination parameters
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

	// Build query parameters with execution_id filter
	param := model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "execution_id", Value: executionID},
		},
		Orders: []model.QueryOrder{
			{Column: "timestamp", Option: "desc"}, // 日志按时间戳倒序
		},
	}

	// Add level filter if provided
	if level := c.Query("level"); level != "" {
		param.Wheres = append(param.Wheres, model.QueryWhere{
			Column: "level",
			Value:  level,
		})
	}

	// Call job.ListLogs function with job_id from execution
	result, err := job.ListLogs(execution.JobID, param, page, pagesize)
	if err != nil {
		log.Error("Failed to list logs for execution %s: %v", executionID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ========================
// Process Handlers
// ========================

// ProcessListLogs process handler for listing logs
func ProcessListLogs(process *process.Process) interface{} {
	// TODO: Implement process handler for listing logs
	args := process.Args
	if len(args) == 0 {
		return map[string]interface{}{"error": "job_id is required"}
	}

	jobID, ok := args[0].(string)
	if !ok {
		return map[string]interface{}{"error": "job_id must be a string"}
	}

	// Default pagination values
	page := 1
	pagesize := 50

	// Parse arguments if provided
	if len(args) > 1 {
		if p, ok := args[1].(int); ok && p > 0 {
			page = p
		}
	}
	if len(args) > 2 {
		if ps, ok := args[2].(int); ok && ps > 0 {
			pagesize = ps
		}
	}

	// Build query parameters
	param := model.QueryParam{}
	if len(args) > 3 {
		if queryParam, ok := args[3].(model.QueryParam); ok {
			param = queryParam
		}
	}

	log.Info("ProcessListLogs called for job: %s (page: %d, pagesize: %d)", jobID, page, pagesize)

	// Call job.ListLogs function
	result, err := job.ListLogs(jobID, param, page, pagesize)
	if err != nil {
		log.Error("Failed to list logs for job %s: %v", jobID, err)
		return map[string]interface{}{"error": err.Error()}
	}

	return result
}
