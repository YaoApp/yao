package kb

import (
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/kb"
	"github.com/yaoapp/yao/openapi/response"
)

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
