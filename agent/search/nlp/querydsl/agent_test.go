package querydsl_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/assistant"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/search/nlp/querydsl"
	"github.com/yaoapp/yao/agent/testutils"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
)

func TestNewAgentProvider(t *testing.T) {
	t.Run("create_provider", func(t *testing.T) {
		provider := querydsl.NewAgentProvider("tests.querydsl-agent")
		assert.NotNil(t, provider)
	})
}

func TestAgentProvider_Generate(t *testing.T) {
	// Skip if running short tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Initialize test environment
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Load the querydsl-agent assistant
	ast, err := assistant.Get("tests.querydsl-agent")
	require.NoError(t, err)
	require.NotNil(t, ast)

	// Create test context
	ctx := newTestContext(t)

	// Create Agent provider for tests.querydsl-agent
	provider := querydsl.NewAgentProvider("tests.querydsl-agent")
	assert.NotNil(t, provider)

	t.Run("verify_fixed_structure", func(t *testing.T) {
		input := &querydsl.Input{
			Query:    "find active users",
			ModelIDs: []string{"user"},
			Limit:    15,
		}

		result, err := provider.Generate(ctx, input)
		if err != nil {
			t.Logf("Generate error: %v", err)
		}
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.DSL, "DSL should not be nil")

		// Verify fixed DSL structure from mock
		// select: ["id", "name", "status", "created_at"]
		assert.Len(t, result.DSL.Select, 4)
		if len(result.DSL.Select) >= 4 {
			assert.Equal(t, "id", result.DSL.Select[0].Field)
			assert.Equal(t, "name", result.DSL.Select[1].Field)
			assert.Equal(t, "status", result.DSL.Select[2].Field)
			assert.Equal(t, "created_at", result.DSL.Select[3].Field)
		}

		// wheres: [{ field: "status", op: "=", value: "active" }]
		assert.Len(t, result.DSL.Wheres, 1)
		if len(result.DSL.Wheres) > 0 {
			assert.Equal(t, "status", result.DSL.Wheres[0].Field.Field)
			assert.Equal(t, "=", result.DSL.Wheres[0].OP)
			assert.Equal(t, "active", result.DSL.Wheres[0].Value)
		}

		// orders: [{ field: "created_at", sort: "desc" }]
		assert.Len(t, result.DSL.Orders, 1)
		if len(result.DSL.Orders) > 0 {
			assert.Equal(t, "created_at", result.DSL.Orders[0].Field.Field)
			assert.Equal(t, "desc", result.DSL.Orders[0].Sort)
		}

		// limit: 15 (from input)
		assert.Equal(t, float64(15), result.DSL.Limit)

		// explain should contain query
		assert.Contains(t, result.Explain, "find active users")

		// warnings should be empty
		assert.Empty(t, result.Warnings)
	})
}

func TestAgentProvider_Generate_Error(t *testing.T) {
	// Skip if running short tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Initialize test environment
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Create test context
	ctx := newTestContext(t)

	t.Run("non-existent_agent", func(t *testing.T) {
		provider := querydsl.NewAgentProvider("tests.nonexistent-agent")
		result, err := provider.Generate(ctx, &querydsl.Input{
			Query:    "test",
			ModelIDs: []string{"user"},
		})
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to get agent")
	})

	t.Run("nil_context", func(t *testing.T) {
		provider := querydsl.NewAgentProvider("tests.querydsl-agent")
		result, err := provider.Generate(nil, &querydsl.Input{
			Query:    "test",
			ModelIDs: []string{"user"},
		})
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "context is required")
	})
}

func TestGenerator_Agent_Integration(t *testing.T) {
	// Skip if running short tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Initialize test environment
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Create test context
	ctx := newTestContext(t)

	// Create generator with Agent mode (assistant ID without mcp: prefix)
	gen := querydsl.NewGenerator("tests.querydsl-agent", nil)

	t.Run("generate_via_agent", func(t *testing.T) {
		input := &querydsl.Input{
			Query:    "find active users",
			ModelIDs: []string{"user"},
			Limit:    10,
		}

		result, err := gen.Generate(ctx, input)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.DSL)

		// Verify structure from agent mock
		assert.Len(t, result.DSL.Select, 4)
		assert.Len(t, result.DSL.Wheres, 1)
		assert.Len(t, result.DSL.Orders, 1)
		assert.Contains(t, result.Explain, "find active users")
	})

	t.Run("allowed_fields_validation", func(t *testing.T) {
		input := &querydsl.Input{
			Query:         "find users",
			ModelIDs:      []string{"user"},
			AllowedFields: []string{"id", "name"}, // Only allow id and name
			Limit:         10,
		}

		result, err := gen.Generate(ctx, input)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.DSL)

		// "status" and "created_at" fields should be filtered out from select
		// since they are not in AllowedFields
		for _, expr := range result.DSL.Select {
			assert.Contains(t, []string{"id", "name"}, expr.Field)
		}

		// Should have warning about removed fields
		assert.NotEmpty(t, result.Warnings)
	})
}

func TestAgentProvider_Generate_WithRetry(t *testing.T) {
	// Skip if running short tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Initialize test environment
	testutils.Prepare(t)
	defer testutils.Clean(t)

	// Load the querydsl-agent-retry assistant
	ast, err := assistant.Get("tests.querydsl-agent-retry")
	require.NoError(t, err)
	require.NotNil(t, ast)

	// Create test context
	ctx := newTestContext(t)

	// Create Agent provider for tests.querydsl-agent-retry
	// This agent returns invalid DSL on first call, valid on second
	provider := querydsl.NewAgentProvider("tests.querydsl-agent-retry")
	assert.NotNil(t, provider)

	t.Run("retry_on_lint_failure", func(t *testing.T) {
		input := &querydsl.Input{
			Query:    "test retry mechanism",
			ModelIDs: []string{"user"},
			Limit:    10,
		}

		// This should succeed after retry
		// First call returns invalid DSL (missing 'from')
		// Second call (with lint_errors) returns valid DSL
		result, err := provider.Generate(ctx, input)
		require.NoError(t, err)
		require.NotNil(t, result)

		if result.DSL != nil {
			// Should have valid DSL after retry
			assert.NotNil(t, result.DSL.From, "DSL should have 'from' field after retry")
			// Explain should indicate this was fixed after receiving lint errors
			assert.Contains(t, result.Explain, "fixed after receiving lint errors")
		}
	})
}

// newTestContext creates a test context with required fields
func newTestContext(t *testing.T) *context.Context {
	t.Helper()
	authorized := &oauthTypes.AuthorizedInfo{
		UserID: "test-user",
	}
	chatID := "test-chat-querydsl"
	ctx := context.New(t.Context(), authorized, chatID)
	return ctx
}
