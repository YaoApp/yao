//go:build integration

package assistant_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

// --- Search Auto Web ---

func TestSearchAutoWeb(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ast, err := assistant.LoadPath("/assistants/tests/search-web")
	require.NoError(t, err)
	require.NotNil(t, ast)

	t.Run("ShouldHaveSearchConfig", func(t *testing.T) {
		assert.NotNil(t, ast.Search, "search config should be set")
		assert.NotNil(t, ast.Search.Web, "web search config should be set")
		assert.Equal(t, "serper", ast.Search.Web.Provider)
		assert.Equal(t, 3, ast.Search.Web.MaxResults)
	})

	t.Run("ShouldHaveUsesConfig", func(t *testing.T) {
		assert.NotNil(t, ast.Uses, "uses config should be set")
		assert.Equal(t, "builtin", ast.Uses.Search)
		assert.Equal(t, "builtin", ast.Uses.Web)
	})

	t.Run("ShouldHaveCitationConfig", func(t *testing.T) {
		assert.NotNil(t, ast.Search.Citation, "citation config should be set")
		assert.Equal(t, "xml", ast.Search.Citation.Format)
		assert.True(t, ast.Search.Citation.AutoInjectPrompt)
	})
}

// --- Search Auto Keyword Not Configured ---

func TestSearchAutoKeywordNotConfigured(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ast, err := assistant.LoadPath("/assistants/tests/search-web")
	require.NoError(t, err)
	require.NotNil(t, ast)

	t.Run("ShouldNotHaveKeywordInUses", func(t *testing.T) {
		if ast.Uses != nil {
			assert.Empty(t, ast.Uses.Keyword, "uses.keyword should be empty")
		}
	})
}

// --- Search Auto Disabled ---

func TestSearchAutoDisabled(t *testing.T) {
	testprepare.PrepareSandbox(t)

	ast, err := assistant.LoadPath("/assistants/tests/search-disabled")
	require.NoError(t, err)
	require.NotNil(t, ast)

	t.Run("ShouldHaveSearchConfig", func(t *testing.T) {
		assert.NotNil(t, ast.Search, "search config should be set")
		assert.NotNil(t, ast.Search.Web, "web search config should be set")
	})

	t.Run("ShouldHaveDisabledUses", func(t *testing.T) {
		assert.NotNil(t, ast.Uses, "uses config should be set")
		assert.Equal(t, "disabled", ast.Uses.Search, "uses.search should be disabled")
	})
}
