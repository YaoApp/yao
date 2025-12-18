package search_test

import (
	"testing"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/search"
	"github.com/yaoapp/yao/agent/search/types"
	"github.com/yaoapp/yao/agent/testutils"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// DB Search JSAPI Integration Tests
// ============================================================================

func TestJSAPI_DB_Integration(t *testing.T) {
	// Skip if running short tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Initialize test environment (loads models, database, query engine, etc.)
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Create test context
	ctx := newJSAPITestContext(t)

	// Verify __yao.role model is loaded
	mod := model.Select("__yao.role")
	require.NotNil(t, mod, "__yao.role model should be loaded")

	// Ensure test data exists
	ensureJSAPITestRole(t, mod)

	t.Run("db_search_with_context", func(t *testing.T) {
		api := search.NewJSAPI(ctx, &types.Config{
			DB: &types.DBConfig{
				Models:     []string{"__yao.role"},
				MaxResults: 10,
			},
		}, &search.Uses{QueryDSL: "builtin"})

		result := api.DB("查询所有角色", map[string]interface{}{
			"models": []interface{}{"__yao.role"},
			"limit":  float64(10),
		})

		require.NotNil(t, result)
		r, ok := result.(*types.Result)
		require.True(t, ok)

		assert.Equal(t, types.SearchTypeDB, r.Type)
		assert.Equal(t, "查询所有角色", r.Query)
		assert.Equal(t, types.SourceHook, r.Source)

		if r.Error != "" {
			t.Logf("Search error: %s", r.Error)
		}
		assert.Empty(t, r.Error, "Should not have error")
		assert.Greater(t, len(r.Items), 0, "Should have results")
	})

	t.Run("db_search_with_scenario", func(t *testing.T) {
		api := search.NewJSAPI(ctx, &types.Config{
			DB: &types.DBConfig{
				Models:     []string{"__yao.role"},
				MaxResults: 5,
			},
		}, &search.Uses{QueryDSL: "builtin"})

		result := api.DB("查询系统角色", map[string]interface{}{
			"models":   []interface{}{"__yao.role"},
			"scenario": "filter",
			"limit":    float64(5),
		})

		require.NotNil(t, result)
		r, ok := result.(*types.Result)
		require.True(t, ok)

		assert.Equal(t, types.SearchTypeDB, r.Type)
		assert.LessOrEqual(t, len(r.Items), 5, "Should respect limit")
	})

	t.Run("db_search_with_select_fields", func(t *testing.T) {
		api := search.NewJSAPI(ctx, &types.Config{
			DB: &types.DBConfig{
				Models:     []string{"__yao.role"},
				MaxResults: 10,
			},
		}, &search.Uses{QueryDSL: "builtin"})

		result := api.DB("查询角色名称", map[string]interface{}{
			"models": []interface{}{"__yao.role"},
			"select": []interface{}{"id", "name", "description"},
			"limit":  float64(10),
		})

		require.NotNil(t, result)
		r, ok := result.(*types.Result)
		require.True(t, ok)

		assert.Equal(t, types.SearchTypeDB, r.Type)
		if r.Error == "" && len(r.Items) > 0 {
			// Verify items have data
			for _, item := range r.Items {
				assert.NotNil(t, item.Data)
				assert.Equal(t, "__yao.role", item.Model)
			}
		}
	})

	t.Run("db_search_all_with_multiple_types", func(t *testing.T) {
		api := search.NewJSAPI(ctx, &types.Config{
			KB: &types.KBConfig{Collections: []string{"docs"}},
			DB: &types.DBConfig{
				Models:     []string{"__yao.role"},
				MaxResults: 10,
			},
		}, &search.Uses{QueryDSL: "builtin"})

		requests := []interface{}{
			map[string]interface{}{
				"type":   "db",
				"query":  "查询角色",
				"models": []interface{}{"__yao.role"},
				"limit":  float64(5),
			},
			map[string]interface{}{
				"type":        "kb",
				"query":       "知识库查询",
				"collections": []interface{}{"docs"},
				"limit":       float64(5),
			},
		}

		results := api.All(requests)
		require.Len(t, results, 2)

		// DB result
		r0, ok := results[0].(*types.Result)
		require.True(t, ok)
		assert.Equal(t, types.SearchTypeDB, r0.Type)

		// KB result
		r1, ok := results[1].(*types.Result)
		require.True(t, ok)
		assert.Equal(t, types.SearchTypeKB, r1.Type)
	})
}

// newJSAPITestContext creates a test context for JSAPI tests
func newJSAPITestContext(t *testing.T) *context.Context {
	t.Helper()
	authorized := &oauthTypes.AuthorizedInfo{
		UserID: "test-user-jsapi",
	}
	chatID := "test-chat-jsapi-db"
	ctx := context.New(t.Context(), authorized, chatID)
	return ctx
}

// ensureJSAPITestRole ensures there's at least one role in the database
func ensureJSAPITestRole(t *testing.T, mod *model.Model) {
	t.Helper()

	// Try to find existing roles
	rows, err := mod.Get(model.QueryParam{Limit: 1})
	if err == nil && len(rows) > 0 {
		return
	}

	// Create a test role
	_, err = mod.Create(map[string]interface{}{
		"role_id":     "jsapi_test_role",
		"name":        "JSAPI Test Role",
		"description": "A test role for JSAPI unit testing",
		"is_active":   true,
		"is_system":   false,
		"level":       1,
	})
	if err != nil {
		t.Logf("Note: Could not create test role: %v", err)
	}
}
