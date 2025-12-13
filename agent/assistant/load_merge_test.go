package assistant_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/testutils"
)

// TestLoadPathMerge tests loading the merge test assistant
// This verifies that global config is properly merged with assistant-specific config
func TestLoadPathMerge(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	ast, err := assistant.LoadPath("/assistants/tests/merge")
	require.NoError(t, err)
	require.NotNil(t, ast)

	assert.Equal(t, "tests.merge", ast.ID)
	assert.Equal(t, "Merge Config Test Assistant", ast.Name)

	// Uses configuration - should merge global with assistant-specific
	// Global (from agent/agent.yml):
	//   vision: "workers.system.vision"
	//   search: "workers.system.search"
	//   fetch: "workers.system.fetch"
	//   audio: (not set)
	//   querydsl: (not set)
	//   rerank: (not set)
	// Assistant:
	//   web: "mcp:custom-web"
	//   keyword: "mcp:custom-keyword"
	// Result: assistant values override, global values inherited
	assert.NotNil(t, ast.Uses)

	// Assistant overrides
	assert.Equal(t, "mcp:custom-web", ast.Uses.Web)         // overridden by assistant
	assert.Equal(t, "mcp:custom-keyword", ast.Uses.Keyword) // overridden by assistant

	// Inherited from global (agent/agent.yml)
	assert.Equal(t, "workers.system.vision", ast.Uses.Vision) // inherited from global
	assert.Equal(t, "workers.system.search", ast.Uses.Search) // inherited from global
	assert.Equal(t, "workers.system.fetch", ast.Uses.Fetch)   // inherited from global

	// Not set in either global or assistant (should be empty)
	assert.Empty(t, ast.Uses.Audio)    // not set anywhere
	assert.Empty(t, ast.Uses.QueryDSL) // not set anywhere
	assert.Empty(t, ast.Uses.Rerank)   // not set anywhere

	// Search configuration - should merge global with assistant-specific
	// Global (from agent/search.yml):
	//   web.provider=tavily, web.max_results=10
	//   kb.threshold=0.7, kb.graph=false
	//   db.max_results=20
	//   keyword.max_keywords=10, keyword.language=auto
	//   rerank.top_n=10
	//   citation.format=#ref:{id}, citation.auto_inject_prompt=true
	//   weights: user=1.0, hook=0.8, auto=0.6
	//   options.skip_threshold=5
	// Assistant:
	//   web.provider=custom-provider, web.max_results=25
	//   kb.collections=[merge-test-kb], kb.threshold=0.85
	assert.NotNil(t, ast.Search)

	// Web config - assistant overrides global
	assert.NotNil(t, ast.Search.Web)
	assert.Equal(t, "custom-provider", ast.Search.Web.Provider) // overridden
	assert.Equal(t, 25, ast.Search.Web.MaxResults)              // overridden

	// KB config - assistant overrides global
	assert.NotNil(t, ast.Search.KB)
	assert.Equal(t, []string{"merge-test-kb"}, ast.Search.KB.Collections) // overridden
	assert.Equal(t, 0.85, ast.Search.KB.Threshold)                        // overridden
	assert.False(t, ast.Search.KB.Graph)                                  // inherited from global

	// DB config - should inherit from global (assistant doesn't define it)
	assert.NotNil(t, ast.Search.DB)
	assert.Equal(t, 20, ast.Search.DB.MaxResults) // inherited from global

	// Keyword config - should inherit from global
	assert.NotNil(t, ast.Search.Keyword)
	assert.Equal(t, 10, ast.Search.Keyword.MaxKeywords)  // inherited from global
	assert.Equal(t, "auto", ast.Search.Keyword.Language) // inherited from global

	// Rerank config - should inherit from global
	assert.NotNil(t, ast.Search.Rerank)
	assert.Equal(t, 10, ast.Search.Rerank.TopN) // inherited from global

	// Citation config - should inherit from global
	assert.NotNil(t, ast.Search.Citation)
	assert.Equal(t, "#ref:{id}", ast.Search.Citation.Format) // inherited from global
	assert.True(t, ast.Search.Citation.AutoInjectPrompt)     // inherited from global

	// Weights config - should inherit from global
	assert.NotNil(t, ast.Search.Weights)
	assert.Equal(t, 1.0, ast.Search.Weights.User) // inherited from global
	assert.Equal(t, 0.8, ast.Search.Weights.Hook) // inherited from global
	assert.Equal(t, 0.6, ast.Search.Weights.Auto) // inherited from global

	// Options config - should inherit from global
	assert.NotNil(t, ast.Search.Options)
	assert.Equal(t, 5, ast.Search.Options.SkipThreshold) // inherited from global
}

