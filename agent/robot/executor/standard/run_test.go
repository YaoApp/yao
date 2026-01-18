package standard_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	agentcontext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/robot/executor/standard"
	"github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/testutils"
)

// ============================================================================
// P3 Run Phase Tests - RunExecution
// ============================================================================

func TestRunExecutionBasic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("executes single task successfully", func(t *testing.T) {
		robot := createRunTestRobot(t)
		exec := createRunTestExecution(robot)

		// Pre-built task (simulating P2 output)
		exec.Tasks = []types.Task{
			{
				ID:           "task-001",
				ExecutorType: types.ExecutorAssistant,
				ExecutorID:   "experts.text-writer",
				Messages: []agentcontext.Message{
					{Role: agentcontext.RoleUser, Content: "Write a short greeting message for a company newsletter. Keep it under 50 words."},
				},
				ExpectedOutput: "A friendly greeting message suitable for a newsletter",
				Order:          0,
				Status:         types.TaskPending,
			},
		}

		e := standard.New()
		err := e.RunExecution(ctx, exec, nil)

		require.NoError(t, err)
		require.Len(t, exec.Results, 1)

		result := exec.Results[0]
		assert.Equal(t, "task-001", result.TaskID)
		assert.True(t, result.Success, "task should succeed")
		assert.NotNil(t, result.Output, "should have output")
		assert.Greater(t, result.Duration, int64(0), "should have duration")

		// Task status should be updated
		assert.Equal(t, types.TaskCompleted, exec.Tasks[0].Status)
		assert.NotNil(t, exec.Tasks[0].StartTime)
		assert.NotNil(t, exec.Tasks[0].EndTime)
	})

	t.Run("executes multiple tasks in order", func(t *testing.T) {
		robot := createRunTestRobot(t)
		exec := createRunTestExecution(robot)

		// Multiple tasks that depend on each other
		exec.Tasks = []types.Task{
			{
				ID:           "task-001",
				ExecutorType: types.ExecutorAssistant,
				ExecutorID:   "experts.data-analyst",
				Messages: []agentcontext.Message{
					{Role: agentcontext.RoleUser, Content: "Analyze this data: Sales Q1: $100K, Q2: $150K, Q3: $120K, Q4: $180K. Calculate the total and average."},
				},
				ExpectedOutput: "JSON with total and average sales figures",
				ValidationRules: []string{
					"output must be valid JSON",
				},
				Order:  0,
				Status: types.TaskPending,
			},
			{
				ID:           "task-002",
				ExecutorType: types.ExecutorAssistant,
				ExecutorID:   "experts.summarizer",
				Messages: []agentcontext.Message{
					{Role: agentcontext.RoleUser, Content: "Summarize the key findings from the previous analysis in 2-3 sentences."},
				},
				ExpectedOutput: "A brief summary of the sales analysis",
				Order:          1,
				Status:         types.TaskPending,
			},
		}

		e := standard.New()
		err := e.RunExecution(ctx, exec, nil)

		require.NoError(t, err)
		require.Len(t, exec.Results, 2)

		// Both tasks should complete
		assert.True(t, exec.Results[0].Success, "first task should succeed")
		assert.True(t, exec.Results[1].Success, "second task should succeed")

		// Second task should have access to first task's result (via context)
		assert.Equal(t, types.TaskCompleted, exec.Tasks[0].Status)
		assert.Equal(t, types.TaskCompleted, exec.Tasks[1].Status)

		t.Logf("Task 1 output: %v", exec.Results[0].Output)
		t.Logf("Task 2 output: %v", exec.Results[1].Output)
	})

	t.Run("passes previous results as context to subsequent tasks", func(t *testing.T) {
		robot := createRunTestRobot(t)
		exec := createRunTestExecution(robot)

		// First task generates data, second task uses it
		exec.Tasks = []types.Task{
			{
				ID:           "task-001",
				ExecutorType: types.ExecutorAssistant,
				ExecutorID:   "experts.text-writer",
				Messages: []agentcontext.Message{
					{Role: agentcontext.RoleUser, Content: "Generate a list of 3 product names for a tech company. Output as JSON array."},
				},
				ExpectedOutput: "JSON array with 3 product names",
				Order:          0,
				Status:         types.TaskPending,
			},
			{
				ID:           "task-002",
				ExecutorType: types.ExecutorAssistant,
				ExecutorID:   "experts.text-writer",
				Messages: []agentcontext.Message{
					{Role: agentcontext.RoleUser, Content: "Using the product names from the previous task, write a one-line tagline for each product."},
				},
				ExpectedOutput: "Taglines for each product",
				Order:          1,
				Status:         types.TaskPending,
			},
		}

		e := standard.New()
		err := e.RunExecution(ctx, exec, nil)

		require.NoError(t, err)
		require.Len(t, exec.Results, 2)

		// Both should succeed
		assert.True(t, exec.Results[0].Success)
		assert.True(t, exec.Results[1].Success)

		// Second task output should reference products from first task
		t.Logf("Task 1 (products): %v", exec.Results[0].Output)
		t.Logf("Task 2 (taglines): %v", exec.Results[1].Output)
	})
}

func TestRunExecutionTaskStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("updates task status during execution", func(t *testing.T) {
		robot := createRunTestRobot(t)
		exec := createRunTestExecution(robot)

		exec.Tasks = []types.Task{
			{
				ID:           "task-001",
				ExecutorType: types.ExecutorAssistant,
				ExecutorID:   "experts.text-writer",
				Messages: []agentcontext.Message{
					{Role: agentcontext.RoleUser, Content: "Say 'Hello World'"},
				},
				Order:  0,
				Status: types.TaskPending,
			},
		}

		// Verify initial status
		assert.Equal(t, types.TaskPending, exec.Tasks[0].Status)

		e := standard.New()
		err := e.RunExecution(ctx, exec, nil)

		require.NoError(t, err)

		// Verify final status
		assert.Equal(t, types.TaskCompleted, exec.Tasks[0].Status)
		assert.NotNil(t, exec.Tasks[0].StartTime)
		assert.NotNil(t, exec.Tasks[0].EndTime)
	})

	t.Run("marks remaining tasks as skipped on failure", func(t *testing.T) {
		robot := createRunTestRobot(t)
		exec := createRunTestExecution(robot)

		// First task uses a non-existent assistant to guarantee failure
		exec.Tasks = []types.Task{
			{
				ID:           "task-001",
				ExecutorType: types.ExecutorAssistant,
				ExecutorID:   "non.existent.assistant.xyz123", // Non-existent assistant
				Messages: []agentcontext.Message{
					{Role: agentcontext.RoleUser, Content: "This will fail"},
				},
				Order:  0,
				Status: types.TaskPending,
			},
			{
				ID:           "task-002",
				ExecutorType: types.ExecutorAssistant,
				ExecutorID:   "experts.text-writer",
				Messages: []agentcontext.Message{
					{Role: agentcontext.RoleUser, Content: "Write another greeting"},
				},
				Order:  1,
				Status: types.TaskPending,
			},
			{
				ID:           "task-003",
				ExecutorType: types.ExecutorAssistant,
				ExecutorID:   "experts.text-writer",
				Messages: []agentcontext.Message{
					{Role: agentcontext.RoleUser, Content: "Write yet another greeting"},
				},
				Order:  2,
				Status: types.TaskPending,
			},
		}

		e := standard.New()
		err := e.RunExecution(ctx, exec, nil)

		// Should return error because first task failed
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "task-001")

		// First task should be failed
		assert.Equal(t, types.TaskFailed, exec.Tasks[0].Status)

		// Remaining tasks should be skipped
		assert.Equal(t, types.TaskSkipped, exec.Tasks[1].Status)
		assert.Equal(t, types.TaskSkipped, exec.Tasks[2].Status)
	})
}

func TestRunExecutionErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("returns error when robot is nil", func(t *testing.T) {
		exec := &types.Execution{
			ID:          "test-exec-1",
			TriggerType: types.TriggerClock,
			Tasks: []types.Task{
				{ID: "task-001", ExecutorID: "test"},
			},
		}
		// Don't set robot

		e := standard.New()
		err := e.RunExecution(ctx, exec, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "robot not found")
	})

	t.Run("returns error when no tasks", func(t *testing.T) {
		robot := createRunTestRobot(t)
		exec := createRunTestExecution(robot)
		exec.Tasks = []types.Task{} // Empty

		e := standard.New()
		err := e.RunExecution(ctx, exec, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no tasks")
	})

	t.Run("returns error for non-existent assistant", func(t *testing.T) {
		robot := createRunTestRobot(t)
		exec := createRunTestExecution(robot)

		exec.Tasks = []types.Task{
			{
				ID:           "task-001",
				ExecutorType: types.ExecutorAssistant,
				ExecutorID:   "non.existent.agent",
				Messages: []agentcontext.Message{
					{Role: agentcontext.RoleUser, Content: "Test"},
				},
				Order:  0,
				Status: types.TaskPending,
			},
		}

		e := standard.New()
		err := e.RunExecution(ctx, exec, nil)

		assert.Error(t, err)
		// Task should be marked as failed
		assert.Equal(t, types.TaskFailed, exec.Tasks[0].Status)
	})
}

func TestRunExecutionContinueOnFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("stops on first failure when ContinueOnFailure is false", func(t *testing.T) {
		robot := createRunTestRobot(t)
		exec := createRunTestExecution(robot)

		// First task will fail (non-existent assistant), second should be skipped
		exec.Tasks = []types.Task{
			{
				ID:           "task-001",
				ExecutorType: types.ExecutorAssistant,
				ExecutorID:   "non.existent.assistant.xyz123",
				Messages: []agentcontext.Message{
					{Role: agentcontext.RoleUser, Content: "This will fail"},
				},
				Order:  0,
				Status: types.TaskPending,
			},
			{
				ID:           "task-002",
				ExecutorType: types.ExecutorAssistant,
				ExecutorID:   "experts.text-writer",
				Messages: []agentcontext.Message{
					{Role: agentcontext.RoleUser, Content: "Write a greeting"},
				},
				Order:  1,
				Status: types.TaskPending,
			},
		}

		// Use default config (ContinueOnFailure = false)
		config := standard.DefaultRunConfig()
		assert.False(t, config.ContinueOnFailure)

		e := standard.New()
		err := e.RunExecution(ctx, exec, config)

		// Should return error
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "task-001")

		// Only first task should have a result
		assert.Len(t, exec.Results, 1)

		// First task failed
		assert.Equal(t, types.TaskFailed, exec.Tasks[0].Status)

		// Second task should be skipped (not executed)
		assert.Equal(t, types.TaskSkipped, exec.Tasks[1].Status)
	})

	t.Run("continues execution when ContinueOnFailure is true", func(t *testing.T) {
		robot := createRunTestRobot(t)
		exec := createRunTestExecution(robot)

		// First task will fail, but second should still execute
		exec.Tasks = []types.Task{
			{
				ID:           "task-001",
				ExecutorType: types.ExecutorAssistant,
				ExecutorID:   "non.existent.assistant.xyz123",
				Messages: []agentcontext.Message{
					{Role: agentcontext.RoleUser, Content: "This will fail"},
				},
				Order:  0,
				Status: types.TaskPending,
			},
			{
				ID:           "task-002",
				ExecutorType: types.ExecutorAssistant,
				ExecutorID:   "experts.text-writer",
				Messages: []agentcontext.Message{
					{Role: agentcontext.RoleUser, Content: "Write a short greeting message"},
				},
				ExpectedOutput: "A greeting message",
				Order:          1,
				Status:         types.TaskPending,
			},
			{
				ID:           "task-003",
				ExecutorType: types.ExecutorAssistant,
				ExecutorID:   "experts.text-writer",
				Messages: []agentcontext.Message{
					{Role: agentcontext.RoleUser, Content: "Write a farewell message"},
				},
				ExpectedOutput: "A farewell message",
				Order:          2,
				Status:         types.TaskPending,
			},
		}

		// Enable ContinueOnFailure
		config := standard.DefaultRunConfig()
		config.ContinueOnFailure = true

		e := standard.New()
		err := e.RunExecution(ctx, exec, config)

		// Should NOT return error when ContinueOnFailure is true
		assert.NoError(t, err)

		// All tasks should have results
		assert.Len(t, exec.Results, 3)

		// First task failed
		assert.Equal(t, types.TaskFailed, exec.Tasks[0].Status)
		assert.False(t, exec.Results[0].Success)

		// Second and third tasks should have executed and completed
		assert.Equal(t, types.TaskCompleted, exec.Tasks[1].Status)
		assert.True(t, exec.Results[1].Success)

		assert.Equal(t, types.TaskCompleted, exec.Tasks[2].Status)
		assert.True(t, exec.Results[2].Success)

		t.Logf("Task 1 (failed): %v", exec.Results[0].Error)
		t.Logf("Task 2 (success): %v", exec.Results[1].Output)
		t.Logf("Task 3 (success): %v", exec.Results[2].Output)
	})

	t.Run("multiple failures with ContinueOnFailure", func(t *testing.T) {
		robot := createRunTestRobot(t)
		exec := createRunTestExecution(robot)

		// Mix of failing and succeeding tasks
		exec.Tasks = []types.Task{
			{
				ID:           "task-001",
				ExecutorType: types.ExecutorAssistant,
				ExecutorID:   "non.existent.assistant.1",
				Messages: []agentcontext.Message{
					{Role: agentcontext.RoleUser, Content: "Fail 1"},
				},
				Order:  0,
				Status: types.TaskPending,
			},
			{
				ID:           "task-002",
				ExecutorType: types.ExecutorAssistant,
				ExecutorID:   "experts.text-writer",
				Messages: []agentcontext.Message{
					{Role: agentcontext.RoleUser, Content: "Say hello"},
				},
				Order:  1,
				Status: types.TaskPending,
			},
			{
				ID:           "task-003",
				ExecutorType: types.ExecutorAssistant,
				ExecutorID:   "non.existent.assistant.2",
				Messages: []agentcontext.Message{
					{Role: agentcontext.RoleUser, Content: "Fail 2"},
				},
				Order:  2,
				Status: types.TaskPending,
			},
			{
				ID:           "task-004",
				ExecutorType: types.ExecutorAssistant,
				ExecutorID:   "experts.text-writer",
				Messages: []agentcontext.Message{
					{Role: agentcontext.RoleUser, Content: "Say goodbye"},
				},
				Order:  3,
				Status: types.TaskPending,
			},
		}

		config := standard.DefaultRunConfig()
		config.ContinueOnFailure = true

		e := standard.New()
		err := e.RunExecution(ctx, exec, config)

		assert.NoError(t, err)
		assert.Len(t, exec.Results, 4)

		// Check status pattern: fail, success, fail, success
		assert.Equal(t, types.TaskFailed, exec.Tasks[0].Status)
		assert.Equal(t, types.TaskCompleted, exec.Tasks[1].Status)
		assert.Equal(t, types.TaskFailed, exec.Tasks[2].Status)
		assert.Equal(t, types.TaskCompleted, exec.Tasks[3].Status)

		// Count successes and failures
		successCount := 0
		failCount := 0
		for _, result := range exec.Results {
			if result.Success {
				successCount++
			} else {
				failCount++
			}
		}
		assert.Equal(t, 2, successCount)
		assert.Equal(t, 2, failCount)
	})
}

func TestRunExecutionValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("validates output with rule-based validation", func(t *testing.T) {
		robot := createRunTestRobot(t)
		exec := createRunTestExecution(robot)

		exec.Tasks = []types.Task{
			{
				ID:           "task-001",
				ExecutorType: types.ExecutorAssistant,
				ExecutorID:   "experts.data-analyst",
				Messages: []agentcontext.Message{
					{Role: agentcontext.RoleUser, Content: "Return a JSON object with fields: name (string), count (number). Example: {\"name\": \"test\", \"count\": 5}"},
				},
				ExpectedOutput: "JSON object with name and count fields",
				ValidationRules: []string{
					"output must be valid JSON",
					`{"type": "type", "value": "object"}`,
				},
				Order:  0,
				Status: types.TaskPending,
			},
		}

		e := standard.New()
		err := e.RunExecution(ctx, exec, nil)

		require.NoError(t, err)
		require.Len(t, exec.Results, 1)

		result := exec.Results[0]
		assert.True(t, result.Success)
		assert.NotNil(t, result.Validation)
		assert.True(t, result.Validation.Passed)
	})

	t.Run("validates output with semantic validation", func(t *testing.T) {
		robot := createRunTestRobot(t)
		exec := createRunTestExecution(robot)

		exec.Tasks = []types.Task{
			{
				ID:           "task-001",
				ExecutorType: types.ExecutorAssistant,
				ExecutorID:   "experts.text-writer",
				Messages: []agentcontext.Message{
					{Role: agentcontext.RoleUser, Content: "Write a professional email greeting for a business context. Start with 'Dear' and end with a comma."},
				},
				ExpectedOutput: "A professional email greeting starting with 'Dear'",
				Order:          0,
				Status:         types.TaskPending,
			},
		}

		e := standard.New()
		err := e.RunExecution(ctx, exec, nil)

		require.NoError(t, err)
		require.Len(t, exec.Results, 1)

		result := exec.Results[0]
		assert.True(t, result.Success)
		assert.NotNil(t, result.Validation)

		t.Logf("Output: %v", result.Output)
		t.Logf("Validation: passed=%v, score=%.2f", result.Validation.Passed, result.Validation.Score)
	})
}

// ============================================================================
// Helper Functions
// ============================================================================

// createRunTestRobot creates a test robot for P3 run tests
func createRunTestRobot(t *testing.T) *types.Robot {
	t.Helper()
	return &types.Robot{
		MemberID:     "test-robot-run",
		TeamID:       "test-team-1",
		DisplayName:  "Test Robot for Run",
		SystemPrompt: "You are a helpful assistant.",
		Config: &types.Config{
			Identity: &types.Identity{
				Role:   "Test Assistant",
				Duties: []string{"Execute tasks", "Generate content"},
			},
			Resources: &types.Resources{
				Phases: map[types.Phase]string{
					types.PhaseRun: "robot.validation",
					"validation":   "robot.validation", // For semantic validation agent
				},
				Agents: []string{
					"experts.data-analyst",
					"experts.summarizer",
					"experts.text-writer",
				},
			},
		},
	}
}

// createRunTestExecution creates a test execution for P3 run tests
func createRunTestExecution(robot *types.Robot) *types.Execution {
	exec := &types.Execution{
		ID:          "test-exec-run-1",
		MemberID:    robot.MemberID,
		TeamID:      robot.TeamID,
		TriggerType: types.TriggerClock,
		StartTime:   time.Now(),
		Status:      types.ExecRunning,
		Phase:       types.PhaseRun,
		Goals: &types.Goals{
			Content: "## Goals\n\n1. Execute test tasks",
		},
	}
	exec.SetRobot(robot)
	return exec
}
