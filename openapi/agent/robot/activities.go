package robot

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/kun/log"
	robotapi "github.com/yaoapp/yao/agent/robot/api"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/response"
)

// ==================== Activities Handler ====================
// Activities are derived from execution status changes across all robots in a team

// ListActivities lists recent activities for the user's team
// GET /v1/agent/robots/activities
func ListActivities(c *gin.Context) {
	// Get authorized information
	authInfo := authorized.GetInfo(c)

	// Get team_id from auth - activities are team-scoped
	teamID := ""
	if authInfo != nil {
		teamID = authInfo.TeamID
		// If no team_id, fall back to user_id for personal users
		if teamID == "" {
			teamID = authInfo.UserID
		}
	}

	if teamID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrAccessDenied.Code,
			ErrorDescription: "Unable to determine team scope",
		}
		response.RespondWithError(c, response.StatusForbidden, errorResp)
		return
	}

	// Parse query parameters
	var filter ActivityFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid query parameters: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Apply defaults
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}

	// Create robot context
	ctx := &robottypes.Context{}

	// Build API query
	query := &robotapi.ActivityQuery{
		TeamID: teamID,
		Limit:  filter.Limit,
		Type:   filter.Type, // Pass type filter
	}

	// Parse 'since' if provided
	if filter.Since != "" {
		since, err := time.Parse(time.RFC3339, filter.Since)
		if err != nil {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: "Invalid 'since' parameter: must be RFC3339 format",
			}
			response.RespondWithError(c, response.StatusBadRequest, errorResp)
			return
		}
		query.Since = &since
	}

	// Call API layer
	result, err := robotapi.ListActivities(ctx, query)
	if err != nil {
		log.Error("Failed to list activities for team %s: %v", teamID, err)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to list activities: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Convert to response
	data := make([]*ActivityResponse, 0, len(result.Data))
	for _, item := range result.Data {
		data = append(data, NewActivityResponse(item))
	}

	resp := &ActivityListResponse{
		Data: data,
	}
	response.RespondWithSuccess(c, response.StatusOK, resp)
}
