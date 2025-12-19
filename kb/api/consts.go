package api

import "github.com/yaoapp/gou/model"

// Collection field definitions
var (
	// AvailableCollectionFields defines all available fields for security filtering
	AvailableCollectionFields = map[string]bool{
		"id": true, "collection_id": true, "name": true, "description": true,
		"status": true, "preset": true, "public": true, "share": true, "sort": true, "cover": true,
		"document_count": true, "embedding_provider_id": true, "embedding_option_id": true,
		"embedding_properties": true, "locale": true, "dimension": true,
		"distance_metric": true, "hnsw_m": true, "ef_construction": true,
		"ef_search": true, "num_lists": true, "num_probes": true,
		"created_at": true, "updated_at": true,
	}

	// DefaultCollectionFields defines the default compact field list
	DefaultCollectionFields = []interface{}{
		"id", "collection_id", "name", "description", "status", "preset", "public", "share",
		"sort", "cover", "document_count", "embedding_provider_id", "embedding_option_id",
		"locale", "dimension", "distance_metric", "created_at", "updated_at",
	}

	// ValidCollectionSortFields defines valid fields for sorting
	ValidCollectionSortFields = map[string]bool{
		"created_at":     true,
		"updated_at":     true,
		"name":           true,
		"sort":           true,
		"document_count": true,
		"status":         true,
	}
)

// Default pagination settings
const (
	DefaultPage     = 1
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// Default sort settings
const (
	DefaultSortField = "created_at"
	DefaultSortOrder = "desc"
)

// DefaultSort defines the default sort order for collection queries
var DefaultSort = []model.QueryOrder{
	{Column: DefaultSortField, Option: DefaultSortOrder},
}

// Default locale
const (
	DefaultLocale = "en"
)

// Document field definitions
var (
	// AvailableDocumentFields defines all available fields for security filtering
	AvailableDocumentFields = map[string]bool{
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

	// DefaultDocumentFields defines the default compact field list
	DefaultDocumentFields = []interface{}{
		"id", "document_id", "collection_id", "name", "description",
		"cover", "tags", "type", "size", "segment_count", "status", "locale",
		"system", "readonly", "file_id", "file_name", "file_mime_type", "uploader_id",
		"url", "url_title", "text_content", "job_id",
		"error_message", "created_at", "updated_at",
	}

	// ValidDocumentSortFields defines valid fields for sorting
	ValidDocumentSortFields = map[string]bool{
		"created_at":    true,
		"updated_at":    true,
		"name":          true,
		"size":          true,
		"segment_count": true,
		"sort":          true,
		"processed_at":  true,
	}
)

// DefaultDocumentSort defines the default sort order for document queries
var DefaultDocumentSort = []model.QueryOrder{
	{Column: DefaultSortField, Option: DefaultSortOrder},
}

// Default uploader
const (
	DefaultUploader = "local"
)
