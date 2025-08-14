package kb

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/yao/attachment"
	"github.com/yaoapp/yao/kb"
	"github.com/yaoapp/yao/openapi/response"
)

// SimpleJob represents a simple job for async operations
// TODO: replace with proper job system later
type SimpleJob struct {
	ID string
}

// NewJob creates a new simple job
func NewJob() *SimpleJob {
	return &SimpleJob{
		ID: uuid.New().String(),
	}
}

// Run executes the job function asynchronously and returns job ID
func (j *SimpleJob) Run(fn func()) string {
	// temporary solution to handle async operations ( TODO: use job queue )
	go fn()
	return j.ID
}

// Document Management Handlers

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

// Validator interface for request validation
type Validator interface {
	Validate() error
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
