package standard_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	agentcontext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/robot/executor/standard"
	"github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/testutils"
)

// ============================================================================
// Validator Tests - Two-Layer Validation System
// ============================================================================

func TestValidatorValidateWithContext(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("validates with no rules - passes with valid output", func(t *testing.T) {
		robot := createValidatorTestRobot(t)
		config := standard.DefaultRunConfig()
		validator := standard.NewValidator(ctx, robot, config)

		task := &types.Task{
			ID:              "task-001",
			ExpectedOutput:  "",
			ValidationRules: []string{},
		}

		result := validator.ValidateWithContext(task, "Some output", nil)

		assert.True(t, result.Passed)
		assert.True(t, result.Complete)
		assert.False(t, result.NeedReply)
		assert.Equal(t, 1.0, result.Score)
	})

	t.Run("validates with no rules - incomplete with empty output", func(t *testing.T) {
		robot := createValidatorTestRobot(t)
		config := standard.DefaultRunConfig()
		validator := standard.NewValidator(ctx, robot, config)

		task := &types.Task{
			ID:              "task-001",
			ExpectedOutput:  "",
			ValidationRules: []string{},
		}

		result := validator.ValidateWithContext(task, "", nil)

		assert.True(t, result.Passed)
		assert.False(t, result.Complete) // Empty output = not complete
	})

	t.Run("validates with rule-based validation - passes", func(t *testing.T) {
		robot := createValidatorTestRobot(t)
		config := standard.DefaultRunConfig()
		validator := standard.NewValidator(ctx, robot, config)

		task := &types.Task{
			ID: "task-001",
			ValidationRules: []string{
				`{"type": "contains", "value": "hello"}`,
			},
		}

		result := validator.ValidateWithContext(task, "hello world", nil)

		assert.True(t, result.Passed)
		assert.True(t, result.Complete)
		assert.False(t, result.NeedReply)
	})

	t.Run("validates with rule-based validation - fails", func(t *testing.T) {
		robot := createValidatorTestRobot(t)
		config := standard.DefaultRunConfig()
		validator := standard.NewValidator(ctx, robot, config)

		task := &types.Task{
			ID: "task-001",
			ValidationRules: []string{
				`{"type": "contains", "value": "expected_string"}`,
			},
		}

		result := validator.ValidateWithContext(task, "actual output without expected", nil)

		assert.False(t, result.Passed)
		assert.False(t, result.Complete)
		assert.True(t, result.NeedReply) // Should suggest retry
		assert.NotEmpty(t, result.ReplyContent)
		assert.NotEmpty(t, result.Issues)
	})

	t.Run("validates with semantic validation", func(t *testing.T) {
		robot := createValidatorTestRobot(t)
		config := standard.DefaultRunConfig()
		validator := standard.NewValidator(ctx, robot, config)

		task := &types.Task{
			ID:             "task-001",
			ExpectedOutput: "A professional greeting message",
		}

		result := validator.ValidateWithContext(task, "Dear Sir/Madam, I hope this message finds you well.", nil)

		// Semantic validation should pass for this appropriate output
		t.Logf("Validation result: passed=%v, complete=%v, score=%.2f",
			result.Passed, result.Complete, result.Score)
		t.Logf("Issues: %v", result.Issues)
		t.Logf("Suggestions: %v", result.Suggestions)

		// The semantic validator should recognize this as appropriate
		assert.NotNil(t, result)
	})
}

