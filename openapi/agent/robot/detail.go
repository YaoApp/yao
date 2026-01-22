package robot

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/kun/log"
	robotapi "github.com/yaoapp/yao/agent/robot/api"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/response"
)

// GetRobot retrieves a single robot by ID
// GET /v1/agent/robots/:id
func GetRobot(c *gin.Context) {
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

	// Get robot via API
	robotResp, err := robotapi.GetRobotResponse(ctx, robotID)
	if err != nil {
		log.Error("Failed to get robot %s: %v", robotID, err)

		// Check for not found error
		if err == robottypes.ErrRobotNotFound {
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

	// Check read permission
	// Permission rules:
	// - No constraints: allow all
	// - OwnerOnly: user must be the creator
	// - TeamOnly: robot must belong to user's team
	if !CanRead(c, authInfo, robotResp.YaoTeamID, robotResp.YaoCreatedBy) {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrAccessDenied.Code,
			ErrorDescription: "Forbidden: No permission to access this robot",
		}
		response.RespondWithError(c, response.StatusForbidden, errorResp)
		return
	}

	// Convert to HTTP response
	resp := NewResponse(robotResp)
	response.RespondWithSuccess(c, response.StatusOK, resp)
}

// GetRobotStatus retrieves the runtime status of a robot
// GET /v1/agent/robots/:id/status
func GetRobotStatus(c *gin.Context) {
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

	// Get robot status via API
	status, err := robotapi.GetRobotStatus(ctx, robotID)
	if err != nil {
		log.Error("Failed to get robot status %s: %v", robotID, err)

		if err == robottypes.ErrRobotNotFound {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: "Robot not found: " + robotID,
			}
			response.RespondWithError(c, response.StatusNotFound, errorResp)
			return
		}

		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to get robot status: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Check read permission
	if !CanRead(c, authInfo, status.YaoTeamID, status.YaoCreatedBy) {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrAccessDenied.Code,
			ErrorDescription: "Forbidden: No permission to access this robot",
		}
		response.RespondWithError(c, response.StatusForbidden, errorResp)
		return
	}

	// Convert to HTTP response
	resp := NewStatusResponse(status)
	response.RespondWithSuccess(c, response.StatusOK, resp)
}

// CreateRobot creates a new robot
// POST /v1/agent/robots
func CreateRobot(c *gin.Context) {
	// Get authorized information
	authInfo := authorized.GetInfo(c)

	// Parse request body
	var req CreateRobotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid request body: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Validate required fields
	if req.DisplayName == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "display_name is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Generate member_id if not provided (follows existing API pattern)
	if req.MemberID == "" {
		generatedID, err := GenerateMemberID(c.Request.Context())
		if err != nil {
			log.Error("Failed to generate member_id: %v", err)
			errorResp := &response.ErrorResponse{
				Code:             response.ErrServerError.Code,
				ErrorDescription: "Failed to generate member_id: " + err.Error(),
			}
			response.RespondWithError(c, response.StatusInternalServerError, errorResp)
			return
		}
		req.MemberID = generatedID
	}

	// Determine effective team_id:
	// - If user has a team selected (authInfo.TeamID), use it
	// - Otherwise, for personal users, use user_id as team_id
	effectiveTeamID := GetEffectiveTeamID(authInfo)
	if req.TeamID == "" {
		req.TeamID = effectiveTeamID
	}

	// Apply team constraint from auth if TeamOnly
	if authInfo != nil && authInfo.Constraints.TeamOnly && authInfo.TeamID != "" {
		// Force team_id to auth team_id
		req.TeamID = authInfo.TeamID
	}

	// Still require team_id after all fallbacks
	if req.TeamID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "team_id is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Convert to API request
	apiReq := req.ToAPICreateRequest()

	// Apply Yao permission fields
	// Key rule: __yao_team_id = authInfo.TeamID if has team, otherwise = authInfo.UserID
	if authInfo != nil {
		yaoTeamID := authInfo.TeamID
		if yaoTeamID == "" {
			// For personal users (no team), use user_id as __yao_team_id
			// This ensures the robot is scoped to the individual user
			yaoTeamID = authInfo.UserID
		}
		apiReq.AuthScope = &robotapi.AuthScope{
			CreatedBy: authInfo.UserID,
			TeamID:    yaoTeamID,
			TenantID:  authInfo.TenantID,
		}
	}

	// Create robot context
	ctx := &robottypes.Context{}

	// Call API layer
	robotResp, err := robotapi.CreateRobot(ctx, apiReq)
	if err != nil {
		log.Error("Failed to create robot: %v", err)

		// Check for duplicate error
		if strings.Contains(err.Error(), "already exists") {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: err.Error(),
			}
			response.RespondWithError(c, response.StatusConflict, errorResp)
			return
		}

		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to create robot: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Convert to HTTP response
	resp := NewResponse(robotResp)
	response.RespondWithSuccess(c, response.StatusCreated, resp)
}

