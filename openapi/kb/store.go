package kb

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/kb"
	"github.com/yaoapp/yao/openapi/response"
)

// Segment Voting, Scoring, Weighting Handlers

// UpdateVote updates votes for segments
func UpdateVote(c *gin.Context) {
	// TODO: Implement update vote logic
	c.JSON(http.StatusOK, gin.H{"message": "Vote updated"})
}

// UpdateScore updates scores for segments
func UpdateScore(c *gin.Context) {
	// TODO: Implement update score logic
	c.JSON(http.StatusOK, gin.H{"message": "Score updated"})
}

// UpdateWeight updates weights for segments
func UpdateWeight(c *gin.Context) {
	var req UpdateWeightRequest

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

	// Perform update weight operation
	updatedCount, err := kb.Instance.UpdateWeight(c.Request.Context(), req.Segments)
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
		"segments":      req.Segments,
		"updated_count": updatedCount,
	}

	response.RespondWithSuccess(c, response.StatusOK, result)
}
