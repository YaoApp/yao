package querydsl

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/query/gou"
	"github.com/yaoapp/yao/agent/search/types"
)

func TestNewGenerator(t *testing.T) {
	tests := []struct {
		name         string
		usesQueryDSL string
		config       *types.QueryDSLConfig
	}{
		{
			name:         "builtin mode",
			usesQueryDSL: "builtin",
			config:       nil,
		},
		{
			name:         "empty defaults to builtin",
			usesQueryDSL: "",
			config:       nil,
		},
		{
			name:         "agent mode",
			usesQueryDSL: "my-querydsl-agent",
			config:       &types.QueryDSLConfig{Strict: true},
		},
		{
			name:         "mcp mode",
			usesQueryDSL: "mcp:nlp.generate_querydsl",
			config:       nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := NewGenerator(tt.usesQueryDSL, tt.config)
			assert.NotNil(t, gen)
			assert.Equal(t, tt.usesQueryDSL, gen.usesQueryDSL)
			assert.Equal(t, tt.config, gen.config)
		})
	}
}

func TestGenerator_Generate_Builtin_RequiresContext(t *testing.T) {
	// Builtin mode now uses __yao.querydsl agent which requires context
	gen := NewGenerator("builtin", nil)

	input := &Input{
		Query:    "find all active users",
		ModelIDs: []string{"user"},
		Limit:    10,
	}

	// Without context, should return error
	_, err := gen.Generate(nil, input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context is required")
}

func TestGenerator_Generate_EmptyMode_RequiresContext(t *testing.T) {
	// Empty mode defaults to __yao.querydsl agent which requires context
	gen := NewGenerator("", nil)

	input := &Input{
		Query:    "search products",
		ModelIDs: []string{"product"},
		Limit:    5,
	}

	// Without context, should return error
	_, err := gen.Generate(nil, input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context is required")
}

func TestGenerator_Generate_AgentMode_RequiresContext(t *testing.T) {
	// Custom agent mode requires context
	gen := NewGenerator("custom.querydsl.agent", nil)

	input := &Input{
		Query:    "find users",
		ModelIDs: []string{"user"},
		Limit:    10,
	}

	_, err := gen.Generate(nil, input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context is required")
}

func TestGenerator_Generate_MCPMode_InvalidFormat(t *testing.T) {
	// Invalid MCP format should fallback to system agent (which requires context)
	gen := NewGenerator("mcp:invalid", nil)

	input := &Input{
		Query:    "find users",
		ModelIDs: []string{"user"},
		Limit:    10,
	}

	_, err := gen.Generate(nil, input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context is required")
}

func TestSystemQueryDSLAgentConstant(t *testing.T) {
	// Verify the system querydsl agent constant
	assert.Equal(t, "__yao.querydsl", SystemQueryDSLAgent)
}

func TestResult(t *testing.T) {
	result := &Result{
		DSL: &gou.QueryDSL{
			Limit: 10,
		},
		Explain:  "Generated query for finding users",
		Warnings: []string{"using placeholder implementation"},
	}

	assert.NotNil(t, result.DSL)
	assert.Equal(t, 10, result.DSL.Limit)
	assert.NotEmpty(t, result.Explain)
	assert.Len(t, result.Warnings, 1)
}

func TestGenerator_ValidateFields(t *testing.T) {
	gen := NewGenerator("", nil)

	t.Run("validate select fields", func(t *testing.T) {
		result := &Result{
			DSL: &gou.QueryDSL{
				Select: []gou.Expression{
					{Field: "id"},
					{Field: "name"},
					{Field: "secret_field"},
				},
			},
		}
		allowedFields := []string{"id", "name", "email"}

		validated := gen.validateFields(result, allowedFields)
		assert.NotNil(t, validated)
		assert.Len(t, validated.DSL.Select, 2)
		assert.Contains(t, validated.Warnings[0], "secret_field")
	})

	t.Run("validate where fields", func(t *testing.T) {
		result := &Result{
			DSL: &gou.QueryDSL{
				Wheres: []gou.Where{
					{
						Condition: gou.Condition{
							Field: &gou.Expression{Field: "status"},
							OP:    "=",
							Value: "active",
						},
					},
					{
						Condition: gou.Condition{
							Field: &gou.Expression{Field: "secret"},
							OP:    "=",
							Value: "hidden",
						},
					},
				},
			},
		}
		allowedFields := []string{"status", "name"}

		validated := gen.validateFields(result, allowedFields)
		assert.NotNil(t, validated)
		assert.Len(t, validated.DSL.Wheres, 1)
		assert.Contains(t, validated.Warnings[0], "secret")
	})

	t.Run("validate order fields", func(t *testing.T) {
		result := &Result{
			DSL: &gou.QueryDSL{
				Orders: gou.Orders{
					{Field: &gou.Expression{Field: "created_at"}, Sort: "desc"},
					{Field: &gou.Expression{Field: "secret_sort"}, Sort: "asc"},
				},
			},
		}
		allowedFields := []string{"created_at", "updated_at"}

		validated := gen.validateFields(result, allowedFields)
		assert.NotNil(t, validated)
		assert.Len(t, validated.DSL.Orders, 1)
		assert.Contains(t, validated.Warnings[0], "secret_sort")
	})

	t.Run("nil DSL", func(t *testing.T) {
		result := &Result{DSL: nil}
		allowedFields := []string{"id", "name"}

		validated := gen.validateFields(result, allowedFields)
		assert.NotNil(t, validated)
		assert.Nil(t, validated.DSL)
	})
}
