package api

import (
	"context"
	"fmt"

	graphragtypes "github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
)

// CreateCollection creates a new collection with the provided parameters
func (instance *KBInstance) CreateCollection(ctx context.Context, params *CreateCollectionParams) (*CreateCollectionResult, error) {

	// Basic validation (before provider settings)
	if params.ID == "" {
		return nil, fmt.Errorf("invalid parameters: id is required")
	}
	if params.EmbeddingProviderID == "" {
		return nil, fmt.Errorf("invalid parameters: embedding_provider_id is required")
	}
	if params.EmbeddingOptionID == "" {
		return nil, fmt.Errorf("invalid parameters: embedding_option_id is required")
	}

	// Get provider settings to resolve dimension and properties
	providerSettings, err := instance.getProviderSettings(params.EmbeddingProviderID, params.EmbeddingOptionID, params.Locale)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve provider settings: %w", err)
	}

	// Set dimension from provider settings
	if params.Config != nil {
		params.Config.Dimension = providerSettings.Dimension
	}

	// Validate full parameters after dimension is set
	if err := validateCreateParams(params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	// Prepare metadata
	metadata := params.Metadata
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	// Add embedding information to metadata
	metadata["__embedding_provider"] = params.EmbeddingProviderID
	metadata["__embedding_option"] = params.EmbeddingOptionID
	if providerSettings.Properties != nil {
		metadata["__embedding_properties"] = providerSettings.Properties
	}
	if params.Locale != "" {
		metadata["__locale"] = params.Locale
	}

	// Prepare database record
	dbData := map[string]interface{}{
		"collection_id":         params.ID,
		"name":                  metadata["name"],
		"description":           metadata["description"],
		"status":                "creating",
		"embedding_provider_id": params.EmbeddingProviderID,
		"embedding_option_id":   params.EmbeddingOptionID,
		"embedding_properties":  providerSettings.Properties,
		"locale":                params.Locale,
	}

	// Add config options to database if provided
	if params.Config != nil {
		if params.Config.Distance != "" {
			dbData["distance"] = params.Config.Distance
		}
		if params.Config.IndexType != "" {
			dbData["index_type"] = params.Config.IndexType
		}
		if params.Config.M > 0 {
			dbData["m"] = params.Config.M
		}
		if params.Config.EfConstruction > 0 {
			dbData["ef_construction"] = params.Config.EfConstruction
		}
		if params.Config.EfSearch > 0 {
			dbData["ef_search"] = params.Config.EfSearch
		}
		if params.Config.NumLists > 0 {
			dbData["num_lists"] = params.Config.NumLists
		}
		if params.Config.NumProbes > 0 {
			dbData["num_probes"] = params.Config.NumProbes
		}
	}

	// Add share field from metadata if provided
	if share, ok := metadata["share"].(string); ok {
		if share == "private" || share == "team" {
			dbData["share"] = share
		}
	}

	// Merge auth scope fields
	if params.AuthScope != nil {
		for k, v := range params.AuthScope {
			dbData[k] = v
		}
	}

	// Create database record first
	_, err = instance.Config.CreateCollection(maps.MapStrAny(dbData))
	if err != nil {
		return nil, fmt.Errorf("failed to save collection metadata: %w", err)
	}

	// Read back the database record to get auto-generated fields (created_at, updated_at)
	dbRecord, err := instance.Config.FindCollection(params.ID, model.QueryParam{})
	if err != nil {
		// Rollback on error
		rollbackErr := instance.Config.RemoveCollection(params.ID)
		if rollbackErr != nil {
			log.Error("Failed to rollback collection database record: %v", rollbackErr)
		}
		return nil, fmt.Errorf("failed to read created collection: %w", err)
	}

	// Add all database fields to metadata for GraphRag
	// This ensures GraphRag metadata contains complete information for vector search filtering

	// Timestamps
	if createdAt, ok := dbRecord["created_at"]; ok {
		metadata["created_at"] = createdAt
		// If updated_at is not set, use created_at (for newly created records)
		if updatedAt, ok := dbRecord["updated_at"]; ok && updatedAt != nil {
			metadata["updated_at"] = updatedAt
		} else {
			metadata["updated_at"] = createdAt
		}
	}

	// Auth scope fields (for permission-based vector search)
	if createdBy, ok := dbRecord["__yao_created_by"]; ok && createdBy != nil {
		metadata["__yao_created_by"] = createdBy
	}
	if teamID, ok := dbRecord["__yao_team_id"]; ok && teamID != nil {
		metadata["__yao_team_id"] = teamID
	}
	if tenantID, ok := dbRecord["__yao_tenant_id"]; ok && tenantID != nil {
		metadata["__yao_tenant_id"] = tenantID
	}

	// Collection ID (for consistency with OpenAPI created collections)
	metadata["collection_id"] = params.ID

	// Collection properties
	if share, ok := dbRecord["share"]; ok && share != nil {
		metadata["share"] = share
	}
	if preset, ok := dbRecord["preset"]; ok {
		metadata["preset"] = preset
	}
	if public, ok := dbRecord["public"]; ok {
		metadata["public"] = public
	}
	if sort, ok := dbRecord["sort"]; ok {
		metadata["sort"] = sort
	}
	if status, ok := dbRecord["status"]; ok && status != nil {
		metadata["status"] = status
	}
	if uid, ok := dbRecord["uid"]; ok {
		metadata["uid"] = uid
	}
	if cover, ok := dbRecord["cover"]; ok {
		metadata["cover"] = cover
	}
	if documentCount, ok := dbRecord["document_count"]; ok {
		metadata["document_count"] = documentCount
	}

	collectionConfig := graphragtypes.CollectionConfig{
		ID:       params.ID,
		Metadata: metadata,
		Config:   params.Config,
	}

	// Create collection in GraphRag
	collectionID, err := instance.GraphRag.CreateCollection(ctx, collectionConfig)
	if err != nil {
		// Rollback: remove the database record
		rollbackErr := instance.Config.RemoveCollection(params.ID)
		if rollbackErr != nil {
			log.Error("Failed to rollback collection database record: %v", rollbackErr)
		}
		return nil, fmt.Errorf("failed to create collection: %w", err)
	}

	// Update status to active after successful creation
	updateErr := instance.updateCollectionWithSync(ctx, params.ID, maps.MapStrAny{"status": "active"})
	if updateErr != nil {
		log.Error("Failed to update collection status to active: %v", updateErr)
	}

	return &CreateCollectionResult{
		CollectionID: collectionID,
		Message:      "Collection created successfully",
	}, nil
}

