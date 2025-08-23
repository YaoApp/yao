package kb

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/kb"
	"github.com/yaoapp/yao/openapi/response"
)

// Score Management Handlers

// UpdateScores updates scores for multiple segments in batch
func UpdateScores(c *gin.Context) {
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

	// Parse request body for batch score updates
	var req UpdateScoresRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid request format: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Validate request
	if len(req.Scores) == 0 {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "At least one score update is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Validate each score entry
	for i, score := range req.Scores {
		if strings.TrimSpace(score.ID) == "" {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: fmt.Sprintf("scores[%d].id is required", i),
			}
			response.RespondWithError(c, response.StatusBadRequest, errorResp)
			return
		}
		if score.Score < 0 {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: fmt.Sprintf("scores[%d].score cannot be negative", i),
			}
			response.RespondWithError(c, response.StatusBadRequest, errorResp)
			return
		}
	}

	// Call GraphRag UpdateScores method (without Compute option)
	updatedCount, err := kb.Instance.UpdateScores(c.Request.Context(), docID, req.Scores)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to update scores: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	result := gin.H{
		"message":       "Scores updated successfully",
		"doc_id":        docID,
		"scores":        req.Scores,
		"updated_count": updatedCount,
	}

	response.RespondWithSuccess(c, response.StatusOK, result)
}
