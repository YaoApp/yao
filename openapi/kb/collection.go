package kb

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/kb"
	kbapi "github.com/yaoapp/yao/kb/api"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/openapi/utils"
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

	// Check if kb.API is available
	if kb.API == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Knowledge base not initialized",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

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
	var authScope map[string]interface{}
	if authInfo != nil {
		collectionData = authInfo.WithCreateScope(collectionData)
		// Extract auth scope fields
		authScope = make(map[string]interface{})
		if createdBy, ok := collectionData["__yao_created_by"]; ok {
			authScope["__yao_created_by"] = createdBy
		}
		if updatedBy, ok := collectionData["__yao_updated_by"]; ok {
			authScope["__yao_updated_by"] = updatedBy
		}
		if teamID, ok := collectionData["__yao_team_id"]; ok {
			authScope["__yao_team_id"] = teamID
		}
	}

	// Build API params
	params := &kbapi.CreateCollectionParams{
		ID:                  req.ID,
		Metadata:            req.Metadata,
		EmbeddingProviderID: req.Config.EmbeddingProviderID,
		EmbeddingOptionID:   req.Config.EmbeddingOptionID,
		Locale:              req.Config.Locale,
		Config:              req.Config.CreateCollectionOptions,
		AuthScope:           authScope,
	}

	// Call API to create collection
	result, err := kb.API.CreateCollection(c.Request.Context(), params)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	successData := gin.H{
		"message":       result.Message,
		"collection_id": result.CollectionID,
	}
	response.RespondWithSuccess(c, response.StatusCreated, successData)
}

// RemoveCollection removes an existing collection
func RemoveCollection(c *gin.Context) {

	// Check if kb.API is available
	if kb.API == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Knowledge base not initialized",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

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

	// Call API to remove collection
	result, err := kb.API.RemoveCollection(c.Request.Context(), collectionID)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	successData := gin.H{
		"message":           result.Message,
		"collection_id":     result.CollectionID,
		"removed":           result.Removed,
		"documents_removed": result.DocumentsRemoved,
	}
	response.RespondWithSuccess(c, response.StatusOK, successData)
}

// CollectionExists checks if a collection exists
func CollectionExists(c *gin.Context) {
	// Check if kb.API is available
	if kb.API == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Knowledge base not initialized",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

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

	// Call API to check collection existence
	result, err := kb.API.CollectionExists(c.Request.Context(), collectionID)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	successData := gin.H{
		"collection_id": result.CollectionID,
		"exists":        result.Exists,
	}
	response.RespondWithSuccess(c, response.StatusOK, successData)
}

// GetCollection retrieves a collection by ID
func GetCollection(c *gin.Context) {
	// Check if kb.API is available
	if kb.API == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Knowledge base not initialized",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	collectionID := c.Param("collectionID")
	if collectionID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Collection ID is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Call API to get collection
	collection, err := kb.API.GetCollection(c.Request.Context(), collectionID)
	if err != nil {
		// Check if it's a "not found" error
		if err.Error() == "collection not found" || err.Error() == fmt.Sprintf("collection with ID '%s' not found", collectionID) {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: "Collection not found",
			}
			response.RespondWithError(c, response.StatusNotFound, errorResp)
			return
		}

		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	response.RespondWithSuccess(c, response.StatusOK, collection)
}

// ListCollections lists collections with pagination
func ListCollections(c *gin.Context) {

	// Check if kb.API is available
	if kb.API == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Knowledge base not initialized",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Get authorized information
	authInfo := authorized.GetInfo(c)

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

	// Parse select parameter
	var selectFields []interface{}
	if selectParam := strings.TrimSpace(c.Query("select")); selectParam != "" {
		requestedFields := strings.Split(selectParam, ",")
		for _, field := range requestedFields {
			field = strings.TrimSpace(field)
			if field != "" && kbapi.AvailableCollectionFields[field] {
				selectFields = append(selectFields, field)
			}
		}
	}

	// Parse sort parameter
	var orders []model.QueryOrder
	if sortParam := strings.TrimSpace(c.Query("sort")); sortParam != "" {
		sortItems := strings.Split(sortParam, ",")
		for _, sortItem := range sortItems {
			sortItem = strings.TrimSpace(sortItem)
			if sortItem == "" {
				continue
			}

			sortParts := strings.Fields(sortItem)
			if len(sortParts) == 0 {
				continue
			}

			sortField := sortParts[0]
			sortOrder := "desc"
			if len(sortParts) >= 2 {
				sortOrder = strings.ToLower(sortParts[1])
			}

			// Validate sort field and order
			if kbapi.ValidCollectionSortFields[sortField] && (sortOrder == "asc" || sortOrder == "desc") {
				orders = append(orders, model.QueryOrder{
					Column: sortField,
					Option: sortOrder,
				})
			}
		}
	}

	// Build filter for API
	filter := &kbapi.ListCollectionsFilter{
		Page:                page,
		PageSize:            pagesize,
		Keywords:            strings.TrimSpace(c.Query("keywords")),
		EmbeddingProviderID: strings.TrimSpace(c.Query("embedding_provider_id")),
		Select:              selectFields,
		Sort:                orders,
		AuthFilters:         AuthFilter(c, authInfo),
	}

	// Parse status parameter
	if statusParam := strings.TrimSpace(c.Query("status")); statusParam != "" {
		statusList := strings.Split(statusParam, ",")
		for _, status := range statusList {
			status = strings.TrimSpace(status)
			if status != "" {
				filter.Status = append(filter.Status, status)
			}
		}
	}

	// Parse system parameter
	if systemParam := strings.TrimSpace(c.Query("system")); systemParam != "" {
		switch systemParam {
		case "true", "1":
			systemVal := true
			filter.System = &systemVal
		case "false", "0":
			systemVal := false
			filter.System = &systemVal
		}
	}

	// Call API to list collections
	result, err := kb.API.ListCollections(c.Request.Context(), filter)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Return the result directly to maintain backward compatibility
	c.JSON(http.StatusOK, gin.H{
		"data":     result.Data,
		"next":     result.Next,
		"prev":     result.Prev,
		"page":     result.Page,
		"pagesize": result.PageSize,
		"total":    result.Total,
		"pagecnt":  result.PageCnt,
	})
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

	// Check if kb.API is available
	if kb.API == nil {
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

	// Build API params
	var authScope map[string]interface{}
	if authInfo != nil {
		authScope = authInfo.WithUpdateScope(maps.MapStrAny{})
	}

	params := &kbapi.UpdateMetadataParams{
		Metadata:  req.Metadata,
		AuthScope: authScope,
	}

	// Call API to update collection metadata
	result, err := kb.API.UpdateCollectionMetadata(c.Request.Context(), collectionID, params)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	successData := gin.H{
		"message":       result.Message,
		"collection_id": result.CollectionID,
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