// RemoveCollection removes an existing collection by ID
func (instance *KBInstance) RemoveCollection(ctx context.Context, collectionID string) (*RemoveCollectionResult, error) {

	if collectionID == "" {
		return nil, fmt.Errorf("collection ID is required")
	}

	// Try to remove from GraphRag (vector/graph stores)
	// Don't fail if collection doesn't exist there - we still want to clean up database
	removed := false
	graphRagErr := error(nil)

	removedFromGraphRag, err := instance.GraphRag.RemoveCollection(ctx, collectionID)
	if err != nil {
		// Log the error but continue to database cleanup
		log.Warn("Failed to remove collection from GraphRag: %v (will continue with database cleanup)", err)
		graphRagErr = err
	} else {
		removed = removedFromGraphRag
	}

	// Always attempt to clean up database, even if GraphRag removal failed
	// This ensures we can recover from inconsistent states
	documentsRemoved := 0

	// Count documents in this collection
	if count, err := instance.Config.DocumentCount(collectionID); err == nil {
		documentsRemoved = count
	}

	// Remove all documents belonging to this collection
	dbCleanupSuccess := true
	if err := instance.Config.RemoveDocumentsByCollectionID(collectionID); err != nil {
		log.Error("Failed to remove documents from collection %s: %v", collectionID, err)
		dbCleanupSuccess = false
	} else {
		log.Info("Removed %d documents from collection %s", documentsRemoved, collectionID)
	}

	// Remove the collection itself from database
	if err := instance.Config.RemoveCollection(collectionID); err != nil {
		log.Error("Failed to remove collection from database: %v", err)
		dbCleanupSuccess = false
	} else {
		log.Info("Successfully removed collection %s and %d documents from database", collectionID, documentsRemoved)
	}

	// Determine final result and error
	// If both GraphRag and database cleanup failed, return error
	if graphRagErr != nil && !dbCleanupSuccess {
		return nil, fmt.Errorf("failed to remove collection: GraphRag error: %v", graphRagErr)
	}

	// If collection didn't exist in GraphRag but was cleaned from database, still consider it successful
	if !removed && dbCleanupSuccess {
		log.Info("Collection %s was not found in GraphRag but was cleaned from database", collectionID)
	}

	return &RemoveCollectionResult{
		CollectionID:     collectionID,
		Removed:          removed || dbCleanupSuccess, // Consider successful if either succeeded
		DocumentsRemoved: documentsRemoved,
		Message:          "Collection removed successfully",
	}, nil
}

