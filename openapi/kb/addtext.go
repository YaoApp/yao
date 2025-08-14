package kb

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/graphrag/utils"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/kb"
	"github.com/yaoapp/yao/openapi/response"
)

// ProcessAddTextRequest processes a text addition request with business logic only
// This function is Gin-agnostic and can be used for both sync and async operations
func ProcessAddTextRequest(ctx context.Context, req *AddTextRequest) error {
	// Check if kb.Instance is available
	if kb.Instance == nil {
		return fmt.Errorf("knowledge base not initialized")
	}

	// Validate request
	if err := req.Validate(); err != nil {
		return err
	}

	// Generate document ID if not provided
	if req.DocID == "" {
		req.DocID = utils.GenDocIDWithCollectionID(req.CollectionID)
	}

	// Get KB config
	config, err := kb.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get KB config: %w", err)
	}

	// Prepare document data for database
	documentData := map[string]interface{}{
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
			documentData["name"] = title
		}
	}

	// Add base request fields
	req.BaseUpsertRequest.AddBaseFields(documentData)

	// First create database record
	_, err = config.CreateDocument(maps.MapStrAny(documentData))
	if err != nil {
		return fmt.Errorf("failed to save document metadata: %w", err)
	}

	// Convert request to UpsertOptions
	upsertOptions, err := req.BaseUpsertRequest.ToUpsertOptions()
	if err != nil {
		// Rollback: remove the database record
		if rollbackErr := config.RemoveDocument(req.DocID); rollbackErr != nil {
			log.Error("Failed to rollback document database record: %v", rollbackErr)
		}
		return fmt.Errorf("failed to convert request to upsert options: %w", err)
	}

	// Perform upsert operation with text
	_, err = kb.Instance.AddText(ctx, req.Text, upsertOptions)
	if err != nil {
		// Update status to error
		config.UpdateDocument(req.DocID, maps.MapStrAny{"status": "error", "error_message": err.Error()})
		return fmt.Errorf("failed to add text: %w", err)
	}

	// Update status to completed after successful processing
	if err := config.UpdateDocument(req.DocID, maps.MapStrAny{"status": "completed"}); err != nil {
		log.Error("Failed to update document status to completed: %v", err)
	}

	return nil
}

// addTextWithRequest processes a text addition with pre-parsed request
func addTextWithRequest(c *gin.Context, req *AddTextRequest) {
	// Prepare request and database data
	_, documentData, err := PrepareAddText(c, req)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
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
	_, err = config.CreateDocument(maps.MapStrAny(documentData))
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to save document metadata: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Convert request to UpsertOptions
	upsertOptions, err := getUpsertOptions(c, &req.BaseUpsertRequest)
	if err != nil {
		// Rollback: remove the database record
		if err := config.RemoveDocument(req.DocID); err != nil {
			log.Error("Failed to rollback document database record: %v", err)
		}
		return
	}

	// Perform upsert operation with text
	_, err = kb.Instance.AddText(c.Request.Context(), req.Text, upsertOptions)
	if err != nil {
		// Update status to error and return error response
		config.UpdateDocument(req.DocID, maps.MapStrAny{"status": "error", "error_message": err.Error()})

		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to add text: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Update status to completed after successful processing
	if err := config.UpdateDocument(req.DocID, maps.MapStrAny{"status": "completed"}); err != nil {
		log.Error("Failed to update document status to completed: %v", err)
	}

	// Return success response
	result := gin.H{
		"message":       "Text added successfully",
		"collection_id": req.CollectionID,
		"doc_id":        req.DocID,
	}

	response.RespondWithSuccess(c, response.StatusCreated, result)
}

// AddText adds text to a collection
func AddText(c *gin.Context) {
	var req AddTextRequest

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

	// Process the request
	addTextWithRequest(c, &req)
}

// AddTextAsync adds text to a collection asynchronously
func AddTextAsync(c *gin.Context) {
	var req AddTextRequest

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

	// Check if kb.Instance is available
	if !checkKBInstance(c) {
		return
	}

	// Convert request to UpsertOptions (just for validation)
	_, err := getUpsertOptions(c, &req.BaseUpsertRequest)
	if err != nil {
		return
	}

	// Handle async processing with parsed request
	handleAsyncWithRequest(c, &req, func(ctx context.Context, r *AddTextRequest) {
		err := ProcessAddTextRequest(ctx, r)
		if err != nil {
			log.Error("Async text processing failed: %v", err)
		}
	})
}
