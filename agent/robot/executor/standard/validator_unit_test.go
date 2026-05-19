//go:build unit

package standard_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent/robot/executor/standard"
	"github.com/yaoapp/yao/agent/robot/types"
)

// ============================================================================
// Validator — hasValidOutput (pure logic, no DB/LLM)
// ============================================================================

func TestValidatorHasValidOutputUnit(t *testing.T) {
	ctx := types.NewContext(nil, nil)
	robot := &types.Robot{
		MemberID: "test-validator",
		Config: &types.Config{
			Identity: &types.Identity{Role: "Test"},
		},
	}
	config := standard.DefaultValidatorConfig()
	v := standard.NewValidator(ctx, robot, config)

	t.Run("nil output is not valid", func(t *testing.T) {
		result := standard.HasValidOutputFn(v, nil)
		assert.False(t, result)
	})

	t.Run("empty string is not valid", func(t *testing.T) {
		result := standard.HasValidOutputFn(v, "")
		assert.False(t, result)
	})

	t.Run("whitespace-only string is not valid", func(t *testing.T) {
		result := standard.HasValidOutputFn(v, "   ")
		assert.False(t, result)
	})

	t.Run("non-empty string is valid", func(t *testing.T) {
		result := standard.HasValidOutputFn(v, "hello")
		assert.True(t, result)
	})

	t.Run("empty slice is not valid", func(t *testing.T) {
		result := standard.HasValidOutputFn(v, []interface{}{})
		assert.False(t, result)
	})

	t.Run("non-empty slice is valid", func(t *testing.T) {
		result := standard.HasValidOutputFn(v, []interface{}{"a"})
		assert.True(t, result)
	})

	t.Run("empty map is not valid", func(t *testing.T) {
		result := standard.HasValidOutputFn(v, map[string]interface{}{})
		assert.False(t, result)
	})

	t.Run("non-empty map is valid", func(t *testing.T) {
		result := standard.HasValidOutputFn(v, map[string]interface{}{"key": "val"})
		assert.True(t, result)
	})

	t.Run("integer is valid", func(t *testing.T) {
		result := standard.HasValidOutputFn(v, 42)
		assert.True(t, result)
	})

	t.Run("bool is valid", func(t *testing.T) {
		result := standard.HasValidOutputFn(v, true)
		assert.True(t, result)
	})
}

// ============================================================================
// Validator — ValidateWithContext (no rules, no expected output — pure logic)
// ============================================================================

func TestValidatorValidateNoRulesUnit(t *testing.T) {
	ctx := types.NewContext(nil, nil)
	robot := &types.Robot{
		MemberID: "test-validator",
		Config: &types.Config{
			Identity: &types.Identity{Role: "Test"},
		},
	}
	config := standard.DefaultValidatorConfig()
	v := standard.NewValidator(ctx, robot, config)

	t.Run("passes with valid output and no rules", func(t *testing.T) {
		task := &types.Task{
			ID:              "task-001",
			ExpectedOutput:  "",
			ValidationRules: []string{},
		}

		result := v.ValidateWithContext(task, "Some output", nil)

		assert.True(t, result.Passed)
		assert.True(t, result.Complete)
		assert.Equal(t, 1.0, result.Score)
	})

	t.Run("incomplete with empty output and no rules", func(t *testing.T) {
		task := &types.Task{
			ID:              "task-001",
			ExpectedOutput:  "",
			ValidationRules: []string{},
		}

		result := v.ValidateWithContext(task, "", nil)

		assert.True(t, result.Passed)
		assert.False(t, result.Complete)
	})

	t.Run("incomplete with nil output and no rules", func(t *testing.T) {
		task := &types.Task{
			ID:              "task-002",
			ExpectedOutput:  "",
			ValidationRules: []string{},
		}

		result := v.ValidateWithContext(task, nil, nil)

		assert.True(t, result.Passed)
		assert.False(t, result.Complete)
	})
}

// ============================================================================
// Validator — Rule-based validation (contains, type, regex — no LLM)
// ============================================================================

