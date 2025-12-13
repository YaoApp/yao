package interfaces

import (
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/query/gou"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/search/types"
)

// KeywordExtractor extracts keywords for web search
type KeywordExtractor interface {
	// Extract extracts search keywords from user message
	// ctx is required for Agent and MCP modes, can be nil for builtin mode
	Extract(ctx *context.Context, content string, opts *types.KeywordOptions) ([]string, error)
}

// QueryDSLGenerator generates QueryDSL for DB search
type QueryDSLGenerator interface {
	// Generate converts natural language to QueryDSL
	Generate(query string, models []*model.Model) (*gou.QueryDSL, error)
}

// Note: Embedding is handled by KB collection's own config (embedding provider + model),
// not defined here. See KB handler for details.
