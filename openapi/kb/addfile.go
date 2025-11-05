package kb

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/graphrag/utils"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/attachment"
	"github.com/yaoapp/yao/job"
	"github.com/yaoapp/yao/kb"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
)

// CreateDocumentRecord creates a document record in the database immediately
// This is called synchronously when the API request comes in
func CreateDocumentRecord(ctx context.Context, authInfo *oauthtypes.AuthorizedInfo, req *AddFileRequest, jobID string) error {
	// Check if kb.Instance is available
	if kb.Instance == nil {
		return fmt.Errorf("knowledge base not initialized")
	}

	// Get file manager
	m, ok := attachment.Managers[req.Uploader]
	if !ok {
		return fmt.Errorf("invalid uploader: %s not found", req.Uploader)
	}

	// Check if the file exists
	exists := m.Exists(ctx, req.FileID)
	if !exists {
		return fmt.Errorf("file not found: %s", req.FileID)
	}

	// Get file info and path
	path, contentType, err := m.LocalPath(ctx, req.FileID)
	if err != nil {
		return fmt.Errorf("failed to get local path: %w", err)
	}

	fileInfo, err := m.Info(ctx, req.FileID)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// Get KB config
	config, err := kb.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get KB config: %w", err)
	}

	// Prepare document data for database
	documentData := map[string]interface{}{
		"document_id":    req.DocID,
		"collection_id":  req.CollectionID,
		"name":           fileInfo.Filename,
		"type":           "file",
		"status":         "pending",
		"uploader_id":    req.Uploader,
		"file_id":        req.FileID,
		"file_name":      fileInfo.Filename,
		"file_path":      path,
		"file_mime_type": contentType,
		"size":           int64(fileInfo.Bytes),
		"job_id":         jobID,
	}

	// With create scope
	if authInfo != nil {
		documentData = authInfo.WithCreateScope(documentData)
	}

	// Add base request fields
	req.BaseUpsertRequest.AddBaseFields(documentData)

	// Create database record
	_, err = config.CreateDocument(maps.MapStrAny(documentData))
	if err != nil {
		return fmt.Errorf("failed to save document metadata: %w", err)
	}

	return nil
}

// HandleFileContent processes the actual file content and updates the knowledge base
// This is called asynchronously by the job system
func HandleFileContent(ctx context.Context, req *AddFileRequest) error {
	// Check if kb.Instance is available
	if kb.Instance == nil {
		return fmt.Errorf("knowledge base not initialized")
	}

	// Get file manager
	m, ok := attachment.Managers[req.Uploader]
	if !ok {
		return fmt.Errorf("invalid uploader: %s not found", req.Uploader)
	}

	// Get file info and path
	path, contentType, err := m.LocalPath(ctx, req.FileID)
	if err != nil {
		return fmt.Errorf("failed to get local path: %w", err)
	}

	// Get KB config
	config, err := kb.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get KB config: %w", err)
	}

	// Convert request to UpsertOptions
	upsertOptions, err := req.BaseUpsertRequest.ToUpsertOptions(path, contentType)
	if err != nil {
		// Update status to error
		config.UpdateDocument(req.DocID, maps.MapStrAny{"status": "error", "error_message": err.Error()})
		return fmt.Errorf("failed to convert request to upsert options: %w", err)
	}

	// Perform upsert operation with file path
	_, err = kb.Instance.AddFile(ctx, path, upsertOptions)
	if err != nil {
		// Update status to error
		config.UpdateDocument(req.DocID, maps.MapStrAny{"status": "error", "error_message": err.Error()})
		return fmt.Errorf("failed to add file: %w", err)
	}

	// Update status to completed after successful processing
	if err := config.UpdateDocument(req.DocID, maps.MapStrAny{"status": "completed"}); err != nil {
		log.Error("Failed to update document status to completed: %v", err)
	}

	// Update segment count for the document
	if segmentCount, err := kb.Instance.SegmentCount(ctx, req.DocID); err != nil {
		log.Error("Failed to get segment count for document %s: %v", req.DocID, err)
	} else {
		log.Info("Got segment count %d for document %s", segmentCount, req.DocID)
		if err := config.UpdateSegmentCount(req.DocID, segmentCount); err != nil {
			log.Error("Failed to update segment count for document %s: %v", req.DocID, err)
		} else {
			log.Info("Successfully updated segment count to %d for document %s", segmentCount, req.DocID)
		}
	}

	// Update document count for the collection and sync to GraphRag
	if err := UpdateDocumentCountWithSync(req.CollectionID, config); err != nil {
		log.Error("Failed to update document count for collection %s: %v", req.CollectionID, err)
	} else {
		log.Info("Successfully updated document count for collection %s", req.CollectionID)
	}

	return nil
}