// TestLoadPathMergeOverride tests loading the merge-override test assistant
// This verifies that assistant config completely overrides global config
func TestLoadPathMergeOverride(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	ast, err := assistant.LoadPath("/assistants/tests/merge-override")
	require.NoError(t, err)
	require.NotNil(t, ast)

	assert.Equal(t, "tests.merge-override", ast.ID)
	assert.Equal(t, "Merge Override Test Assistant", ast.Name)

	// Uses configuration - all fields should be overridden by assistant
	assert.NotNil(t, ast.Uses)
	assert.Equal(t, "mcp:custom-vision", ast.Uses.Vision)
	assert.Equal(t, "mcp:custom-audio", ast.Uses.Audio)
	assert.Equal(t, "mcp:custom-search", ast.Uses.Search)
	assert.Equal(t, "mcp:custom-fetch", ast.Uses.Fetch)
	assert.Equal(t, "mcp:custom-web", ast.Uses.Web)
	assert.Equal(t, "mcp:custom-keyword", ast.Uses.Keyword)
	assert.Equal(t, "mcp:custom-querydsl", ast.Uses.QueryDSL)
	assert.Equal(t, "mcp:custom-rerank", ast.Uses.Rerank)

	// Search configuration - all fields should be overridden by assistant
	assert.NotNil(t, ast.Search)

	// Web config - all overridden
	assert.NotNil(t, ast.Search.Web)
	assert.Equal(t, "override-provider", ast.Search.Web.Provider)
	assert.Equal(t, "$ENV.OVERRIDE_API_KEY", ast.Search.Web.APIKeyEnv)
	assert.Equal(t, 100, ast.Search.Web.MaxResults)

	// KB config - all overridden
	assert.NotNil(t, ast.Search.KB)
	assert.Equal(t, []string{"override-kb"}, ast.Search.KB.Collections)
	assert.Equal(t, 0.99, ast.Search.KB.Threshold)
	assert.True(t, ast.Search.KB.Graph)

	// DB config - all overridden
	assert.NotNil(t, ast.Search.DB)
	assert.Equal(t, []string{"override-model"}, ast.Search.DB.Models)
	assert.Equal(t, 200, ast.Search.DB.MaxResults)

	// Keyword config - all overridden
	assert.NotNil(t, ast.Search.Keyword)
	assert.Equal(t, 20, ast.Search.Keyword.MaxKeywords)
	assert.Equal(t, "zh", ast.Search.Keyword.Language)

	// QueryDSL config - overridden
	assert.NotNil(t, ast.Search.QueryDSL)
	assert.True(t, ast.Search.QueryDSL.Strict)

	// Rerank config - overridden
	assert.NotNil(t, ast.Search.Rerank)
	assert.Equal(t, 20, ast.Search.Rerank.TopN)

	// Citation config - all overridden
	assert.NotNil(t, ast.Search.Citation)
	assert.Equal(t, "[override:{id}]", ast.Search.Citation.Format)
	assert.False(t, ast.Search.Citation.AutoInjectPrompt)
	assert.Equal(t, "Override citation prompt", ast.Search.Citation.CustomPrompt)

	// Weights config - all overridden
	assert.NotNil(t, ast.Search.Weights)
	assert.Equal(t, 2.0, ast.Search.Weights.User)
	assert.Equal(t, 1.5, ast.Search.Weights.Hook)
	assert.Equal(t, 1.0, ast.Search.Weights.Auto)

	// Options config - overridden
	assert.NotNil(t, ast.Search.Options)
	assert.Equal(t, 10, ast.Search.Options.SkipThreshold)
}

