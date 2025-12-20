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

// ========== Document Types ==========

// ListDocumentsFilter represents the filter options for listing documents
type ListDocumentsFilter struct {
	Page         int                `json:"page" yaml:"page"`
	PageSize     int                `json:"pagesize" yaml:"pagesize"`
	CollectionID string             `json:"collection_id,omitempty" yaml:"collection_id,omitempty"`
	Keywords     string             `json:"keywords,omitempty" yaml:"keywords,omitempty"`
	Tag          string             `json:"tag,omitempty" yaml:"tag,omitempty"`
	Status       []string           `json:"status,omitempty" yaml:"status,omitempty"`
	StatusNot    []string           `json:"status_not,omitempty" yaml:"status_not,omitempty"`
	Select       []interface{}      `json:"select,omitempty" yaml:"select,omitempty"`
	Sort         []model.QueryOrder `json:"sort,omitempty" yaml:"sort,omitempty"`
	AuthFilters  []model.QueryWhere `json:"-" yaml:"-"` // Internal: authentication filters
}

// ListDocumentsResult represents the result of listing documents
type ListDocumentsResult struct {
	Data     []map[string]interface{} `json:"data" yaml:"data"`
	Next     int                      `json:"next" yaml:"next"`
	Prev     int                      `json:"prev" yaml:"prev"`
	Page     int                      `json:"page" yaml:"page"`
	PageSize int                      `json:"pagesize" yaml:"pagesize"`
	Total    int                      `json:"total" yaml:"total"`
	PageCnt  int                      `json:"pagecnt" yaml:"pagecnt"`
}

// GetDocumentParams represents the parameters for getting a document
type GetDocumentParams struct {
	Select []interface{} `json:"select,omitempty" yaml:"select,omitempty"`
}

// RemoveDocumentsParams represents the parameters for removing documents
type RemoveDocumentsParams struct {
	DocumentIDs []string `json:"document_ids" yaml:"document_ids"`
}

// RemoveDocumentsResult represents the result of removing documents
type RemoveDocumentsResult struct {
	Message        string `json:"message" yaml:"message"`
	DeletedCount   int    `json:"deleted_count" yaml:"deleted_count"`
	RequestedCount int    `json:"requested_count" yaml:"requested_count"`
	DBDeletedCount int    `json:"db_deleted_count" yaml:"db_deleted_count"`
}

// AddFileParams represents the parameters for adding a file
type AddFileParams struct {
	CollectionID string                 `json:"collection_id" yaml:"collection_id"`
	FileID       string                 `json:"file_id" yaml:"file_id"`
	Uploader     string                 `json:"uploader,omitempty" yaml:"uploader,omitempty"`
	DocID        string                 `json:"doc_id,omitempty" yaml:"doc_id,omitempty"`
	Locale       string                 `json:"locale,omitempty" yaml:"locale,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Chunking     *ProviderConfigParams  `json:"chunking" yaml:"chunking"`
	Embedding    *ProviderConfigParams  `json:"embedding" yaml:"embedding"`
	Extraction   *ProviderConfigParams  `json:"extraction,omitempty" yaml:"extraction,omitempty"`
	Fetcher      *ProviderConfigParams  `json:"fetcher,omitempty" yaml:"fetcher,omitempty"`
	Converter    *ProviderConfigParams  `json:"converter,omitempty" yaml:"converter,omitempty"`
	Job          *JobOptionsParams      `json:"job,omitempty" yaml:"job,omitempty"`
	AuthScope    map[string]interface{} `json:"-" yaml:"-"` // Internal: authentication scope fields
}

// AddTextParams represents the parameters for adding text
type AddTextParams struct {
	CollectionID string                 `json:"collection_id" yaml:"collection_id"`
	Text         string                 `json:"text" yaml:"text"`
	DocID        string                 `json:"doc_id,omitempty" yaml:"doc_id,omitempty"`
	Locale       string                 `json:"locale,omitempty" yaml:"locale,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Chunking     *ProviderConfigParams  `json:"chunking" yaml:"chunking"`
	Embedding    *ProviderConfigParams  `json:"embedding" yaml:"embedding"`
	Extraction   *ProviderConfigParams  `json:"extraction,omitempty" yaml:"extraction,omitempty"`
	Fetcher      *ProviderConfigParams  `json:"fetcher,omitempty" yaml:"fetcher,omitempty"`
	Converter    *ProviderConfigParams  `json:"converter,omitempty" yaml:"converter,omitempty"`
	Job          *JobOptionsParams      `json:"job,omitempty" yaml:"job,omitempty"`
	AuthScope    map[string]interface{} `json:"-" yaml:"-"` // Internal: authentication scope fields
}

