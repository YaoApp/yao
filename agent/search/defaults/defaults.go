package defaults

import "github.com/yaoapp/yao/agent/search/types"

// SystemDefaults provides hardcoded default values
// Used by agent/load.go for merging with agent/search.yao
var SystemDefaults = &types.Config{
	// Web search defaults
	Web: &types.WebConfig{
		Provider:   "tavily",
		MaxResults: 10,
	},

	// KB search defaults
	KB: &types.KBConfig{
		Threshold: 0.7,
		Graph:     false,
	},

	// DB search defaults
	DB: &types.DBConfig{
		MaxResults: 20,
	},

	// Keyword extraction options (uses.keyword)
	Keyword: &types.KeywordConfig{
		MaxKeywords: 10,
		Language:    "auto",
	},

	// QueryDSL generation options (uses.querydsl)
	QueryDSL: &types.QueryDSLConfig{
		Strict: false,
	},

	// Rerank options (uses.rerank)
	Rerank: &types.RerankConfig{
		TopN: 10,
	},

	// Citation
	Citation: &types.CitationConfig{
		Format:           "#ref:{id}",
		AutoInjectPrompt: true,
	},

	// Source weights
	Weights: &types.WeightsConfig{
		User: 1.0,
		Hook: 0.8,
		Auto: 0.6,
	},

	// Behavior options
	Options: &types.OptionsConfig{
		SkipThreshold: 5,
	},
}

// GetWeight returns the weight for a source type using default config
func GetWeight(source types.SourceType) float64 {
	return SystemDefaults.GetWeight(source)
}
