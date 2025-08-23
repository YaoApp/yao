package kb

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/graphrag/utils"
	"github.com/yaoapp/yao/attachment"
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

	// Add context fields (permissions, user info, etc.)
	addContextFields(c, data)

	return &req, data, nil
}

// PrepareAddFile prepares AddFile request and database data
func PrepareAddFile(c *gin.Context, req *AddFileRequest) (*AddFileRequest, map[string]interface{}, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, nil, err
	}

	// Validate file and get path
	path, contentType, err := validateFileAndGetPath(c, req)
	if err != nil {
		return nil, nil, err
	}

	// Get file info
	m, _ := attachment.Managers[req.Uploader]
	fileInfo, _ := m.Info(c.Request.Context(), req.FileID)

	// Generate document ID if not provided
	if req.DocID == "" {
		req.DocID = utils.GenDocIDWithCollectionID(req.CollectionID)
	}

	// Prepare document data for database
	data := map[string]interface{}{
		"document_id":    req.DocID,
		"collection_id":  req.CollectionID,
		"name":           fileInfo.Filename,
		"type":           "file",
		"status":         "pending",
		"uploader_id":    req.Uploader,
		"file_name":      fileInfo.Filename,
		"file_path":      path,
		"file_mime_type": contentType,
		"size":           int64(fileInfo.Bytes),
	}

	req.BaseUpsertRequest.AddBaseFields(data)
	addContextFields(c, data)

	return req, data, nil
}

// PrepareAddText prepares AddText request and database data
func PrepareAddText(c *gin.Context, req *AddTextRequest) (*AddTextRequest, map[string]interface{}, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, nil, err
	}

	// Generate document ID if not provided
	if req.DocID == "" {
		req.DocID = utils.GenDocIDWithCollectionID(req.CollectionID)
	}

	// Prepare document data for database
	data := map[string]interface{}{
		"document_id":   req.DocID,
		"collection_id": req.CollectionID,
		"name":          "Text Document",
		"type":          "text",
		"status":        "pending",
		"text_content":  req.Text,
		"size":          int64(len(req.Text)),
	}

	// Use title from metadata if available
	if req.Metadata != nil {
		if title, ok := req.Metadata["title"].(string); ok && title != "" {
			data["name"] = title
		}
	}

	req.BaseUpsertRequest.AddBaseFields(data)
	addContextFields(c, data)

	return req, data, nil
}

// PrepareAddURL prepares AddURL request and database data
func PrepareAddURL(c *gin.Context, req *AddURLRequest) (*AddURLRequest, map[string]interface{}, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, nil, err
	}

	// Generate document ID if not provided
	if req.DocID == "" {
		req.DocID = utils.GenDocIDWithCollectionID(req.CollectionID)
	}

	// Prepare document data for database
	data := map[string]interface{}{
		"document_id":   req.DocID,
		"collection_id": req.CollectionID,
		"name":          req.URL,
		"type":          "url",
		"status":        "pending",
		"url":           req.URL,
	}

	// Use title from metadata if available
	if req.Metadata != nil {
		if title, ok := req.Metadata["title"].(string); ok && title != "" {
			data["name"] = title
			data["url_title"] = title
		}
	}

	req.BaseUpsertRequest.AddBaseFields(data)
	addContextFields(c, data)

	return req, data, nil
}

// addContextFields adds context-specific fields like permissions, user info
func addContextFields(c *gin.Context, data map[string]interface{}) {
	// TODO: Add permission-related fields from Guard
	// Example: data["user_id"] = c.GetString("user_id")
	// Example: data["permissions"] = c.Get("permissions")
	// Example: data["tenant_id"] = c.GetString("tenant_id")
}
