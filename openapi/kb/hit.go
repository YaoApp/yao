package kb

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/openapi/response"
)

// Hit Management Handlers

// ScrollHits scrolls hits with iterator-style pagination for a specific segment
func ScrollHits(c *gin.Context) {
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

	// Extract segmentID from URL path
	segmentID := c.Param("segmentID")
	if segmentID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Segment ID is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Parse query parameters for scroll options
	options := map[string]interface{}{
		"doc_id":     docID,
		"segment_id": segmentID,
		"limit":      100, // Default limit
	}

	// Parse limit (default: 100)
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			options["limit"] = limit
		}
	}

	// Parse scroll_id parameter for continuing pagination
	if scrollID := strings.TrimSpace(c.Query("scroll_id")); scrollID != "" {
		options["scroll_id"] = scrollID
	}

	// Parse order_by parameter
	if orderBy := strings.TrimSpace(c.Query("order_by")); orderBy != "" {
		orderByFields := strings.Split(orderBy, ",")
		// Trim spaces from each field
		for i, field := range orderByFields {
			orderByFields[i] = strings.TrimSpace(field)
		}
		options["order_by"] = orderByFields
	}

	// Parse filter parameters
	filter := make(map[string]interface{})
	if hitType := c.Query("hit_type"); hitType != "" {
		filter["hit_type"] = hitType
	}
	if userID := c.Query("user_id"); userID != "" {
		filter["user_id"] = userID
	}
	if sessionID := c.Query("session_id"); sessionID != "" {
		filter["session_id"] = sessionID
	}
	if len(filter) > 0 {
		options["filter"] = filter
	}

	// TODO: Implement document permission validation for docID
	// TODO: Implement scroll hits logic with GraphRag or database
	// TODO: Call kb.Instance.ScrollHits(c.Request.Context(), segmentID, options)

	// Return mock response for now
	result := gin.H{
		"hits":      []interface{}{},
		"scroll_id": nil,
		"has_more":  false,
		"total":     0,
		"options":   options,
	}

	response.RespondWithSuccess(c, response.StatusOK, result)
}

// GetHits gets hits for a specific segment (simple list, no pagination)
func GetHits(c *gin.Context) {
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

	// Extract segmentID from URL path
	segmentID := c.Param("segmentID")
	if segmentID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Segment ID is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Parse basic filter parameters
	filter := make(map[string]interface{})
	if hitType := c.Query("hit_type"); hitType != "" {
		filter["hit_type"] = hitType
	}
	if userID := c.Query("user_id"); userID != "" {
		filter["user_id"] = userID
	}
	if sessionID := c.Query("session_id"); sessionID != "" {
		filter["session_id"] = sessionID
	}

	// Parse limit parameter (optional, for basic limiting without pagination)
	var limit int
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// TODO: Implement document permission validation for docID
	// TODO: Implement get hits logic (simple query without pagination)
	// TODO: Call kb.Instance.GetHits(c.Request.Context(), segmentID, filter, limit)

	// Return mock response for now
	result := gin.H{
		"hits":       []interface{}{},
		"doc_id":     docID,
		"segment_id": segmentID,
		"total":      0,
	}

	if len(filter) > 0 {
		result["filter"] = filter
	}
	if limit > 0 {
		result["limit"] = limit
	}

	response.RespondWithSuccess(c, response.StatusOK, result)
}

// GetHit gets a specific hit by ID
func GetHit(c *gin.Context) {
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

	// Extract segmentID from URL path
	segmentID := c.Param("segmentID")
	if segmentID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Segment ID is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Extract hitID from URL path
	hitID := c.Param("hitID")
	if hitID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Hit ID is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// TODO: Implement document permission validation for docID
	// TODO: Implement get hit detail logic
	c.JSON(http.StatusOK, gin.H{
		"hit":        nil,
		"doc_id":     docID,
		"segment_id": segmentID,
		"hit_id":     hitID,
	})
}

// AddHits adds new hits to a segment
func AddHits(c *gin.Context) {
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

	// Extract segmentID from URL path
	segmentID := c.Param("segmentID")
	if segmentID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Segment ID is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// TODO: Implement document permission validation for docID
	// TODO: Implement add hit logic
	c.JSON(http.StatusOK, gin.H{
		"message":    "Hit added successfully",
		"doc_id":     docID,
		"segment_id": segmentID,
		"hit_id":     "placeholder-hit-id",
	})
}

// UpdateHits updates hits in batch
func UpdateHits(c *gin.Context) {
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

	// Extract segmentID from URL path
	segmentID := c.Param("segmentID")
	if segmentID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Segment ID is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Parse hit_ids from query parameter or request body
	var hitIDs []string

	// Try query parameter first (comma-separated)
	if hitIDsParam := strings.TrimSpace(c.Query("hit_ids")); hitIDsParam != "" {
		hitIDs = strings.Split(hitIDsParam, ",")
		for i, id := range hitIDs {
			hitIDs[i] = strings.TrimSpace(id)
		}
	}

	// TODO: Also support request body with hit data for batch updates
	// TODO: Implement document permission validation for docID
	// TODO: Implement batch update hit logic

	result := gin.H{
		"message":       "Hits updated successfully",
		"doc_id":        docID,
		"segment_id":    segmentID,
		"updated_count": len(hitIDs),
	}

	if len(hitIDs) > 0 {
		result["hit_ids"] = hitIDs
	}

	response.RespondWithSuccess(c, response.StatusOK, result)
}

// RemoveHits removes hits from a segment in batch
func RemoveHits(c *gin.Context) {
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

	// Extract segmentID from URL path
	segmentID := c.Param("segmentID")
	if segmentID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Segment ID is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Parse hit_ids from query parameter (comma-separated)
	hitIDsParam := strings.TrimSpace(c.Query("hit_ids"))
	if hitIDsParam == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "hit_ids query parameter is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Split comma-separated hit IDs
	hitIDs := strings.Split(hitIDsParam, ",")
	var validHitIDs []string
	for _, id := range hitIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			validHitIDs = append(validHitIDs, id)
		}
	}

	if len(validHitIDs) == 0 {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "At least one valid hit ID is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// TODO: Implement document permission validation for docID
	// TODO: Implement batch remove hit logic

	result := gin.H{
		"message":       "Hits removed successfully",
		"doc_id":        docID,
		"segment_id":    segmentID,
		"hit_ids":       validHitIDs,
		"removed_count": len(validHitIDs),
	}

	response.RespondWithSuccess(c, response.StatusOK, result)
}
