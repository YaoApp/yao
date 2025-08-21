package kb

import (
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/kb"
	"github.com/yaoapp/yao/openapi/response"
)

// Graph Management Handlers

// GetSegmentGraph gets the graph (entities and relationships) for a specific segment
func GetSegmentGraph(c *gin.Context) {
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

	// Parse query parameters for graph options
	options := make(map[string]interface{})

	// Include entities (default: true)
	if includeEntities := c.Query("include_entities"); includeEntities == "false" {
		options["include_entities"] = false
	} else {
		options["include_entities"] = true
	}

	// Include relationships (default: true)
	if includeRelationships := c.Query("include_relationships"); includeRelationships == "false" {
		options["include_relationships"] = false
	} else {
		options["include_relationships"] = true
	}

	// Include metadata (default: true)
	if includeMetadata := c.Query("include_metadata"); includeMetadata == "false" {
		options["include_metadata"] = false
	} else {
		options["include_metadata"] = true
	}

	// TODO: Implement document permission validation for docID
	// TODO: Implement get segment graph logic
	// TODO: Call kb.Instance.GetSegmentGraph(c.Request.Context(), segmentID, options)

	// Return mock response for now
	result := gin.H{
		"entities":      []interface{}{},
		"relationships": []interface{}{},
		"doc_id":        docID,
		"segment_id":    segmentID,
		"options":       options,
	}

	response.RespondWithSuccess(c, response.StatusOK, result)
}

// ExtractSegmentGraph re-extracts entities and relationships for a specific segment (synchronous)
func ExtractSegmentGraph(c *gin.Context) {
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

	// Parse extraction options from request body (optional)
	var extractOptions map[string]interface{}
	if err := c.ShouldBindJSON(&extractOptions); err != nil {
		// If no body provided, use default options
		extractOptions = make(map[string]interface{})
	}

	// TODO: Implement document permission validation for docID
	// TODO: Implement extract segment graph logic
	// TODO: Call kb.Instance.ExtractSegmentGraph(c.Request.Context(), segmentID, extractOptions)

	// Return mock response for now
	result := gin.H{
		"message":             "Entities and relationships extracted successfully",
		"doc_id":              docID,
		"segment_id":          segmentID,
		"entities_count":      0,
		"relationships_count": 0,
		"extraction_options":  extractOptions,
	}

	response.RespondWithSuccess(c, response.StatusOK, result)
}

// ExtractSegmentGraphAsync re-extracts entities and relationships for a specific segment (asynchronous)
func ExtractSegmentGraphAsync(c *gin.Context) {
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

	// Parse extraction options from request body (optional)
	var extractOptions map[string]interface{}
	if err := c.ShouldBindJSON(&extractOptions); err != nil {
		// If no body provided, use default options
		extractOptions = make(map[string]interface{})
	}

	// TODO: Implement document permission validation for docID

	// Create and run job
	job := NewJob()
	jobID := job.Run(func() {
		// TODO: Implement async extract segment graph logic
		// err := ExtractSegmentGraphProcess(context.Background(), segmentID, extractOptions, job.ID)
		// For now, just simulate async processing
		// if err != nil {
		//     log.Error("Async graph extraction failed: %v", err)
		// }
	})

	// Return job ID for status tracking
	result := gin.H{
		"job_id":     jobID,
		"message":    "Graph extraction started",
		"doc_id":     docID,
		"segment_id": segmentID,
	}

	response.RespondWithSuccess(c, response.StatusCreated, result)
}
