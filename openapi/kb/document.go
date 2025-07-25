package kb

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/kb"
	"github.com/yaoapp/yao/openapi/response"
)

// Document Management Handlers

// AddFile adds a file to a collection
func AddFile(c *gin.Context) {
	var req AddFileRequest

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
	if kb.Instance == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Knowledge base not initialized",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// TODO: Call external function to get file info
	// filename, contentType, err := GetFileInfo(req.FileID)
	// For now, use hardcoded values
	filename := "document.pdf"
	contentType := "application/pdf"

	// Convert request to UpsertOptions
	upsertOptions, err := req.BaseUpsertRequest.ToUpsertOptions(filename, contentType)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Failed to convert request to upsert options: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Perform upsert operation with file ID
	// Note: In a real implementation, you would need to fetch the file content
	// using req.FileID and pass it to the upsert operation
	docID, err := kb.Instance.AddFile(c.Request.Context(), req.FileID, upsertOptions)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to upsert file: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Return success response
	result := gin.H{
		"message":       "File added successfully",
		"collection_id": req.CollectionID,
		"file_id":       req.FileID,
		"doc_id":        docID,
	}

	response.RespondWithSuccess(c, response.StatusCreated, result)
}

// AddText adds text to a collection
func AddText(c *gin.Context) {
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
	if kb.Instance == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Knowledge base not initialized",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Convert request to UpsertOptions
	upsertOptions, err := req.BaseUpsertRequest.ToUpsertOptions()
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Failed to convert request to upsert options: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Perform upsert operation with text
	docID, err := kb.Instance.AddText(c.Request.Context(), req.Text, upsertOptions)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to upsert text: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Return success response
	result := gin.H{
		"message":       "Text added successfully",
		"collection_id": req.CollectionID,
		"doc_id":        docID,
	}

	response.RespondWithSuccess(c, response.StatusCreated, result)
}

// AddURL adds a URL to a collection
func AddURL(c *gin.Context) {
	var req AddURLRequest

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
	if kb.Instance == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Knowledge base not initialized",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Convert request to UpsertOptions
	upsertOptions, err := req.BaseUpsertRequest.ToUpsertOptions()
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Failed to convert request to upsert options: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Perform upsert operation with URL
	docID, err := kb.Instance.AddURL(c.Request.Context(), req.URL, upsertOptions)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to upsert URL: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Return success response
	result := gin.H{
		"message":       "URL added successfully",
		"collection_id": req.CollectionID,
		"url":           req.URL,
		"doc_id":        docID,
	}

	response.RespondWithSuccess(c, response.StatusCreated, result)
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
