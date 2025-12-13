package interfaces

import (
	"github.com/yaoapp/yao/agent/search/types"
)

// KeywordExtractor extracts keywords for web search
type KeywordExtractor interface {
	// Extract extracts search keywords from user message
	Extract(content string, opts *types.KeywordOptions) ([]string, error)
}

// QueryDSLGenerator generates QueryDSL for DB search
type QueryDSLGenerator interface {
	// Generate converts natural language to QueryDSL
	Generate(query string, schemas []*types.ModelSchema) (*types.QueryDSL, error)
}

// Note: Embedding is handled by KB collection's own config (embedding provider + model),
// not defined here. See KB handler for details.
