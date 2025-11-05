package kb

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/kb"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/openapi/utils"
)

// Collection Management Handlers

// Collection field definitions
var (
	// availableCollectionFields defines all available fields for security filtering
	availableCollectionFields = map[string]bool{
		"id": true, "collection_id": true, "name": true, "description": true,
		"status": true, "preset": true, "public": true, "share": true, "sort": true, "cover": true,
		"document_count": true, "embedding_provider_id": true, "embedding_option_id": true,
		"embedding_properties": true, "locale": true, "dimension": true,
		"distance_metric": true, "hnsw_m": true, "ef_construction": true,
		"ef_search": true, "num_lists": true, "num_probes": true,
		"created_at": true, "updated_at": true,
	}

	// defaultCollectionFields defines the default compact field list
	defaultCollectionFields = []interface{}{
		"id", "collection_id", "name", "description", "status", "preset", "public", "share",
		"sort", "cover", "document_count", "embedding_provider_id", "embedding_option_id",
		"locale", "dimension", "distance_metric", "created_at", "updated_at",
	}

	// validCollectionSortFields defines valid fields for sorting
	validCollectionSortFields = map[string]bool{
		"created_at":     true,
		"updated_at":     true,
		"name":           true,
		"sort":           true,
		"document_count": true,
		"status":         true,
	}
)

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

	// Attach create scope to the collection data
	authInfo := authorized.GetInfo(c)
	if authInfo != nil {
		collectionData = authInfo.WithCreateScope(collectionData)
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

	// Update status to active after successful creation and sync to GraphRag
	updateErr := UpdateCollectionWithSync(req.ID, maps.MapStrAny{"status": "active"}, config)
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

	authInfo := authorized.GetInfo(c)

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

	// Check remove permission
	hasPermission, err := checkCollectionPermission(authInfo, collectionID)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: err.Error(),
		}
		response.RespondWithError(c, response.StatusForbidden, errorResp)
		return
	}

	// 403 Forbidden
	if !hasPermission {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrAccessDenied.Code,
			ErrorDescription: "Forbidden: No permission to remove collection",
		}
		response.RespondWithError(c, response.StatusForbidden, errorResp)
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

	// Remove collection and all its documents from database after successful GraphRag removal
	documentsRemoved := 0
	if config, err := kb.GetConfig(); err == nil {
		// First, count documents in this collection (for reporting)
		if count, err := config.DocumentCount(collectionID); err == nil {
			documentsRemoved = count
		}

		// Remove all documents belonging to this collection
		if err := config.RemoveDocumentsByCollectionID(collectionID); err != nil {
			log.Error("Failed to remove documents from collection %s: %v", collectionID, err)
		} else {
			log.Info("Removed %d documents from collection %s", documentsRemoved, collectionID)
		}

		// Then remove the collection itself
		if err := config.RemoveCollection(collectionID); err != nil {
			log.Error("Failed to remove collection from database: %v", err)
		} else {
			log.Info("Successfully removed collection %s and %d documents", collectionID, documentsRemoved)
		}
	}

	successData := gin.H{
		"message":           "Collection removed successfully",
		"collection_id":     collectionID,
		"removed":           removed,
		"documents_removed": documentsRemoved,
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

// ListCollections lists collections with pagination
func ListCollections(c *gin.Context) {

	// Get authorized information
	authInfo := authorized.GetInfo(c)

	// Check if kb.Instance is available
	if kb.Instance == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Knowledge base not initialized",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Parse pagination parameters
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	pagesize := 20
	if pagesizeStr := c.Query("pagesize"); pagesizeStr != "" {
		if ps, err := strconv.Atoi(pagesizeStr); err == nil && ps > 0 && ps <= 100 {
			pagesize = ps
		}
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

	// Parse select parameter
	var selectFields []interface{}
	if selectParam := strings.TrimSpace(c.Query("select")); selectParam != "" {
		requestedFields := strings.Split(selectParam, ",")
		for _, field := range requestedFields {
			field = strings.TrimSpace(field)
			if field != "" && availableCollectionFields[field] {
				selectFields = append(selectFields, field)
			}
		}
		// If no valid fields found, use default
		if len(selectFields) == 0 {
			selectFields = defaultCollectionFields
		}
	} else {
		selectFields = defaultCollectionFields
	}

	// Build query parameters
	param := model.QueryParam{Select: selectFields}

	// Add filters
	var wheres []model.QueryWhere

	// Apply permission-based filtering
	wheres = append(wheres, AuthFilter(c, authInfo)...)

	// Filter by keywords (search in name and description)
	if keywords := strings.TrimSpace(c.Query("keywords")); keywords != "" {
		wheres = append(wheres, model.QueryWhere{
			Column: "name",
			Value:  "%" + keywords + "%",
			OP:     "like",
		})
		wheres = append(wheres, model.QueryWhere{
			Column: "description",
			Value:  "%" + keywords + "%",
			OP:     "like",
			Wheres: []model.QueryWhere{},
			Method: "orwhere",
		})
	}

	// Filter by status (support multiple values separated by comma)
	if statusParam := strings.TrimSpace(c.Query("status")); statusParam != "" {
		statusList := strings.Split(statusParam, ",")
		var statusValues []interface{}
		for _, status := range statusList {
			status = strings.TrimSpace(status)
			if status != "" {
				statusValues = append(statusValues, status)
			}
		}

		if len(statusValues) > 0 {
			if len(statusValues) == 1 {
				// Single status
				wheres = append(wheres, model.QueryWhere{
					Column: "status",
					Value:  statusValues[0],
				})
			} else {
				// Multiple status - use IN clause
				wheres = append(wheres, model.QueryWhere{
					Column: "status",
					Value:  statusValues,
					OP:     "in",
				})
			}
		}
	}

	// Filter by system flag
	if systemParam := strings.TrimSpace(c.Query("system")); systemParam != "" {
		switch systemParam {
		case "true", "1":
			wheres = append(wheres, model.QueryWhere{
				Column: "system",
				Value:  true,
			})
		case "false", "0":
			wheres = append(wheres, model.QueryWhere{
				Column: "system",
				Value:  false,
			})
		}
	}

	// Filter by embedding_provider_id
	if providerID := strings.TrimSpace(c.Query("embedding_provider_id")); providerID != "" {
		wheres = append(wheres, model.QueryWhere{
			Column: "embedding_provider_id",
			Value:  providerID,
		})
	}

	param.Wheres = wheres

	// Add ordering
	sortParam := strings.TrimSpace(c.Query("sort"))
	if sortParam == "" {
		sortParam = "created_at desc" // Default sort
	}

	// Parse sort parameter (format: "field1 direction1,field2 direction2")
	var orders []model.QueryOrder
	sortItems := strings.Split(sortParam, ",")

	for _, sortItem := range sortItems {
		sortItem = strings.TrimSpace(sortItem)
		if sortItem == "" {
			continue
		}

		// Parse each sort item (format: "field direction")
		sortParts := strings.Fields(sortItem)
		sortField := "created_at" // Default field
		sortOrder := "desc"       // Default order

		if len(sortParts) >= 1 {
			sortField = sortParts[0]
		}
		if len(sortParts) >= 2 {
			sortOrder = strings.ToLower(sortParts[1])
		}

		// Validate sort field
		if !validCollectionSortFields[sortField] {
			continue // Skip invalid fields
		}

		// Validate sort order
		if sortOrder != "asc" && sortOrder != "desc" {
			sortOrder = "desc" // Default order
		}

		orders = append(orders, model.QueryOrder{
			Column: sortField,
			Option: sortOrder,
		})
	}

	// If no valid orders found, use default
	if len(orders) == 0 {
		orders = []model.QueryOrder{
			{Column: "created_at", Option: "desc"},
		}
	}

	param.Orders = orders

	// Query collections using KB config
	result, err := config.SearchCollections(param, page, pagesize)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to search collections: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	c.JSON(http.StatusOK, result)
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

	// Check update permission
	authInfo := authorized.GetInfo(c)
	hasPermission, err := checkCollectionPermission(authInfo, collectionID)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: err.Error(),
		}
		response.RespondWithError(c, response.StatusForbidden, errorResp)
		return
	}

	// 403 Forbidden
	if !hasPermission {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrAccessDenied.Code,
			ErrorDescription: "Forbidden: No permission to update collection",
		}
		response.RespondWithError(c, response.StatusForbidden, errorResp)
		return
	}

	// Call the actual UpdateCollectionMetadata method
	err = kb.Instance.UpdateCollectionMetadata(c.Request.Context(), collectionID, req.Metadata)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to update collection metadata: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Update collection metadata in database after successful GraphRag update
	// Note: Only update database here, don't sync to GraphRag again (already done above)
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

		// Update __yao_updated_by
		updateData = authInfo.WithUpdateScope(updateData)
		if len(updateData) > 0 {
			// Only update database, don't sync to GraphRag again to avoid duplicate updates
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

// checkCollectionPermission checks if the user has permission to access the collection
func checkCollectionPermission(authInfo *oauthtypes.AuthorizedInfo, collectionID string, readable ...bool) (bool, error) {

	// Team Permission validation)
	if authInfo == nil {
		return true, nil
	}

	// No constraints, allow access
	if !authInfo.Constraints.TeamOnly && !authInfo.Constraints.OwnerOnly {
		return true, nil
	}

	// Get KB config
	config, err := kb.GetConfig()
	if err != nil {
		return false, fmt.Errorf("failed to get KB config: %v", err)
	}

	collection, err := config.FindCollection(collectionID, model.QueryParam{
		Select: []interface{}{"collection_id", "__yao_created_by", "__yao_updated_by", "__yao_team_id", "public", "share"},
		Wheres: []model.QueryWhere{
			{Column: "collection_id", Value: collectionID},
		},
		Limit: 1,
	})

	if len(collection) == 0 {
		return false, fmt.Errorf("collection not found: %s", collectionID)
	}

	if err != nil {
		return false, fmt.Errorf("failed to find collection: %v", err)
	}

	// if readable is true, check if the collection is readable
	if len(readable) > 0 && readable[0] {
		if utils.ToBool(collection["public"]) {
			return true, nil
		}

		// Team only permission validation
		if collection["share"] == "team" && authInfo.Constraints.TeamOnly {
			return true, nil
		}
	}

	// Combined Team and Owner permission validation
	if authInfo.Constraints.TeamOnly && authInfo.Constraints.OwnerOnly {
		if collection["__yao_created_by"] == authInfo.UserID && collection["__yao_team_id"] == authInfo.TeamID {
			return true, nil
		}
	}

	// Owner only permission validation
	if authInfo.Constraints.OwnerOnly && collection["__yao_created_by"] == authInfo.UserID {
		return true, nil
	}

	// Team only permission validation
	if authInfo.Constraints.TeamOnly && collection["__yao_team_id"] == authInfo.TeamID {
		return true, nil
	}

	return false, fmt.Errorf("no permission to access collection: %s", collectionID)
}
