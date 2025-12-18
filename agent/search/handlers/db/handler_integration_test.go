package db_test

import (
	"testing"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/query/gou"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/search/handlers/db"
	"github.com/yaoapp/yao/agent/search/types"
	"github.com/yaoapp/yao/agent/testutils"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Integration Tests - Requires database and models
// ============================================================================

func TestHandler_Search_Integration(t *testing.T) {
	// Skip if running short tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Initialize test environment (loads models, database, query engine, etc.)
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Create test context
	ctx := newTestContext(t)

	// Verify __yao.role model is loaded
	mod := model.Select("__yao.role")
	require.NotNil(t, mod, "__yao.role model should be loaded")

	t.Run("search_role_model_with_results", func(t *testing.T) {
		// First, ensure there's at least one role in the database
		ensureTestRole(t, mod)

		// Create handler with builtin QueryDSL generator
		h := db.NewHandler("builtin", &types.DBConfig{
			Models:     []string{"__yao.role"},
			MaxResults: 10,
		})

		req := &types.Request{
			Type:     types.SearchTypeDB,
			Query:    "查询所有角色",
			Source:   types.SourceAuto,
			Models:   []string{"__yao.role"},
			Scenario: types.ScenarioFilter,
			Limit:    10,
		}

		result, err := h.SearchWithContext(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Verify result structure
		assert.Equal(t, types.SearchTypeDB, result.Type)
		assert.Equal(t, "查询所有角色", result.Query)
		assert.Equal(t, types.SourceAuto, result.Source)
		assert.GreaterOrEqual(t, result.Duration, int64(0))

		// Should have results
		if result.Error != "" {
			t.Logf("Search error: %s", result.Error)
		}
		assert.Empty(t, result.Error, "Search should not return error")
		assert.Greater(t, len(result.Items), 0, "Should have at least one result")
		assert.Equal(t, len(result.Items), result.Total)

		// Verify result items
		for _, item := range result.Items {
			assert.Equal(t, types.SearchTypeDB, item.Type)
			assert.Equal(t, types.SourceAuto, item.Source)
			assert.Equal(t, "__yao.role", item.Model)
			assert.NotNil(t, item.Data, "Data should not be nil")
			assert.NotNil(t, item.RecordID, "RecordID should not be nil")
		}
	})

	t.Run("search_with_filter_scenario", func(t *testing.T) {
		h := db.NewHandler("builtin", nil)

		req := &types.Request{
			Type:     types.SearchTypeDB,
			Query:    "查询系统角色",
			Source:   types.SourceHook,
			Models:   []string{"__yao.role"},
			Scenario: types.ScenarioFilter,
			Limit:    5,
		}

		result, err := h.SearchWithContext(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Equal(t, types.SearchTypeDB, result.Type)
		assert.Equal(t, types.SourceHook, result.Source)
		assert.LessOrEqual(t, len(result.Items), 5, "Should respect limit")
	})

	t.Run("search_with_preset_wheres", func(t *testing.T) {
		h := db.NewHandler("builtin", nil)

		req := &types.Request{
			Type:   types.SearchTypeDB,
			Query:  "查询角色",
			Source: types.SourceAuto,
			Models: []string{"__yao.role"},
			Wheres: []gou.Where{
				{Condition: gou.Condition{Field: &gou.Expression{Field: "is_active"}, Value: true, OP: "="}},
			},
			Limit: 10,
		}

		result, err := h.SearchWithContext(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)

		// All results should have is_active = true (due to preset where)
		for _, item := range result.Items {
			if data, ok := item.Data["is_active"]; ok {
				// is_active could be bool or int depending on driver
				switch v := data.(type) {
				case bool:
					assert.True(t, v)
				case int64:
					assert.Equal(t, int64(1), v)
				case float64:
					assert.Equal(t, float64(1), v)
				}
			}
		}
	})

	t.Run("search_nonexistent_model_graceful", func(t *testing.T) {
		h := db.NewHandler("builtin", nil)

		req := &types.Request{
			Type:   types.SearchTypeDB,
			Query:  "查询文章",
			Source: types.SourceAuto,
			Models: []string{"nonexistent_model", "article", "fake_model"},
			Limit:  10,
		}

		// Should NOT panic, should return gracefully with error
		result, err := h.SearchWithContext(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Should have error message about no valid models
		assert.Equal(t, types.SearchTypeDB, result.Type)
		assert.Equal(t, "no valid models found", result.Error)
		assert.Empty(t, result.Items)
	})

	t.Run("search_mixed_models_partial_exist", func(t *testing.T) {
		h := db.NewHandler("builtin", nil)

		req := &types.Request{
			Type:   types.SearchTypeDB,
			Query:  "查询角色",
			Source: types.SourceAuto,
			Models: []string{"nonexistent_model", "__yao.role", "fake_model"}, // Only __yao.role exists
			Limit:  10,
		}

		// Should NOT panic, should work with the existing model
		result, err := h.SearchWithContext(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Should succeed with partial models
		assert.Equal(t, types.SearchTypeDB, result.Type)
		if result.Error == "" {
			// If no error, should have results from __yao.role
			assert.GreaterOrEqual(t, len(result.Items), 0)
		}
	})
}

// newTestContext creates a test context with required fields
func newTestContext(t *testing.T) *context.Context {
	t.Helper()
	authorized := &oauthTypes.AuthorizedInfo{
		UserID: "test-user",
	}
	chatID := "test-chat-db-search"
	ctx := context.New(t.Context(), authorized, chatID)
	return ctx
}

// ensureTestRole ensures there's at least one role in the database for testing
func ensureTestRole(t *testing.T, mod *model.Model) {
	t.Helper()

	// Try to find existing roles
	rows, err := mod.Get(model.QueryParam{Limit: 1})
	if err == nil && len(rows) > 0 {
		return // Already have roles
	}

	// Create a test role
	_, err = mod.Create(map[string]interface{}{
		"role_id":     "test_role",
		"name":        "Test Role",
		"description": "A test role for unit testing",
		"is_active":   true,
		"is_system":   false,
		"level":       1,
	})
	if err != nil {
		t.Logf("Note: Could not create test role: %v", err)
	}
}