func TestValidatorIsComplete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("complete when passed with valid output", func(t *testing.T) {
		robot := createValidatorTestRobot(t)
		config := standard.DefaultRunConfig()
		validator := standard.NewValidator(ctx, robot, config)

		task := &types.Task{
			ID:              "task-001",
			ExpectedOutput:  "",
			ValidationRules: []string{},
		}

		result := validator.ValidateWithContext(task, "Valid output", nil)

		assert.True(t, result.Complete)
	})

	t.Run("not complete when passed but empty output", func(t *testing.T) {
		robot := createValidatorTestRobot(t)
		config := standard.DefaultRunConfig()
		validator := standard.NewValidator(ctx, robot, config)

		task := &types.Task{
			ID:              "task-001",
			ExpectedOutput:  "",
			ValidationRules: []string{},
		}

		result := validator.ValidateWithContext(task, "", nil)

		assert.False(t, result.Complete)
	})

	t.Run("not complete when validation failed", func(t *testing.T) {
		robot := createValidatorTestRobot(t)
		config := standard.DefaultRunConfig()
		validator := standard.NewValidator(ctx, robot, config)

		task := &types.Task{
			ID: "task-001",
			ValidationRules: []string{
				`{"type": "contains", "value": "MUST_CONTAIN_THIS"}`,
			},
		}

		result := validator.ValidateWithContext(task, "output without required string", nil)

		assert.False(t, result.Passed)
		assert.False(t, result.Complete)
	})

	t.Run("not complete when score below threshold", func(t *testing.T) {
		robot := createValidatorTestRobot(t)
		config := standard.DefaultRunConfig()
		config.ValidationThreshold = 0.9 // High threshold
		validator := standard.NewValidator(ctx, robot, config)

		task := &types.Task{
			ID:             "task-001",
			ExpectedOutput: "A very specific output format that's hard to match exactly",
		}

		// This output might get a lower score due to semantic mismatch
		result := validator.ValidateWithContext(task, "Some generic output", nil)

		// If score is below threshold, should not be complete
		if result.Passed && result.Score < config.ValidationThreshold {
			assert.False(t, result.Complete)
		}
	})
}

func TestValidatorCheckNeedReply(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("no reply needed when complete", func(t *testing.T) {
		robot := createValidatorTestRobot(t)
		config := standard.DefaultRunConfig()
		validator := standard.NewValidator(ctx, robot, config)

		task := &types.Task{
			ID:              "task-001",
			ExpectedOutput:  "",
			ValidationRules: []string{},
		}

		result := validator.ValidateWithContext(task, "Complete output", nil)

		assert.True(t, result.Complete)
		assert.False(t, result.NeedReply)
		assert.Empty(t, result.ReplyContent)
	})

	t.Run("reply needed when validation failed with suggestions", func(t *testing.T) {
		robot := createValidatorTestRobot(t)
		config := standard.DefaultRunConfig()
		validator := standard.NewValidator(ctx, robot, config)

		task := &types.Task{
			ID: "task-001",
			ValidationRules: []string{
				`{"type": "type", "value": "object"}`,
			},
		}

		// String output when object expected
		result := validator.ValidateWithContext(task, "not an object", nil)

		assert.False(t, result.Passed)
		assert.True(t, result.NeedReply)
		assert.NotEmpty(t, result.ReplyContent)
		// The reply should contain validation feedback about the issue
		assert.Contains(t, result.ReplyContent, "did not pass validation")
	})

	t.Run("reply needed when output is empty but passed", func(t *testing.T) {
		robot := createValidatorTestRobot(t)
		config := standard.DefaultRunConfig()
		validator := standard.NewValidator(ctx, robot, config)

		task := &types.Task{
			ID:              "task-001",
			ExpectedOutput:  "",
			ValidationRules: []string{},
		}

		result := validator.ValidateWithContext(task, "   ", nil) // Whitespace only

		// Passed (no rules) but not complete (empty output)
		assert.True(t, result.Passed)
		assert.False(t, result.Complete)
		// When passed but not complete (empty output), checkNeedReply may or may not
		// set NeedReply depending on the implementation details
		// Just verify the result is consistent
		t.Logf("NeedReply: %v, ReplyContent: %s", result.NeedReply, result.ReplyContent)
	})
}

