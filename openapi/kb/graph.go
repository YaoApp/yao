package kb

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/graphrag/utils"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/kb"
	"github.com/yaoapp/yao/kb/providers/factory"
	kbtypes "github.com/yaoapp/yao/kb/types"
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

	// Parse query parameters for filtering options
	includeEntities := c.DefaultQuery("include_entities", "true") != "false"
	includeRelationships := c.DefaultQuery("include_relationships", "true") != "false"

	// Prepare the response
	result := gin.H{
		"doc_id":     docID,
		"segment_id": segmentID,
	}

	// Get entities if requested
	if includeEntities {
		entities, err := kb.Instance.GetSegmentEntities(c.Request.Context(), docID, segmentID)
		if err != nil {
			errorResp := &response.ErrorResponse{
				Code:             "segment_entities_error",
				ErrorDescription: fmt.Sprintf("Failed to get segment entities: %v", err),
			}
			response.RespondWithError(c, response.StatusNotFound, errorResp)
			return
		}
		result["entities"] = entities
		result["entities_count"] = len(entities)
	}

	// Get relationships if requested (using entity-based query for better results)
	if includeRelationships {
		relationships, err := kb.Instance.GetSegmentRelationshipsByEntities(c.Request.Context(), docID, segmentID)
		if err != nil {
			errorResp := &response.ErrorResponse{
				Code:             "segment_relationships_error",
				ErrorDescription: fmt.Sprintf("Failed to get segment relationships: %v", err),
			}
			response.RespondWithError(c, response.StatusNotFound, errorResp)
			return
		}
		result["relationships"] = relationships
		result["relationships_count"] = len(relationships)
		result["query_type"] = "by_entities" // Indicate we're using entity-based relationship query
	}

	response.RespondWithSuccess(c, response.StatusOK, result)
}

// GetSegmentEntities gets the entities for a specific segment
func GetSegmentEntities(c *gin.Context) {
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

	// Call the GraphRag instance to get segment entities
	entities, err := kb.Instance.GetSegmentEntities(c.Request.Context(), docID, segmentID)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             "segment_entities_error",
			ErrorDescription: fmt.Sprintf("Failed to get segment entities: %v", err),
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return
	}

	// Prepare the response
	result := gin.H{
		"doc_id":         docID,
		"segment_id":     segmentID,
		"entities":       entities,
		"entities_count": len(entities),
	}

	response.RespondWithSuccess(c, response.StatusOK, result)
}

// GetSegmentRelationships gets the relationships for a specific segment
func GetSegmentRelationships(c *gin.Context) {
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

	// Call the GraphRag instance to get segment relationships
	relationships, err := kb.Instance.GetSegmentRelationships(c.Request.Context(), docID, segmentID)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             "segment_relationships_error",
			ErrorDescription: fmt.Sprintf("Failed to get segment relationships: %v", err),
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return
	}

	// Prepare the response
	result := gin.H{
		"doc_id":              docID,
		"segment_id":          segmentID,
		"relationships":       relationships,
		"relationships_count": len(relationships),
	}

	response.RespondWithSuccess(c, response.StatusOK, result)
}