func TestValidatorRuleBasedUnit(t *testing.T) {
	ctx := types.NewContext(nil, nil)
	robot := &types.Robot{
		MemberID: "test-validator",
		Config: &types.Config{
			Identity: &types.Identity{Role: "Test"},
		},
	}
	config := standard.DefaultValidatorConfig()
	v := standard.NewValidator(ctx, robot, config)

	t.Run("contains rule passes", func(t *testing.T) {
		task := &types.Task{
			ID:              "task-001",
			ValidationRules: []string{`{"type": "contains", "value": "hello"}`},
		}

		result := v.ValidateWithContext(task, "hello world", nil)
		assert.True(t, result.Passed)
		assert.True(t, result.Complete)
	})

	t.Run("contains rule fails", func(t *testing.T) {
		task := &types.Task{
			ID:              "task-001",
			ValidationRules: []string{`{"type": "contains", "value": "expected_string"}`},
		}

		result := v.ValidateWithContext(task, "actual output without expected", nil)
		assert.False(t, result.Passed)
		assert.False(t, result.Complete)
		assert.True(t, result.NeedReply)
		assert.NotEmpty(t, result.ReplyContent)
		assert.NotEmpty(t, result.Issues)
	})

	t.Run("type check passes for object", func(t *testing.T) {
		task := &types.Task{
			ID:              "task-001",
			ValidationRules: []string{`{"type": "type", "value": "object"}`},
		}

		result := v.ValidateWithContext(task, map[string]interface{}{"key": "value"}, nil)
		assert.True(t, result.Passed)
	})

	t.Run("type check fails for non-object", func(t *testing.T) {
		task := &types.Task{
			ID:              "task-001",
			ValidationRules: []string{`{"type": "type", "value": "object"}`},
		}

		result := v.ValidateWithContext(task, "not an object", nil)
		assert.False(t, result.Passed)
		assert.True(t, result.NeedReply)
	})

	t.Run("regex rule passes", func(t *testing.T) {
		task := &types.Task{
			ID:              "task-001",
			ValidationRules: []string{`{"type": "regex", "value": "^[A-Z][a-z]+$"}`},
		}

		result := v.ValidateWithContext(task, "Hello", nil)
		assert.True(t, result.Passed)
	})

	t.Run("regex rule fails", func(t *testing.T) {
		task := &types.Task{
			ID:              "task-001",
			ValidationRules: []string{`{"type": "regex", "value": "^[A-Z][a-z]+$"}`},
		}

		result := v.ValidateWithContext(task, "hello", nil)
		assert.False(t, result.Passed)
	})

	t.Run("type check with path passes for array", func(t *testing.T) {
		task := &types.Task{
			ID:              "task-001",
			ValidationRules: []string{`{"type": "type", "path": "items", "value": "array"}`},
		}

		result := v.ValidateWithContext(task, map[string]interface{}{
			"items": []interface{}{"a", "b"},
		}, nil)
		assert.True(t, result.Passed)
	})

	t.Run("type check with path fails for non-array", func(t *testing.T) {
		task := &types.Task{
			ID:              "task-001",
			ValidationRules: []string{`{"type": "type", "path": "items", "value": "array"}`},
		}

		result := v.ValidateWithContext(task, map[string]interface{}{
			"items": "not an array",
		}, nil)
		assert.False(t, result.Passed)
	})

	t.Run("equals rule passes", func(t *testing.T) {
		task := &types.Task{
			ID:              "task-001",
			ValidationRules: []string{`{"type": "equals", "value": "expected"}`},
		}

		result := v.ValidateWithContext(task, "expected", nil)
		assert.True(t, result.Passed)
	})

	t.Run("equals rule fails", func(t *testing.T) {
		task := &types.Task{
			ID:              "task-001",
			ValidationRules: []string{`{"type": "equals", "value": "expected"}`},
		}

		result := v.ValidateWithContext(task, "different", nil)
		assert.False(t, result.Passed)
	})
}

// ============================================================================
// Validator — convertStringRule (pure logic)
// ============================================================================

func TestValidatorConvertStringRuleUnit(t *testing.T) {
	ctx := types.NewContext(nil, nil)
	robot := &types.Robot{
		MemberID: "test-validator",
		Config: &types.Config{
			Identity: &types.Identity{Role: "Test"},
		},
	}
	config := standard.DefaultValidatorConfig()
	v := standard.NewValidator(ctx, robot, config)

	t.Run("converts 'valid JSON' rule", func(t *testing.T) {
		a := standard.ConvertStringRuleFn(v, "output must be valid JSON")
		assert.NotNil(t, a)
		assert.Equal(t, "type", a.Type)
		assert.Equal(t, "object", a.Value)
	})

	t.Run("converts 'json array' rule", func(t *testing.T) {
		a := standard.ConvertStringRuleFn(v, "must be json array")
		assert.NotNil(t, a)
		assert.Equal(t, "type", a.Type)
		assert.Equal(t, "array", a.Value)
	})

	t.Run("converts 'must contain' rule with single quotes", func(t *testing.T) {
		a := standard.ConvertStringRuleFn(v, "must contain 'success'")
		assert.NotNil(t, a)
		assert.Equal(t, "contains", a.Type)
		assert.Equal(t, "success", a.Value)
	})

	t.Run("converts 'must contain' rule with double quotes", func(t *testing.T) {
		a := standard.ConvertStringRuleFn(v, `must contain "hello"`)
		assert.NotNil(t, a)
		assert.Equal(t, "contains", a.Type)
		assert.Equal(t, "hello", a.Value)
	})

	t.Run("converts 'not empty' rule", func(t *testing.T) {
		task := &types.Task{
			ID:              "task-not-empty",
			ValidationRules: []string{"output must not be empty"},
		}
		result := v.ValidateWithContext(task, "Some content", nil)
		assert.True(t, result.Passed)
	})

	t.Run("converts 'non-empty' rule", func(t *testing.T) {
		task := &types.Task{
			ID:              "task-non-empty",
			ValidationRules: []string{"must be non-empty"},
		}
		result := v.ValidateWithContext(task, "content here", nil)
		assert.True(t, result.Passed)
	})

	t.Run("returns nil for unknown rules", func(t *testing.T) {
		a := standard.ConvertStringRuleFn(v, "count > 0")
		assert.Nil(t, a, "unknown rules should be handled by semantic validation")
	})
}

