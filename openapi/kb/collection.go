package kb

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
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

// CreateCollection creates a new collection
func CreateCollection(c *gin.Context) {
	// Prepare request and database data
	req, collectionData, err := PrepareCreateCollection(c)
	if err != nil {
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

	// Get KB config
	config, err := kb.GetConfig()
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to get KB config: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// First create database record
	_, err = config.CreateCollection(maps.MapStrAny(collectionData))
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to save collection metadata: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Create CollectionConfig for GraphRag
	collectionConfig := types.CollectionConfig{
		ID:       req.ID,
		Metadata: req.Metadata,
		Config:   req.Config.CreateCollectionOptions,
	}

	// Call the actual CreateCollection method
	collectionID, err := kb.Instance.CreateCollection(c.Request.Context(), collectionConfig)
	if err != nil {
		// Rollback: remove the database record
		rollbackErr := config.RemoveCollection(req.ID)
		if rollbackErr != nil {
			log.Error("Failed to rollback collection database record: %v", rollbackErr)
		}

		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to create collection: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Update status to active after successful creation
	updateErr := config.UpdateCollection(req.ID, maps.MapStrAny{"status": "active"})
	if updateErr != nil {
		log.Error("Failed to update collection status to active: %v", updateErr)
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

	// Remove collection from database after successful GraphRag removal
	if config, err := kb.GetConfig(); err == nil {
		if err := config.RemoveCollection(collectionID); err != nil {
			log.Error("Failed to remove collection from database: %v", err)
		}
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

// GetCollection retrieves a collection by ID
func GetCollection(c *gin.Context) {
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

	// Use the dedicated GetCollection method
	collection, err := kb.Instance.GetCollection(c.Request.Context(), collectionID)
	if err != nil {
		// Check if it's a "not found" error
		if err.Error() == fmt.Sprintf("collection with ID '%s' not found", collectionID) {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: "Collection not found",
			}
			response.RespondWithError(c, response.StatusNotFound, errorResp)
			return
		}

		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to get collection: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	response.RespondWithSuccess(c, response.StatusOK, collection)
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

	// Update collection metadata in database after successful GraphRag update
	if config, err := kb.GetConfig(); err == nil {
		// Prepare update data from metadata
		updateData := maps.MapStrAny{}
		if name, ok := req.Metadata["name"]; ok {
			updateData["name"] = name
		}
		if description, ok := req.Metadata["description"]; ok {
			updateData["description"] = description
		}
		if status, ok := req.Metadata["status"]; ok {
			updateData["status"] = status
		}

		if len(updateData) > 0 {
			if err := config.UpdateCollection(collectionID, updateData); err != nil {
				log.Error("Failed to update collection in database: %v", err)
			}
		}
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
	EmbeddingProviderID string `json:"embedding_provider_id" binding:"required"` // embedding provider id
	EmbeddingOptionID   string `json:"embedding_option_id" binding:"required"`   // embedding option id
	Locale              string `json:"locale,omitempty"`                         // locale for provider reading
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
