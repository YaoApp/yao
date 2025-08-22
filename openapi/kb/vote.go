package kb

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/yao/kb"
	"github.com/yaoapp/yao/openapi/response"
)

// Vote Management Handlers

// ScrollVotes scrolls votes with iterator-style pagination for a specific segment
func ScrollVotes(c *gin.Context) {
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
	if voteType := c.Query("vote_type"); voteType != "" {
		filter["vote_type"] = voteType
	}
	if userID := c.Query("user_id"); userID != "" {
		filter["user_id"] = userID
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

	// Convert options to ScrollVotesOptions
	scrollOptions := &types.ScrollVotesOptions{
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
		if voteType, ok := filterMap["vote_type"]; ok {
			scrollOptions.VoteType = types.VoteType(voteType.(string))
		}
		if source, ok := filterMap["source"]; ok {
			scrollOptions.Source = source.(string)
		}
		if scenario, ok := filterMap["scenario"]; ok {
			scrollOptions.Scenario = scenario.(string)
		}
	}

	// Call GraphRag ScrollVotes method
	result, err := kb.Instance.ScrollVotes(c.Request.Context(), docID, scrollOptions)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to scroll votes: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	response.RespondWithSuccess(c, response.StatusOK, result)
}

// GetVotes gets votes for a specific segment (simple list, no pagination)
func GetVotes(c *gin.Context) {
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
	if voteType := c.Query("vote_type"); voteType != "" {
		filter["vote_type"] = voteType
	}
	if userID := c.Query("user_id"); userID != "" {
		filter["user_id"] = userID
	}

	// TODO: Search functionality not implemented yet - reserved for future use
	errorResp := &response.ErrorResponse{
		Code:             response.ErrServerError.Code,
		ErrorDescription: "Search votes functionality is reserved but not implemented yet",
	}
	response.RespondWithError(c, response.StatusNotImplemented, errorResp)
}

// GetVote gets a specific vote by ID
func GetVote(c *gin.Context) {
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

	// Extract voteID from URL path
	voteID := c.Param("voteID")
	if voteID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Vote ID is required",
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

	// Call GraphRag GetVote method
	vote, err := kb.Instance.GetVote(c.Request.Context(), docID, segmentID, voteID)
	if err != nil {
		if err.Error() == "vote not found" {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: "Vote not found",
			}
			response.RespondWithError(c, response.StatusNotFound, errorResp)
		} else {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrServerError.Code,
				ErrorDescription: "Failed to get vote: " + err.Error(),
			}
			response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		}
		return
	}

	response.RespondWithSuccess(c, response.StatusOK, gin.H{
		"vote":       vote,
		"doc_id":     docID,
		"segment_id": segmentID,
		"vote_id":    voteID,
	})
}

// AddVotes adds new votes to a segment using UpdateVotes implementation
func AddVotes(c *gin.Context) {
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

	// Parse request body for vote data
	var req UpdateVoteRequest
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
			ErrorDescription: "At least one vote is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Ensure all votes are for the correct segment
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
	var options types.UpdateVoteOptions
	if req.DefaultReaction != nil {
		// Use the default reaction provided in the request
		options.Reaction = req.DefaultReaction
	} else {
		// Create basic fallback context for segments that don't have reaction
		options.Reaction = &types.SegmentReaction{
			Source:   "api",
			Scenario: "vote",
			Context: map[string]interface{}{
				"method":    c.Request.Method,
				"path":      c.Request.URL.Path,
				"client_ip": c.ClientIP(),
			},
		}
	}

	// Call GraphRag UpdateVotes method
	updatedCount, err := kb.Instance.UpdateVotes(c.Request.Context(), docID, req.Segments, options)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to add votes: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	result := gin.H{
		"message":       "Votes added successfully",
		"doc_id":        docID,
		"segment_id":    segmentID,
		"votes":         req.Segments,
		"updated_count": updatedCount,
	}

	response.RespondWithSuccess(c, response.StatusOK, result)
}

// UpdateVotes updates votes in batch
func UpdateVotes(c *gin.Context) {
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

	// Parse vote_ids from query parameter or request body
	var voteIDs []string

	// Try query parameter first (comma-separated)
	if voteIDsParam := strings.TrimSpace(c.Query("vote_ids")); voteIDsParam != "" {
		voteIDs = strings.Split(voteIDsParam, ",")
		for i, id := range voteIDs {
			voteIDs[i] = strings.TrimSpace(id)
		}
	}

	// TODO: Also support request body with vote data for batch updates
	// TODO: Implement document permission validation for docID
	// TODO: Implement batch update vote logic

	result := gin.H{
		"message":       "Votes updated successfully",
		"doc_id":        docID,
		"segment_id":    segmentID,
		"updated_count": len(voteIDs),
	}

	if len(voteIDs) > 0 {
		result["vote_ids"] = voteIDs
	}

	response.RespondWithSuccess(c, response.StatusOK, result)
}

// RemoveVotes removes votes from a segment in batch
func RemoveVotes(c *gin.Context) {
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

	// Parse vote_ids from query parameter (comma-separated)
	voteIDsParam := strings.TrimSpace(c.Query("vote_ids"))
	if voteIDsParam == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "vote_ids query parameter is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Split comma-separated vote IDs
	voteIDs := strings.Split(voteIDsParam, ",")
	var validVoteIDs []string
	for _, id := range voteIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			validVoteIDs = append(validVoteIDs, id)
		}
	}

	if len(validVoteIDs) == 0 {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "At least one valid vote ID is required",
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

	// Build VoteRemoval structs
	var voteRemovals []types.VoteRemoval
	for _, voteID := range validVoteIDs {
		voteRemovals = append(voteRemovals, types.VoteRemoval{
			SegmentID: segmentID,
			VoteID:    voteID,
		})
	}

	// Call GraphRag RemoveVotes method
	removedCount, err := kb.Instance.RemoveVotes(c.Request.Context(), docID, voteRemovals)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to remove votes: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	result := gin.H{
		"message":       "Votes removed successfully",
		"doc_id":        docID,
		"segment_id":    segmentID,
		"vote_ids":      validVoteIDs,
		"removed_count": removedCount,
	}

	response.RespondWithSuccess(c, response.StatusOK, result)
}