// ============================================================================
// Validator — hasAgentRules (pure logic)
// ============================================================================

func TestValidatorHasAgentRulesUnit(t *testing.T) {
	ctx := types.NewContext(nil, nil)
	robot := &types.Robot{
		MemberID: "test-validator",
		Config: &types.Config{
			Identity: &types.Identity{Role: "Test"},
		},
	}
	config := standard.DefaultValidatorConfig()
	v := standard.NewValidator(ctx, robot, config)

	t.Run("detects agent-type rules", func(t *testing.T) {
		rules := []string{
			`{"type": "agent", "agent": "validation.agent"}`,
		}
		assert.True(t, standard.HasAgentRulesFn(v, rules))
	})

	t.Run("returns false for non-agent rules", func(t *testing.T) {
		rules := []string{
			`{"type": "contains", "value": "hello"}`,
			"must be valid JSON",
		}
		assert.False(t, standard.HasAgentRulesFn(v, rules))
	})

	t.Run("returns false for empty rules", func(t *testing.T) {
		assert.False(t, standard.HasAgentRulesFn(v, nil))
		assert.False(t, standard.HasAgentRulesFn(v, []string{}))
	})
}

// ============================================================================
// Validator — getSemanticRules (pure logic)
// ============================================================================

func TestValidatorGetSemanticRulesUnit(t *testing.T) {
	ctx := types.NewContext(nil, nil)
	robot := &types.Robot{
		MemberID: "test-validator",
		Config: &types.Config{
			Identity: &types.Identity{Role: "Test"},
		},
	}
	config := standard.DefaultValidatorConfig()
	v := standard.NewValidator(ctx, robot, config)

	t.Run("returns only semantic rules", func(t *testing.T) {
		rules := []string{
			`{"type": "contains", "value": "hello"}`,
			"output must be valid JSON",
			"count > 0",
			"results should be meaningful",
		}
		semanticRules := standard.GetSemanticRulesFn(v, rules)

		assert.Len(t, semanticRules, 2)
		assert.Contains(t, semanticRules, "count > 0")
		assert.Contains(t, semanticRules, "results should be meaningful")
	})

	t.Run("returns nil for all convertible rules", func(t *testing.T) {
		rules := []string{
			`{"type": "regex", "value": ".*"}`,
			"must be valid JSON",
		}
		semanticRules := standard.GetSemanticRulesFn(v, rules)
		assert.Empty(t, semanticRules)
	})
}

// ============================================================================
// Validator — generateFeedbackReply (pure logic)
// ============================================================================

func TestValidatorGenerateFeedbackReplyUnit(t *testing.T) {
	ctx := types.NewContext(nil, nil)
	robot := &types.Robot{
		MemberID: "test-validator",
		Config: &types.Config{
			Identity: &types.Identity{Role: "Test"},
		},
	}
	config := standard.DefaultValidatorConfig()
	v := standard.NewValidator(ctx, robot, config)

	t.Run("includes issues and suggestions", func(t *testing.T) {
		result := &types.ValidationResult{
			Issues:      []string{"Missing required field", "Wrong format"},
			Suggestions: []string{"Add 'name' field", "Use JSON format"},
		}
		reply := standard.GenerateFeedbackReplyFn(v, result)

		assert.Contains(t, reply, "## Validation Feedback")
		assert.Contains(t, reply, "Missing required field")
		assert.Contains(t, reply, "Wrong format")
		assert.Contains(t, reply, "Add 'name' field")
		assert.Contains(t, reply, "Use JSON format")
		assert.Contains(t, reply, "improved response")
	})

	t.Run("handles issues only", func(t *testing.T) {
		result := &types.ValidationResult{
			Issues: []string{"Type mismatch"},
		}
		reply := standard.GenerateFeedbackReplyFn(v, result)

		assert.Contains(t, reply, "Type mismatch")
		assert.NotContains(t, reply, "### Suggestions")
	})
}

// ============================================================================
// Validator — DefaultValidatorConfig
// ============================================================================

func TestDefaultValidatorConfigUnit(t *testing.T) {
	config := standard.DefaultValidatorConfig()
	assert.NotNil(t, config)
	assert.Equal(t, 0.6, config.ValidationThreshold)
}
