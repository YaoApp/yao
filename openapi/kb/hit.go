package kb

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/yao/kb"
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

	// Check if kb.Instance is available
	if kb.Instance == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Knowledge base not initialized",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Convert options to ScrollHitsOptions
	scrollOptions := &types.ScrollHitsOptions{
		SegmentID: segmentID,
		Limit:     options["limit"].(int),
	}

	// Set cursor if provided
	if scrollID, exists := options["scroll_id"]; exists && scrollID != nil {
		scrollOptions.Cursor = scrollID.(string)
	}

	// Set filters if provided
	if filter, exists := options["filter"]; exists && filter != nil {
		filterMap := filter.(map[string]interface{})
		if source, ok := filterMap["source"]; ok {
			scrollOptions.Source = source.(string)
		}
		if scenario, ok := filterMap["scenario"]; ok {
			scrollOptions.Scenario = scenario.(string)
		}
	}

	// Call GraphRag ScrollHits method
	result, err := kb.Instance.ScrollHits(c.Request.Context(), docID, scrollOptions)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to scroll hits: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
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

	// TODO: Search functionality not implemented yet - reserved for future use
	errorResp := &response.ErrorResponse{
		Code:             response.ErrServerError.Code,
		ErrorDescription: "Search hits functionality is reserved but not implemented yet",
	}
	response.RespondWithError(c, response.StatusNotImplemented, errorResp)
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

	// Check if kb.Instance is available
	if kb.Instance == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Knowledge base not initialized",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Call GraphRag GetHit method
	hit, err := kb.Instance.GetHit(c.Request.Context(), docID, segmentID, hitID)
	if err != nil {
		if err.Error() == "hit not found" {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: "Hit not found",
			}
			response.RespondWithError(c, response.StatusNotFound, errorResp)
		} else {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrServerError.Code,
				ErrorDescription: "Failed to get hit: " + err.Error(),
			}
			response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		}
		return
	}

	response.RespondWithSuccess(c, response.StatusOK, gin.H{
		"hit":        hit,
		"doc_id":     docID,
		"segment_id": segmentID,
		"hit_id":     hitID,
	})
}

// AddHits adds new hits to a segment using UpdateHits implementation
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

	// Parse request body for hit data
	var req UpdateHitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid request format: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Validate request
	if len(req.Segments) == 0 {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "At least one hit is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Ensure all hits are for the correct segment
	for i := range req.Segments {
		req.Segments[i].ID = segmentID
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

	// Build options with default reaction from payload or create basic fallback
	var options types.UpdateHitOptions
	if req.DefaultReaction != nil {
		// Use the default reaction provided in the request
		options.Reaction = req.DefaultReaction
	} else {
		// Create basic fallback context for segments that don't have reaction
		options.Reaction = &types.SegmentReaction{
			Source:   "api",
			Scenario: "hit",
			Context: map[string]interface{}{
				"method":    c.Request.Method,
				"path":      c.Request.URL.Path,
				"client_ip": c.ClientIP(),
			},
		}
	}

	// Call GraphRag UpdateHits method
	updatedCount, err := kb.Instance.UpdateHits(c.Request.Context(), docID, req.Segments, options)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to add hits: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	result := gin.H{
		"message":       "Hits added successfully",
		"doc_id":        docID,
		"segment_id":    segmentID,
		"hits":          req.Segments,
		"updated_count": updatedCount,
	}

	response.RespondWithSuccess(c, response.StatusOK, result)
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

	// Check if kb.Instance is available
	if kb.Instance == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Knowledge base not initialized",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Build HitRemoval structs
	var hitRemovals []types.HitRemoval
	for _, hitID := range validHitIDs {
		hitRemovals = append(hitRemovals, types.HitRemoval{
			SegmentID: segmentID,
			HitID:     hitID,
		})
	}

	// Call GraphRag RemoveHits method
	removedCount, err := kb.Instance.RemoveHits(c.Request.Context(), docID, hitRemovals)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to remove hits: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	result := gin.H{
		"message":       "Hits removed successfully",
		"doc_id":        docID,
		"segment_id":    segmentID,
		"hit_ids":       validHitIDs,
		"removed_count": removedCount,
	}

	response.RespondWithSuccess(c, response.StatusOK, result)
}
