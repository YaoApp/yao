package keyword_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent/search/nlp/keyword"
	"github.com/yaoapp/yao/agent/search/types"
)

func TestExtractor_BuiltinMode_RequiresContext(t *testing.T) {
	// Test builtin mode requires context (now uses __yao.keyword agent)
	extractor := keyword.NewExtractor("builtin", &types.KeywordConfig{
		MaxKeywords: 5,
		Language:    "auto",
	})

	// Without context, should return error
	_, err := extractor.Extract(nil, "How to build a search engine with Elasticsearch?", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context is required")
}

func TestExtractor_EmptyUsesKeyword_RequiresContext(t *testing.T) {
	// Empty uses.keyword should default to __yao.keyword agent
	extractor := keyword.NewExtractor("", nil)

	// Without context, should return error
	_, err := extractor.Extract(nil, "Machine learning algorithms", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context is required")
}

func TestExtractor_AgentMode_RequiresContext(t *testing.T) {
	// Custom agent mode requires context
	extractor := keyword.NewExtractor("custom.keyword.agent", nil)

	_, err := extractor.Extract(nil, "Test query", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context is required")
}

func TestExtractor_MCPMode_InvalidFormat(t *testing.T) {
	// Invalid MCP format should fallback to system agent (which requires context)
	extractor := keyword.NewExtractor("mcp:invalid", nil)

	_, err := extractor.Extract(nil, "Test query", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context is required")
}

func TestExtractor_SystemKeywordAgentConstant(t *testing.T) {
	// Verify the system keyword agent constant
	assert.Equal(t, "__yao.keyword", keyword.SystemKeywordAgent)
}