// GetCollection retrieves a collection by ID
// Reads from database first, then merges with GraphRag metadata
func (instance *KBInstance) GetCollection(ctx context.Context, collectionID string) (map[string]interface{}, error) {

	if collectionID == "" {
		return nil, fmt.Errorf("collection ID is required")
	}

	// Read from database (source of truth for existence and permissions)
	dbRecord, err := instance.Config.FindCollection(collectionID, model.QueryParam{})
	if err != nil {
		return nil, fmt.Errorf("collection not found")
	}

	// Convert database record to result map (flatten to top level)
	result := make(map[string]interface{})
	for k, v := range dbRecord {
		result[k] = v
	}

	// Set standard ID fields
	result["id"] = collectionID
	result["collection_id"] = collectionID

	// Read from GraphRag and merge (for config and metadata object)
	graphRagCollection, err := instance.GraphRag.GetCollection(ctx, collectionID)
	if err == nil && graphRagCollection != nil {
		// Set GraphRag config (vector store configuration)
		if graphRagCollection.Config != nil {
			result["config"] = graphRagCollection.Config
		}

		// Set GraphRag metadata as nested object (for backward compatibility)
		// This allows access via collection["metadata"]["field"]
		if graphRagCollection.Metadata != nil {
			result["metadata"] = graphRagCollection.Metadata

			// Also flatten GraphRag metadata fields to top level
			// Only add fields that don't exist in database record
			for k, v := range graphRagCollection.Metadata {
				if _, exists := result[k]; !exists {
					result[k] = v
				}
			}
		}
	}

	return result, nil
}

// CollectionExists checks if a collection exists by ID
// Checks both database and GraphRag for consistency
func (instance *KBInstance) CollectionExists(ctx context.Context, collectionID string) (*CollectionExistsResult, error) {

	if collectionID == "" {
		return nil, fmt.Errorf("collection ID is required")
	}

	// Check database (source of truth for existence)
	_, dbErr := instance.Config.FindCollection(collectionID, model.QueryParam{})
	dbExists := dbErr == nil

	// Check GraphRag for consistency
	graphRagExists, _ := instance.GraphRag.CollectionExists(ctx, collectionID)

	// Collection exists if it exists in database
	// Log warning if there's inconsistency (for debugging)
	if dbExists != graphRagExists {
		log.Warn("Collection %s existence mismatch: database=%v, graphrag=%v", collectionID, dbExists, graphRagExists)
	}

	return &CollectionExistsResult{
		CollectionID: collectionID,
		Exists:       dbExists,
	}, nil
}