// AddURLParams represents the parameters for adding a URL
type AddURLParams struct {
	CollectionID string                 `json:"collection_id" yaml:"collection_id"`
	URL          string                 `json:"url" yaml:"url"`
	DocID        string                 `json:"doc_id,omitempty" yaml:"doc_id,omitempty"`
	Locale       string                 `json:"locale,omitempty" yaml:"locale,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Chunking     *ProviderConfigParams  `json:"chunking" yaml:"chunking"`
	Embedding    *ProviderConfigParams  `json:"embedding" yaml:"embedding"`
	Extraction   *ProviderConfigParams  `json:"extraction,omitempty" yaml:"extraction,omitempty"`
	Fetcher      *ProviderConfigParams  `json:"fetcher,omitempty" yaml:"fetcher,omitempty"`
	Converter    *ProviderConfigParams  `json:"converter,omitempty" yaml:"converter,omitempty"`
	Job          *JobOptionsParams      `json:"job,omitempty" yaml:"job,omitempty"`
	AuthScope    map[string]interface{} `json:"-" yaml:"-"` // Internal: authentication scope fields
}

// ProviderConfigParams represents a provider configuration
type ProviderConfigParams struct {
	ProviderID string                 `json:"provider_id" yaml:"provider_id"`
	OptionID   string                 `json:"option_id,omitempty" yaml:"option_id,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty" yaml:"properties,omitempty"`
}

// JobOptionsParams contains job options for async operations
type JobOptionsParams struct {
	Name        string `json:"name,omitempty" yaml:"name,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Icon        string `json:"icon,omitempty" yaml:"icon,omitempty"`
	Category    string `json:"category,omitempty" yaml:"category,omitempty"`
}

// AddDocumentResult represents the result of adding a document (sync)
type AddDocumentResult struct {
	Message      string `json:"message" yaml:"message"`
	CollectionID string `json:"collection_id" yaml:"collection_id"`
	DocID        string `json:"doc_id" yaml:"doc_id"`
	FileID       string `json:"file_id,omitempty" yaml:"file_id,omitempty"`
	URL          string `json:"url,omitempty" yaml:"url,omitempty"`
}

// AddDocumentAsyncResult represents the result of adding a document (async)
type AddDocumentAsyncResult struct {
	JobID string `json:"job_id" yaml:"job_id"`
	DocID string `json:"doc_id" yaml:"doc_id"`
}

// ========== Search Types ==========

// SearchMode defines the search strategy
type SearchMode string

const (
	// SearchModeVector performs pure vector similarity search
	SearchModeVector SearchMode = "vector"
	// SearchModeGraph performs graph traversal to find related segments
	SearchModeGraph SearchMode = "graph"
	// SearchModeExpand uses graph to expand/associate entities, then enhances vector search
	// This enables deeper semantic connections through entity relationships
	SearchModeExpand SearchMode = "expand"
)

// Query represents a single search query
type Query struct {
	// CollectionID is the collection to search in (required)
	CollectionID string `json:"collection_id" yaml:"collection_id"`

	// Input is the direct search query text (e.g., LLM-summarized query)
	// Either Input or Messages is required; Input takes precedence if both provided
	Input string `json:"input,omitempty" yaml:"input,omitempty"`

	// Messages is the conversation history for context-aware search
	// The last user message is used as the query if Input is empty
	Messages []types.ChatMessage `json:"messages,omitempty" yaml:"messages,omitempty"`

	// Mode determines the search strategy (optional, defaults to collection config or "expand")
	// - vector: pure vector similarity search
	// - graph: graph traversal to find related segments
	// - expand: graph-based entity expansion/association + vector search
	Mode SearchMode `json:"mode,omitempty" yaml:"mode,omitempty"`

	// DocumentID filters results to a specific document (optional)
	DocumentID string `json:"document_id,omitempty" yaml:"document_id,omitempty"`

	// Threshold filters results below this similarity threshold (optional)
	Threshold float64 `json:"threshold,omitempty" yaml:"threshold,omitempty"`

	// Metadata filters segments by metadata fields (optional)
	Metadata map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	// Graph search options (used when Mode is graph or expand)
	MaxDepth int `json:"max_depth,omitempty" yaml:"max_depth,omitempty"` // Max traversal depth (default: 2)

	// Pagination options
	// If not specified, returns default number of results
	Page     int    `json:"page,omitempty" yaml:"page,omitempty"`         // Page number (1-based), 0 means no pagination
	PageSize int    `json:"pagesize,omitempty" yaml:"pagesize,omitempty"` // Number of results per page
	Cursor   string `json:"cursor,omitempty" yaml:"cursor,omitempty"`     // Cursor for cursor-based pagination
}

// GraphData contains graph-specific search results
type GraphData struct {
	Nodes         []types.GraphNode         `json:"nodes,omitempty" yaml:"nodes,omitempty"`
	Relationships []types.GraphRelationship `json:"relationships,omitempty" yaml:"relationships,omitempty"`
}

// SearchResult represents the merged result of search operations
type SearchResult struct {
	// Segments contains the merged and deduplicated text segments with scores
	Segments []types.Segment `json:"segments" yaml:"segments"`

	// Graph contains merged nodes and relationships (only for graph/hybrid mode)
	Graph *GraphData `json:"graph,omitempty" yaml:"graph,omitempty"`

	// Pagination info
	Page       int    `json:"page,omitempty" yaml:"page,omitempty"`         // Current page number
	PageSize   int    `json:"pagesize,omitempty" yaml:"pagesize,omitempty"` // Results per page
	Total      int    `json:"total" yaml:"total"`                           // Total number of results
	TotalPages int    `json:"pagecnt,omitempty" yaml:"pagecnt,omitempty"`   // Total pages
	Next       int    `json:"next,omitempty" yaml:"next,omitempty"`         // Next page number
	Prev       int    `json:"prev,omitempty" yaml:"prev,omitempty"`         // Previous page number
	Cursor     string `json:"cursor,omitempty" yaml:"cursor,omitempty"`     // Cursor for next page
}