// TestLoadPathMergeEmpty tests loading the merge-empty test assistant
// This verifies that assistant with no uses/search config inherits all from global
func TestLoadPathMergeEmpty(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	ast, err := assistant.LoadPath("/assistants/tests/merge-empty")
	require.NoError(t, err)
	require.NotNil(t, ast)

	assert.Equal(t, "tests.merge-empty", ast.ID)
	assert.Equal(t, "Merge Empty Test Assistant", ast.Name)

	// Uses configuration - all inherited from global (agent/agent.yml)
	assert.NotNil(t, ast.Uses)
	assert.Equal(t, "workers.system.vision", ast.Uses.Vision) // from global
	assert.Equal(t, "workers.system.search", ast.Uses.Search) // from global
	assert.Equal(t, "workers.system.fetch", ast.Uses.Fetch)   // from global
	assert.Empty(t, ast.Uses.Audio)                           // not set in global
	assert.Empty(t, ast.Uses.Web)                             // not set in global
	assert.Empty(t, ast.Uses.Keyword)                         // not set in global
	assert.Empty(t, ast.Uses.QueryDSL)                        // not set in global
	assert.Empty(t, ast.Uses.Rerank)                          // not set in global

	// Search configuration - all inherited from global (agent/search.yml)
	assert.NotNil(t, ast.Search)

	// Web config - from global
	assert.NotNil(t, ast.Search.Web)
	assert.Equal(t, "tavily", ast.Search.Web.Provider)
	assert.Equal(t, 10, ast.Search.Web.MaxResults)

	// KB config - from global
	assert.NotNil(t, ast.Search.KB)
	assert.Equal(t, 0.7, ast.Search.KB.Threshold)
	assert.False(t, ast.Search.KB.Graph)

	// DB config - from global
	assert.NotNil(t, ast.Search.DB)
	assert.Equal(t, 20, ast.Search.DB.MaxResults)

	// Keyword config - from global
	assert.NotNil(t, ast.Search.Keyword)
	assert.Equal(t, 10, ast.Search.Keyword.MaxKeywords)
	assert.Equal(t, "auto", ast.Search.Keyword.Language)

	// Rerank config - from global
	assert.NotNil(t, ast.Search.Rerank)
	assert.Equal(t, 10, ast.Search.Rerank.TopN)

	// Citation config - from global
	assert.NotNil(t, ast.Search.Citation)
	assert.Equal(t, "#ref:{id}", ast.Search.Citation.Format)
	assert.True(t, ast.Search.Citation.AutoInjectPrompt)

	// Weights config - from global
	assert.NotNil(t, ast.Search.Weights)
	assert.Equal(t, 1.0, ast.Search.Weights.User)
	assert.Equal(t, 0.8, ast.Search.Weights.Hook)
	assert.Equal(t, 0.6, ast.Search.Weights.Auto)

	// Options config - from global
	assert.NotNil(t, ast.Search.Options)
	assert.Equal(t, 5, ast.Search.Options.SkipThreshold)
}

// TestLoadPathUsesAndSearchMerge tests loading fullfields assistant
// This verifies that uses and search configs are properly loaded and merged
func TestLoadPathUsesAndSearchMerge(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	ast, err := assistant.LoadPath("/assistants/tests/fullfields")
	require.NoError(t, err)
	require.NotNil(t, ast)

	// Uses configuration - assistant-specific values
	assert.NotNil(t, ast.Uses)
	assert.Equal(t, "agent", ast.Uses.Vision)
	assert.Equal(t, "mcp:audio-server", ast.Uses.Audio)
	assert.Equal(t, "agent", ast.Uses.Fetch)
	assert.Equal(t, "builtin", ast.Uses.Web)
	assert.Equal(t, "builtin", ast.Uses.Keyword)
	assert.Equal(t, "builtin", ast.Uses.QueryDSL)
	assert.Equal(t, "builtin", ast.Uses.Rerank)

	// Search configuration - assistant-specific values
	assert.NotNil(t, ast.Search)

	// Web config - from assistant
	assert.NotNil(t, ast.Search.Web)
	assert.Equal(t, "tavily", ast.Search.Web.Provider)
	assert.Equal(t, 15, ast.Search.Web.MaxResults)

	// KB config - from assistant
	assert.NotNil(t, ast.Search.KB)
	assert.Equal(t, []string{"docs", "faq"}, ast.Search.KB.Collections)
	assert.Equal(t, 0.8, ast.Search.KB.Threshold)
	assert.True(t, ast.Search.KB.Graph)

	// DB config - from assistant
	assert.NotNil(t, ast.Search.DB)
	assert.Equal(t, []string{"user", "product"}, ast.Search.DB.Models)
	assert.Equal(t, 50, ast.Search.DB.MaxResults)

	// Citation config - from assistant
	assert.NotNil(t, ast.Search.Citation)
	assert.Equal(t, "#ref:{id}", ast.Search.Citation.Format)
	assert.True(t, ast.Search.Citation.AutoInjectPrompt)

	// Weights config - from assistant
	assert.NotNil(t, ast.Search.Weights)
	assert.Equal(t, 1.0, ast.Search.Weights.User)
	assert.Equal(t, 0.9, ast.Search.Weights.Hook)
	assert.Equal(t, 0.7, ast.Search.Weights.Auto)
}