// ListCollections lists collections with pagination and filtering
func (instance *KBInstance) ListCollections(ctx context.Context, filter *ListCollectionsFilter) (*ListCollectionsResult, error) {

	page := filter.Page
	if page <= 0 {
		page = DefaultPage
	}

	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	} else if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	// Process select fields
	selectFields := filter.Select
	if len(selectFields) == 0 {
		selectFields = DefaultCollectionFields
	} else {
		// Filter valid fields
		validFields := []interface{}{}
		for _, field := range selectFields {
			if fieldStr, ok := field.(string); ok && AvailableCollectionFields[fieldStr] {
				validFields = append(validFields, field)
			}
		}
		if len(validFields) == 0 {
			selectFields = DefaultCollectionFields
		} else {
			selectFields = validFields
		}
	}

	// Build query parameters
	param := model.QueryParam{Select: selectFields}

	// Build wheres
	var wheres []model.QueryWhere

	// Add auth filters
	if len(filter.AuthFilters) > 0 {
		wheres = append(wheres, filter.AuthFilters...)
	}

	// Filter by keywords (search in name and description)
	if filter.Keywords != "" {
		wheres = append(wheres, model.QueryWhere{
			Column: "name",
			Value:  "%" + filter.Keywords + "%",
			OP:     "like",
		})
		wheres = append(wheres, model.QueryWhere{
			Column: "description",
			Value:  "%" + filter.Keywords + "%",
			OP:     "like",
			Method: "orwhere",
		})
	}

	// Filter by status
	if len(filter.Status) > 0 {
		statusValues := []interface{}{}
		for _, status := range filter.Status {
			if status != "" {
				statusValues = append(statusValues, status)
			}
		}

		if len(statusValues) > 0 {
			if len(statusValues) == 1 {
				wheres = append(wheres, model.QueryWhere{
					Column: "status",
					Value:  statusValues[0],
				})
			} else {
				wheres = append(wheres, model.QueryWhere{
					Column: "status",
					Value:  statusValues,
					OP:     "in",
				})
			}
		}
	}

	// Filter by system flag
	if filter.System != nil {
		wheres = append(wheres, model.QueryWhere{
			Column: "system",
			Value:  *filter.System,
		})
	}

	// Filter by embedding_provider_id
	if filter.EmbeddingProviderID != "" {
		wheres = append(wheres, model.QueryWhere{
			Column: "embedding_provider_id",
			Value:  filter.EmbeddingProviderID,
		})
	}

	param.Wheres = wheres

	// Process sort orders
	orders := filter.Sort
	if len(orders) == 0 {
		orders = DefaultSort
	} else {
		// Validate sort fields
		validOrders := []model.QueryOrder{}
		for _, order := range orders {
			if ValidCollectionSortFields[order.Column] {
				validOrders = append(validOrders, order)
			}
		}
		if len(validOrders) == 0 {
			orders = DefaultSort
		} else {
			orders = validOrders
		}
	}

	param.Orders = orders

	// Query collections
	result, err := instance.Config.SearchCollections(param, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to search collections: %w", err)
	}

	// Convert maps.MapStr result to ListCollectionsResult
	listResult := &ListCollectionsResult{
		Page:     page,
		PageSize: pageSize,
		Data:     make([]map[string]interface{}, 0), // Initialize as empty array, not nil
	}

	// Extract pagination data from result
	if data, ok := result["data"].([]map[string]interface{}); ok {
		listResult.Data = data
	} else if data, ok := result["data"].([]interface{}); ok {
		// Convert []interface{} to []map[string]interface{}
		converted := make([]map[string]interface{}, 0, len(data))
		for _, item := range data {
			if mapItem, ok := item.(map[string]interface{}); ok {
				converted = append(converted, mapItem)
			}
		}
		listResult.Data = converted
	} else if data, ok := result["data"].([]maps.MapStr); ok {
		// Handle []maps.MapStr type (most likely from model.Paginate)
		converted := make([]map[string]interface{}, 0, len(data))
		for _, item := range data {
			converted = append(converted, map[string]interface{}(item))
		}
		listResult.Data = converted
	}

	if next, ok := result["next"].(int); ok {
		listResult.Next = next
	}
	if prev, ok := result["prev"].(int); ok {
		listResult.Prev = prev
	}
	if total, ok := result["total"].(int); ok {
		listResult.Total = total
	}
	if pagecnt, ok := result["pagecnt"].(int); ok {
		listResult.PageCnt = pagecnt
	}

	return listResult, nil
}

