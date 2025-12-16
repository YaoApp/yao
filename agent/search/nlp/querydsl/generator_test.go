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

func TestGenerator_Generate_Builtin(t *testing.T) {
	gen := NewGenerator("builtin", nil)

	// Note: In real usage, models are loaded internally via model.Select()
	// For this test, we just verify the basic flow works without models
	input := &Input{
		Query:    "find all active users",
		ModelIDs: []string{"user"},
		Limit:    10,
	}

	result, err := gen.Generate(nil, input)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.DSL)
	assert.NotEmpty(t, result.Explain)
	assert.NotEmpty(t, result.Warnings)
}

func TestGenerator_Generate_EmptyMode(t *testing.T) {
	// Empty mode should default to builtin
	gen := NewGenerator("", nil)

	input := &Input{
		Query:    "search products",
		ModelIDs: []string{"product"},
		Limit:    5,
	}

	result, err := gen.Generate(nil, input)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestBuiltinGenerator_Generate(t *testing.T) {
	gen := NewBuiltinGenerator()

	t.Run("empty query", func(t *testing.T) {
		result, err := gen.Generate(&Input{})
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Nil(t, result.DSL)
		assert.Contains(t, result.Warnings, "empty query, returning empty DSL")
	})

	t.Run("nil input", func(t *testing.T) {
		result, err := gen.Generate(nil)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Nil(t, result.DSL)
	})

	t.Run("basic query without models loaded", func(t *testing.T) {
		// Models are loaded internally via model.Select()
		// When model is not found, it still generates basic DSL
		result, err := gen.Generate(&Input{
			Query:    "find users",
			ModelIDs: []string{"user"},
			Limit:    10,
		})
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotNil(t, result.DSL)
		assert.Equal(t, 10, result.DSL.Limit)
	})

	t.Run("query with pre-defined wheres", func(t *testing.T) {
		preWheres := []gou.Where{
			{
				Condition: gou.Condition{
					Field: &gou.Expression{Field: "status"},
					OP:    "=",
					Value: "active",
				},
			},
		}
		result, err := gen.Generate(&Input{
			Query:    "find users",
			ModelIDs: []string{"user"},
			Wheres:   preWheres,
			Limit:    10,
		})
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotNil(t, result.DSL)
		// Should use pre-defined wheres
		assert.Equal(t, preWheres, result.DSL.Wheres)
	})

	t.Run("query with orders", func(t *testing.T) {
		orders := gou.Orders{
			{Field: &gou.Expression{Field: "created_at"}, Sort: "desc"},
		}
		result, err := gen.Generate(&Input{
			Query:    "find users",
			ModelIDs: []string{"user"},
			Orders:   orders,
			Limit:    10,
		})
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotNil(t, result.DSL)
		assert.Equal(t, orders, result.DSL.Orders)
	})

	t.Run("query with allowed fields", func(t *testing.T) {
		result, err := gen.Generate(&Input{
			Query:         "find users",
			ModelIDs:      []string{"user"},
			AllowedFields: []string{"id", "name", "email"},
			Limit:         10,
		})
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotNil(t, result.DSL)
	})

	t.Run("default limit", func(t *testing.T) {
		result, err := gen.Generate(&Input{
			Query:    "find users",
			ModelIDs: []string{"user"},
		})
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotNil(t, result.DSL)
		assert.Equal(t, 20, result.DSL.Limit)
	})

	t.Run("multi-model query", func(t *testing.T) {
		// Models are loaded internally via model.Select()
		result, err := gen.Generate(&Input{
			Query:    "find user orders",
			ModelIDs: []string{"user", "order"},
			Limit:    10,
		})
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotNil(t, result.DSL)
	})
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