// AddFileHandler processes a file addition request with business logic only
// This function combines both document creation and content processing for sync operations
func AddFileHandler(ctx context.Context, authInfo *oauthtypes.AuthorizedInfo, req *AddFileRequest, jobID ...string) error {
	// Validate request
	if err := req.Validate(); err != nil {
		return err
	}

	// DocID should be generated by the caller before calling this function
	if req.DocID == "" {
		return fmt.Errorf("document ID is required")
	}

	// For sync operations, create document record and process content immediately
	var jid string
	if len(jobID) > 0 {
		jid = jobID[0]
	}

	// Create document record
	if err := CreateDocumentRecord(ctx, authInfo, req, jid); err != nil {
		return err
	}

	// Process file content
	return HandleFileContent(ctx, req)
}

// addFileWithRequest processes a file addition with pre-parsed request using Gin context
func addFileWithRequest(c *gin.Context, req *AddFileRequest) {

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

	// Use the business logic function
	err = AddFileHandler(c.Request.Context(), authInfo, req)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Return success response
	result := gin.H{
		"message":       "File added successfully",
		"collection_id": req.CollectionID,
		"file_id":       req.FileID,
		"doc_id":        req.DocID,
	}

	response.RespondWithSuccess(c, response.StatusCreated, result)
}

