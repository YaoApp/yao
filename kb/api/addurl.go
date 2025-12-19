package api

import (
	"context"
	"fmt"

	"github.com/yaoapp/gou/graphrag/utils"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/job"
)

// AddURL adds a URL to a collection (sync)
func (instance *KBInstance) AddURL(ctx context.Context, params *AddURLParams) (*AddDocumentResult, error) {
	// Validate required parameters
	if params.CollectionID == "" {
		return nil, fmt.Errorf("collection_id is required")
	}
	if params.URL == "" {
		return nil, fmt.Errorf("url is required")
	}
	if params.Chunking == nil {
		return nil, fmt.Errorf("chunking configuration is required")
	}
	if params.Embedding == nil {
		return nil, fmt.Errorf("embedding configuration is required")
	}

	// Generate document ID if not provided
	docID := params.DocID
	if docID == "" {
		docID = utils.GenDocIDWithCollectionID(params.CollectionID)
	}

	// Create document record
	documentData := map[string]interface{}{
		"document_id":   docID,
		"collection_id": params.CollectionID,
		"name":          "URL Document",
		"type":          "url",
		"status":        "pending",
		"url":           params.URL,
	}

	// Use title from metadata if available
	if params.Metadata != nil {
		if title, ok := params.Metadata["title"].(string); ok && title != "" {
			documentData["name"] = title
		}
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
	_, err := instance.Config.CreateDocument(maps.MapStrAny(documentData))
	if err != nil {
		return nil, fmt.Errorf("failed to save document metadata: %w", err)
	}

	// Process URL content
	params.DocID = docID // Ensure docID is set
	err = instance.processURL(ctx, docID, params)
	if err != nil {
		return nil, err
	}

	return &AddDocumentResult{
		Message:      "URL added successfully",
		CollectionID: params.CollectionID,
		DocID:        docID,
		URL:          params.URL,
	}, nil
}

// AddURLAsync adds a URL to a collection (async)
func (instance *KBInstance) AddURLAsync(ctx context.Context, params *AddURLParams) (*AddDocumentAsyncResult, error) {
	// Validate required parameters
	if params.CollectionID == "" {
		return nil, fmt.Errorf("collection_id is required")
	}
	if params.URL == "" {
		return nil, fmt.Errorf("url is required")
	}
	if params.Chunking == nil {
		return nil, fmt.Errorf("chunking configuration is required")
	}
	if params.Embedding == nil {
		return nil, fmt.Errorf("embedding configuration is required")
	}

	// Generate document ID if not provided
	docID := params.DocID
	if docID == "" {
		docID = utils.GenDocIDWithCollectionID(params.CollectionID)
	}

	// Get job options with defaults
	jobName, jobDescription, jobIcon, jobCategory := getJobOptions(params.Job,
		"Knowledge Base Web Content Processing",
		"Fetching and indexing web content for knowledge base search",
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
		"document_id":   docID,
		"collection_id": params.CollectionID,
		"name":          "URL Document",
		"type":          "url",
		"status":        "pending",
		"url":           params.URL,
		"job_id":        j.JobID,
	}

	// Use title from metadata if available
	if params.Metadata != nil {
		if title, ok := params.Metadata["title"].(string); ok && title != "" {
			documentData["name"] = title
		}
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
	asyncParams := &AddURLParams{
		CollectionID: params.CollectionID,
		URL:          params.URL,
		Locale:       params.Locale,
		Chunking:     params.Chunking,
		Embedding:    params.Embedding,
		Extraction:   params.Extraction,
		Fetcher:      params.Fetcher,
		Converter:    params.Converter,
	}

	// Add function execution to job
	err = j.AddFunc(&job.ExecutionOptions{Priority: 1}, "kb.addurl", func(execCtx *job.ExecutionContext) error {
		return instance.processURL(execCtx.Ctx, asyncDocID, asyncParams)
	}, map[string]interface{}{
		"doc_id":        asyncDocID,
		"collection_id": params.CollectionID,
		"url":           params.URL,
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

// processURL processes URL content and updates the knowledge base
func (instance *KBInstance) processURL(ctx context.Context, docID string, params *AddURLParams) error {
	// Convert to UpsertOptions
	upsertOptions, err := instance.toUpsertOptions(docID, params.CollectionID, params.Locale, "", "", params.Chunking, params.Embedding, params.Extraction, params.Fetcher, params.Converter)
	if err != nil {
		instance.Config.UpdateDocument(docID, maps.MapStrAny{"status": "error", "error_message": err.Error()})
		return fmt.Errorf("failed to convert to upsert options: %w", err)
	}

	// Add URL to GraphRag
	_, err = instance.GraphRag.AddURL(ctx, params.URL, upsertOptions)
	if err != nil {
		instance.Config.UpdateDocument(docID, maps.MapStrAny{"status": "error", "error_message": err.Error()})
		return fmt.Errorf("failed to add URL: %w", err)
	}

	// Update status and segment count
	instance.updateDocumentAfterProcessing(ctx, docID, params.CollectionID)

	return nil
}
