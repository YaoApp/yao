package querydsl

import (
	stdContext "context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

// newTestContext creates a test context for MCP testing
func newTestContext() *agentContext.Context {
	ctx := agentContext.New(stdContext.Background(), nil, "test-chat")
	ctx.AssistantID = "test-assistant"
	ctx.Locale = "en"
	ctx.Referer = agentContext.RefererAPI
	stack, _, _ := agentContext.EnterStack(ctx, "test-assistant", &agentContext.Options{})
	ctx.Stack = stack
	return ctx
}

func TestMCPProvider_Generate(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Create context
	ctx := newTestContext()

	// Create MCP provider for search.generate_querydsl
	provider, err := NewMCPProvider("search.generate_querydsl")
	assert.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, "search", provider.serverID)
	assert.Equal(t, "generate_querydsl", provider.toolName)

	t.Run("verify_fixed_structure", func(t *testing.T) {
		input := &Input{
			Query:    "find active users",
			ModelIDs: []string{"user"},
			Limit:    10,
		}

		result, err := provider.Generate(ctx, input)
		if err != nil {
			t.Logf("Generate error: %v", err)
		}
		assert.NoError(t, err)
		assert.NotNil(t, result)

		if result == nil {
			t.Fatal("result is nil")
		}

		if !assert.NotNil(t, result.DSL, "DSL should not be nil") {
			t.Logf("Result: Explain=%s, Warnings=%v", result.Explain, result.Warnings)
			return
		}

		// Verify fixed DSL structure from mock
		// select: ["id", "name", "status"] - parsed as Expression with Field property
		assert.Len(t, result.DSL.Select, 3)
		if len(result.DSL.Select) >= 3 {
			assert.Equal(t, "id", result.DSL.Select[0].Field)
			assert.Equal(t, "name", result.DSL.Select[1].Field)
			assert.Equal(t, "status", result.DSL.Select[2].Field)
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

		// limit: 10 (from input, returned as float64 from JSON)
		assert.Equal(t, float64(10), result.DSL.Limit)

		// explain should contain query
		assert.Contains(t, result.Explain, "find active users")

		// warnings should be empty
		assert.Empty(t, result.Warnings)
	})
}

func TestNewMCPProvider(t *testing.T) {
	t.Run("valid format", func(t *testing.T) {
		provider, err := NewMCPProvider("nlp.generate_querydsl")
		assert.NoError(t, err)
		assert.NotNil(t, provider)
		assert.Equal(t, "nlp", provider.serverID)
		assert.Equal(t, "generate_querydsl", provider.toolName)
	})

	t.Run("invalid format - no dot", func(t *testing.T) {
		provider, err := NewMCPProvider("invalid")
		assert.Error(t, err)
		assert.Nil(t, provider)
		assert.Contains(t, err.Error(), "invalid MCP format")
	})

	t.Run("complex tool name", func(t *testing.T) {
		provider, err := NewMCPProvider("server.tool.with.dots")
		assert.NoError(t, err)
		assert.NotNil(t, provider)
		assert.Equal(t, "server", provider.serverID)
		assert.Equal(t, "tool.with.dots", provider.toolName)
	})
}

func TestMCPProvider_Generate_Error(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := newTestContext()

	t.Run("non-existent server", func(t *testing.T) {
		provider, _ := NewMCPProvider("nonexistent.tool")
		result, err := provider.Generate(ctx, &Input{
			Query:    "test",
			ModelIDs: []string{"user"},
		})
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestGenerator_MCP_Integration(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// Skip if not in integration test mode
	if os.Getenv("YAO_TEST_MCP") != "true" {
		t.Skip("Skipping MCP integration test (set YAO_TEST_MCP=true to run)")
	}

	ctx := newTestContext()

	// Create generator with MCP mode
	gen := NewGenerator("mcp:search.generate_querydsl", nil)

	t.Run("generate_via_mcp", func(t *testing.T) {
		input := &Input{
			Query:    "find active users",
			ModelIDs: []string{"user"},
			Limit:    15,
		}

		result, err := gen.Generate(ctx, input)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotNil(t, result.DSL)

		// Verify fixed structure is correctly parsed
		assert.Len(t, result.DSL.Select, 3)
		assert.Len(t, result.DSL.Wheres, 1)
		assert.Len(t, result.DSL.Orders, 1)
		assert.Equal(t, float64(15), result.DSL.Limit)
		assert.Contains(t, result.Explain, "find active users")
	})

	t.Run("allowed_fields_validation", func(t *testing.T) {
		input := &Input{
			Query:         "find users",
			ModelIDs:      []string{"user"},
			AllowedFields: []string{"id", "name"}, // Only allow id and name
			Limit:         10,
		}

		result, err := gen.Generate(ctx, input)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotNil(t, result.DSL)

		// "status" field should be filtered out from select and wheres
		// since it's not in AllowedFields
		for _, expr := range result.DSL.Select {
			assert.Contains(t, []string{"id", "name"}, expr.Field)
		}

		// Should have warning about removed fields
		assert.NotEmpty(t, result.Warnings)
	})
}

func TestMCPProvider_Generate_WithRetry(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	ctx := newTestContext()

	// Create MCP provider for search.generate_querydsl_with_retry
	// This tool returns invalid DSL on first call, valid on second
	provider, err := NewMCPProvider("search.generate_querydsl_with_retry")
	assert.NoError(t, err)
	assert.NotNil(t, provider)

	t.Run("retry_on_lint_failure", func(t *testing.T) {
		input := &Input{
			Query:    "test retry mechanism",
			ModelIDs: []string{"user"},
			Limit:    10,
		}

		// This should succeed after retry
		// First call returns invalid DSL (missing 'from')
		// Second call (with lint_errors) returns valid DSL
		result, err := provider.Generate(ctx, input)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		if result != nil && result.DSL != nil {
			// Should have valid DSL after retry
			assert.NotNil(t, result.DSL.From, "DSL should have 'from' field after retry")
			// Explain should indicate this was fixed after receiving lint errors
			assert.Contains(t, result.Explain, "fixed after receiving lint errors")
		}
	})
}
