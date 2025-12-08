package api

import (
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/model"
)

// CreateCollectionParams represents the parameters for creating a collection
type CreateCollectionParams struct {
	ID                  string                         `json:"id" yaml:"id"`
	Metadata            map[string]interface{}         `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	EmbeddingProviderID string                         `json:"embedding_provider_id" yaml:"embedding_provider_id"`
	EmbeddingOptionID   string                         `json:"embedding_option_id" yaml:"embedding_option_id"`
	Locale              string                         `json:"locale,omitempty" yaml:"locale,omitempty"`
	Config              *types.CreateCollectionOptions `json:"config,omitempty" yaml:"config,omitempty"`
	AuthScope           map[string]interface{}         `json:"-" yaml:"-"` // Internal: authentication scope fields
}

// CreateCollectionResult represents the result of creating a collection
type CreateCollectionResult struct {
	CollectionID string `json:"collection_id" yaml:"collection_id"`
	Message      string `json:"message" yaml:"message"`
}

// RemoveCollectionResult represents the result of removing a collection
type RemoveCollectionResult struct {
	CollectionID     string `json:"collection_id" yaml:"collection_id"`
	Removed          bool   `json:"removed" yaml:"removed"`
	DocumentsRemoved int    `json:"documents_removed" yaml:"documents_removed"`
	Message          string `json:"message" yaml:"message"`
}

// CollectionExistsResult represents the result of checking if a collection exists
type CollectionExistsResult struct {
	CollectionID string `json:"collection_id" yaml:"collection_id"`
	Exists       bool   `json:"exists" yaml:"exists"`
}

// ListCollectionsFilter represents the filter options for listing collections
type ListCollectionsFilter struct {
	Page                int                `json:"page" yaml:"page"`
	PageSize            int                `json:"pagesize" yaml:"pagesize"`
	Keywords            string             `json:"keywords,omitempty" yaml:"keywords,omitempty"`
	Status              []string           `json:"status,omitempty" yaml:"status,omitempty"`
	System              *bool              `json:"system,omitempty" yaml:"system,omitempty"`
	EmbeddingProviderID string             `json:"embedding_provider_id,omitempty" yaml:"embedding_provider_id,omitempty"`
	Select              []interface{}      `json:"select,omitempty" yaml:"select,omitempty"`
	Sort                []model.QueryOrder `json:"sort,omitempty" yaml:"sort,omitempty"`
	AuthFilters         []model.QueryWhere `json:"-" yaml:"-"` // Internal: authentication filters
}

// ListCollectionsResult represents the result of listing collections
type ListCollectionsResult struct {
	Data     []map[string]interface{} `json:"data" yaml:"data"`
	Next     int                      `json:"next" yaml:"next"`
	Prev     int                      `json:"prev" yaml:"prev"`
	Page     int                      `json:"page" yaml:"page"`
	PageSize int                      `json:"pagesize" yaml:"pagesize"`
	Total    int                      `json:"total" yaml:"total"`
	PageCnt  int                      `json:"pagecnt" yaml:"pagecnt"`
}

// UpdateMetadataParams represents the parameters for updating collection metadata
type UpdateMetadataParams struct {
	Metadata  map[string]interface{} `json:"metadata" yaml:"metadata"`
	AuthScope map[string]interface{} `json:"-" yaml:"-"` // Internal: authentication scope fields for update
}

// UpdateMetadataResult represents the result of updating collection metadata
type UpdateMetadataResult struct {
	CollectionID string `json:"collection_id" yaml:"collection_id"`
	Message      string `json:"message" yaml:"message"`
}
