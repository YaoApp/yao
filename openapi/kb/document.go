package kb

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/attachment"
	"github.com/yaoapp/yao/kb"
	"github.com/yaoapp/yao/openapi/response"
)

// Document field definitions
var (
	// availableDocumentFields defines all available fields for security filtering
	availableDocumentFields = map[string]bool{
		"id": true, "document_id": true, "collection_id": true, "name": true,
		"description": true, "status": true, "type": true, "size": true,
		"segment_count": true, "job_id": true, "uploader_id": true, "tags": true,
		"locale": true, "system": true, "readonly": true, "sort": true, "cover": true,
		"file_id": true, "file_name": true, "file_mime_type": true,
		"url": true, "url_title": true, "text_content": true,
		"converter_provider_id": true, "converter_option_id": true, "converter_properties": true,
		"fetcher_provider_id": true, "fetcher_option_id": true, "fetcher_properties": true,
		"chunking_provider_id": true, "chunking_option_id": true, "chunking_properties": true,
		"extraction_provider_id": true, "extraction_option_id": true, "extraction_properties": true,
		"processed_at": true, "error_message": true, "created_at": true, "updated_at": true,
	}

	// defaultDocumentFields defines the default compact field list
	defaultDocumentFields = []interface{}{
		"id", "document_id", "collection_id", "name", "description",
		"cover", "tags", "type", "size", "segment_count", "status", "locale",
		"system", "readonly", "file_id", "file_name", "file_mime_type", "uploader_id",
		"url", "url_title", "text_content", // 添加 URL 和文本内容字段
		"error_message", "created_at", "updated_at",
	}

	// validSortFields defines valid fields for sorting
	validSortFields = map[string]bool{
		"created_at":    true,
		"updated_at":    true,
		"name":          true,
		"size":          true,
		"segment_count": true,
		"sort":          true,
		"processed_at":  true,
	}
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
	// Check if kb.Instance is available
	if !checkKBInstance(c) {
		return
	}

	// Parse pagination parameters
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	pagesize := 20
	if pagesizeStr := c.Query("pagesize"); pagesizeStr != "" {
		if ps, err := strconv.Atoi(pagesizeStr); err == nil && ps > 0 && ps <= 100 {
			pagesize = ps
		}
	}

	// Get KB instance and config
	kbInstance := kb.Instance.(*kb.KnowledgeBase)
	config := kbInstance.Config

	// Parse select parameter
	var selectFields []interface{}
	if selectParam := strings.TrimSpace(c.Query("select")); selectParam != "" {
		requestedFields := strings.Split(selectParam, ",")
		for _, field := range requestedFields {
			field = strings.TrimSpace(field)
			if field != "" && availableDocumentFields[field] {
				selectFields = append(selectFields, field)
			}
		}
		// If no valid fields found, use default
		if len(selectFields) == 0 {
			selectFields = defaultDocumentFields
		}
	} else {
		selectFields = defaultDocumentFields
	}

	// Build query parameters
	param := model.QueryParam{
		Select: selectFields,
	}

	// Add filters
	var wheres []model.QueryWhere

	// Filter by keywords (search in name and description)
	if keywords := strings.TrimSpace(c.Query("keywords")); keywords != "" {
		wheres = append(wheres, model.QueryWhere{
			Column: "name",
			Value:  "%" + keywords + "%",
			OP:     "like",
		})
		wheres = append(wheres, model.QueryWhere{
			Column: "description",
			Value:  "%" + keywords + "%",
			OP:     "like",
			Wheres: []model.QueryWhere{},
			Method: "orwhere",
		})
	}

	// Filter by tag
	if tag := strings.TrimSpace(c.Query("tag")); tag != "" {
		wheres = append(wheres, model.QueryWhere{
			Column: "tags",
			Value:  "%" + tag + "%",
			OP:     "like",
		})
	}

	// Filter by collection_id
	if collectionID := strings.TrimSpace(c.Query("collection_id")); collectionID != "" {
		wheres = append(wheres, model.QueryWhere{
			Column: "collection_id",
			Value:  collectionID,
		})
	}

	// Filter by status (support multiple values separated by comma)
	if statusParam := strings.TrimSpace(c.Query("status")); statusParam != "" {
		statusList := strings.Split(statusParam, ",")
		var statusValues []interface{}
		for _, status := range statusList {
			status = strings.TrimSpace(status)
			if status != "" {
				statusValues = append(statusValues, status)
			}
		}

		if len(statusValues) > 0 {
			if len(statusValues) == 1 {
				// Single status
				wheres = append(wheres, model.QueryWhere{
					Column: "status",
					Value:  statusValues[0],
				})
			} else {
				// Multiple status - use IN clause
				wheres = append(wheres, model.QueryWhere{
					Column: "status",
					Value:  statusValues,
					OP:     "in",
				})
			}
		}
	}

	// Filter by status_not (exclude specific statuses)
	if statusNotParam := strings.TrimSpace(c.Query("status_not")); statusNotParam != "" {
		statusNotList := strings.Split(statusNotParam, ",")
		var statusNotValues []interface{}
		for _, status := range statusNotList {
			status = strings.TrimSpace(status)
			if status != "" {
				statusNotValues = append(statusNotValues, status)
			}
		}

		if len(statusNotValues) > 0 {
			if len(statusNotValues) == 1 {
				// Single status exclusion
				wheres = append(wheres, model.QueryWhere{
					Column: "status",
					Value:  statusNotValues[0],
					OP:     "!=",
				})
			} else {
				// Multiple status exclusion - use NOT IN clause
				// Since gou/model doesn't support "notin" OP directly,
				// we need to use a different approach or multiple != conditions
				for _, status := range statusNotValues {
					wheres = append(wheres, model.QueryWhere{
						Column: "status",
						Value:  status,
						OP:     "!=",
					})
				}
			}
		}
	}

	param.Wheres = wheres

	// Add ordering
	sortParam := strings.TrimSpace(c.Query("sort"))
	if sortParam == "" {
		sortParam = "created_at desc" // Default sort
	}

	// Parse sort parameter (format: "field1 direction1,field2 direction2")
	var orders []model.QueryOrder
	sortItems := strings.Split(sortParam, ",")

	for _, sortItem := range sortItems {
		sortItem = strings.TrimSpace(sortItem)
		if sortItem == "" {
			continue
		}

		// Parse each sort item (format: "field direction")
		sortParts := strings.Fields(sortItem)
		sortField := "created_at" // Default field
		sortOrder := "desc"       // Default order

		if len(sortParts) >= 1 {
			sortField = sortParts[0]
		}
		if len(sortParts) >= 2 {
			sortOrder = strings.ToLower(sortParts[1])
		}

		// Validate sort field
		if !validSortFields[sortField] {
			continue // Skip invalid fields
		}

		// Validate sort order
		if sortOrder != "asc" && sortOrder != "desc" {
			sortOrder = "desc" // Default order
		}

		orders = append(orders, model.QueryOrder{
			Column: sortField,
			Option: sortOrder,
		})
	}

	// If no valid orders found, use default
	if len(orders) == 0 {
		orders = []model.QueryOrder{
			{Column: "created_at", Option: "desc"},
		}
	}

	param.Orders = orders

	// Query documents using KB config
	result, err := config.SearchDocuments(param, page, pagesize)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to search documents: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetDocument gets document details by document ID
func GetDocument(c *gin.Context) {
	// Check if kb.Instance is available
	if !checkKBInstance(c) {
		return
	}

	docID := c.Param("docID")
	if docID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Document ID is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Get KB instance and config
	kbInstance := kb.Instance.(*kb.KnowledgeBase)
	config := kbInstance.Config

	// Parse select parameter - same logic as ListDocuments
	var selectFields []interface{}
	if selectParam := strings.TrimSpace(c.Query("select")); selectParam != "" {
		requestedFields := strings.Split(selectParam, ",")
		for _, field := range requestedFields {
			field = strings.TrimSpace(field)
			if field != "" && availableDocumentFields[field] {
				selectFields = append(selectFields, field)
			}
		}
		// If no valid fields found, use default
		if len(selectFields) == 0 {
			selectFields = defaultDocumentFields
		}
	} else {
		selectFields = defaultDocumentFields
	}

	// Build query parameters
	param := model.QueryParam{
		Select: selectFields,
	}

	// Query single document using KB config
	result, err := config.FindDocument(docID, param)
	if err != nil {
		if strings.Contains(err.Error(), "document not found") {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrInvalidRequest.Code,
				ErrorDescription: "Document not found: " + docID,
			}
			response.RespondWithError(c, response.StatusNotFound, errorResp)
			return
		}

		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to get document: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	c.JSON(http.StatusOK, result)
}

// RemoveDocs removes documents by IDs
func RemoveDocs(c *gin.Context) {
	// Check if kb.Instance is available
	if !checkKBInstance(c) {
		return
	}

	// Parse document_ids from query parameter (comma-separated string)
	docIDsParam := strings.TrimSpace(c.Query("document_ids"))
	if docIDsParam == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "document_ids query parameter is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Split comma-separated document IDs
	docIDs := strings.Split(docIDsParam, ",")
	var validDocIDs []string
	for _, id := range docIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			validDocIDs = append(validDocIDs, id)
		}
	}

	if len(validDocIDs) == 0 {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "No valid document IDs provided",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Get KB config for database operations
	config, err := kb.GetConfig()
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to get KB config: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Remove documents using GraphRAG
	deletedCount, err := kb.Instance.RemoveDocs(c.Request.Context(), validDocIDs)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to remove documents: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Also remove documents from the database and track collections to update
	dbDeletedCount := 0
	collectionsToUpdate := make(map[string]bool) // Track unique collection IDs

	for _, docID := range validDocIDs {
		// Get document info before deletion to track collection
		if docInfo, err := config.FindDocument(docID, model.QueryParam{
			Select: []interface{}{"collection_id"},
		}); err == nil && docInfo != nil {
			if collectionID, ok := docInfo["collection_id"].(string); ok && collectionID != "" {
				collectionsToUpdate[collectionID] = true
			}
		}

		if err := config.RemoveDocument(docID); err != nil {
			// Log the error but don't fail the entire operation
			// since the document was already removed from GraphRAG
			errorResp := &response.ErrorResponse{
				Code:             response.ErrServerError.Code,
				ErrorDescription: "Failed to remove document from database: " + err.Error(),
			}
			response.RespondWithError(c, response.StatusInternalServerError, errorResp)
			return
		}
		dbDeletedCount++
	}

	// Update document counts for affected collections and sync to GraphRag
	for collectionID := range collectionsToUpdate {
		if err := UpdateDocumentCountWithSync(collectionID, config); err != nil {
			// Log error but don't fail the operation
			// TODO: Add proper logging
			// log.Error("Failed to update document count for collection %s: %v", collectionID, err)
		}
	}

	// Return success response with deletion count
	c.JSON(http.StatusOK, gin.H{
		"message":          "Documents removed successfully",
		"deleted_count":    deletedCount,
		"requested_count":  len(validDocIDs),
		"db_deleted_count": dbDeletedCount,
	})
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
