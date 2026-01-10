package api

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
)

// ListDocuments lists documents with pagination and filtering
func (instance *KBInstance) ListDocuments(ctx context.Context, filter *ListDocumentsFilter) (*ListDocumentsResult, error) {
	page := filter.Page
	if page <= 0 {
		page = DefaultPage
	}

	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	} else if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	// Process select fields
	selectFields := filter.Select
	if len(selectFields) == 0 {
		selectFields = DefaultDocumentFields
	} else {
		// Filter valid fields
		validFields := []interface{}{}
		for _, field := range selectFields {
			if fieldStr, ok := field.(string); ok && AvailableDocumentFields[fieldStr] {
				validFields = append(validFields, field)
			}
		}
		if len(validFields) == 0 {
			selectFields = DefaultDocumentFields
		} else {
			selectFields = validFields
		}
	}

	// Build query parameters
	param := model.QueryParam{Select: selectFields}

	// Build wheres
	var wheres []model.QueryWhere

	// Add auth filters
	if len(filter.AuthFilters) > 0 {
		wheres = append(wheres, filter.AuthFilters...)
	}

	// Filter by collection_id
	if filter.CollectionID != "" {
		wheres = append(wheres, model.QueryWhere{
			Column: "collection_id",
			Value:  filter.CollectionID,
		})
	}

	// Filter by keywords (search in name and description)
	if filter.Keywords != "" {
		wheres = append(wheres, model.QueryWhere{
			Column: "name",
			Value:  "%" + filter.Keywords + "%",
			OP:     "like",
		})
		wheres = append(wheres, model.QueryWhere{
			Column: "description",
			Value:  "%" + filter.Keywords + "%",
			OP:     "like",
			Method: "orwhere",
		})
	}

	// Filter by tag
	if filter.Tag != "" {
		wheres = append(wheres, model.QueryWhere{
			Column: "tags",
			Value:  "%" + filter.Tag + "%",
			OP:     "like",
		})
	}

	// Filter by status
	if len(filter.Status) > 0 {
		statusValues := []interface{}{}
		for _, status := range filter.Status {
			if status != "" {
				statusValues = append(statusValues, status)
			}
		}

		if len(statusValues) > 0 {
			if len(statusValues) == 1 {
				wheres = append(wheres, model.QueryWhere{
					Column: "status",
					Value:  statusValues[0],
				})
			} else {
				wheres = append(wheres, model.QueryWhere{
					Column: "status",
					Value:  statusValues,
					OP:     "in",
				})
			}
		}
	}

	// Filter by status_not (exclude specific statuses)
	if len(filter.StatusNot) > 0 {
		for _, status := range filter.StatusNot {
			if status != "" {
				wheres = append(wheres, model.QueryWhere{
					Column: "status",
					Value:  status,
					OP:     "!=",
				})
			}
		}
	}

	param.Wheres = wheres

	// Process sort orders
	orders := filter.Sort
	if len(orders) == 0 {
		orders = DefaultDocumentSort
	} else {
		// Validate sort fields
		validOrders := []model.QueryOrder{}
		for _, order := range orders {
			if ValidDocumentSortFields[order.Column] {
				// Validate sort order
				if order.Option != "asc" && order.Option != "desc" {
					order.Option = "desc"
				}
				validOrders = append(validOrders, order)
			}
		}
		if len(validOrders) == 0 {
			orders = DefaultDocumentSort
		} else {
			orders = validOrders
		}
	}

	param.Orders = orders

	// Query documents
	result, err := instance.Config.SearchDocuments(param, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to search documents: %w", err)
	}

	// Convert result to ListDocumentsResult
	listResult := &ListDocumentsResult{
		Page:     page,
		PageSize: pageSize,
		Data:     make([]map[string]interface{}, 0),
	}

	// Extract pagination data from result
	if data, ok := result["data"].([]map[string]interface{}); ok {
		listResult.Data = data
	} else if data, ok := result["data"].([]interface{}); ok {
		converted := make([]map[string]interface{}, 0, len(data))
		for _, item := range data {
			if mapItem, ok := item.(map[string]interface{}); ok {
				converted = append(converted, mapItem)
			}
		}
		listResult.Data = converted
	} else if data, ok := result["data"].([]maps.MapStr); ok {
		converted := make([]map[string]interface{}, 0, len(data))
		for _, item := range data {
			converted = append(converted, map[string]interface{}(item))
		}
		listResult.Data = converted
	}

	if next, ok := result["next"].(int); ok {
		listResult.Next = next
	}
	if prev, ok := result["prev"].(int); ok {
		listResult.Prev = prev
	}
	if total, ok := result["total"].(int); ok {
		listResult.Total = total
	}
	if pagecnt, ok := result["pagecnt"].(int); ok {
		listResult.PageCnt = pagecnt
	}

	return listResult, nil
}

