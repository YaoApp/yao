package robot

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/kun/log"
	agentcontext "github.com/yaoapp/yao/agent/context"
	robotapi "github.com/yaoapp/yao/agent/robot/api"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/response"
)

// ==================== Trigger Handlers ====================
// Permission Note: Same as execution - check robot's permission.

// TriggerRobot triggers a robot execution
// POST /v1/agent/robots/:id/trigger
func TriggerRobot(c *gin.Context) {
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

	// Parse request body
	var req TriggerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid request body: " + err.Error(),
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

	// Check write permission on robot (trigger requires write permission)
	if !CanWrite(c, authInfo, robotResp.YaoTeamID, robotResp.YaoCreatedBy) {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrAccessDenied.Code,
			ErrorDescription: "Forbidden: No permission to trigger this robot",
		}
		response.RespondWithError(c, response.StatusForbidden, errorResp)
		return
	}

	// Build API trigger request
	apiReq := buildAPITriggerRequest(&req)

	// Call API layer
	result, err := robotapi.Trigger(ctx, robotID, apiReq)
	if err != nil {
		log.Error("Failed to trigger robot %s: %v", robotID, err)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to trigger robot: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Convert to response
	resp := &TriggerResponse{
		Accepted:    result.Accepted,
		ExecutionID: result.ExecutionID,
		Queued:      result.Queued,
		Message:     result.Message,
	}

	if result.Accepted {
		response.RespondWithSuccess(c, response.StatusOK, resp)
	} else {
		// Trigger was not accepted (e.g., queue full, robot paused)
		response.RespondWithSuccess(c, response.StatusOK, resp)
	}
}

// InterveneRobot performs human intervention on a robot
// POST /v1/agent/robots/:id/intervene
func InterveneRobot(c *gin.Context) {
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

	// Parse request body
	var req InterveneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid request body: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Validate action
	if req.Action == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "action is required",
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

	// Check write permission on robot (intervention requires write permission)
	if !CanWrite(c, authInfo, robotResp.YaoTeamID, robotResp.YaoCreatedBy) {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrAccessDenied.Code,
			ErrorDescription: "Forbidden: No permission to intervene with this robot",
		}
		response.RespondWithError(c, response.StatusForbidden, errorResp)
		return
	}

	// Build API trigger request for intervention
	apiReq := &robotapi.TriggerRequest{
		Type:   robottypes.TriggerHuman,
		Action: robottypes.InterventionAction(req.Action),
		PlanAt: req.PlanAt,
	}

	// Convert messages
	if len(req.Messages) > 0 {
		apiReq.Messages = convertMessagesToContext(req.Messages)
	}

	// Call API layer (Intervene uses TriggerHuman internally)
	result, err := robotapi.Intervene(ctx, robotID, apiReq)
	if err != nil {
		log.Error("Failed to intervene with robot %s: %v", robotID, err)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to intervene: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Convert to response
	resp := &InterveneResponse{
		Accepted:    result.Accepted,
		ExecutionID: result.ExecutionID,
		Message:     result.Message,
	}

	response.RespondWithSuccess(c, response.StatusOK, resp)
}

// ==================== Helper Functions ====================

// buildAPITriggerRequest builds robotapi.TriggerRequest from HTTP request
func buildAPITriggerRequest(req *TriggerRequest) *robotapi.TriggerRequest {
	apiReq := &robotapi.TriggerRequest{}

	// Set trigger type (default to human)
	switch req.TriggerType {
	case "event":
		apiReq.Type = robottypes.TriggerEvent
	case "clock":
		apiReq.Type = robottypes.TriggerClock
	default:
		apiReq.Type = robottypes.TriggerHuman
	}

	// Human intervention fields
	if req.Action != "" {
		apiReq.Action = robottypes.InterventionAction(req.Action)
	}
	if len(req.Messages) > 0 {
		apiReq.Messages = convertMessagesToContext(req.Messages)
	}

	// Event fields
	if req.Source != "" {
		apiReq.Source = robottypes.EventSource(req.Source)
	}
	if req.EventType != "" {
		apiReq.EventType = req.EventType
	}
	if req.Data != nil {
		apiReq.Data = req.Data
	}

	// Executor mode
	if req.ExecutorMode != "" {
		apiReq.ExecutorMode = robottypes.ExecutorMode(req.ExecutorMode)
	}

	// i18n locale
	if req.Locale != "" {
		apiReq.Locale = req.Locale
	}

	return apiReq
}

// convertMessagesToContext converts MessageItem slice to agent context messages
func convertMessagesToContext(msgs []MessageItem) []agentcontext.Message {
	result := make([]agentcontext.Message, 0, len(msgs))
	for _, m := range msgs {
		result = append(result, agentcontext.Message{
			Role:    agentcontext.MessageRole(m.Role),
			Content: m.Content,
		})
	}
	return result
}
