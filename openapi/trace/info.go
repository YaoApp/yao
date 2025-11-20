package trace

import (
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/trace"
)

// GetInfo retrieves trace information
// GET /api/__yao/openapi/v1/trace/traces/:traceID/info
func GetInfo(c *gin.Context) {
	// Get trace ID from URL parameter
	traceID := c.Param("traceID")
	if traceID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Trace ID is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Load trace manager with permission checking
	manager, _, shouldRelease, err := loadTraceManager(c, traceID)
	if err != nil {
		respondWithLoadError(c, err)
		return
	}

	// Release after use if we loaded it temporarily
	if shouldRelease {
		defer trace.Release(traceID)
	}

	// Get trace info from manager (reads from storage)
	info, err := manager.GetTraceInfo()
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to get trace info: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Prepare response data
	infoData := gin.H{
		"id":         info.ID,
		"driver":     info.Driver,
		"status":     info.Status,
		"created_at": info.CreatedAt,
		"updated_at": info.UpdatedAt,
		"archived":   info.Archived,
	}

	if info.ArchivedAt != nil {
		infoData["archived_at"] = *info.ArchivedAt
	}

	if info.Metadata != nil {
		infoData["metadata"] = info.Metadata
	}

	// Add user/team info if available
	if info.CreatedBy != "" {
		infoData["created_by"] = info.CreatedBy
	}
	if info.TeamID != "" {
		infoData["team_id"] = info.TeamID
	}
	if info.TenantID != "" {
		infoData["tenant_id"] = info.TenantID
	}

	response.RespondWithSuccess(c, response.StatusOK, infoData)
}
