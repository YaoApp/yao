package kb

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/graphrag/utils"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/kb"
	"github.com/yaoapp/yao/kb/providers/factory"
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

	// Parse CollectionID from docID to find the right collection
	collectionID, _ := utils.ExtractCollectionIDFromDocID(docID)
	if collectionID == "" {
		collectionID = "default"
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

	// Get Embedding Provider ID from collection
	knowledgeBase := kb.Instance.(*kb.KnowledgeBase)

	// Get Extraction Provider ID from document
	document, err := knowledgeBase.Config.FindDocument(docID, model.QueryParam{Select: []interface{}{
		"collection_id",
		"embedding_provider_id", "embedding_option_id", "embedding_properties",
		"extraction_provider_id", "extraction_option_id", "extraction_properties",
	}})
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Failed to find document: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

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

	// Validate segment texts
	if len(req.SegmentTexts) == 0 {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "segment_texts is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	for i, segmentText := range req.SegmentTexts {
		if strings.TrimSpace(segmentText.Text) == "" {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: fmt.Sprintf("segment_texts[%d].text cannot be empty", i),
			}
			response.RespondWithError(c, response.StatusBadRequest, errorResp)
			return
		}
		if strings.TrimSpace(segmentText.ID) == "" {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: fmt.Sprintf("segment_texts[%d].id cannot be empty", i),
			}
			response.RespondWithError(c, response.StatusBadRequest, errorResp)
			return
		}
	}

	// Construct UpsertOptions from database document configuration
	upsertOptions := &types.UpsertOptions{
		CollectionID: document["collection_id"].(string),
		DocID:        docID,
		Metadata:     make(map[string]interface{}),
	}

	// Build Embedding provider configuration from document using Factory
	if embeddingProviderID, ok := document["embedding_provider_id"].(string); ok && embeddingProviderID != "" {
		embeddingConfig := &ProviderConfig{
			ProviderID: embeddingProviderID,
		}

		if embeddingOptionID, ok := document["embedding_option_id"].(string); ok && embeddingOptionID != "" {
			embeddingConfig.OptionID = embeddingOptionID
		}

		// Use Factory to resolve and create embedding provider
		embeddingOption, err := embeddingConfig.ProviderOption("embedding", "en")
		if err != nil {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: "Failed to resolve embedding provider: " + err.Error(),
			}
			response.RespondWithError(c, response.StatusBadRequest, errorResp)
			return
		}

		embeddingProvider, err := factory.MakeEmbedding(embeddingProviderID, embeddingOption)
		if err != nil {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: "Failed to create embedding provider: " + err.Error(),
			}
			response.RespondWithError(c, response.StatusBadRequest, errorResp)
			return
		}

		upsertOptions.Embedding = embeddingProvider
	}

	// Build Extraction provider configuration from document (if available)
	if extractionProviderID, ok := document["extraction_provider_id"].(string); ok && extractionProviderID != "" {
		extractionConfig := &ProviderConfig{
			ProviderID: extractionProviderID,
		}

		if extractionOptionID, ok := document["extraction_option_id"].(string); ok && extractionOptionID != "" {
			extractionConfig.OptionID = extractionOptionID
		}

		// Use Factory to resolve and create extraction provider
		extractionOption, err := extractionConfig.ProviderOption("extraction", "en")
		if err != nil {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: "Failed to resolve extraction provider: " + err.Error(),
			}
			response.RespondWithError(c, response.StatusBadRequest, errorResp)
			return
		}

		extractionProvider, err := factory.MakeExtraction(extractionProviderID, extractionOption)
		if err != nil {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: "Failed to create extraction provider: " + err.Error(),
			}
			response.RespondWithError(c, response.StatusBadRequest, errorResp)
			return
		}

		upsertOptions.Extraction = extractionProvider
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
		"collection_id":  upsertOptions.CollectionID,
		"updated_count":  updatedCount,
		"segments_count": len(req.SegmentTexts),
	}

	response.RespondWithSuccess(c, response.StatusOK, result)
}