// GetDocument retrieves a document by ID
func (instance *KBInstance) GetDocument(ctx context.Context, docID string, params *GetDocumentParams) (map[string]interface{}, error) {
	if docID == "" {
		return nil, fmt.Errorf("document ID is required")
	}

	// Process select fields
	var selectFields []interface{}
	if params != nil && len(params.Select) > 0 {
		for _, field := range params.Select {
			if fieldStr, ok := field.(string); ok && AvailableDocumentFields[fieldStr] {
				selectFields = append(selectFields, field)
			}
		}
	}
	if len(selectFields) == 0 {
		selectFields = DefaultDocumentFields
	}

	// Build query parameters
	param := model.QueryParam{
		Select: selectFields,
	}

	// Query single document
	result, err := instance.Config.FindDocument(docID, param)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// RemoveDocuments removes documents by IDs
func (instance *KBInstance) RemoveDocuments(ctx context.Context, params *RemoveDocumentsParams) (*RemoveDocumentsResult, error) {
	if len(params.DocumentIDs) == 0 {
		return nil, fmt.Errorf("document IDs are required")
	}

	// Remove documents using GraphRag
	deletedCount, err := instance.GraphRag.RemoveDocs(ctx, params.DocumentIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to remove documents from GraphRag: %w", err)
	}

	// Also remove documents from the database and track collections to update
	dbDeletedCount := 0
	collectionsToUpdate := make(map[string]bool)

	for _, docID := range params.DocumentIDs {
		// Get document info before deletion to track collection
		if docInfo, err := instance.Config.FindDocument(docID, model.QueryParam{
			Select: []interface{}{"collection_id"},
		}); err == nil && docInfo != nil {
			if collectionID, ok := docInfo["collection_id"].(string); ok && collectionID != "" {
				collectionsToUpdate[collectionID] = true
			}
		}

		if err := instance.Config.RemoveDocument(docID); err != nil {
			return nil, fmt.Errorf("failed to remove document from database: %w", err)
		}
		dbDeletedCount++
	}

	// Update document counts for affected collections and sync to GraphRag
	for collectionID := range collectionsToUpdate {
		if err := instance.updateDocumentCountWithSync(ctx, collectionID); err != nil {
			log.Error("Failed to update document count for collection %s: %v", collectionID, err)
		}
	}

	return &RemoveDocumentsResult{
		Message:        "Documents removed successfully",
		DeletedCount:   deletedCount,
		RequestedCount: len(params.DocumentIDs),
		DBDeletedCount: dbDeletedCount,
	}, nil
}

// GetDocumentsContent retrieves content for multiple documents by IDs
// Returns document info with content (only text-based files are supported)
func (instance *KBInstance) GetDocumentsContent(ctx context.Context, docIDs []string) ([]map[string]interface{}, error) {
	if len(docIDs) == 0 {
		return nil, fmt.Errorf("document IDs are required")
	}

	// Get document model
	modelName := "__yao.kb.document"
	if instance.Config != nil && instance.Config.DocumentModel != "" {
		modelName = instance.Config.DocumentModel
	}

	mod := model.Select(modelName)
	if mod == nil {
		return nil, fmt.Errorf("document model not found: %s", modelName)
	}

	results := make([]map[string]interface{}, 0, len(docIDs))
	for _, docID := range docIDs {
		param := model.QueryParam{
			Select: []interface{}{"document_id", "collection_id", "name", "text_content", "type", "status", "file_path", "file_mime_type"},
			Wheres: []model.QueryWhere{
				{Column: "document_id", Value: docID},
			},
			Limit: 1,
		}

		docs, err := mod.Get(param)
		if err != nil {
			log.Warn("Failed to get document %s: %v", docID, err)
			continue
		}

		if len(docs) == 0 {
			log.Warn("Document not found: %s", docID)
			continue
		}

		doc := docs[0]
		content := ""
		contentType := "text/plain"

		// Get content type
		filePath, _ := doc["file_path"].(string)
		if mimeType, ok := doc["file_mime_type"].(string); ok && mimeType != "" {
			contentType = mimeType
		} else if filePath != "" {
			contentType = inferContentType(filePath)
		}

		// Only process text-based files
		if isTextContentType(contentType) {
			// 1. Try text_content first
			if textContent, ok := doc["text_content"].(string); ok && textContent != "" {
				content = textContent
			} else if filePath != "" {
				// 2. Read from file_path
				fileContent, err := readFileContent(filePath)
				if err != nil {
					log.Warn("Failed to read file content for %s: %v", docID, err)
				} else {
					content = fileContent
				}
			}
		}

		results = append(results, map[string]interface{}{
			"document_id":   docID,
			"collection_id": doc["collection_id"],
			"name":          doc["name"],
			"content":       content,
			"content_type":  contentType,
			"type":          doc["type"],
			"status":        doc["status"],
		})
	}

	return results, nil
}

// isTextContentType checks if the content type is text-based
func isTextContentType(contentType string) bool {
	textTypes := []string{
		"text/",
		"application/json",
		"application/xml",
		"application/javascript",
	}
	for _, tt := range textTypes {
		if strings.HasPrefix(contentType, tt) {
			return true
		}
	}
	return false
}

// inferContentType infers content type from file extension
func inferContentType(filePath string) string {
	lower := strings.ToLower(filePath)
	switch {
	case strings.HasSuffix(lower, ".md"):
		return "text/markdown"
	case strings.HasSuffix(lower, ".txt"):
		return "text/plain"
	case strings.HasSuffix(lower, ".html"), strings.HasSuffix(lower, ".htm"):
		return "text/html"
	case strings.HasSuffix(lower, ".json"):
		return "application/json"
	case strings.HasSuffix(lower, ".xml"):
		return "application/xml"
	case strings.HasSuffix(lower, ".csv"):
		return "text/csv"
	case strings.HasSuffix(lower, ".pdf"):
		return "application/pdf"
	default:
		return "text/plain"
	}
}

// readFileContent reads the content of a file
func readFileContent(filePath string) (string, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", err
	}

	// Read file content
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	// Convert to string and handle encoding
	content := string(data)

	// Basic cleanup - remove null bytes and normalize line endings
	content = strings.ReplaceAll(content, "\x00", "")
	content = strings.ReplaceAll(content, "\r\n", "\n")

	return content, nil
}
