package kb

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/kb"
	kbtypes "github.com/yaoapp/yao/kb/types"
	apiutils "github.com/yaoapp/yao/openapi/utils"
)

// PrepareCreateCollection prepares CreateCollection request and database data
func PrepareCreateCollection(c *gin.Context) (*CreateCollectionRequest, map[string]interface{}, error) {
	var req CreateCollectionRequest

	// Parse and bind JSON request
	if err := c.ShouldBindJSON(&req); err != nil {
		return nil, nil, fmt.Errorf("invalid request format: %w", err)
	}

	// Get provider settings first to resolve dimension
	providerSettings, err := getProviderSettings(req.Config.EmbeddingProviderID, req.Config.EmbeddingOptionID, req.Config.Locale)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve provider settings: %w", err)
	}

	// Set dimension from provider settings
	req.Config.Dimension = providerSettings.Dimension

	// Store embedding properties if available
	var embeddingProperties map[string]interface{} = nil
	if providerSettings.Properties != nil {
		embeddingProperties = providerSettings.Properties
	}

	// Add metadata with provider information
	if req.Metadata == nil {
		req.Metadata = make(map[string]interface{})
	}
	req.Metadata["__embedding_provider"] = req.Config.EmbeddingProviderID
	req.Metadata["__embedding_option"] = req.Config.EmbeddingOptionID

	if embeddingProperties != nil {
		req.Metadata["__embedding_properties"] = embeddingProperties
	}

	if req.Config.Locale != "" {
		req.Metadata["__locale"] = req.Config.Locale
	}

	// Now validate request parameters (after dimension and metadata are set)
	if err := validateCreateCollectionRequest(&req); err != nil {
		return nil, nil, err
	}

	// Prepare collection data for database
	data := map[string]interface{}{
		"collection_id":         req.ID,
		"name":                  req.Metadata["name"],
		"description":           req.Metadata["description"],
		"status":                "creating",
		"embedding_provider_id": req.Config.EmbeddingProviderID,
		"embedding_option_id":   req.Config.EmbeddingOptionID,
		"embedding_properties":  embeddingProperties,
		"locale":                req.Config.Locale,
		"distance":              req.Config.Distance,
		"index_type":            req.Config.IndexType,
	}

	// Add share field from metadata if provided
	share := apiutils.ToString(req.Metadata["share"])
	if share == "private" || share == "team" {
		data["share"] = share
	}

	// Add optional HNSW parameters
	if req.Config.M > 0 {
		data["m"] = req.Config.M
	}
	if req.Config.EfConstruction > 0 {
		data["ef_construction"] = req.Config.EfConstruction
	}
	if req.Config.EfSearch > 0 {
		data["ef_search"] = req.Config.EfSearch
	}

	// Add optional IVF parameters
	if req.Config.NumLists > 0 {
		data["num_lists"] = req.Config.NumLists
	}
	if req.Config.NumProbes > 0 {
		data["num_probes"] = req.Config.NumProbes
	}

	return &req, data, nil
}

// UpdateCollectionWithSync updates collection metadata in database and syncs to GraphRag
func UpdateCollectionWithSync(collectionID string, data maps.MapStrAny, config *kbtypes.Config) error {
	// Create a copy of data for GraphRag to avoid contamination from database operations
	// This is necessary because Gou's UpdateWhere method modifies the input data parameter
	originalData := make(maps.MapStrAny)
	for k, v := range data {
		originalData[k] = v
	}

	// Update collection in database
	if err := config.UpdateCollection(collectionID, data); err != nil {
		return fmt.Errorf("failed to update collection in database: %w", err)
	}

	// Sync to GraphRag metadata if kb.Instance is available
	if kb.Instance != nil {
		// Convert the original (unmodified) data to map[string]interface{}
		metadata := make(map[string]interface{})
		for k, v := range originalData {
			metadata[k] = v
		}

		// Update GraphRag metadata
		ctx := context.Background()
		if err := kb.Instance.UpdateCollectionMetadata(ctx, collectionID, metadata); err != nil {
			return fmt.Errorf("failed to sync collection metadata to GraphRag: %w", err)
		}
	}

	return nil
}

// UpdateDocumentCountWithSync updates document count in database and syncs to GraphRag metadata
func UpdateDocumentCountWithSync(collectionID string, config *kbtypes.Config) error {
	// Update document count in database
	if err := config.UpdateDocumentCount(collectionID); err != nil {
		return fmt.Errorf("failed to update document count in database: %w", err)
	}

	// Sync to GraphRag metadata if kb.Instance is available
	if kb.Instance != nil {
		// Get the updated document count
		count, err := config.DocumentCount(collectionID)
		if err != nil {
			return fmt.Errorf("failed to get document count for sync: %w", err)
		}

		// Prepare metadata for GraphRag
		metadata := map[string]interface{}{
			"document_count": count,
		}

		// Update GraphRag metadata
		ctx := context.Background()
		if err := kb.Instance.UpdateCollectionMetadata(ctx, collectionID, metadata); err != nil {
			return fmt.Errorf("failed to sync document count to GraphRag: %w", err)
		}
	}

	return nil
}