// RemoveSegments removes segments by IDs
func RemoveSegments(c *gin.Context) {
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

	// Parse segment_ids from query parameter (comma-separated string)
	segmentIDsParam := strings.TrimSpace(c.Query("segment_ids"))
	if segmentIDsParam == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "segment_ids query parameter is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Split comma-separated segment IDs
	segmentIDs := strings.Split(segmentIDsParam, ",")
	var validSegmentIDs []string
	for _, id := range segmentIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			validSegmentIDs = append(validSegmentIDs, id)
		}
	}

	if len(validSegmentIDs) == 0 {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "No valid segment IDs provided",
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

	// Perform remove segments operation
	removedCount, err := kb.Instance.RemoveSegments(c.Request.Context(), docID, validSegmentIDs)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to remove segments: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Return success response
	result := gin.H{
		"message":       "Segments removed successfully",
		"segment_ids":   validSegmentIDs,
		"removed_count": removedCount,
	}

	response.RespondWithSuccess(c, response.StatusOK, result)
}

// RemoveSegmentsByDocID removes all segments of a document
func RemoveSegmentsByDocID(c *gin.Context) {
	// Parse docID from URL path parameter
	docID := c.Param("docID")
	if docID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "docID is required",
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

	// Perform remove segments by document ID operation
	removedCount, err := kb.Instance.RemoveSegmentsByDocID(c.Request.Context(), docID)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to remove segments by document ID: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Return success response
	result := gin.H{
		"message":       "Segments removed successfully",
		"doc_id":        docID,
		"removed_count": removedCount,
	}

	response.RespondWithSuccess(c, response.StatusOK, result)
}

// GetSegments gets segments by IDs
func GetSegments(c *gin.Context) {
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

	// TODO: Implement document permission validation for docID
	// TODO: Implement get segments logic
	c.JSON(http.StatusOK, gin.H{
		"segments": []interface{}{},
		"doc_id":   docID,
	})
}

// GetSegment gets a single segment by ID
func GetSegment(c *gin.Context) {
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

	// Check if KB instance exists
	if kb.Instance == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Knowledge base instance is not initialized",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// TODO: Implement document permission validation for docID

	// Get the segment using KB interface
	segment, err := kb.Instance.GetSegment(c.Request.Context(), docID, segmentID)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: fmt.Sprintf("Failed to get segment: %v", err),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	if segment == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Segment not found",
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return
	}

	// Verify that the segment belongs to the specified document
	if segment.DocumentID != docID {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrAccessDenied.Code,
			ErrorDescription: "Segment does not belong to the specified document",
		}
		response.RespondWithError(c, response.StatusForbidden, errorResp)
		return
	}

	result := gin.H{
		"segment":    segment,
		"doc_id":     docID,
		"segment_id": segmentID,
	}

	response.RespondWithSuccess(c, response.StatusOK, result)
}

// ScrollSegments scrolls segments with iterator-style pagination
func ScrollSegments(c *gin.Context) {
	// Parse docID from URL path parameter
	docID := c.Param("docID")
	if docID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "docID is required",
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

	// Parse query parameters for scroll options
	options := &types.ScrollSegmentsOptions{
		IncludeMetadata: true, // Default to include metadata
	}

	// Parse limit (default: 100)
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			options.Limit = limit
		}
	}
	if options.Limit == 0 {
		options.Limit = 100 // Default limit
	}

	// Parse scroll_id parameter for continuing pagination
	if scrollID := strings.TrimSpace(c.Query("scroll_id")); scrollID != "" {
		options.ScrollID = scrollID
	}

	// Parse order_by parameter
	if orderBy := strings.TrimSpace(c.Query("order_by")); orderBy != "" {
		options.OrderBy = strings.Split(orderBy, ",")
		// Trim spaces from each field
		for i, field := range options.OrderBy {
			options.OrderBy[i] = strings.TrimSpace(field)
		}
	}

	// Parse fields parameter
	if fields := strings.TrimSpace(c.Query("fields")); fields != "" {
		options.Fields = strings.Split(fields, ",")
		// Trim spaces from each field
		for i, field := range options.Fields {
			options.Fields[i] = strings.TrimSpace(field)
		}
	}

	// Parse include options
	if includeNodes := c.Query("include_nodes"); includeNodes == "true" {
		options.IncludeNodes = true
	}
	if includeRelationships := c.Query("include_relationships"); includeRelationships == "true" {
		options.IncludeRelationships = true
	}
	if includeMetadata := c.Query("include_metadata"); includeMetadata == "false" {
		options.IncludeMetadata = false
	}

	// Parse filter parameters (basic implementation for common filters)
	filter := make(map[string]interface{})
	if score := c.Query("score"); score != "" {
		if scoreVal, err := strconv.ParseFloat(score, 64); err == nil {
			filter["score"] = scoreVal
		}
	}
	if weight := c.Query("weight"); weight != "" {
		if weightVal, err := strconv.ParseFloat(weight, 64); err == nil {
			filter["weight"] = weightVal
		}
	}
	if vote := c.Query("vote"); vote != "" {
		if voteVal, err := strconv.Atoi(vote); err == nil {
			filter["vote"] = voteVal
		}
	}
	if len(filter) > 0 {
		options.Filter = filter
	}

	// Call GraphRag ScrollSegments method
	result, err := kb.Instance.ScrollSegments(c.Request.Context(), docID, options)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to scroll segments: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Return success response
	response.RespondWithSuccess(c, response.StatusOK, result)
}

