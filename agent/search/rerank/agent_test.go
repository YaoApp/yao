package rerank_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/search/rerank"
	"github.com/yaoapp/yao/agent/search/types"
	"github.com/yaoapp/yao/agent/testutils"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
)

func TestAgentProviderWithAssistantConfig(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Load the rerank-agent assistant
	ast, err := assistant.Get("tests.rerank-agent")
	require.NoError(t, err)
	require.NotNil(t, ast)

	// Create test context
	ctx := newTestContext(t)

	// Create provider with test assistant
	provider := rerank.NewAgentProvider("tests.rerank-agent")

	items := []*types.ResultItem{
		{CitationID: "ref_001", Score: 0.9, Weight: 1.0, Title: "First"},
		{CitationID: "ref_002", Score: 0.8, Weight: 1.0, Title: "Second"},
		{CitationID: "ref_003", Score: 0.7, Weight: 1.0, Title: "Third"},
	}

	result, err := provider.Rerank(ctx, "test query", items, &types.RerankOptions{TopN: 10})

	require.NoError(t, err)
	assert.NotEmpty(t, result)

	// The mock agent reverses the order
	// So we expect: ref_003, ref_002, ref_001
	assert.Len(t, result, 3)
	assert.Equal(t, "ref_003", result[0].CitationID)
	assert.Equal(t, "ref_002", result[1].CitationID)
	assert.Equal(t, "ref_001", result[2].CitationID)
}

func TestAgentProviderWithTopN(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := newTestContext(t)
	provider := rerank.NewAgentProvider("tests.rerank-agent")

	items := []*types.ResultItem{
		{CitationID: "ref_001", Score: 0.9, Weight: 1.0},
		{CitationID: "ref_002", Score: 0.8, Weight: 1.0},
		{CitationID: "ref_003", Score: 0.7, Weight: 1.0},
		{CitationID: "ref_004", Score: 0.6, Weight: 1.0},
		{CitationID: "ref_005", Score: 0.5, Weight: 1.0},
	}

	// Request top 2 only
	result, err := provider.Rerank(ctx, "test query", items, &types.RerankOptions{TopN: 2})

	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestAgentProviderWithoutContext(t *testing.T) {
	provider := rerank.NewAgentProvider("tests.rerank-agent")

	items := []*types.ResultItem{
		{CitationID: "ref_001", Score: 0.9, Weight: 1.0},
	}

	_, err := provider.Rerank(nil, "test query", items, &types.RerankOptions{TopN: 10})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context is required")
}

func TestAgentProviderAgentNotFound(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := newTestContext(t)
	provider := rerank.NewAgentProvider("non-existent-agent")

	items := []*types.ResultItem{
		{CitationID: "ref_001", Score: 0.9, Weight: 1.0},
	}

	_, err := provider.Rerank(ctx, "test query", items, &types.RerankOptions{TopN: 10})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get agent")
}

func TestAgentProviderEmptyItems(t *testing.T) {
	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := newTestContext(t)
	provider := rerank.NewAgentProvider("tests.rerank-agent")

	result, err := provider.Rerank(ctx, "test query", []*types.ResultItem{}, &types.RerankOptions{TopN: 10})

	require.NoError(t, err)
	assert.Empty(t, result)
}

// newTestContext creates a test context with required fields
func newTestContext(t *testing.T) *context.Context {
	t.Helper()
	authorized := &oauthTypes.AuthorizedInfo{
		UserID: "test-user",
	}
	chatID := "test-chat-rerank"
	return context.New(t.Context(), authorized, chatID)
}
