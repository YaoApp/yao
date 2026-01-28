package robot

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/kun/log"
	robotapi "github.com/yaoapp/yao/agent/robot/api"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/response"
)

// ==================== Execution Handlers ====================
// Permission Note: Execution permissions are inherited from the parent robot.
// Check robot's __yao_team_id and __yao_created_by for access control.

// ListExecutions lists executions for a robot
// GET /v1/agent/robots/:id/executions
func ListExecutions(c *gin.Context) {
	// Get authorized information
	authInfo := authorized.GetInfo(c)

	// Get robot ID from URL parameter
	robotID := c.Param("id")
	if robotID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "robot id is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Create robot context
	ctx := &robottypes.Context{}

	// Check robot permission first (executions inherit robot permission)
	robotResp, err := robotapi.GetRobotResponse(ctx, robotID)
	if err != nil {
		if errors.Is(err, robottypes.ErrRobotNotFound) {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: "Robot not found: " + robotID,
			}
			response.RespondWithError(c, response.StatusNotFound, errorResp)
			return
		}
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to get robot: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Check read permission on robot
	if !CanRead(c, authInfo, robotResp.YaoTeamID, robotResp.YaoCreatedBy) {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrAccessDenied.Code,
			ErrorDescription: "Forbidden: No permission to access this robot's executions",
		}
		response.RespondWithError(c, response.StatusForbidden, errorResp)
		return
	}

	// Parse query parameters
	var filter ExecutionFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid query parameters: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Apply defaults
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}

	// Build API query
	query := &robotapi.ExecutionQuery{
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}
	if filter.Status != "" {
		query.Status = robottypes.ExecStatus(filter.Status)
	}
	if filter.TriggerType != "" {
		query.Trigger = robottypes.TriggerType(filter.TriggerType)
	}

	// Call API layer
	result, err := robotapi.ListExecutions(ctx, robotID, query)
	if err != nil {
		log.Error("Failed to list executions for robot %s: %v", robotID, err)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to list executions: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Convert to response (brief format for list)
	data := make([]*ExecutionResponse, 0, len(result.Data))
	for _, exec := range result.Data {
		data = append(data, NewExecutionResponseBrief(exec))
	}

	resp := &ExecutionListResponse{
		Data:     data,
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	}
	response.RespondWithSuccess(c, response.StatusOK, resp)
}

// GetExecution gets a single execution by ID
// GET /v1/agent/robots/:id/executions/:exec_id
func GetExecution(c *gin.Context) {
	// Get authorized information
	authInfo := authorized.GetInfo(c)

	// Get robot ID and execution ID from URL parameters
	robotID := c.Param("id")
	execID := c.Param("exec_id")

	if robotID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "robot id is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}
	if execID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "execution id is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Create robot context
	ctx := &robottypes.Context{}

	// Check robot permission first
	robotResp, err := robotapi.GetRobotResponse(ctx, robotID)
	if err != nil {
		if errors.Is(err, robottypes.ErrRobotNotFound) {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: "Robot not found: " + robotID,
			}
			response.RespondWithError(c, response.StatusNotFound, errorResp)
			return
		}
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to get robot: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Check read permission on robot
	if !CanRead(c, authInfo, robotResp.YaoTeamID, robotResp.YaoCreatedBy) {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrAccessDenied.Code,
			ErrorDescription: "Forbidden: No permission to access this robot's executions",
		}
		response.RespondWithError(c, response.StatusForbidden, errorResp)
		return
	}

	// Get execution
	exec, err := robotapi.GetExecution(ctx, execID)
	if err != nil {
		log.Error("Failed to get execution %s: %v", execID, err)

		if err.Error() == "execution not found: "+execID {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: "Execution not found: " + execID,
			}
			response.RespondWithError(c, response.StatusNotFound, errorResp)
			return
		}

		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to get execution: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Verify execution belongs to this robot
	if exec.MemberID != robotID {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Execution does not belong to this robot",
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return
	}

	// Convert to response (full format for detail)
	resp := NewExecutionResponseFromExecution(exec)
	response.RespondWithSuccess(c, response.StatusOK, resp)
}

// PauseExecution pauses a running execution
// POST /v1/agent/robots/:id/executions/:exec_id/pause
func PauseExecution(c *gin.Context) {
	handleExecutionControl(c, "pause")
}

// ResumeExecution resumes a paused execution
// POST /v1/agent/robots/:id/executions/:exec_id/resume
func ResumeExecution(c *gin.Context) {
	handleExecutionControl(c, "resume")
}

// CancelExecution cancels/stops an execution
// POST /v1/agent/robots/:id/executions/:exec_id/cancel
func CancelExecution(c *gin.Context) {
	handleExecutionControl(c, "cancel")
}

// handleExecutionControl handles pause/resume/cancel operations
func handleExecutionControl(c *gin.Context, action string) {
	// Get authorized information
	authInfo := authorized.GetInfo(c)

	// Get robot ID and execution ID from URL parameters
	robotID := c.Param("id")
	execID := c.Param("exec_id")

	if robotID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "robot id is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}
	if execID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "execution id is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Create robot context
	ctx := &robottypes.Context{}

	// Check robot permission first
	robotResp, err := robotapi.GetRobotResponse(ctx, robotID)
	if err != nil {
		if errors.Is(err, robottypes.ErrRobotNotFound) {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: "Robot not found: " + robotID,
			}
			response.RespondWithError(c, response.StatusNotFound, errorResp)
			return
		}
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to get robot: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Check write permission on robot (control operations require write)
	if !CanWrite(c, authInfo, robotResp.YaoTeamID, robotResp.YaoCreatedBy) {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrAccessDenied.Code,
			ErrorDescription: "Forbidden: No permission to control this robot's executions",
		}
		response.RespondWithError(c, response.StatusForbidden, errorResp)
		return
	}

	// Execute the control action
	var controlErr error
	switch action {
	case "pause":
		controlErr = robotapi.PauseExecution(ctx, execID)
	case "resume":
		controlErr = robotapi.ResumeExecution(ctx, execID)
	case "cancel":
		controlErr = robotapi.StopExecution(ctx, execID)
	default:
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid action: " + action,
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	if controlErr != nil {
		log.Error("Failed to %s execution %s: %v", action, execID, controlErr)

		// Check for common errors
		errMsg := controlErr.Error()
		if errMsg == "execution_id is required" || strings.Contains(errMsg, "execution not found") {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: "Execution not found or not running: " + execID,
			}
			response.RespondWithError(c, response.StatusNotFound, errorResp)
			return
		}

		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to " + action + " execution: " + controlErr.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Return success
	resp := &ExecutionControlResponse{
		ExecutionID: execID,
		Action:      action + "d", // paused, resumed, cancelled
		Success:     true,
		Message:     "Execution " + action + "d successfully",
	}
	response.RespondWithSuccess(c, response.StatusOK, resp)
}