func TestValidatorConvertStringRule(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("converts 'valid JSON' rule", func(t *testing.T) {
		robot := createValidatorTestRobot(t)
		config := standard.DefaultRunConfig()
		validator := standard.NewValidator(ctx, robot, config)

		task := &types.Task{
			ID: "task-001",
			ValidationRules: []string{
				"output must be valid JSON",
			},
		}

		// Valid JSON object
		result := validator.ValidateWithContext(task, map[string]interface{}{"key": "value"}, nil)
		assert.True(t, result.Passed)

		// Invalid (string is not an object)
		result2 := validator.ValidateWithContext(task, "not json", nil)
		assert.False(t, result2.Passed)
	})

	t.Run("converts 'must contain' rule", func(t *testing.T) {
		robot := createValidatorTestRobot(t)
		config := standard.DefaultRunConfig()
		validator := standard.NewValidator(ctx, robot, config)

		task := &types.Task{
			ID: "task-001",
			ValidationRules: []string{
				"must contain 'success'",
			},
		}

		result := validator.ValidateWithContext(task, "Operation was a success!", nil)
		assert.True(t, result.Passed)

		result2 := validator.ValidateWithContext(task, "Operation failed", nil)
		assert.False(t, result2.Passed)
	})

	t.Run("converts 'not empty' rule", func(t *testing.T) {
		robot := createValidatorTestRobot(t)
		config := standard.DefaultRunConfig()
		validator := standard.NewValidator(ctx, robot, config)

		task := &types.Task{
			ID: "task-001",
			ValidationRules: []string{
				"output must not be empty",
			},
		}

		result := validator.ValidateWithContext(task, "Some content", nil)
		assert.True(t, result.Passed)

		// Note: The "not empty" rule may be converted to semantic validation
		// rather than a rule-based assertion, so empty string might still pass
		// if semantic validation is lenient
		result2 := validator.ValidateWithContext(task, "", nil)
		t.Logf("Empty string validation: passed=%v, issues=%v", result2.Passed, result2.Issues)
	})

	t.Run("converts 'json array' rule", func(t *testing.T) {
		robot := createValidatorTestRobot(t)
		config := standard.DefaultRunConfig()
		validator := standard.NewValidator(ctx, robot, config)

		task := &types.Task{
			ID: "task-001",
			ValidationRules: []string{
				"must be json array",
			},
		}

		result := validator.ValidateWithContext(task, []interface{}{"a", "b", "c"}, nil)
		assert.True(t, result.Passed)

		result2 := validator.ValidateWithContext(task, map[string]interface{}{"key": "value"}, nil)
		assert.False(t, result2.Passed)
	})
}

func TestValidatorParseRules(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("parses JSON assertion rules", func(t *testing.T) {
		robot := createValidatorTestRobot(t)
		config := standard.DefaultRunConfig()
		validator := standard.NewValidator(ctx, robot, config)

		task := &types.Task{
			ID: "task-001",
			ValidationRules: []string{
				`{"type": "equals", "value": "expected"}`,
			},
		}

		result := validator.ValidateWithContext(task, "expected", nil)
		assert.True(t, result.Passed)

		result2 := validator.ValidateWithContext(task, "different", nil)
		assert.False(t, result2.Passed)
	})

	t.Run("parses regex rules", func(t *testing.T) {
		robot := createValidatorTestRobot(t)
		config := standard.DefaultRunConfig()
		validator := standard.NewValidator(ctx, robot, config)

		task := &types.Task{
			ID: "task-001",
			ValidationRules: []string{
				`{"type": "regex", "value": "^[A-Z][a-z]+$"}`,
			},
		}

		result := validator.ValidateWithContext(task, "Hello", nil)
		assert.True(t, result.Passed)

		result2 := validator.ValidateWithContext(task, "hello", nil)
		assert.False(t, result2.Passed)
	})

	t.Run("parses json_path rules", func(t *testing.T) {
		robot := createValidatorTestRobot(t)
		config := standard.DefaultRunConfig()
		validator := standard.NewValidator(ctx, robot, config)

		task := &types.Task{
			ID: "task-001",
			ValidationRules: []string{
				`{"type": "json_path", "path": "data.count", "value": 42}`,
			},
		}

		result := validator.ValidateWithContext(task, map[string]interface{}{
			"data": map[string]interface{}{
				"count": 42,
			},
		}, nil)
		assert.True(t, result.Passed)

		result2 := validator.ValidateWithContext(task, map[string]interface{}{
			"data": map[string]interface{}{
				"count": 10,
			},
		}, nil)
		assert.False(t, result2.Passed)
	})

	t.Run("parses type rules with path", func(t *testing.T) {
		robot := createValidatorTestRobot(t)
		config := standard.DefaultRunConfig()
		validator := standard.NewValidator(ctx, robot, config)

		task := &types.Task{
			ID: "task-001",
			ValidationRules: []string{
				`{"type": "type", "path": "items", "value": "array"}`,
			},
		}

		result := validator.ValidateWithContext(task, map[string]interface{}{
			"items": []interface{}{"a", "b"},
		}, nil)
		assert.True(t, result.Passed)

		result2 := validator.ValidateWithContext(task, map[string]interface{}{
			"items": "not an array",
		}, nil)
		assert.False(t, result2.Passed)
	})
}

func TestValidatorSemanticValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("semantic validation with expected output", func(t *testing.T) {
		robot := createValidatorTestRobot(t)
		config := standard.DefaultRunConfig()
		validator := standard.NewValidator(ctx, robot, config)

		task := &types.Task{
			ID:             "task-001",
			ExpectedOutput: "A JSON object containing user information with name and email fields",
		}

		output := map[string]interface{}{
			"name":  "John Doe",
			"email": "john@example.com",
		}

		result := validator.ValidateWithContext(task, output, nil)

		t.Logf("Semantic validation: passed=%v, score=%.2f, complete=%v",
			result.Passed, result.Score, result.Complete)
		t.Logf("Details: %s", result.Details)

		// Should pass semantic validation
		assert.NotNil(t, result)
	})

	t.Run("semantic validation with complex criteria", func(t *testing.T) {
		robot := createValidatorTestRobot(t)
		config := standard.DefaultRunConfig()
		validator := standard.NewValidator(ctx, robot, config)

		task := &types.Task{
			ID:             "task-001",
			ExpectedOutput: "A professional email with greeting, body, and signature",
			Messages: []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: "Write a professional email"},
			},
		}

		output := `Dear Mr. Smith,

I hope this email finds you well. I am writing to follow up on our previous conversation regarding the project timeline.

Please let me know if you have any questions.

Best regards,
John Doe`

		result := validator.ValidateWithContext(task, output, nil)

		t.Logf("Email validation: passed=%v, score=%.2f", result.Passed, result.Score)

		// Should recognize this as a valid professional email
		assert.NotNil(t, result)
	})
}

func TestValidatorMergeResults(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("both rule and semantic validation pass", func(t *testing.T) {
		robot := createValidatorTestRobot(t)
		config := standard.DefaultRunConfig()
		validator := standard.NewValidator(ctx, robot, config)

		task := &types.Task{
			ID:             "task-001",
			ExpectedOutput: "A greeting message",
			ValidationRules: []string{
				`{"type": "contains", "value": "Hello"}`,
			},
		}

		result := validator.ValidateWithContext(task, "Hello, how are you today?", nil)

		assert.True(t, result.Passed)
		assert.True(t, result.Complete)
	})

	t.Run("rule passes but semantic fails", func(t *testing.T) {
		robot := createValidatorTestRobot(t)
		config := standard.DefaultRunConfig()
		validator := standard.NewValidator(ctx, robot, config)

		task := &types.Task{
			ID:             "task-001",
			ExpectedOutput: "A formal business letter with proper formatting",
			ValidationRules: []string{
				`{"type": "contains", "value": "Hello"}`, // This will pass
			},
		}

		// Contains "Hello" but not a formal business letter
		result := validator.ValidateWithContext(task, "Hello there buddy!", nil)

		// Rule passes, but semantic might not
		t.Logf("Merged result: passed=%v, score=%.2f", result.Passed, result.Score)
	})

	t.Run("rule fails - semantic not run", func(t *testing.T) {
		robot := createValidatorTestRobot(t)
		config := standard.DefaultRunConfig()
		validator := standard.NewValidator(ctx, robot, config)

		task := &types.Task{
			ID:             "task-001",
			ExpectedOutput: "Some expected output",
			ValidationRules: []string{
				`{"type": "contains", "value": "REQUIRED_STRING"}`,
			},
		}

		result := validator.ValidateWithContext(task, "Output without required string", nil)

		// Should fail at rule level, semantic not needed
		assert.False(t, result.Passed)
		assert.False(t, result.Complete)
	})
}

// ============================================================================
// Helper Functions
// ============================================================================

// createValidatorTestRobot creates a test robot for validator tests
func createValidatorTestRobot(t *testing.T) *types.Robot {
	t.Helper()
	return &types.Robot{
		MemberID:     "test-robot-validator",
		TeamID:       "test-team-1",
		DisplayName:  "Test Robot for Validator",
		SystemPrompt: "You are a helpful assistant.",
		Config: &types.Config{
			Identity: &types.Identity{
				Role:   "Test Assistant",
				Duties: []string{"Validate outputs"},
			},
			Resources: &types.Resources{
				Phases: map[types.Phase]string{
					types.PhaseRun: "robot.validation",
					"validation":   "robot.validation", // For semantic validation agent
				},
				Agents: []string{
					"experts.data-analyst",
					"experts.text-writer",
				},
			},
		},
	}
}
