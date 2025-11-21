package trace

import (
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/trace"
)

// GetSpaces retrieves all spaces in the trace (metadata only, without key-value data)
// GET /api/__yao/openapi/v1/trace/traces/:traceID/spaces
func GetSpaces(c *gin.Context) {
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

	// Get all spaces from manager (reads from storage)
	spaces, err := manager.GetAllSpaces()
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to get spaces: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Prepare response - return flat list of spaces with metadata only
	spaceList := make([]gin.H, 0, len(spaces))
	for _, space := range spaces {
		spaceInfo := gin.H{
			"id":          space.ID,
			"label":       space.Label,
			"type":        space.Type,
			"icon":        space.Icon,
			"description": space.Description,
			"ttl":         space.TTL,
			"created_at":  space.CreatedAt,
			"updated_at":  space.UpdatedAt,
		}

		if space.Metadata != nil {
			spaceInfo["metadata"] = space.Metadata
		}

		spaceList = append(spaceList, spaceInfo)
	}

	response.RespondWithSuccess(c, response.StatusOK, gin.H{
		"trace_id": traceID,
		"spaces":   spaceList,
		"count":    len(spaceList),
	})
}

// GetSpace retrieves a single space by ID with all key-value data
// GET /api/__yao/openapi/v1/trace/traces/:traceID/spaces/:spaceID
func GetSpace(c *gin.Context) {
	// Get trace ID and space ID from URL parameters
	traceID := c.Param("traceID")
	spaceID := c.Param("spaceID")

	if traceID == "" || spaceID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Trace ID and Space ID are required",
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

	// Get space by ID from manager (reads from storage with all data)
	spaceData, err := manager.GetSpaceByID(spaceID)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to get space: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	if spaceData == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Space not found",
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return
	}

	// Prepare detailed space response with all key-value data
	responseData := gin.H{
		"id":          spaceData.ID,
		"label":       spaceData.Label,
		"type":        spaceData.Type,
		"icon":        spaceData.Icon,
		"description": spaceData.Description,
		"ttl":         spaceData.TTL,
		"created_at":  spaceData.CreatedAt,
		"updated_at":  spaceData.UpdatedAt,
		"data":        spaceData.Data, // Include all key-value pairs
	}

	if spaceData.Metadata != nil {
		responseData["metadata"] = spaceData.Metadata
	}

	response.RespondWithSuccess(c, response.StatusOK, responseData)
}
