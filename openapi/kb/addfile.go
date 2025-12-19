package kb

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/graphrag/utils"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/attachment"
	"github.com/yaoapp/yao/kb"
	kbapi "github.com/yaoapp/yao/kb/api"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
)

// AddFile adds a file to a collection (sync)
func AddFile(c *gin.Context) {
	var req AddFileRequest

	// Check if kb.API is available
	if !checkKBAPI(c) {
		return
	}

	// Parse and bind JSON request
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid request format: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Generate document ID if not provided
	if req.DocID == "" {
		req.DocID = utils.GenDocIDWithCollectionID(req.CollectionID)
	}

	// Check collection permission
	authInfo := authorized.GetInfo(c)
	hasPermission, err := checkCollectionPermission(authInfo, req.CollectionID)
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

	// Convert request to API params
	params := convertAddFileRequest(&req, authInfo)

	// Call kb.API
	result, err := kb.API.AddFile(c.Request.Context(), params)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Return success response
	response.RespondWithSuccess(c, response.StatusCreated, result)
}

// AddFileAsync adds file to a collection asynchronously
func AddFileAsync(c *gin.Context) {
	var req AddFileRequest

	log.Info("AddFileAsync: Starting async file addition")

	// Check if kb.API is available
	if !checkKBAPI(c) {
		log.Error("AddFileAsync: KB API check failed")
		return
	}

	// Parse and bind JSON request
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("AddFileAsync: JSON binding failed: %v", err)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid request format: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	log.Info("AddFileAsync: Request parsed successfully")

	// Validate request
	if err := req.Validate(); err != nil {
		log.Error("AddFileAsync: Request validation failed: %v", err)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	log.Info("AddFileAsync: Request validation passed")

	// Validate file exists
	if err := validateFileExists(c, &req); err != nil {
		log.Error("AddFileAsync: File validation failed: %v", err)
		return
	}

	log.Info("AddFileAsync: File validation passed")

	// Generate document ID if not provided
	if req.DocID == "" {
		req.DocID = utils.GenDocIDWithCollectionID(req.CollectionID)
	}

	log.Info("AddFileAsync: Generated doc_id: %s", req.DocID)

	// Check collection permission
	authInfo := authorized.GetInfo(c)
	hasPermission, err := checkCollectionPermission(authInfo, req.CollectionID)
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

	// Convert request to API params
	params := convertAddFileRequest(&req, authInfo)

	// Call kb.API async
	result, err := kb.API.AddFileAsync(c.Request.Context(), params)
	if err != nil {
		log.Error("AddFileAsync: Failed to add file async: %v", err)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	log.Info("AddFileAsync: Job created with ID: %s", result.JobID)

	// Return job_id and doc_id
	response.RespondWithSuccess(c, response.StatusCreated, result)
}

// ProcessAddFile documents.addfile Knowledge Base add file processor
// Args[0] map: Request parameters {"collection_id": "collection", "file_id": "file123", "uploader": "local", ...}
// Return: map: Response data {"doc_id": "document_id"}
func ProcessAddFile(process *process.Process) interface{} {
	process.ValidateArgNums(1)

	// Get parameters
	reqMap := process.ArgsMap(0)

	// Check knowledge base API
	if kb.API == nil {
		exception.New("knowledge base API not initialized", 500).Throw()
	}

	// Convert parameters to AddFileParams
	params := parseAddFileParams(reqMap)

	// Get context
	ctx := process.Context
	if ctx == nil {
		ctx = context.Background()
	}

	// Call kb.API
	result, err := kb.API.AddFile(ctx, params)
	if err != nil {
		exception.New("failed to add file: %s", 500, err.Error()).Throw()
	}

	// Return result
	return maps.MapStrAny{
		"doc_id": result.DocID,
	}
}

// convertAddFileRequest converts AddFileRequest to kbapi.AddFileParams
func convertAddFileRequest(req *AddFileRequest, authInfo *oauthtypes.AuthorizedInfo) *kbapi.AddFileParams {
	params := &kbapi.AddFileParams{
		CollectionID: req.CollectionID,
		FileID:       req.FileID,
		Uploader:     req.Uploader,
		DocID:        req.DocID,
		Locale:       req.Locale,
		Metadata:     req.Metadata,
	}

	// Convert provider configs
	if req.Chunking != nil {
		params.Chunking = &kbapi.ProviderConfigParams{
			ProviderID: req.Chunking.ProviderID,
			OptionID:   req.Chunking.OptionID,
		}
	}

	if req.Embedding != nil {
		params.Embedding = &kbapi.ProviderConfigParams{
			ProviderID: req.Embedding.ProviderID,
			OptionID:   req.Embedding.OptionID,
		}
	}

	if req.Extraction != nil {
		params.Extraction = &kbapi.ProviderConfigParams{
			ProviderID: req.Extraction.ProviderID,
			OptionID:   req.Extraction.OptionID,
		}
	}

	if req.Fetcher != nil {
		params.Fetcher = &kbapi.ProviderConfigParams{
			ProviderID: req.Fetcher.ProviderID,
			OptionID:   req.Fetcher.OptionID,
		}
	}

	if req.Converter != nil {
		params.Converter = &kbapi.ProviderConfigParams{
			ProviderID: req.Converter.ProviderID,
			OptionID:   req.Converter.OptionID,
		}
	}

	if req.Job != nil {
		params.Job = &kbapi.JobOptionsParams{
			Name:        req.Job.Name,
			Description: req.Job.Description,
			Icon:        req.Job.Icon,
			Category:    req.Job.Category,
		}
	}

	// Set auth scope
	if authInfo != nil {
		params.AuthScope = authInfo.WithCreateScope(nil)
	}

	return params
}

// parseAddFileParams parses request map into kbapi.AddFileParams
func parseAddFileParams(reqMap map[string]interface{}) *kbapi.AddFileParams {
	params := &kbapi.AddFileParams{}

	// Required fields
	if collectionID, ok := reqMap["collection_id"].(string); ok {
		params.CollectionID = collectionID
	} else {
		exception.New("collection_id is required", 400).Throw()
	}

	if fileID, ok := reqMap["file_id"].(string); ok {
		params.FileID = fileID
	} else {
		exception.New("file_id is required", 400).Throw()
	}

	// Optional fields
	if uploader, ok := reqMap["uploader"].(string); ok {
		params.Uploader = uploader
	} else {
		params.Uploader = "local" // Default to local uploader
	}

	if locale, ok := reqMap["locale"].(string); ok {
		params.Locale = locale
	}

	if docID, ok := reqMap["doc_id"].(string); ok {
		params.DocID = docID
	}

	// Generate doc_id if not provided
	if params.DocID == "" {
		params.DocID = utils.GenDocIDWithCollectionID(params.CollectionID)
	}

	// Handle metadata
	if metadata, ok := reqMap["metadata"].(map[string]interface{}); ok {
		params.Metadata = metadata
	}

	// Handle chunking configuration
	if chunkingMap, ok := reqMap["chunking"].(map[string]interface{}); ok {
		chunking := &kbapi.ProviderConfigParams{}
		if providerID, ok := chunkingMap["provider_id"].(string); ok {
			chunking.ProviderID = providerID
		} else {
			exception.New("chunking.provider_id is required", 400).Throw()
		}
		if optionID, ok := chunkingMap["option_id"].(string); ok {
			chunking.OptionID = optionID
		}
		params.Chunking = chunking
	} else {
		exception.New("chunking configuration is required", 400).Throw()
	}

	// Handle embedding configuration
	if embeddingMap, ok := reqMap["embedding"].(map[string]interface{}); ok {
		embedding := &kbapi.ProviderConfigParams{}
		if providerID, ok := embeddingMap["provider_id"].(string); ok {
			embedding.ProviderID = providerID
		} else {
			exception.New("embedding.provider_id is required", 400).Throw()
		}
		if optionID, ok := embeddingMap["option_id"].(string); ok {
			embedding.OptionID = optionID
		}
		params.Embedding = embedding
	} else {
		exception.New("embedding configuration is required", 400).Throw()
	}

	// Handle optional extraction configuration
	if extractionMap, ok := reqMap["extraction"].(map[string]interface{}); ok {
		extraction := &kbapi.ProviderConfigParams{}
		if providerID, ok := extractionMap["provider_id"].(string); ok {
			extraction.ProviderID = providerID
		}
		if optionID, ok := extractionMap["option_id"].(string); ok {
			extraction.OptionID = optionID
		}
		params.Extraction = extraction
	}

	// Handle optional fetcher configuration
	if fetcherMap, ok := reqMap["fetcher"].(map[string]interface{}); ok {
		fetcher := &kbapi.ProviderConfigParams{}
		if providerID, ok := fetcherMap["provider_id"].(string); ok {
			fetcher.ProviderID = providerID
		}
		if optionID, ok := fetcherMap["option_id"].(string); ok {
			fetcher.OptionID = optionID
		}
		params.Fetcher = fetcher
	}

	// Handle optional converter configuration
	if converterMap, ok := reqMap["converter"].(map[string]interface{}); ok {
		converter := &kbapi.ProviderConfigParams{}
		if providerID, ok := converterMap["provider_id"].(string); ok {
			converter.ProviderID = providerID
		}
		if optionID, ok := converterMap["option_id"].(string); ok {
			converter.OptionID = optionID
		}
		params.Converter = converter
	}

	// Handle job options
	if jobMap, ok := reqMap["job"].(map[string]interface{}); ok {
		job := &kbapi.JobOptionsParams{}
		if name, ok := jobMap["name"].(string); ok {
			job.Name = name
		}
		if description, ok := jobMap["description"].(string); ok {
			job.Description = description
		}
		if icon, ok := jobMap["icon"].(string); ok {
			job.Icon = icon
		}
		if category, ok := jobMap["category"].(string); ok {
			job.Category = category
		}
		params.Job = job
	}

	return params
}

// validateFileExists validates that the file exists in the attachment manager
func validateFileExists(c *gin.Context, req *AddFileRequest) error {
	// Get file manager
	m, ok := attachment.Managers[req.Uploader]
	if !ok {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid uploader: " + req.Uploader + " not found",
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return fmt.Errorf("invalid uploader: %s not found", req.Uploader)
	}

	// Check if the file exists
	exists := m.Exists(c.Request.Context(), req.FileID)
	if !exists {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "File not found: " + req.FileID,
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return fmt.Errorf("file not found: %s", req.FileID)
	}

	return nil
}
