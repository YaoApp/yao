package kb

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/attachment"
	"github.com/yaoapp/yao/kb"
	"github.com/yaoapp/yao/openapi/response"
)

// Document Management Handlers

// Validator interface for request validation
type Validator interface {
	Validate() error
}

// validateRequest validates a request by parsing JSON and calling Validate()
func validateRequest[T Validator](c *gin.Context, req T) error {
	// Parse and bind JSON request
	if err := c.ShouldBindJSON(req); err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid request format: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return err
	}

	// Validate request
	if err := req.Validate(); err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return err
	}

	return nil
}

// checkKBInstance checks if kb.Instance is available
func checkKBInstance(c *gin.Context) bool {
	if kb.Instance == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Knowledge base not initialized",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return false
	}
	return true
}

// getUpsertOptions converts BaseUpsertRequest to UpsertOptions with optional file info
func getUpsertOptions(c *gin.Context, req *BaseUpsertRequest, fileInfo ...string) (*types.UpsertOptions, error) {
	upsertOptions, err := req.ToUpsertOptions(fileInfo...)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Failed to convert request to upsert options: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return nil, err
	}
	return upsertOptions, nil
}

// validateFileAndGetPath validates file manager, file existence and gets local path
func validateFileAndGetPath(c *gin.Context, req *AddFileRequest) (string, string, error) {
	// Get file manager
	m, ok := attachment.Managers[req.Uploader]
	if !ok {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Invalid uploader: " + req.Uploader + " not found",
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return "", "", response.ErrInvalidRequest
	}

	// Check if the file exists
	exists := m.Exists(c.Request.Context(), req.FileID)
	if !exists {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "File not found: " + req.FileID,
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return "", "", response.ErrInvalidRequest
	}

	// Get the options of the manager
	path, contentType, err := m.LocalPath(c.Request.Context(), req.FileID)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to get local path: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return "", "", err
	}

	return path, contentType, nil
}

// handleAsync handles async processing for any handler function
func handleAsync(c *gin.Context, syncHandler func(*gin.Context)) {
	jobid := uuid.New().String()

	// temporary solution to handle async operations ( TODO: use job queue )
	go func() { syncHandler(c) }()

	response.RespondWithSuccess(c, response.StatusCreated, gin.H{"job_id": jobid})
}

// AddFile adds a file to a collection
func AddFile(c *gin.Context) {
	// Check if kb.Instance is available
	if !checkKBInstance(c) {
		return
	}

	// Prepare request and database data
	req, documentData, err := PrepareAddFile(c)
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
	path, contentType, err := validateFileAndGetPath(c, req)
	if err != nil {
		// Rollback: remove the database record
		if err := config.RemoveDocument(req.DocID); err != nil {
			log.Error("Failed to rollback document database record: %v", err)
		}
		return
	}

	upsertOptions, err := getUpsertOptions(c, &req.BaseUpsertRequest, path, contentType)
	if err != nil {
		// Rollback: remove the database record
		if err := config.RemoveDocument(req.DocID); err != nil {
			log.Error("Failed to rollback document database record: %v", err)
		}
		return
	}

	// Perform upsert operation with file ID
	_, err = kb.Instance.AddFile(c.Request.Context(), req.FileID, upsertOptions)
	if err != nil {
		// Update status to error and return error response
		config.UpdateDocument(req.DocID, maps.MapStrAny{"status": "error", "error_message": err.Error()})

		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to add file: " + err.Error(),
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
		"message":       "File added successfully",
		"collection_id": req.CollectionID,
		"file_id":       req.FileID,
		"doc_id":        req.DocID,
	}

	response.RespondWithSuccess(c, response.StatusCreated, result)
}

// AddFileAsync adds file to a collection asynchronously
func AddFileAsync(c *gin.Context) {
	var req AddFileRequest

	// Check if kb.Instance is available
	if !checkKBInstance(c) {
		return
	}

	// Validate request
	if err := validateRequest(c, &req); err != nil {
		return
	}

	// Validate file and get path
	_, _, err := validateFileAndGetPath(c, &req)
	if err != nil {
		return
	}

	// Convert request to UpsertOptions (just for validation)
	_, err = getUpsertOptions(c, &req.BaseUpsertRequest)
	if err != nil {
		return
	}

	// Handle async processing
	handleAsync(c, AddFile)
}

// AddText adds text to a collection
func AddText(c *gin.Context) {
	// Check if kb.Instance is available
	if !checkKBInstance(c) {
		return
	}

	// Prepare request and database data
	req, documentData, err := PrepareAddText(c)
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

// AddTextAsync adds text to a collection asynchronously
func AddTextAsync(c *gin.Context) {
	var req AddTextRequest

	// Validate request
	if err := validateRequest(c, &req); err != nil {
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

	// Handle async processing
	handleAsync(c, AddText)
}

// AddURL adds a URL to a collection
func AddURL(c *gin.Context) {
	// Check if kb.Instance is available
	if !checkKBInstance(c) {
		return
	}

	// Prepare request and database data
	req, documentData, err := PrepareAddURL(c)
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

	// Perform upsert operation with URL
	_, err = kb.Instance.AddURL(c.Request.Context(), req.URL, upsertOptions)
	if err != nil {
		// Update status to error and return error response
		config.UpdateDocument(req.DocID, maps.MapStrAny{"status": "error", "error_message": err.Error()})

		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to add URL: " + err.Error(),
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
		"message":       "URL added successfully",
		"collection_id": req.CollectionID,
		"url":           req.URL,
		"doc_id":        req.DocID,
	}

	response.RespondWithSuccess(c, response.StatusCreated, result)
}

// AddURLAsync adds a URL to a collection asynchronously
func AddURLAsync(c *gin.Context) {
	var req AddURLRequest

	// Validate request
	if err := validateRequest(c, &req); err != nil {
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

	// Handle async processing
	handleAsync(c, AddURL)
}

// ListDocuments lists documents with pagination
func ListDocuments(c *gin.Context) {
	// TODO: Implement list documents logic
	// Query parameters for pagination: page, limit, filter, etc.
	c.JSON(http.StatusOK, gin.H{
		"documents": []interface{}{},
		"total":     0,
		"page":      1,
		"limit":     20,
	})
}

// ScrollDocuments scrolls through documents with iterator-style pagination
func ScrollDocuments(c *gin.Context) {
	// TODO: Implement scroll documents logic
	// Query parameters: cursor, limit, filter, etc.
	c.JSON(http.StatusOK, gin.H{
		"documents": []interface{}{},
		"cursor":    "",
		"hasMore":   false,
	})
}

// GetDocument gets document details by document ID
func GetDocument(c *gin.Context) {
	// TODO: Implement get document logic
	// Note: This might need to be implemented based on your document storage structure
	// as the GraphRag interface doesn't directly provide a GetDocument method
	docID := c.Param("docID")
	if docID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Document ID is required"})
		return
	}

	// TODO: Implement actual document retrieval logic
	// This could involve querying your document storage or getting document metadata
	c.JSON(http.StatusOK, gin.H{
		"docID":   docID,
		"message": "Document details retrieved",
		// Add actual document fields here when implementing
	})
}

// RemoveDocs removes documents by IDs
func RemoveDocs(c *gin.Context) {
	// TODO: Implement remove documents logic
	c.JSON(http.StatusOK, gin.H{"message": "Documents removed"})
}
