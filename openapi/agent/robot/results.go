package robot

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/kun/log"
	robotapi "github.com/yaoapp/yao/agent/robot/api"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/response"
)

// ==================== Results Handlers ====================
// Results are completed executions with delivery content

// ListResults lists results (deliveries) for a robot
// GET /v1/agent/robots/:id/results
func ListResults(c *gin.Context) {
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
			ErrorDescription: "Forbidden: No permission to access this robot's results",
		}
		response.RespondWithError(c, response.StatusForbidden, errorResp)
		return
	}

	// Parse query parameters
	var filter ResultFilter
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
	query := &robotapi.ResultQuery{
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}
	if filter.TriggerType != "" {
		query.TriggerType = robottypes.TriggerType(filter.TriggerType)
	}
	if filter.Keyword != "" {
		query.Keyword = filter.Keyword
	}

	// Call API layer
	result, err := robotapi.ListResults(ctx, robotID, query)
	if err != nil {
		log.Error("Failed to list results for robot %s: %v", robotID, err)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to list results: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Convert to response
	data := make([]*ResultResponse, 0, len(result.Data))
	for _, item := range result.Data {
		data = append(data, NewResultResponse(item))
	}

	resp := &ResultListResponse{
		Data:     data,
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	}
	response.RespondWithSuccess(c, response.StatusOK, resp)
}

// GetResult gets a single result by execution ID
// GET /v1/agent/robots/:id/results/:result_id
func GetResult(c *gin.Context) {
	// Get authorized information
	authInfo := authorized.GetInfo(c)

	// Get robot ID and result ID from URL parameters
	robotID := c.Param("id")
	resultID := c.Param("result_id")

	if robotID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "robot id is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}
	if resultID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "result id is required",
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
			ErrorDescription: "Forbidden: No permission to access this robot's results",
		}
		response.RespondWithError(c, response.StatusForbidden, errorResp)
		return
	}

	// Get result
	result, err := robotapi.GetResult(ctx, resultID)
	if err != nil {
		log.Error("Failed to get result %s: %v", resultID, err)

		if err.Error() == "result not found: "+resultID || err.Error() == "result not found: "+resultID+" (no delivery content)" {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: "Result not found: " + resultID,
			}
			response.RespondWithError(c, response.StatusNotFound, errorResp)
			return
		}

		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to get result: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Verify result belongs to this robot
	if result.MemberID != robotID {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Result does not belong to this robot",
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return
	}

	// Convert to response
	resp := NewResultDetailResponse(result)
	response.RespondWithSuccess(c, response.StatusOK, resp)
}
