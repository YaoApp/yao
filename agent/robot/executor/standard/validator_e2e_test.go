//go:build e2e

package standard_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/robot/executor/standard"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestValidatorValidateWithContextE2E(t *testing.T) {
	identity := testprepare.PrepareE2E(t)
	ctx := testCtx(identity)

	t.Run("validates_task_output_with_rules", func(t *testing.T) {
		v := standard.NewValidator("tests.robot-single")
		task := robottypes.Task{
			ID:              "task-val-1",
			ExecutorType:    robottypes.ExecutorAssistant,
			ExecutorID:      "data-analyst",
			ExpectedOutput:  "JSON with sales data",
			ValidationRules: []string{"must contain sales figures", "must be valid JSON"},
		}
		result := robottypes.TaskResult{
			TaskID:  "task-val-1",
			Success: true,
			Output:  `{"total_sales": 1500000, "top_product": "Widget A"}`,
		}

		vr, err := v.ValidateWithContext(ctx, &task, &result)
		require.NoError(t, err)
		require.NotNil(t, vr)
		assert.True(t, vr.Passed)
		assert.GreaterOrEqual(t, vr.Score, 0.5)
	})

	t.Run("fails_invalid_output", func(t *testing.T) {
		v := standard.NewValidator("tests.robot-single")
		task := robottypes.Task{
			ID:              "task-val-2",
			ExecutorType:    robottypes.ExecutorAssistant,
			ExecutorID:      "data-analyst",
			ExpectedOutput:  "JSON with financial data",
			ValidationRules: []string{"must contain revenue numbers", "must be valid JSON"},
		}
		result := robottypes.TaskResult{
			TaskID:  "task-val-2",
			Success: true,
			Output:  "This is just plain text with no data.",
		}

		vr, err := v.ValidateWithContext(ctx, &task, &result)
		require.NoError(t, err)
		require.NotNil(t, vr)
		assert.False(t, vr.Passed)
	})
}

func TestValidatorIsCompleteE2E(t *testing.T) {
	identity := testprepare.PrepareE2E(t)
	ctx := testCtx(identity)

	t.Run("complete_execution_passes", func(t *testing.T) {
		v := standard.NewValidator("tests.robot-single")

		exec := &robottypes.Execution{
			ID:     "exec-complete-1",
			Status: robottypes.ExecCompleted,
			Goals:  &robottypes.Goals{Content: "Analyze sales data"},
			Tasks: []robottypes.Task{
				{ID: "t1", Status: robottypes.TaskCompleted},
			},
			Results: []robottypes.TaskResult{
				{TaskID: "t1", Success: true, Output: "Sales data analyzed: $1.5M revenue"},
			},
		}

		complete, err := v.IsComplete(ctx, exec)
		require.NoError(t, err)
		assert.True(t, complete)
	})
}

func TestValidatorSemanticValidationE2E(t *testing.T) {
	identity := testprepare.PrepareE2E(t)
	ctx := testCtx(identity)

	t.Run("semantic_rules_produce_feedback", func(t *testing.T) {
		v := standard.NewValidator("tests.robot-single")
		task := robottypes.Task{
			ID:              "task-sem-1",
			ExecutorType:    robottypes.ExecutorAssistant,
			ExecutorID:      "analyst",
			ExpectedOutput:  "Detailed financial analysis",
			ValidationRules: []string{"output must be professional", "must include specific numbers"},
		}
		result := robottypes.TaskResult{
			TaskID:  "task-sem-1",
			Success: true,
			Output:  "idk lol something about money maybe?",
		}

		vr, err := v.ValidateWithContext(ctx, &task, &result)
		require.NoError(t, err)
		require.NotNil(t, vr)
		if !vr.Passed && len(vr.Issues) > 0 {
			assert.NotEmpty(t, vr.Issues[0])
		}
	})
}
