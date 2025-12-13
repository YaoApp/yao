package keyword_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent/search/nlp/keyword"
	"github.com/yaoapp/yao/agent/search/types"
)

func TestExtractor_BuiltinMode(t *testing.T) {
	// Test builtin mode (no external dependencies)
	extractor := keyword.NewExtractor("builtin", &types.KeywordConfig{
		MaxKeywords: 5,
		Language:    "auto",
	})

	keywords, err := extractor.Extract(nil, "How to build a search engine with Elasticsearch?", nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, keywords)
	assert.LessOrEqual(t, len(keywords), 5)
}

func TestExtractor_EmptyUsesKeyword(t *testing.T) {
	// Empty uses.keyword should default to builtin
	extractor := keyword.NewExtractor("", nil)

	keywords, err := extractor.Extract(nil, "Machine learning algorithms", nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, keywords)
}

func TestExtractor_RuntimeOptionsOverride(t *testing.T) {
	// Config has max_keywords=10, but runtime opts override to 3
	extractor := keyword.NewExtractor("builtin", &types.KeywordConfig{
		MaxKeywords: 10,
	})

	keywords, err := extractor.Extract(nil, "one two three four five six seven eight nine ten", &types.KeywordOptions{
		MaxKeywords: 3,
	})
	assert.NoError(t, err)
	assert.LessOrEqual(t, len(keywords), 3)
}

func TestExtractor_ConfigDefaults(t *testing.T) {
	// No config, should use defaults
	extractor := keyword.NewExtractor("builtin", nil)

	keywords, err := extractor.Extract(nil, "Test query for keyword extraction", nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, keywords)
	assert.LessOrEqual(t, len(keywords), 10) // default max_keywords is 10
}

func TestExtractor_InvalidMCPFormat(t *testing.T) {
	// Invalid MCP format should fallback to builtin
	extractor := keyword.NewExtractor("mcp:invalid", nil)

	keywords, err := extractor.Extract(nil, "Test query", nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, keywords)
}
