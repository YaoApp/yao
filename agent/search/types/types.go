package types

import (
	"github.com/yaoapp/gou/query/gou"
)

// SearchType represents the type of search
type SearchType string

// SearchType constants
const (
	SearchTypeWeb SearchType = "web" // Web/Internet search
	SearchTypeKB  SearchType = "kb"  // Knowledge base vector search
	SearchTypeDB  SearchType = "db"  // Database search (Yao Model/QueryDSL)
)

// ScenarioType represents the QueryDSL generation scenario
type ScenarioType string

// ScenarioType constants for QueryDSL generation
const (
	ScenarioFilter      ScenarioType = "filter"      // Simple filtering queries
	ScenarioAggregation ScenarioType = "aggregation" // Aggregation/grouping queries
	ScenarioJoin        ScenarioType = "join"        // Multi-table join queries
	ScenarioComplex     ScenarioType = "complex"     // Complex queries combining multiple features
)

// SourceType represents where the search result came from
type SourceType string

// SourceType constants
const (
	SourceUser SourceType = "user" // User-provided DataContent (highest priority)
	SourceHook SourceType = "hook" // Hook ctx.search.*() results
	SourceAuto SourceType = "auto" // Auto search results (lowest priority)
)

// Request represents a search request
type Request struct {
	// Common fields
	Query  string     `json:"query"`           // Search query (natural language)
	Type   SearchType `json:"type"`            // Search type: "web", "kb", or "db"
	Limit  int        `json:"limit,omitempty"` // Max results (default: 10)
	Source SourceType `json:"source"`          // Source of this request (user/hook/auto)

	// Web search specific
	Sites     []string `json:"sites,omitempty"`      // Restrict to specific sites
	TimeRange string   `json:"time_range,omitempty"` // "day", "week", "month", "year"

	// Knowledge base specific
	Collections []string               `json:"collections,omitempty"` // KB collection IDs
	Threshold   float64                `json:"threshold,omitempty"`   // Similarity threshold (0-1)
	Graph       bool                   `json:"graph,omitempty"`       // Enable graph association
	Metadata    map[string]interface{} `json:"metadata,omitempty"`    // Metadata filter for KB search

	// Database search specific
	Models   []string     `json:"models,omitempty"`   // Model IDs (e.g., "user", "agents.mybot.product")
	Scenario ScenarioType `json:"scenario,omitempty"` // QueryDSL scenario: "filter", "aggregation", "join", "complex"
	Wheres   []gou.Where  `json:"wheres,omitempty"`   // Pre-defined filters (optional), uses GOU QueryDSL Where
	Orders   gou.Orders   `json:"orders,omitempty"`   // Sort orders (optional), uses GOU QueryDSL Orders
	Select   []string     `json:"select,omitempty"`   // Fields to return (optional)

	// Reranking
	Rerank *RerankOptions `json:"rerank,omitempty"`
}

// RerankOptions controls result reranking
// Reranker type is determined by uses.rerank in agent/agent.yml
type RerankOptions struct {
	TopN int `json:"top_n,omitempty"` // Return top N after reranking
}

// Result represents the search result
type Result struct {
	Type     SearchType    `json:"type"`            // Search type
	Query    string        `json:"query"`           // Original query
	Source   SourceType    `json:"source"`          // Source of this result
	Items    []*ResultItem `json:"items"`           // Result items
	Total    int           `json:"total"`           // Total matches
	Duration int64         `json:"duration_ms"`     // Search duration in ms
	Error    string        `json:"error,omitempty"` // Error message if failed

	// Graph associations (KB only, if enabled)
	GraphNodes []*GraphNode `json:"graph_nodes,omitempty"`

	// DB specific
	DSL map[string]interface{} `json:"dsl,omitempty"` // Generated QueryDSL (DB only)
}

// ResultItem represents a single search result item
type ResultItem struct {
	// Citation
	CitationID string `json:"citation_id"` // Unique ID for LLM reference: "ref_001"

	// Weighting
	Source SourceType `json:"source"`          // Source type: "user", "hook", "auto"
	Weight float64    `json:"weight"`          // Source weight (from config)
	Score  float64    `json:"score,omitempty"` // Relevance score (0-1)

	// Common fields
	Type    SearchType `json:"type"`            // Search type for this item
	Title   string     `json:"title,omitempty"` // Title/headline
	Content string     `json:"content"`         // Main content/snippet
	URL     string     `json:"url,omitempty"`   // Source URL

	// KB specific
	DocumentID string `json:"document_id,omitempty"` // Source document ID
	Collection string `json:"collection,omitempty"`  // Collection name

	// DB specific
	Model    string                 `json:"model,omitempty"`     // Model ID
	RecordID interface{}            `json:"record_id,omitempty"` // Record primary key
	Data     map[string]interface{} `json:"data,omitempty"`      // Full record data

	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"` // Additional metadata
}

// ProcessedQuery represents a processed query ready for execution
type ProcessedQuery struct {
	Type     SearchType    `json:"type"`
	Keywords []string      `json:"keywords,omitempty"` // For web search
	Vector   []float32     `json:"vector,omitempty"`   // For KB search
	DSL      *gou.QueryDSL `json:"dsl,omitempty"`      // For DB search, uses GOU QueryDSL
}

// Keyword represents an extracted keyword with weight
type Keyword struct {
	K string  `json:"k"` // Keyword text
	W float64 `json:"w"` // Weight (0.1-1.0), higher means more relevant
}

// Note: For QueryDSL and Model types, use GOU types directly:
// - github.com/yaoapp/gou/query/gou.QueryDSL
// - github.com/yaoapp/gou/model.Model
// - github.com/yaoapp/gou/model.Column
