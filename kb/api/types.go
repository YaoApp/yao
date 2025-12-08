package api

import (
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/model"
)

// CreateCollectionParams represents the parameters for creating a collection
type CreateCollectionParams struct {
	ID                  string                         `json:"id"`
	Metadata            map[string]interface{}         `json:"metadata,omitempty"`
	EmbeddingProviderID string                         `json:"embedding_provider_id"`
	EmbeddingOptionID   string                         `json:"embedding_option_id"`
	Locale              string                         `json:"locale,omitempty"`
	Config              *types.CreateCollectionOptions `json:"config,omitempty"`
	AuthScope           map[string]interface{}         `json:"-"` // Internal: authentication scope fields
}

// CreateCollectionResult represents the result of creating a collection
type CreateCollectionResult struct {
	CollectionID string `json:"collection_id"`
	Message      string `json:"message"`
}

// RemoveCollectionResult represents the result of removing a collection
type RemoveCollectionResult struct {
	CollectionID     string `json:"collection_id"`
	Removed          bool   `json:"removed"`
	DocumentsRemoved int    `json:"documents_removed"`
	Message          string `json:"message"`
}

// CollectionExistsResult represents the result of checking if a collection exists
type CollectionExistsResult struct {
	CollectionID string `json:"collection_id"`
	Exists       bool   `json:"exists"`
}

// ListCollectionsFilter represents the filter options for listing collections
type ListCollectionsFilter struct {
	Page                int                `json:"page"`
	PageSize            int                `json:"pagesize"`
	Keywords            string             `json:"keywords,omitempty"`
	Status              []string           `json:"status,omitempty"`
	System              *bool              `json:"system,omitempty"`
	EmbeddingProviderID string             `json:"embedding_provider_id,omitempty"`
	Select              []interface{}      `json:"select,omitempty"`
	Sort                []model.QueryOrder `json:"sort,omitempty"`
	AuthFilters         []model.QueryWhere `json:"-"` // Internal: authentication filters
}

// ListCollectionsResult represents the result of listing collections
type ListCollectionsResult struct {
	Data     []map[string]interface{} `json:"data"`
	Next     int                      `json:"next"`
	Prev     int                      `json:"prev"`
	Page     int                      `json:"page"`
	PageSize int                      `json:"pagesize"`
	Total    int                      `json:"total"`
	PageCnt  int                      `json:"pagecnt"`
}

// UpdateMetadataParams represents the parameters for updating collection metadata
type UpdateMetadataParams struct {
	Metadata  map[string]interface{} `json:"metadata"`
	AuthScope map[string]interface{} `json:"-"` // Internal: authentication scope fields for update
}

// UpdateMetadataResult represents the result of updating collection metadata
type UpdateMetadataResult struct {
	CollectionID string `json:"collection_id"`
	Message      string `json:"message"`
}
