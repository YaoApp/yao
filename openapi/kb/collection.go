package kb

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/yao/kb"
)

// Collection Management Handlers

// CreateCollection creates a new collection
func CreateCollection(c *gin.Context) {
	var req CreateCollectionRequest

	// Parse and bind JSON request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format: " + err.Error()})
		return
	}

	// Validate request parameters
	if err := validateCreateCollectionRequest(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if kb.Instance is available
	if kb.Instance == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Knowledge base not initialized"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create collection: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":       "Collection created successfully",
		"collection_id": collectionID,
	})
}

// RemoveCollection removes an existing collection
func RemoveCollection(c *gin.Context) {
	// TODO: Implement remove collection logic
	c.JSON(http.StatusOK, gin.H{"message": "Collection removed"})
}

// CollectionExists checks if a collection exists
func CollectionExists(c *gin.Context) {
	// TODO: Implement collection exists check logic
	c.JSON(http.StatusOK, gin.H{"exists": false})
}

// GetCollections retrieves collections with optional filtering
func GetCollections(c *gin.Context) {
	collections, err := kb.Instance.GetCollections(c.Request.Context(), map[string]interface{}{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, collections)
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
