//go:build integration

package assistant_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/assistant"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestParseSearchField(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		result := assistant.ExportParseSearchField(nil)
		assert.Nil(t, result)
	})

	t.Run("BoolTrue", func(t *testing.T) {
		result := assistant.ExportParseSearchField(true)
		require.NotNil(t, result)
		assert.True(t, result.NeedSearch)
		assert.Contains(t, result.SearchTypes, "web")
	})

	t.Run("BoolFalse", func(t *testing.T) {
		result := assistant.ExportParseSearchField(false)
		require.NotNil(t, result)
		assert.False(t, result.NeedSearch)
	})

	t.Run("MapWithNeedSearchAndTypes", func(t *testing.T) {
		input := map[string]any{
			"need_search":  true,
			"search_types": []any{"web", "kb"},
			"confidence":   0.9,
			"reason":       "user asked about current events",
		}
		result := assistant.ExportParseSearchField(input)
		require.NotNil(t, result)
		assert.True(t, result.NeedSearch)
		assert.Equal(t, []string{"web", "kb"}, result.SearchTypes)
		assert.InDelta(t, 0.9, result.Confidence, 0.01)
		assert.Equal(t, "user asked about current events", result.Reason)
	})

	t.Run("SearchIntentStruct", func(t *testing.T) {
		intent := &agentContext.SearchIntent{
			NeedSearch:  true,
			SearchTypes: []string{"web"},
			Confidence:  0.8,
			Reason:      "direct struct",
		}
		result := assistant.ExportParseSearchField(intent)
		require.NotNil(t, result)
		assert.True(t, result.NeedSearch)
		assert.Equal(t, []string{"web"}, result.SearchTypes)
		assert.InDelta(t, 0.8, result.Confidence, 0.01)
	})
}

func TestShouldAutoSearchDisabled(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ast, err := assistant.Get("tests.search-disabled")
	require.NoError(t, err)

	ctx := newTestContext("test-search-disabled", "tests.search-disabled")
	messages := []agentContext.Message{
		{Role: agentContext.RoleUser, Content: "What is the weather today?"},
	}

	// Search is disabled via uses.search = "disabled"
	result := assistant.ExportShouldAutoSearch(ast, ctx, messages, nil, nil)
	assert.Nil(t, result, "should return nil when search is disabled")
}

func TestShouldAutoSearchSkipByOpts(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ast, err := assistant.Get("tests.search-web")
	require.NoError(t, err)

	ctx := newTestContext("test-search-skip-opts", "tests.search-web")
	messages := []agentContext.Message{
		{Role: agentContext.RoleUser, Content: "Search something"},
	}

	// Skip search via opts
	opts := &agentContext.Options{
		Skip: &agentContext.Skip{Search: true},
	}

	result := assistant.ExportShouldAutoSearch(ast, ctx, messages, nil, opts)
	assert.Nil(t, result, "should return nil when opts.Skip.Search is true")
}

func TestShouldAutoSearchHookControl(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ast, err := assistant.Get("tests.search-hook")
	require.NoError(t, err)

	ctx := newTestContext("test-search-hook-control", "tests.search-hook")
	messages := []agentContext.Message{
		{Role: agentContext.RoleUser, Content: "Test search hook"},
	}

	t.Run("HookDisablesSearch", func(t *testing.T) {
		// Simulate createResponse.Search = false (hook disables search)
		createResponse := &agentContext.HookCreateResponse{
			Search: false,
		}

		result := assistant.ExportShouldAutoSearch(ast, ctx, messages, createResponse, nil)
		assert.Nil(t, result, "should return nil when hook sets search=false")
	})

	t.Run("HookEnablesWebSearch", func(t *testing.T) {
		// Simulate createResponse.Search = {need_search: true, search_types: ["web"]}
		createResponse := &agentContext.HookCreateResponse{
			Search: map[string]any{
				"need_search":  true,
				"search_types": []any{"web"},
				"confidence":   0.95,
				"reason":       "hook determined search needed",
			},
		}

		result := assistant.ExportShouldAutoSearch(ast, ctx, messages, createResponse, nil)
		require.NotNil(t, result, "should return SearchIntent when hook enables search")
		assert.True(t, result.NeedSearch)
		assert.Contains(t, result.SearchTypes, "web")
	})
}

func TestShouldAutoSearchNoConfig(t *testing.T) {
	testprepare.PrepareSandbox(t)

	// tests.no-prompt has no search configuration
	ast, err := assistant.Get("tests.no-prompt")
	require.NoError(t, err)

	ctx := newTestContext("test-search-no-config", "tests.no-prompt")
	messages := []agentContext.Message{
		{Role: agentContext.RoleUser, Content: "Hello"},
	}

	result := assistant.ExportShouldAutoSearch(ast, ctx, messages, nil, nil)
	assert.Nil(t, result, "should return nil when assistant has no search configuration")
}

func TestGetMergedSearchUses(t *testing.T) {
	testprepare.PrepareSandbox(t)

	t.Run("AssistantUsesOnly", func(t *testing.T) {
		ast, err := assistant.Get("tests.search-web")
		require.NoError(t, err)

		result := assistant.ExportGetMergedSearchUses(ast, nil)
		require.NotNil(t, result)
		assert.Equal(t, "builtin", result.Search)
		assert.Equal(t, "builtin", result.Web)
	})

	t.Run("OptsOverride", func(t *testing.T) {
		ast, err := assistant.Get("tests.search-web")
		require.NoError(t, err)

		opts := &agentContext.Options{
			Uses: &agentContext.Uses{
				Search: "mcp:custom-search",
				Web:    "mcp:custom-web",
			},
		}

		result := assistant.ExportGetMergedSearchUses(ast, nil, opts)
		require.NotNil(t, result)
		assert.Equal(t, "mcp:custom-search", result.Search)
		assert.Equal(t, "mcp:custom-web", result.Web)
	})

	t.Run("CreateResponseOverride", func(t *testing.T) {
		ast, err := assistant.Get("tests.search-web")
		require.NoError(t, err)

		createResponse := &agentContext.HookCreateResponse{
			Uses: &agentContext.Uses{
				Search: "agent-search",
				Web:    "agent-web",
			},
		}

		opts := &agentContext.Options{
			Uses: &agentContext.Uses{
				Search: "opts-search",
				Web:    "opts-web",
			},
		}

		// createResponse has highest priority
		result := assistant.ExportGetMergedSearchUses(ast, createResponse, opts)
		require.NotNil(t, result)
		assert.Equal(t, "agent-search", result.Search)
		assert.Equal(t, "agent-web", result.Web)
	})
}
