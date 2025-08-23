package kb

import (
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/kb"
	"github.com/yaoapp/yao/openapi/response"
)

// Weight Management Handlers

// UpdateWeights updates weights for multiple segments in batch
func UpdateWeights(c *gin.Context) {
	// Extract docID from URL path
	docID := c.Param("docID")
	if docID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Document ID is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	var req UpdateWeightsRequest

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

	// TODO: Implement document permission validation for docID

	// Call GraphRag UpdateWeights method (without Compute option)
	updatedCount, err := kb.Instance.UpdateWeights(c.Request.Context(), docID, req.Weights)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to update segment weights: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Return success response
	result := gin.H{
		"message":       "Segment weights updated successfully",
		"doc_id":        docID,
		"weights":       req.Weights,
		"updated_count": updatedCount,
	}

	response.RespondWithSuccess(c, response.StatusOK, result)
}