// TestLoadPathSearchAssistant tests loading the dedicated search test assistant
func TestLoadPathSearchAssistant(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	ast, err := assistant.LoadPath("/assistants/tests/search")
	require.NoError(t, err)
	require.NotNil(t, ast)

	assert.Equal(t, "tests.search", ast.ID)
	assert.Equal(t, "Search Config Test Assistant", ast.Name)

	// Uses configuration
	assert.NotNil(t, ast.Uses)
	assert.Equal(t, "builtin", ast.Uses.Web)
	assert.Equal(t, "builtin", ast.Uses.Keyword)
	assert.Equal(t, "builtin", ast.Uses.QueryDSL)
	assert.Equal(t, "builtin", ast.Uses.Rerank)

	// Search configuration
	assert.NotNil(t, ast.Search)

	// Web config
	assert.NotNil(t, ast.Search.Web)
	assert.Equal(t, "serper", ast.Search.Web.Provider)
	assert.Equal(t, "$ENV.SERPER_API_KEY", ast.Search.Web.APIKeyEnv)
	assert.Equal(t, 20, ast.Search.Web.MaxResults)

	// KB config
	assert.NotNil(t, ast.Search.KB)
	assert.Equal(t, []string{"knowledge-base", "documents"}, ast.Search.KB.Collections)
	assert.Equal(t, 0.75, ast.Search.KB.Threshold)
	assert.False(t, ast.Search.KB.Graph)

	// DB config
	assert.NotNil(t, ast.Search.DB)
	assert.Equal(t, []string{"article", "comment"}, ast.Search.DB.Models)
	assert.Equal(t, 30, ast.Search.DB.MaxResults)

	// Keyword config
	assert.NotNil(t, ast.Search.Keyword)
	assert.Equal(t, 8, ast.Search.Keyword.MaxKeywords)
	assert.Equal(t, "auto", ast.Search.Keyword.Language)

	// QueryDSL config
	assert.NotNil(t, ast.Search.QueryDSL)
	assert.True(t, ast.Search.QueryDSL.Strict)

	// Rerank config
	assert.NotNil(t, ast.Search.Rerank)
	assert.Equal(t, 5, ast.Search.Rerank.TopN)

	// Citation config
	assert.NotNil(t, ast.Search.Citation)
	assert.Equal(t, "#cite:{id}", ast.Search.Citation.Format)
	assert.False(t, ast.Search.Citation.AutoInjectPrompt)
	assert.Equal(t, "Please cite sources using #cite:{id} format.", ast.Search.Citation.CustomPrompt)

	// Weights config
	assert.NotNil(t, ast.Search.Weights)
	assert.Equal(t, 1.0, ast.Search.Weights.User)
	assert.Equal(t, 0.85, ast.Search.Weights.Hook)
	assert.Equal(t, 0.65, ast.Search.Weights.Auto)

	// Options config
	assert.NotNil(t, ast.Search.Options)
	assert.Equal(t, 3, ast.Search.Options.SkipThreshold)
}
