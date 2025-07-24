package kb

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/yao/kb"
	"github.com/yaoapp/yao/openapi/response"
)

// Collection Management Handlers

// CreateCollection creates a new collection
func CreateCollection(c *gin.Context) {
	var req CreateCollectionRequest

	// Parse and bind JSON request
	if err := c.ShouldBindJSON(&req); err != nil {
		// Create a custom error with the same structure but specific message
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid request format: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Validate request parameters
	if err := validateCreateCollectionRequest(&req); err != nil {
		// Create a custom error with the same structure but specific message
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Check if kb.Instance is available
	if kb.Instance == nil {
		// Create a custom error with the same structure but specific message
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Knowledge base not initialized",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Create CollectionConfig
	collectionConfig := types.CollectionConfig{
		ID:       req.ID,
		Metadata: req.Metadata,
		Config:   req.Config,
	}

	// Call the actual CreateCollection method
	collectionID, err := kb.Instance.CreateCollection(c.Request.Context(), collectionConfig)
	if err != nil {
		// Create a custom error with the same structure but specific message
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to create collection: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	successData := gin.H{
		"message":       "Collection created successfully",
		"collection_id": collectionID,
	}
	response.RespondWithSuccess(c, response.StatusCreated, successData)
}

// RemoveCollection removes an existing collection
func RemoveCollection(c *gin.Context) {
	// Get collection ID from URL parameter
	collectionID := c.Param("collectionID")
	if collectionID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Collection ID is required",
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

	// Call the actual RemoveCollection method
	removed, err := kb.Instance.RemoveCollection(c.Request.Context(), collectionID)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to remove collection: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	if !removed {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Collection not found or could not be removed",
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return
	}

	successData := gin.H{
		"message":       "Collection removed successfully",
		"collection_id": collectionID,
		"removed":       removed,
	}
	response.RespondWithSuccess(c, response.StatusOK, successData)
}

// CollectionExists checks if a collection exists
func CollectionExists(c *gin.Context) {
	// Get collection ID from URL parameter
	collectionID := c.Param("collectionID")
	if collectionID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Collection ID is required",
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

	// Call the actual CollectionExists method
	exists, err := kb.Instance.CollectionExists(c.Request.Context(), collectionID)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to check collection existence: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	successData := gin.H{
		"collection_id": collectionID,
		"exists":        exists,
	}
	response.RespondWithSuccess(c, response.StatusOK, successData)
}

// GetCollections retrieves collections with optional filtering
func GetCollections(c *gin.Context) {
	// Check if kb.Instance is available
	if kb.Instance == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Knowledge base not initialized",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Build filter from query parameters
	filter := make(map[string]interface{})

	// Extract all query parameters as potential filter conditions
	// This allows filtering by any metadata field, e.g.:
	// GET /collections?category=documents&status=active
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			// Use the first value if multiple values are provided
			filter[key] = values[0]
		}
	}

	collections, err := kb.Instance.GetCollections(c.Request.Context(), filter)
	if err != nil {
		// Create a custom error with the same structure but specific message
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}
	response.RespondWithSuccess(c, response.StatusOK, collections)
}

// CreateCollectionRequest represents the request structure for creating a collection
type CreateCollectionRequest struct {
	ID       string                         `json:"id" binding:"required"`
	Metadata map[string]interface{}         `json:"metadata"`
	Config   *types.CreateCollectionOptions `json:"config" binding:"required"`
}

// validateCreateCollectionRequest validates the incoming request for creating a collection
func validateCreateCollectionRequest(req *CreateCollectionRequest) error {
	if req.ID == "" {
		return fmt.Errorf("id is required")
	}

	if req.Config == nil {
		return fmt.Errorf("config is required")
	}

	// Validate CreateCollectionOptions
	if err := req.Config.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	return nil
}
