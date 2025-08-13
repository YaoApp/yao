package kb

import (
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/kb"
	"github.com/yaoapp/yao/openapi/response"
)

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
