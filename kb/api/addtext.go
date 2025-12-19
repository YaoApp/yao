package api

import (
	"context"
	"fmt"

	"github.com/yaoapp/gou/graphrag/utils"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/job"
)

// AddText adds text to a collection (sync)
func (instance *KBInstance) AddText(ctx context.Context, params *AddTextParams) (*AddDocumentResult, error) {
	// Validate required parameters
	if params.CollectionID == "" {
		return nil, fmt.Errorf("collection_id is required")
	}
	if params.Text == "" {
		return nil, fmt.Errorf("text is required")
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
		"name":          "Text Document",
		"type":          "text",
		"status":        "pending",
		"text_content":  params.Text,
		"size":          int64(len(params.Text)),
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

	// Process text content
	params.DocID = docID // Ensure docID is set
	err = instance.processText(ctx, docID, params)
	if err != nil {
		return nil, err
	}

	return &AddDocumentResult{
		Message:      "Text added successfully",
		CollectionID: params.CollectionID,
		DocID:        docID,
	}, nil
}

// AddTextAsync adds text to a collection (async)
func (instance *KBInstance) AddTextAsync(ctx context.Context, params *AddTextParams) (*AddDocumentAsyncResult, error) {
	// Validate required parameters
	if params.CollectionID == "" {
		return nil, fmt.Errorf("collection_id is required")
	}
	if params.Text == "" {
		return nil, fmt.Errorf("text is required")
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
		"Knowledge Base Text Processing",
		"Processing and indexing text content for knowledge base search",
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
		"name":          "Text Document",
		"type":          "text",
		"status":        "pending",
		"text_content":  params.Text,
		"size":          int64(len(params.Text)),
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
	asyncParams := &AddTextParams{
		CollectionID: params.CollectionID,
		Text:         params.Text,
		Locale:       params.Locale,
		Chunking:     params.Chunking,
		Embedding:    params.Embedding,
		Extraction:   params.Extraction,
		Fetcher:      params.Fetcher,
		Converter:    params.Converter,
	}

	// Add function execution to job
	err = j.AddFunc(&job.ExecutionOptions{Priority: 1}, "kb.addtext", func(execCtx *job.ExecutionContext) error {
		return instance.processText(execCtx.Ctx, asyncDocID, asyncParams)
	}, map[string]interface{}{
		"doc_id":        asyncDocID,
		"collection_id": params.CollectionID,
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

// processText processes text content and updates the knowledge base
func (instance *KBInstance) processText(ctx context.Context, docID string, params *AddTextParams) error {
	// Convert to UpsertOptions
	upsertOptions, err := instance.toUpsertOptions(docID, params.CollectionID, params.Locale, "", "", params.Chunking, params.Embedding, params.Extraction, params.Fetcher, params.Converter)
	if err != nil {
		instance.Config.UpdateDocument(docID, maps.MapStrAny{"status": "error", "error_message": err.Error()})
		return fmt.Errorf("failed to convert to upsert options: %w", err)
	}

	// Add text to GraphRag
	_, err = instance.GraphRag.AddText(ctx, params.Text, upsertOptions)
	if err != nil {
		instance.Config.UpdateDocument(docID, maps.MapStrAny{"status": "error", "error_message": err.Error()})
		return fmt.Errorf("failed to add text: %w", err)
	}

	// Update status and segment count
	instance.updateDocumentAfterProcessing(ctx, docID, params.CollectionID)

	return nil
}
