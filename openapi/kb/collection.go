package kb

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/yao/kb"
	"github.com/yaoapp/yao/openapi/response"
)

// Collection Management Handlers

// ProviderSettings represents the resolved provider configuration
type ProviderSettings struct {
	Dimension  int                    `json:"dimension"`
	Connector  string                 `json:"connector"`
	Properties map[string]interface{} `json:"properties"`
}

// getProviderSettings reads and resolves provider settings by provider ID and option value
func getProviderSettings(providerID, optionValue, locale string) (*ProviderSettings, error) {
	// Default locale to "en" if empty
	if locale == "" {
		locale = "en"
	}

	// Get the specific provider using KB API
	provider, err := kb.GetProviderWithLanguage("embedding", providerID, locale)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider %s: %v", providerID, err)
	}

	// Find the target option
	targetOption, found := provider.GetOption(optionValue)
	if !found {
		return nil, fmt.Errorf("option not found: %s for provider %s", optionValue, providerID)
	}

	// Extract settings from option properties
	settings := &ProviderSettings{
		Properties: make(map[string]interface{}),
	}

	// Copy all properties
	if targetOption.Properties != nil {
		for key, value := range targetOption.Properties {
			settings.Properties[key] = value
		}
	}

	// Extract dimension
	if dim, ok := targetOption.Properties["dimensions"]; ok {
		if dimInt, ok := dim.(int); ok {
			settings.Dimension = dimInt
		} else if dimFloat, ok := dim.(float64); ok {
			settings.Dimension = int(dimFloat)
		}
	}

	// Extract connector
	if connector, ok := targetOption.Properties["connector"]; ok {
		if connStr, ok := connector.(string); ok {
			settings.Connector = connStr
		}
	}

	return settings, nil
}

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

	// Get provider settings by provider id and option value
	providerSettings, err := getProviderSettings(req.Config.EmbeddingProvider, req.Config.EmbeddingOption, req.Config.Locale)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: fmt.Sprintf("Failed to resolve provider settings: %v", err),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Set dimension by provider settings and add original provider id and option value to metadata with prefix __
	req.Config.Dimension = providerSettings.Dimension
	if req.Metadata == nil {
		req.Metadata = make(map[string]interface{})
	}
	req.Metadata["__embedding_provider"] = req.Config.EmbeddingProvider
	req.Metadata["__embedding_option"] = req.Config.EmbeddingOption
	if req.Config.Locale != "" {
		req.Metadata["__locale"] = req.Config.Locale
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
		Config:   req.Config.CreateCollectionOptions,
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

// UpdateCollectionMetadata updates the metadata of an existing collection
func UpdateCollectionMetadata(c *gin.Context) {
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

	var req UpdateCollectionMetadataRequest

	// Parse and bind JSON request
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid request format: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Validate request parameters
	if err := validateUpdateCollectionMetadataRequest(&req); err != nil {
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

	// Call the actual UpdateCollectionMetadata method
	err := kb.Instance.UpdateCollectionMetadata(c.Request.Context(), collectionID, req.Metadata)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to update collection metadata: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	successData := gin.H{
		"message":       "Collection metadata updated successfully",
		"collection_id": collectionID,
	}
	response.RespondWithSuccess(c, response.StatusOK, successData)
}

// CreateCollectionRequest represents the request structure for creating a collection
type CreateCollectionRequest struct {
	ID       string                  `json:"id" binding:"required"`
	Metadata map[string]interface{}  `json:"metadata"`
	Config   *CreateCollectionConfig `json:"config" binding:"required"`
}

// CreateCollectionConfig represents the request structure for creating a collection
type CreateCollectionConfig struct {
	EmbeddingProvider string `json:"embedding_provider" binding:"required"` // embedding provider id
	EmbeddingOption   string `json:"embedding_option" binding:"required"`   // embedding option value
	Locale            string `json:"locale,omitempty"`                      // locale for provider reading
	*types.CreateCollectionOptions
}

// UpdateCollectionMetadataRequest represents the request structure for updating collection metadata
type UpdateCollectionMetadataRequest struct {
	Metadata map[string]interface{} `json:"metadata" binding:"required"`
}

// validateCreateCollectionRequest validates the incoming request for creating a collection
func validateCreateCollectionRequest(req *CreateCollectionRequest) error {
	if req.ID == "" {
		return fmt.Errorf("id is required")
	}

	if req.Config == nil {
		return fmt.Errorf("config is required")
	}

	// Validate CreateCollectionOptions (ignore collection name cannot be empty error)
	if err := req.Config.Validate(); err != nil && err.Error() != "collection name cannot be empty" {
		return fmt.Errorf("invalid config: %w", err)
	}

	return nil
}

// validateUpdateCollectionMetadataRequest validates the incoming request for updating collection metadata
func validateUpdateCollectionMetadataRequest(req *UpdateCollectionMetadataRequest) error {
	if req.Metadata == nil {
		return fmt.Errorf("metadata is required")
	}

	if len(req.Metadata) == 0 {
		return fmt.Errorf("metadata cannot be empty")
	}

	return nil
}