// GetSegmentRelationshipsByEntities gets all relationships connected to entities in this segment
func GetSegmentRelationshipsByEntities(c *gin.Context) {
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

	// Call the GraphRag instance to get segment relationships by entities
	relationships, err := kb.Instance.GetSegmentRelationshipsByEntities(c.Request.Context(), docID, segmentID)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             "segment_relationships_by_entities_error",
			ErrorDescription: fmt.Sprintf("Failed to get segment relationships by entities: %v", err),
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return
	}

	// Prepare the response
	result := gin.H{
		"doc_id":              docID,
		"segment_id":          segmentID,
		"relationships":       relationships,
		"relationships_count": len(relationships),
		"query_type":          "by_entities", // Indicate this is entity-based query
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

	// Parse CollectionID from docID to find the right collection
	collectionID, _ := utils.ExtractCollectionIDFromDocID(docID)
	if collectionID == "" {
		collectionID = "default"
	}

	// Get Extraction Provider ID from document
	knowledgeBase := kb.Instance.(*kb.KnowledgeBase)
	document, err := knowledgeBase.Config.FindDocument(docID, model.QueryParam{Select: []interface{}{
		"collection_id",
		"extraction_provider_id", "extraction_option_id", "extraction_properties",
		"locale",
	}})
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Failed to find document: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Parse extraction options from request body (optional, will override document config)
	var extractOptions map[string]interface{}
	if err := c.ShouldBindJSON(&extractOptions); err != nil {
		// If no body provided, use default options
		extractOptions = make(map[string]interface{})
	}

	// Build ExtractionOptions from document configuration
	var options *types.ExtractionOptions
	if document != nil {
		options = &types.ExtractionOptions{}

		// Get extraction provider from document
		if extractionProviderID, ok := document["extraction_provider_id"].(string); ok && extractionProviderID != "" {
			// Get extraction option ID from document
			var extractionOptionID string
			if optionID, ok := document["extraction_option_id"].(string); ok {
				extractionOptionID = optionID
			}

			// Get extraction properties from document
			var extractionProperties map[string]interface{}
			if props, ok := document["extraction_properties"].(map[string]interface{}); ok {
				extractionProperties = props
			}

			// Create extraction provider configuration
			extractionConfig := &ProviderConfig{
				ProviderID: extractionProviderID,
				OptionID:   extractionOptionID,
				// Don't set Option directly when OptionID is provided
				// Let ProviderOption method resolve it from the provider
			}

			// If we have custom properties but no OptionID, set them directly
			if extractionOptionID == "" && len(extractionProperties) > 0 {
				extractionConfig.Option = &kbtypes.ProviderOption{
					Properties: extractionProperties,
				}
			}

			// Get locale from document (default to "en" if not set)
			locale := "en"
			if docLocale, ok := document["locale"].(string); ok && docLocale != "" {
				locale = docLocale
			}

			// Get provider option using the same pattern as ToUpsertOptions
			extractionOption, err := extractionConfig.ProviderOption("extraction", locale)
			if err != nil {
				errorResp := &response.ErrorResponse{
					Code:             "extraction_provider_error",
					ErrorDescription: fmt.Sprintf("Failed to resolve extraction provider: %v", err),
				}
				response.RespondWithError(c, response.StatusInternalServerError, errorResp)
				return
			}

			// Use factory to create extraction provider
			extractor, err := factory.MakeExtraction(extractionProviderID, extractionOption)
			if err != nil {
				errorResp := &response.ErrorResponse{
					Code:             "extraction_provider_error",
					ErrorDescription: fmt.Sprintf("Failed to create extraction provider %s: %v", extractionProviderID, err),
				}
				response.RespondWithError(c, response.StatusInternalServerError, errorResp)
				return
			}

			// Set the extractor in options
			options.Use = extractor
		}
	}

	// Allow request body to override extraction options
	if len(extractOptions) > 0 {
		if options == nil {
			options = &types.ExtractionOptions{}
		}
		// TODO: Map extractOptions from request body to override document settings if needed
		// For now, document settings take precedence
	}

	// Call ExtractSegmentGraph
	extractionResult, err := kb.Instance.ExtractSegmentGraph(c.Request.Context(), docID, segmentID, options)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             "extraction_failed",
			ErrorDescription: fmt.Sprintf("Failed to extract segment graph: %v", err),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Build response using the simplified SegmentExtractionResult structure
	result := map[string]interface{}{
		"message":             "Entities and relationships extracted successfully",
		"doc_id":              extractionResult.DocID,
		"segment_id":          extractionResult.SegmentID,
		"entities_count":      extractionResult.EntitiesCount,      // Use count from structure
		"relationships_count": extractionResult.RelationshipsCount, // Use count from structure
		"extraction_model":    extractionResult.ExtractionModel,
		"extraction_options":  extractOptions,
		// Note: Detailed entities and relationships are no longer returned
		// Frontend should use separate APIs (GetSegmentEntities/GetSegmentRelationships) if needed
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

	// TODO: Implement async extract segment graph logic using Job system
	// err := ExtractSegmentGraphProcess(context.Background(), segmentID, extractOptions, job.ID)

	// Temporary response until async implementation is completed
	result := gin.H{
		"message":    "Async graph extraction not yet implemented",
		"status":     "pending_implementation",
		"doc_id":     docID,
		"segment_id": segmentID,
	}

	response.RespondWithSuccess(c, response.StatusCreated, result)
}
