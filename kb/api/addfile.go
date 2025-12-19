package api

import (
	"context"
	"fmt"

	"github.com/yaoapp/gou/graphrag/utils"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/attachment"
	"github.com/yaoapp/yao/job"
)

// AddFile adds a file to a collection (sync)
func (instance *KBInstance) AddFile(ctx context.Context, params *AddFileParams) (*AddDocumentResult, error) {
	// Validate required parameters
	if params.CollectionID == "" {
		return nil, fmt.Errorf("collection_id is required")
	}
	if params.FileID == "" {
		return nil, fmt.Errorf("file_id is required")
	}
	if params.Chunking == nil {
		return nil, fmt.Errorf("chunking configuration is required")
	}
	if params.Embedding == nil {
		return nil, fmt.Errorf("embedding configuration is required")
	}

	// Set default uploader
	uploader := params.Uploader
	if uploader == "" {
		uploader = DefaultUploader
	}

	// Generate document ID if not provided
	docID := params.DocID
	if docID == "" {
		docID = utils.GenDocIDWithCollectionID(params.CollectionID)
	}

	// Get file manager
	m, ok := attachment.Managers[uploader]
	if !ok {
		return nil, fmt.Errorf("invalid uploader: %s not found", uploader)
	}

	// Check if the file exists
	exists := m.Exists(ctx, params.FileID)
	if !exists {
		return nil, fmt.Errorf("file not found: %s", params.FileID)
	}

	// Get file info and path
	path, contentType, err := m.LocalPath(ctx, params.FileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get local path: %w", err)
	}

	fileInfo, err := m.Info(ctx, params.FileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Create document record
	documentData := map[string]interface{}{
		"document_id":    docID,
		"collection_id":  params.CollectionID,
		"name":           fileInfo.Filename,
		"type":           "file",
		"status":         "pending",
		"uploader_id":    uploader,
		"file_id":        params.FileID,
		"file_name":      fileInfo.Filename,
		"file_path":      path,
		"file_mime_type": contentType,
		"size":           int64(fileInfo.Bytes),
	}

	// Add auth scope fields
	if params.AuthScope != nil {
		for k, v := range params.AuthScope {
			documentData[k] = v
		}
	}

	// Add base fields
	addBaseFieldsFromParams(documentData, params.Locale, params.Metadata, params.Chunking, params.Embedding, params.Extraction, params.Fetcher, params.Converter)

	// Create database record
	_, err = instance.Config.CreateDocument(maps.MapStrAny(documentData))
	if err != nil {
		return nil, fmt.Errorf("failed to save document metadata: %w", err)
	}

	// Process file content
	params.DocID = docID // Ensure docID is set
	err = instance.processFile(ctx, docID, params)
	if err != nil {
		return nil, err
	}

	return &AddDocumentResult{
		Message:      "File added successfully",
		CollectionID: params.CollectionID,
		DocID:        docID,
		FileID:       params.FileID,
	}, nil
}

// AddFileAsync adds a file to a collection (async)
func (instance *KBInstance) AddFileAsync(ctx context.Context, params *AddFileParams) (*AddDocumentAsyncResult, error) {
	// Validate required parameters
	if params.CollectionID == "" {
		return nil, fmt.Errorf("collection_id is required")
	}
	if params.FileID == "" {
		return nil, fmt.Errorf("file_id is required")
	}
	if params.Chunking == nil {
		return nil, fmt.Errorf("chunking configuration is required")
	}
	if params.Embedding == nil {
		return nil, fmt.Errorf("embedding configuration is required")
	}

	// Set default uploader
	uploader := params.Uploader
	if uploader == "" {
		uploader = DefaultUploader
	}

	// Generate document ID if not provided
	docID := params.DocID
	if docID == "" {
		docID = utils.GenDocIDWithCollectionID(params.CollectionID)
	}

	// Get file manager
	m, ok := attachment.Managers[uploader]
	if !ok {
		return nil, fmt.Errorf("invalid uploader: %s not found", uploader)
	}

	// Check if the file exists
	exists := m.Exists(ctx, params.FileID)
	if !exists {
		return nil, fmt.Errorf("file not found: %s", params.FileID)
	}

	// Get file info and path
	path, contentType, err := m.LocalPath(ctx, params.FileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get local path: %w", err)
	}

	fileInfo, err := m.Info(ctx, params.FileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Get job options with defaults
	jobName, jobDescription, jobIcon, jobCategory := getJobOptions(params.Job,
		"Knowledge Base File Processing",
		"Processing and indexing file content for knowledge base search",
		"library_add",
		"Knowledge Base",
	)

	// Create job data
	jobCreateData := map[string]interface{}{
		"name":          jobName,
		"description":   jobDescription,
		"category_name": jobCategory,
	}
	if jobIcon != "" {
		jobCreateData["icon"] = jobIcon
	}

	// Add auth scope fields
	if params.AuthScope != nil {
		for k, v := range params.AuthScope {
			jobCreateData[k] = v
		}
	}

	// Create and save Job
	j, err := job.OnceAndSave(job.GOROUTINE, jobCreateData)
	if err != nil {
		return nil, fmt.Errorf("failed to create and save job: %w", err)
	}

	// Create document record
	documentData := map[string]interface{}{
		"document_id":    docID,
		"collection_id":  params.CollectionID,
		"name":           fileInfo.Filename,
		"type":           "file",
		"status":         "pending",
		"uploader_id":    uploader,
		"file_id":        params.FileID,
		"file_name":      fileInfo.Filename,
		"file_path":      path,
		"file_mime_type": contentType,
		"size":           int64(fileInfo.Bytes),
		"job_id":         j.JobID,
	}

	// Add auth scope fields
	if params.AuthScope != nil {
		for k, v := range params.AuthScope {
			documentData[k] = v
		}
	}

	// Add base fields
	addBaseFieldsFromParams(documentData, params.Locale, params.Metadata, params.Chunking, params.Embedding, params.Extraction, params.Fetcher, params.Converter)

	// Create database record
	_, err = instance.Config.CreateDocument(maps.MapStrAny(documentData))
	if err != nil {
		return nil, fmt.Errorf("failed to save document metadata: %w", err)
	}

	// Capture parameters for the async function
	asyncDocID := docID
	asyncParams := &AddFileParams{
		CollectionID: params.CollectionID,
		FileID:       params.FileID,
		Uploader:     uploader,
		Locale:       params.Locale,
		Chunking:     params.Chunking,
		Embedding:    params.Embedding,
		Extraction:   params.Extraction,
		Fetcher:      params.Fetcher,
		Converter:    params.Converter,
	}

	// Add function execution to job
	err = j.AddFunc(&job.ExecutionOptions{Priority: 1}, "kb.addfile", func(execCtx *job.ExecutionContext) error {
		return instance.processFile(execCtx.Ctx, asyncDocID, asyncParams)
	}, map[string]interface{}{
		"doc_id":        asyncDocID,
		"collection_id": params.CollectionID,
		"file_id":       params.FileID,
	})
	if err != nil {
		// Rollback: remove document record
		instance.Config.RemoveDocument(docID)
		return nil, fmt.Errorf("failed to add job execution: %w", err)
	}

	// Push the job to execution queue
	err = j.Push()
	if err != nil {
		// Rollback: remove document record
		instance.Config.RemoveDocument(docID)
		return nil, fmt.Errorf("failed to push job: %w", err)
	}

	return &AddDocumentAsyncResult{
		JobID: j.JobID,
		DocID: docID,
	}, nil
}

// processFile processes file content and updates the knowledge base
func (instance *KBInstance) processFile(ctx context.Context, docID string, params *AddFileParams) error {
	uploader := params.Uploader
	if uploader == "" {
		uploader = DefaultUploader
	}

	// Get file manager
	m, ok := attachment.Managers[uploader]
	if !ok {
		return fmt.Errorf("invalid uploader: %s not found", uploader)
	}

	// Get file path
	path, contentType, err := m.LocalPath(ctx, params.FileID)
	if err != nil {
		return fmt.Errorf("failed to get local path: %w", err)
	}

	// Convert to UpsertOptions
	upsertOptions, err := instance.toUpsertOptions(docID, params.CollectionID, params.Locale, path, contentType, params.Chunking, params.Embedding, params.Extraction, params.Fetcher, params.Converter)
	if err != nil {
		instance.Config.UpdateDocument(docID, maps.MapStrAny{"status": "error", "error_message": err.Error()})
		return fmt.Errorf("failed to convert to upsert options: %w", err)
	}

	// Add file to GraphRag
	_, err = instance.GraphRag.AddFile(ctx, path, upsertOptions)
	if err != nil {
		instance.Config.UpdateDocument(docID, maps.MapStrAny{"status": "error", "error_message": err.Error()})
		return fmt.Errorf("failed to add file: %w", err)
	}

	// Update status and segment count
	instance.updateDocumentAfterProcessing(ctx, docID, params.CollectionID)

	return nil
}
