package kb

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/kb"
	"github.com/yaoapp/yao/openapi/response"
)

// Segment Management Handlers

// AddSegments adds segments to a document
func AddSegments(c *gin.Context) {
	var req AddSegmentsRequest

	// Parse and bind JSON request
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid request format: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Check if kb.Instance is available
	if kb.Instance == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Knowledge base not initialized",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Convert request to UpsertOptions
	upsertOptions, err := req.BaseUpsertRequest.ToUpsertOptions()
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Failed to convert request to upsert options: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Perform add segments operation
	segmentIDs, err := kb.Instance.AddSegments(c.Request.Context(), req.DocID, req.SegmentTexts, upsertOptions)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to add segments: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Return success response
	result := gin.H{
		"message":        "Segments added successfully",
		"collection_id":  req.CollectionID,
		"doc_id":         req.DocID,
		"segment_ids":    segmentIDs,
		"segments_count": len(segmentIDs),
	}

	response.RespondWithSuccess(c, response.StatusCreated, result)
}

// UpdateSegments updates segments manually
func UpdateSegments(c *gin.Context) {
	var req UpdateSegmentsRequest

	// Parse and bind JSON request
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid request format: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Check if kb.Instance is available
	if kb.Instance == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Knowledge base not initialized",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Convert request to UpsertOptions
	upsertOptions, err := req.BaseUpsertRequest.ToUpsertOptions()
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Failed to convert request to upsert options: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Perform update segments operation
	updatedCount, err := kb.Instance.UpdateSegments(c.Request.Context(), req.SegmentTexts, upsertOptions)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to update segments: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Return success response
	result := gin.H{
		"message":        "Segments updated successfully",
		"collection_id":  req.CollectionID,
		"updated_count":  updatedCount,
		"segments_count": len(req.SegmentTexts),
	}

	response.RespondWithSuccess(c, response.StatusOK, result)
}

// RemoveSegments removes segments by IDs
func RemoveSegments(c *gin.Context) {
	// TODO: Implement remove segments logic
	c.JSON(http.StatusOK, gin.H{"message": "Segments removed"})
}

// RemoveSegmentsByDocID removes all segments of a document
func RemoveSegmentsByDocID(c *gin.Context) {
	// TODO: Implement remove segments by document ID logic
	c.JSON(http.StatusOK, gin.H{"message": "Segments removed by document ID"})
}

// GetSegments gets segments by IDs
func GetSegments(c *gin.Context) {
	// TODO: Implement get segments logic
	c.JSON(http.StatusOK, gin.H{"segments": []interface{}{}})
}

// GetSegment gets a single segment by ID
func GetSegment(c *gin.Context) {
	// TODO: Implement get single segment logic
	c.JSON(http.StatusOK, gin.H{"segment": nil})
}

// ListSegments lists segments with pagination
func ListSegments(c *gin.Context) {
	// TODO: Implement list segments with pagination logic
	c.JSON(http.StatusOK, gin.H{"segments": []interface{}{}, "total": 0, "page": 1})
}

// ScrollSegments scrolls segments with iterator-style pagination
func ScrollSegments(c *gin.Context) {
	// TODO: Implement scroll segments logic
	c.JSON(http.StatusOK, gin.H{"segments": []interface{}{}, "cursor": ""})
}