// AddFile adds a file to a collection
func AddFile(c *gin.Context) {
	var req AddFileRequest

	// Check if kb.Instance is available
	if !checkKBInstance(c) {
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

	// Process the request
	addFileWithRequest(c, &req)
}

// AddFileAsync adds file to a collection asynchronously
func AddFileAsync(c *gin.Context) {
	var req AddFileRequest

	log.Info("AddFileAsync: Starting async file addition")

	// Check if kb.Instance is available
	if !checkKBInstance(c) {
		log.Error("AddFileAsync: KB instance check failed")
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

	log.Info("AddFileAsync: Request parsed successfully: %+v", req)

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

	// Validate file and get path
	_, _, err := validateFileAndGetPath(c, &req)
	if err != nil {
		log.Error("AddFileAsync: File validation failed: %v", err)
		return
	}

	log.Info("AddFileAsync: File validation passed")

	// Convert request to UpsertOptions (just for validation)
	_, err = getUpsertOptions(c, &req.BaseUpsertRequest)
	if err != nil {
		log.Error("AddFileAsync: UpsertOptions validation failed: %v", err)
		return
	}

	log.Info("AddFileAsync: UpsertOptions validation passed")

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

	// Step 1: Get job options with defaults
	jobName, jobDescription, jobIcon, jobCategory := req.GetJobOptions(
		"Knowledge Base File Processing",                                 // default name
		"Processing and indexing file content for knowledge base search", // default description
		"library_add",    // default icon (Material Icon)
		"Knowledge Base", // default category
	)

	// Create job data
	jobCreateData := map[string]interface{}{
		"name":          jobName,
		"description":   jobDescription,
		"category_name": jobCategory, // Pass category name directly, let SaveJob handle it
	}
	if jobIcon != "" {
		jobCreateData["icon"] = jobIcon
	}

	// With create scope
	if authInfo != nil {
		jobCreateData = authInfo.WithCreateScope(jobCreateData)
	}

	// Create and save Job in one step to get JobID
	j, err := job.OnceAndSave(job.GOROUTINE, jobCreateData)
	if err != nil {
		log.Error("AddFileAsync: Job creation and save failed: %v", err)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to create and save job: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	log.Info("AddFileAsync: Job created and saved with ID: %s", j.JobID)

	// Step 2: Create document record immediately with job_id
	err = CreateDocumentRecord(c.Request.Context(), authInfo, &req, j.JobID)
	if err != nil {
		log.Error("AddFileAsync: Failed to create document record: %v", err)
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to create document record: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	log.Info("AddFileAsync: Document record created successfully")

	// Step 4: Add execution to job
	jobData := map[string]interface{}{
		"collection_id": req.CollectionID,
		"file_id":       req.FileID,
		"uploader":      req.Uploader,
		"locale":        req.Locale,
		"doc_id":        req.DocID,
		"metadata":      req.Metadata,
		"chunking":      req.Chunking,
		"embedding":     req.Embedding,
		"extraction":    req.Extraction,
		"fetcher":       req.Fetcher,
		"converter":     req.Converter,
	}

	err = j.Add(&job.ExecutionOptions{
		Priority: 1,
	}, "kb.documents.addfile", jobData)
	if err != nil {
		log.Error("AddFileAsync: Failed to add job execution: %v", err)
		// Rollback: remove document record
		if config, err := kb.GetConfig(); err == nil {
			config.RemoveDocument(req.DocID)
		}
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to add job execution: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Step 5: Push the job to execution queue
	err = j.Push()
	if err != nil {
		log.Error("AddFileAsync: Failed to push job: %v", err)
		// Rollback: remove document record
		if config, err := kb.GetConfig(); err == nil {
			config.RemoveDocument(req.DocID)
		}
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to push job: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	log.Info("AddFileAsync: Job pushed successfully")

	// Return job_id and doc_id
	response.RespondWithSuccess(c, response.StatusCreated, gin.H{
		"job_id": j.JobID,
		"doc_id": req.DocID,
	})
}

// ProcessAddFile documents.addfile Knowledge Base add file processor
// Args[0] map: Request parameters {"collection_id": "collection", "file_id": "file123", "uploader": "local", ...}
// Return: map: Response data {"doc_id": "document_id"}
func ProcessAddFile(process *process.Process) interface{} {
	process.ValidateArgNums(1)

	// Get parameters
	reqMap := process.ArgsMap(0)

	// Check knowledge base instance
	if kb.Instance == nil {
		exception.New("knowledge base not initialized", 500).Throw()
	}

	// Convert parameters to AddFileRequest structure
	req := parseAddFileRequest(reqMap)

	// Get KB config to check if document exists
	config, err := kb.GetConfig()
	if err != nil {
		exception.New("failed to get KB config: %s", 500, err.Error()).Throw()
	}

	// Check if document already exists
	ctx := process.Context
	if ctx == nil {
		ctx = context.Background()
	}

	existingDoc, err := config.FindDocument(req.DocID, model.QueryParam{})
	if err != nil || existingDoc == nil {
		// Document doesn't exist, create it first (sync scenario)
		log.Info("ProcessAddFile: Document %s not found, creating new record", req.DocID)

		// Get job_id from request if provided (for async scenario)
		var jobID string
		if jid, ok := reqMap["job_id"].(string); ok {
			jobID = jid
		}
		err = CreateDocumentRecord(ctx, authorized.ProcessAuthInfo(process), req, jobID)
		if err != nil {
			exception.New("failed to create document record: %s", 500, err.Error()).Throw()
		}
	} else {
		log.Info("ProcessAddFile: Document %s already exists, processing content only", req.DocID)
	}

	// Process file content
	err = HandleFileContent(ctx, req)
	if err != nil {
		exception.New("failed to process file: %s", 500, err.Error()).Throw()
	}

	// Return result
	return maps.MapStrAny{
		"doc_id": req.DocID,
	}
}

// parseAddFileRequest parses request map into AddFileRequest structure
func parseAddFileRequest(reqMap map[string]interface{}) *AddFileRequest {
	req := &AddFileRequest{}

	// Required fields
	if collectionID, ok := reqMap["collection_id"].(string); ok {
		req.CollectionID = collectionID
	} else {
		exception.New("collection_id is required", 400).Throw()
	}

	if fileID, ok := reqMap["file_id"].(string); ok {
		req.FileID = fileID
	} else {
		exception.New("file_id is required", 400).Throw()
	}

	// Optional fields
	if uploader, ok := reqMap["uploader"].(string); ok {
		req.Uploader = uploader
	} else {
		req.Uploader = "local" // Default to local uploader
	}

	if locale, ok := reqMap["locale"].(string); ok {
		req.Locale = locale
	}

	if docID, ok := reqMap["doc_id"].(string); ok {
		req.DocID = docID
	}

	// Generate doc_id if not provided
	if req.DocID == "" {
		req.DocID = utils.GenDocIDWithCollectionID(req.CollectionID)
	}

	// Handle metadata
	if metadata, ok := reqMap["metadata"].(map[string]interface{}); ok {
		req.Metadata = metadata
	}

	// Handle chunking configuration
	if chunkingMap, ok := reqMap["chunking"].(map[string]interface{}); ok {
		chunking := &ProviderConfig{}
		if providerID, ok := chunkingMap["provider_id"].(string); ok {
			chunking.ProviderID = providerID
		} else {
			exception.New("chunking.provider_id is required", 400).Throw()
		}
		if optionID, ok := chunkingMap["option_id"].(string); ok {
			chunking.OptionID = optionID
		}
		req.Chunking = chunking
	} else {
		exception.New("chunking configuration is required", 400).Throw()
	}

	// Handle embedding configuration
	if embeddingMap, ok := reqMap["embedding"].(map[string]interface{}); ok {
		embedding := &ProviderConfig{}
		if providerID, ok := embeddingMap["provider_id"].(string); ok {
			embedding.ProviderID = providerID
		} else {
			exception.New("embedding.provider_id is required", 400).Throw()
		}
		if optionID, ok := embeddingMap["option_id"].(string); ok {
			embedding.OptionID = optionID
		}
		req.Embedding = embedding
	} else {
		exception.New("embedding configuration is required", 400).Throw()
	}

	// Handle optional extraction configuration
	if extractionMap, ok := reqMap["extraction"].(map[string]interface{}); ok {
		extraction := &ProviderConfig{}
		if providerID, ok := extractionMap["provider_id"].(string); ok {
			extraction.ProviderID = providerID
		}
		if optionID, ok := extractionMap["option_id"].(string); ok {
			extraction.OptionID = optionID
		}
		req.Extraction = extraction
	}

	// Handle optional fetcher configuration
	if fetcherMap, ok := reqMap["fetcher"].(map[string]interface{}); ok {
		fetcher := &ProviderConfig{}
		if providerID, ok := fetcherMap["provider_id"].(string); ok {
			fetcher.ProviderID = providerID
		}
		if optionID, ok := fetcherMap["option_id"].(string); ok {
			fetcher.OptionID = optionID
		}
		req.Fetcher = fetcher
	}

	// Handle optional converter configuration
	if converterMap, ok := reqMap["converter"].(map[string]interface{}); ok {
		converter := &ProviderConfig{}
		if providerID, ok := converterMap["provider_id"].(string); ok {
			converter.ProviderID = providerID
		}
		if optionID, ok := converterMap["option_id"].(string); ok {
			converter.OptionID = optionID
		}
		req.Converter = converter
	}

	// Handle job options
	if jobMap, ok := reqMap["job"].(map[string]interface{}); ok {
		job := &JobOptions{}
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
		req.Job = job
	}

	return req
}
