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
