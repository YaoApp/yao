//go:build integration

package web_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/search/handlers/web"
	"github.com/yaoapp/yao/agent/search/types"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestNewHandler(t *testing.T) {
	testprepare.PrepareSandbox(t)

	t.Run("builtin mode", func(t *testing.T) {
		h := web.NewHandler("builtin", nil)
		require.NotNil(t, h)
		assert.Equal(t, types.SearchTypeWeb, h.Type())
	})

	t.Run("builtin mode with empty string", func(t *testing.T) {
		h := web.NewHandler("", nil)
		require.NotNil(t, h)
		assert.Equal(t, types.SearchTypeWeb, h.Type())
	})

	t.Run("agent mode", func(t *testing.T) {
		h := web.NewHandler("tests.web-agent", nil)
		require.NotNil(t, h)
		assert.Equal(t, types.SearchTypeWeb, h.Type())
	})

	t.Run("mcp mode", func(t *testing.T) {
		h := web.NewHandler("mcp:search.web_search", nil)
		require.NotNil(t, h)
		assert.Equal(t, types.SearchTypeWeb, h.Type())
	})

	t.Run("with web config", func(t *testing.T) {
		cfg := &types.WebConfig{
			Provider:   "serper",
			MaxResults: 5,
		}
		h := web.NewHandler("builtin", cfg)
		require.NotNil(t, h)
	})
}

func TestHandler_Search_Builtin(t *testing.T) {
	testprepare.PrepareSandbox(t)

	h := web.NewHandler("builtin", nil)
	req := &types.Request{
		Type:   types.SearchTypeWeb,
		Query:  "test query",
		Source: types.SourceAuto,
		Limit:  3,
	}

	result, err := h.Search(req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, types.SearchTypeWeb, result.Type)
	assert.Equal(t, "test query", result.Query)
	assert.Equal(t, types.SourceAuto, result.Source)
}

func TestHandler_Search_AgentWithoutContext(t *testing.T) {
	testprepare.PrepareSandbox(t)

	h := web.NewHandler("tests.web-agent", nil)
	req := &types.Request{
		Type:   types.SearchTypeWeb,
		Query:  "test",
		Source: types.SourceAuto,
	}

	result, err := h.Search(req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.Error)
	assert.Contains(t, result.Error, "requires context")
}

func TestHandler_Search_MCPInvalidFormat(t *testing.T) {
	testprepare.PrepareSandbox(t)

	h := web.NewHandler("mcp:invalid", nil)
	req := &types.Request{
		Type:   types.SearchTypeWeb,
		Query:  "test",
		Source: types.SourceAuto,
	}

	result, err := h.Search(req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.Error)
	assert.Contains(t, result.Error, "Invalid MCP format")
}

func TestNewMCPProvider_InvalidFormat(t *testing.T) {
	testprepare.PrepareSandbox(t)

	_, err := web.NewMCPProvider("invalid")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid MCP format")

	_, err = web.NewMCPProvider("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid MCP format")
}

func TestNewMCPProvider_ValidFormat(t *testing.T) {
	testprepare.PrepareSandbox(t)

	p, err := web.NewMCPProvider("search.web_search")
	require.NoError(t, err)
	require.NotNil(t, p)
}

func TestNewAgentProvider(t *testing.T) {
	testprepare.PrepareSandbox(t)

	p := web.NewAgentProvider("tests.web-agent")
	require.NotNil(t, p)
}

func TestAgentProvider_SearchWithoutContext(t *testing.T) {
	testprepare.PrepareSandbox(t)

	p := web.NewAgentProvider("tests.web-agent")
	req := &types.Request{
		Type:   types.SearchTypeWeb,
		Query:  "test query",
		Source: types.SourceAuto,
	}

	result, err := p.Search(nil, req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.Error)
	assert.Contains(t, result.Error, "requires context")
}