// UpdateCollectionMetadata updates the metadata of an existing collection
func (instance *KBInstance) UpdateCollectionMetadata(ctx context.Context, collectionID string, params *UpdateMetadataParams) (*UpdateMetadataResult, error) {

	if collectionID == "" {
		return nil, fmt.Errorf("collection ID is required")
	}

	if len(params.Metadata) == 0 {
		return nil, fmt.Errorf("metadata is required and cannot be empty")
	}

	err := instance.GraphRag.UpdateCollectionMetadata(ctx, collectionID, params.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to update collection metadata: %w", err)
	}

	// Update collection metadata in database after successful GraphRag update
	// Prepare update data from metadata
	updateData := maps.MapStrAny{}
	if name, ok := params.Metadata["name"]; ok {
		updateData["name"] = name
	}
	if description, ok := params.Metadata["description"]; ok {
		updateData["description"] = description
	}
	if status, ok := params.Metadata["status"]; ok {
		updateData["status"] = status
	}

	// Merge auth scope fields
	if params.AuthScope != nil {
		for k, v := range params.AuthScope {
			updateData[k] = v
		}
	}

	if len(updateData) > 0 {
		// Only update database, don't sync to GraphRag again
		if err := instance.Config.UpdateCollection(collectionID, updateData); err != nil {
			log.Error("Failed to update collection in database: %v", err)
		}
	}

	return &UpdateMetadataResult{
		CollectionID: collectionID,
		Message:      "Collection metadata updated successfully",
	}, nil
}

// Helper methods

// validateCreateParams validates the create collection parameters
func validateCreateParams(params *CreateCollectionParams) error {
	if params.ID == "" {
		return fmt.Errorf("id is required")
	}

	if params.EmbeddingProviderID == "" {
		return fmt.Errorf("embedding_provider_id is required")
	}

	if params.EmbeddingOptionID == "" {
		return fmt.Errorf("embedding_option_id is required")
	}

	// Validate CreateCollectionOptions if provided
	if params.Config != nil {
		if err := params.Config.Validate(); err != nil && err.Error() != "collection name cannot be empty" {
			return fmt.Errorf("invalid config: %w", err)
		}
	}

	return nil
}

// ProviderSettings represents the resolved provider configuration
type ProviderSettings struct {
	Dimension  int                    `json:"dimension"`
	Connector  string                 `json:"connector"`
	Properties map[string]interface{} `json:"properties"`
}

// getProviderSettings reads and resolves provider settings by provider ID and option value
func (instance *KBInstance) getProviderSettings(providerID, optionValue, locale string) (*ProviderSettings, error) {
	// Default locale to "en" if empty
	if locale == "" {
		locale = DefaultLocale
	}

	// Get the specific provider from instance
	provider, err := instance.Providers.GetProvider("embedding", providerID, locale)
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

// updateCollectionWithSync updates collection metadata in database and syncs to GraphRag
func (instance *KBInstance) updateCollectionWithSync(ctx context.Context, collectionID string, data maps.MapStrAny) error {
	// Create a copy of data for GraphRag to avoid contamination from database operations
	originalData := make(maps.MapStrAny)
	for k, v := range data {
		originalData[k] = v
	}

	// Update collection in database
	if err := instance.Config.UpdateCollection(collectionID, data); err != nil {
		return fmt.Errorf("failed to update collection in database: %w", err)
	}

	// Sync to GraphRag metadata
	// Convert the original (unmodified) data to map[string]interface{}
	metadata := make(map[string]interface{})
	for k, v := range originalData {
		metadata[k] = v
	}

	// Update GraphRag metadata
	if err := instance.GraphRag.UpdateCollectionMetadata(ctx, collectionID, metadata); err != nil {
		return fmt.Errorf("failed to sync collection metadata to GraphRag: %w", err)
	}

	return nil
}