// AddSegmentsAsync adds segments to a document asynchronously
func AddSegmentsAsync(c *gin.Context) {
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

	// Check if kb.Instance is available
	if kb.Instance == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Knowledge base not initialized",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

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

	// Set docID from URL parameter
	req.DocID = docID

	// Validate request
	if err := req.Validate(); err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// TODO: Implement document permission validation for docID

	// Create and run job
	job := NewJob()
	jobID := job.Run(func() {
		// TODO: Implement async add segments logic
		// This should call the same logic as AddSegments but in background
		// err := AddSegmentsProcess(context.Background(), &req, job.ID)
		// For now, just simulate async processing
		// if err != nil {
		//     log.Error("Async segments addition failed: %v", err)
		// }
	})

	// Return job ID for status tracking
	result := gin.H{
		"job_id":  jobID,
		"message": "Segments addition started",
		"doc_id":  docID,
	}

	response.RespondWithSuccess(c, response.StatusCreated, result)
}

// UpdateSegmentsAsync updates segments in a document asynchronously
func UpdateSegmentsAsync(c *gin.Context) {
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

	// Check if kb.Instance is available
	if kb.Instance == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Knowledge base not initialized",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Parse request body for segments data
	var requestBody map[string]interface{}
	if err := c.ShouldBindJSON(&requestBody); err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid request format: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// TODO: Validate request body
	// TODO: Implement document permission validation for docID

	// Create and run job
	job := NewJob()
	jobID := job.Run(func() {
		// TODO: Implement async update segments logic
		// This should call the same logic as UpdateSegments but in background
		// err := UpdateSegmentsProcess(context.Background(), docID, requestBody, job.ID)
		// For now, just simulate async processing
		// if err != nil {
		//     log.Error("Async segments update failed: %v", err)
		// }
	})

	// Return job ID for status tracking
	result := gin.H{
		"job_id":  jobID,
		"message": "Segments update started",
		"doc_id":  docID,
	}

	response.RespondWithSuccess(c, response.StatusCreated, result)
}

// GetSegmentParents gets the parent segments for a specific segment
func GetSegmentParents(c *gin.Context) {
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

	// Check if kb.Instance is available
	if kb.Instance == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Knowledge base not initialized",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Parse query parameters for parent options
	options := make(map[string]interface{})

	// Include metadata (default: true)
	if includeMetadata := c.Query("include_metadata"); includeMetadata == "false" {
		options["include_metadata"] = false
	} else {
		options["include_metadata"] = true
	}

	// Depth level (default: 1 - direct parents only)
	depth := 1
	if depthStr := c.Query("depth"); depthStr != "" {
		if d, err := strconv.Atoi(depthStr); err == nil && d > 0 {
			depth = d
		}
	}
	options["depth"] = depth

	// TODO: Implement document permission validation for docID
	// TODO: Implement get segment parents logic
	// TODO: Call kb.Instance.GetSegmentParents(c.Request.Context(), segmentID, options)

	// Return mock response for now
	result := gin.H{
		"parents":    []interface{}{},
		"doc_id":     docID,
		"segment_id": segmentID,
		"depth":      depth,
		"total":      0,
	}

	response.RespondWithSuccess(c, response.StatusOK, result)
}