// UpdateRobot updates an existing robot
// PUT /v1/agent/robots/:id
func UpdateRobot(c *gin.Context) {
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
	var req UpdateRobotRequest
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

	// Check permission - first get the robot to verify ownership/team
	existingRobot, err := robotapi.GetRobotResponse(ctx, robotID)
	if err != nil {
		if err == robottypes.ErrRobotNotFound {
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

	// Check write permission (only creator can update)
	if !CanWrite(c, authInfo, existingRobot.YaoTeamID, existingRobot.YaoCreatedBy) {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrAccessDenied.Code,
			ErrorDescription: "Forbidden: No permission to update this robot",
		}
		response.RespondWithError(c, response.StatusForbidden, errorResp)
		return
	}

	// Convert to API request
	apiReq := req.ToAPIUpdateRequest()

	// Apply Yao permission fields
	if authInfo != nil {
		apiReq.AuthScope = &robotapi.AuthScope{
			UpdatedBy: authInfo.UserID,
		}
	}

	// Call API layer
	robotResp, err := robotapi.UpdateRobot(ctx, robotID, apiReq)
	if err != nil {
		log.Error("Failed to update robot %s: %v", robotID, err)

		if err == robottypes.ErrRobotNotFound {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: "Robot not found: " + robotID,
			}
			response.RespondWithError(c, response.StatusNotFound, errorResp)
			return
		}

		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to update robot: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Convert to HTTP response
	resp := NewResponse(robotResp)
	response.RespondWithSuccess(c, response.StatusOK, resp)
}

// DeleteRobot deletes a robot
// DELETE /v1/agent/robots/:id
func DeleteRobot(c *gin.Context) {
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

	// Check permission - first get the robot to verify ownership/team
	existingRobot, err := robotapi.GetRobotResponse(ctx, robotID)
	if err != nil {
		if err == robottypes.ErrRobotNotFound {
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

	// Check write permission (only creator can delete)
	if !CanWrite(c, authInfo, existingRobot.YaoTeamID, existingRobot.YaoCreatedBy) {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrAccessDenied.Code,
			ErrorDescription: "Forbidden: No permission to delete this robot",
		}
		response.RespondWithError(c, response.StatusForbidden, errorResp)
		return
	}

	// Call API layer
	err = robotapi.RemoveRobot(ctx, robotID)
	if err != nil {
		log.Error("Failed to delete robot %s: %v", robotID, err)

		// Check for running executions
		if strings.Contains(err.Error(), "running executions") {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: err.Error(),
			}
			response.RespondWithError(c, response.StatusConflict, errorResp)
			return
		}

		if err == robottypes.ErrRobotNotFound {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: "Robot not found: " + robotID,
			}
			response.RespondWithError(c, response.StatusNotFound, errorResp)
			return
		}

		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to delete robot: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Return success with no content
	response.RespondWithSuccess(c, response.StatusOK, map[string]interface{}{
		"member_id": robotID,
		"deleted":   true,
	})
}
