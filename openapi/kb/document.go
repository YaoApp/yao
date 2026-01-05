package kb

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/kb"
	kbapi "github.com/yaoapp/yao/kb/api"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/response"
)

// Document Management Handlers

// ListDocuments lists documents with pagination
func ListDocuments(c *gin.Context) {
	// Check if kb.API is available
	if !checkKBAPI(c) {
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

	// Parse select parameter
	var selectFields []interface{}
	if selectParam := strings.TrimSpace(c.Query("select")); selectParam != "" {
		requestedFields := strings.Split(selectParam, ",")
		for _, field := range requestedFields {
			field = strings.TrimSpace(field)
			if field != "" && kbapi.AvailableDocumentFields[field] {
				selectFields = append(selectFields, field)
			}
		}
		// If no valid fields found, use default
		if len(selectFields) == 0 {
			selectFields = kbapi.DefaultDocumentFields
		}
	} else {
		selectFields = kbapi.DefaultDocumentFields
	}

	// Get authorized information
	authInfo := authorized.GetInfo(c)

	// Build filter for kb.API
	filter := &kbapi.ListDocumentsFilter{
		Page:     page,
		PageSize: pagesize,
		Keywords: strings.TrimSpace(c.Query("keywords")),
		Tag:      strings.TrimSpace(c.Query("tag")),
		Select:   selectFields,
	}

	// Filter by collection_id
	collectionID := strings.TrimSpace(c.Query("collection_id"))
	if collectionID != "" {
		// Validate collection permission
		hasPermission, err := checkCollectionPermission(authInfo, collectionID, true)
		if err != nil {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrServerError.Code,
				ErrorDescription: err.Error(),
			}
			response.RespondWithError(c, response.StatusForbidden, errorResp)
			return
		}

		// 403 Forbidden
		if !hasPermission {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrAccessDenied.Code,
				ErrorDescription: "Forbidden: No permission to view collection",
			}
			response.RespondWithError(c, response.StatusForbidden, errorResp)
			return
		}

		filter.CollectionID = collectionID
	} else {
		// Filter by authorization constraints
		filter.AuthFilters = AuthFilter(c, authInfo)
	}

	// Filter by status (support multiple values separated by comma)
	if statusParam := strings.TrimSpace(c.Query("status")); statusParam != "" {
		statusList := strings.Split(statusParam, ",")
		var statusValues []string
		for _, status := range statusList {
			status = strings.TrimSpace(status)
			if status != "" {
				statusValues = append(statusValues, status)
			}
		}
		filter.Status = statusValues
	}

	// Filter by status_not (exclude specific statuses)
	if statusNotParam := strings.TrimSpace(c.Query("status_not")); statusNotParam != "" {
		statusNotList := strings.Split(statusNotParam, ",")
		var statusNotValues []string
		for _, status := range statusNotList {
			status = strings.TrimSpace(status)
			if status != "" {
				statusNotValues = append(statusNotValues, status)
			}
		}
		filter.StatusNot = statusNotValues
	}

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
		if !kbapi.ValidDocumentSortFields[sortField] {
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
	filter.Sort = orders

	// Query documents using kb.API
	result, err := kb.API.ListDocuments(c.Request.Context(), filter)
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
	// Check if kb.API is available
	if !checkKBAPI(c) {
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

	// Parse select parameter - same logic as ListDocuments
	var selectFields []interface{}
	if selectParam := strings.TrimSpace(c.Query("select")); selectParam != "" {
		requestedFields := strings.Split(selectParam, ",")
		for _, field := range requestedFields {
			field = strings.TrimSpace(field)
			if field != "" && kbapi.AvailableDocumentFields[field] {
				selectFields = append(selectFields, field)
			}
		}
		// If no valid fields found, use default
		if len(selectFields) == 0 {
			selectFields = kbapi.DefaultDocumentFields
		}
	} else {
		selectFields = kbapi.DefaultDocumentFields
	}

	// Build params for kb.API
	params := &kbapi.GetDocumentParams{
		Select: selectFields,
	}

	// Query single document using kb.API
	result, err := kb.API.GetDocument(c.Request.Context(), docID, params)
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
	// Check if kb.API is available
	if !checkKBAPI(c) {
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

	// Validate document permissions
	authInfo := authorized.GetInfo(c)
	checkedCollections := make(map[string]bool)
	for _, docID := range validDocIDs {
		collectionID := extractCollectionIDFromDocID(docID)
		if collectionID == "" {
			collectionID = "default"
		}

		// Skip if already checked
		if checkedCollections[collectionID] {
			continue
		}
		checkedCollections[collectionID] = true

		// Check update permission
		hasPermission, err := checkCollectionPermission(authInfo, collectionID)
		if err != nil {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrServerError.Code,
				ErrorDescription: err.Error(),
			}
			response.RespondWithError(c, response.StatusForbidden, errorResp)
			return
		}

		// 403 Forbidden
		if !hasPermission {
			errorResp := &response.ErrorResponse{
				Code:             response.ErrAccessDenied.Code,
				ErrorDescription: "Forbidden: No permission to update collection",
			}
			response.RespondWithError(c, response.StatusForbidden, errorResp)
			return
		}
	}

	// Remove documents using kb.API
	result, err := kb.API.RemoveDocuments(c.Request.Context(), &kbapi.RemoveDocumentsParams{
		DocumentIDs: validDocIDs,
	})
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to remove documents: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Return success response with deletion count
	c.JSON(http.StatusOK, result)
}

// checkKBAPI checks if kb.API is available
func checkKBAPI(c *gin.Context) bool {
	if kb.API == nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Knowledge base API not initialized",
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return false
	}
	return true
}

// extractCollectionIDFromDocID extracts collection ID from document ID
// Document ID format: {prefix}_{collection_id}__{random_id}
func extractCollectionIDFromDocID(docID string) string {
	parts := strings.Split(docID, "__")
	if len(parts) < 2 {
		return ""
	}

	return parts[0]
	// // First part contains prefix_collection_id
	// prefix := parts[0]
	// // Find the first underscore to skip the prefix
	// idx := strings.Index(prefix, "_")
	// if idx == -1 {
	// 	return prefix
	// }
	// return prefix[idx+1:]
}
