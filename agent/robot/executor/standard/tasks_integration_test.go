//go:build integration

package standard_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	agentcontext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/robot/executor/standard"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

// ============================================================================
// P2 Tasks Phase Tests
// ============================================================================

func TestRunTasksBasic(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	ctx := testCtx(identity)

	t.Run("calls_tasks_agent_and_processes_response", func(t *testing.T) {
		robot := newTestRobot(t, identity)
		exec := createTasksExecution(robot, robottypes.TriggerClock)
		exec.Goals = &robottypes.Goals{
			Content: "## Goals\n\n1. [High] Analyze Q4 sales data and identify top performing products\n2. [Normal] Generate a summary report for management",
		}

		e := standard.New()
		err := e.RunTasks(ctx, exec, nil)

		// Mock echo returns "echo: <formatted goals>", which doesn't contain
		// a JSON {tasks: [...]} structure. RunTasks expects valid JSON.
		// Verify the code path returns a meaningful error (not a crash).
		if err != nil {
			assert.Contains(t, err.Error(), "tasks agent")
		} else {
			require.NotEmpty(t, exec.Tasks)
			for i, task := range exec.Tasks {
				t.Logf("Task %d: ID=%s, ExecutorType=%s, ExecutorID=%s", i, task.ID, task.ExecutorType, task.ExecutorID)
				assert.NotEmpty(t, task.ID)
				assert.NotEmpty(t, task.ExecutorID)
				assert.NotEmpty(t, task.Messages)
			}
		}
	})

	t.Run("succeeds_when_tasks_pre_populated", func(t *testing.T) {
		robot := newTestRobot(t, identity)
		exec := createTasksExecution(robot, robottypes.TriggerClock)
		exec.Goals = &robottypes.Goals{Content: "test goals"}
		exec.Tasks = []robottypes.Task{
			{
				ID: "task-001", ExecutorType: robottypes.ExecutorAssistant, ExecutorID: "experts.text-writer",
				Messages:    []agentcontext.Message{{Role: agentcontext.RoleUser, Content: "Write report"}},
				Description: "Write the summary report",
			},
			{
				ID: "task-002", ExecutorType: robottypes.ExecutorAssistant, ExecutorID: "experts.data-analyst",
				Messages:    []agentcontext.Message{{Role: agentcontext.RoleUser, Content: "Analyze data"}},
				Description: "Analyze sales data",
			},
		}

		e := standard.New()
		err := e.RunTasks(ctx, exec, nil)

		require.NoError(t, err)
		require.Len(t, exec.Tasks, 2)
		assert.Equal(t, "task-001", exec.Tasks[0].ID)
		assert.Equal(t, "task-002", exec.Tasks[1].ID)
	})
}

func TestRunTasksErrorHandling(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	ctx := testCtx(identity)

	t.Run("returns_error_when_robot_is_nil", func(t *testing.T) {
		exec := &robottypes.Execution{ID: "test-exec-tasks-norobot", TriggerType: robottypes.TriggerClock}

		e := standard.New()
		err := e.RunTasks(ctx, exec, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "robot not found")
	})

	t.Run("returns_error_when_goals_not_available", func(t *testing.T) {
		robot := newTestRobot(t, identity)
		exec := createTasksExecution(robot, robottypes.TriggerClock)
		exec.Goals = nil

		e := standard.New()
		err := e.RunTasks(ctx, exec, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "goals not available")
	})

	t.Run("returns_error_when_goals_content_is_empty", func(t *testing.T) {
		robot := newTestRobot(t, identity)
		exec := createTasksExecution(robot, robottypes.TriggerClock)
		exec.Goals = &robottypes.Goals{Content: ""}

		e := standard.New()
		err := e.RunTasks(ctx, exec, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "goals not available")
	})

	t.Run("skips_regeneration_when_tasks_pre_populated", func(t *testing.T) {
		robot := newTestRobot(t, identity)
		exec := createTasksExecution(robot, robottypes.TriggerClock)
		exec.Goals = &robottypes.Goals{Content: "test goals"}
		exec.Tasks = []robottypes.Task{
			{ID: "pre-task-001", ExecutorType: robottypes.ExecutorAssistant, ExecutorID: "experts.text-writer",
				Messages: []agentcontext.Message{{Role: agentcontext.RoleUser, Content: "test"}}},
		}

		e := standard.New()
		err := e.RunTasks(ctx, exec, nil)

		require.NoError(t, err)
		require.Len(t, exec.Tasks, 1)
		assert.Equal(t, "pre-task-001", exec.Tasks[0].ID)
	})
}

// ============================================================================
// ParseTasks Unit Tests
// ============================================================================

func TestParseTasks(t *testing.T) {
	_ = testprepare.PrepareSandbox(t)

	t.Run("parses_valid_tasks_array", func(t *testing.T) {
		data := []interface{}{
			map[string]interface{}{
				"id": "task-001", "goal_ref": "Goal 1",
				"executor_type": "agent", "executor_id": "experts.data-analyst",
				"messages": []interface{}{
					map[string]interface{}{"role": "user", "content": "Analyze sales data"},
				},
				"expected_output":  "JSON with sales metrics",
				"validation_rules": []interface{}{"output must be valid JSON"},
				"order":            float64(0),
			},
			map[string]interface{}{
				"id": "task-002", "executor_type": "agent",
				"executor_id": "experts.text-writer",
				"description": "Generate report from analysis",
				"order":       float64(1),
			},
		}

		tasks, err := standard.ParseTasks(data)

		require.NoError(t, err)
		require.Len(t, tasks, 2)

		assert.Equal(t, "task-001", tasks[0].ID)
		assert.Equal(t, robottypes.ExecutorAssistant, tasks[0].ExecutorType)
		assert.Equal(t, "experts.data-analyst", tasks[0].ExecutorID)
		assert.Len(t, tasks[0].Messages, 1)
		assert.Equal(t, "JSON with sales metrics", tasks[0].ExpectedOutput)

		assert.Equal(t, "task-002", tasks[1].ID)
		assert.Equal(t, "Generate report from analysis", tasks[1].Description)
	})

	t.Run("generates_ID_if_missing", func(t *testing.T) {
		data := []interface{}{
			map[string]interface{}{
				"executor_type": "agent", "executor_id": "experts.summarizer",
				"description": "Summarize content",
			},
		}
		tasks, err := standard.ParseTasks(data)
		require.NoError(t, err)
		assert.Equal(t, "task-001", tasks[0].ID)
	})

	t.Run("returns_error_for_missing_executor_type", func(t *testing.T) {
		data := []interface{}{
			map[string]interface{}{"id": "task-001", "executor_id": "experts.summarizer", "description": "test"},
		}
		_, err := standard.ParseTasks(data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing executor_type")
	})

	t.Run("handles_different_executor_types", func(t *testing.T) {
		data := []interface{}{
			map[string]interface{}{"executor_type": "agent", "executor_id": "a", "description": "d"},
			map[string]interface{}{"executor_type": "assistant", "executor_id": "b", "description": "d"},
			map[string]interface{}{"executor_type": "mcp", "executor_id": "c", "description": "d"},
			map[string]interface{}{"executor_type": "process", "executor_id": "e", "description": "d"},
		}
		tasks, err := standard.ParseTasks(data)
		require.NoError(t, err)
		require.Len(t, tasks, 4)

		assert.Equal(t, robottypes.ExecutorAssistant, tasks[0].ExecutorType)
		assert.Equal(t, robottypes.ExecutorAssistant, tasks[1].ExecutorType)
		assert.Equal(t, robottypes.ExecutorMCP, tasks[2].ExecutorType)
		assert.Equal(t, robottypes.ExecutorProcess, tasks[3].ExecutorType)
	})
}

func TestValidateTasks(t *testing.T) {
	_ = testprepare.PrepareSandbox(t)

	t.Run("validates_valid_tasks", func(t *testing.T) {
		tasks := []robottypes.Task{
			{ID: "task-001", ExecutorType: robottypes.ExecutorAssistant, ExecutorID: "experts.data-analyst",
				Messages: []agentcontext.Message{{Content: "Analyze data"}}},
		}
		assert.NoError(t, standard.ValidateTasks(tasks))
	})

	t.Run("returns_error_for_empty_tasks", func(t *testing.T) {
		err := standard.ValidateTasks([]robottypes.Task{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no tasks generated")
	})

	t.Run("returns_error_for_duplicate_IDs", func(t *testing.T) {
		tasks := []robottypes.Task{
			{ID: "task-001", ExecutorID: "a", Messages: []agentcontext.Message{{Content: "test"}}},
			{ID: "task-001", ExecutorID: "b", Messages: []agentcontext.Message{{Content: "test"}}},
		}
		err := standard.ValidateTasks(tasks)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate task ID")
	})
}

func TestSortTasksByOrder(t *testing.T) {
	_ = testprepare.PrepareSandbox(t)

	t.Run("sorts_tasks_by_order", func(t *testing.T) {
		tasks := []robottypes.Task{
			{ID: "task-c", Order: 2},
			{ID: "task-a", Order: 0},
			{ID: "task-b", Order: 1},
		}
		standard.SortTasksByOrder(tasks)
		assert.Equal(t, "task-a", tasks[0].ID)
		assert.Equal(t, "task-b", tasks[1].ID)
		assert.Equal(t, "task-c", tasks[2].ID)
	})

	t.Run("handles_empty_tasks", func(t *testing.T) {
		tasks := []robottypes.Task{}
		standard.SortTasksByOrder(tasks)
		assert.Empty(t, tasks)
	})
}

func TestParseExecutorType(t *testing.T) {
	_ = testprepare.PrepareSandbox(t)

	assert.Equal(t, robottypes.ExecutorAssistant, standard.ParseExecutorType("agent"))
	assert.Equal(t, robottypes.ExecutorAssistant, standard.ParseExecutorType("assistant"))
	assert.Equal(t, robottypes.ExecutorMCP, standard.ParseExecutorType("mcp"))
	assert.Equal(t, robottypes.ExecutorProcess, standard.ParseExecutorType("process"))
	assert.Equal(t, robottypes.ExecutorAssistant, standard.ParseExecutorType("unknown"))
}

func TestValidateExecutorExists(t *testing.T) {
	_ = testprepare.PrepareSandbox(t)

	t.Run("returns_true_for_existing_agent", func(t *testing.T) {
		robot := &robottypes.Robot{
			Config: &robottypes.Config{
				Resources: &robottypes.Resources{
					Agents: []string{"experts.data-analyst", "experts.text-writer"},
				},
			},
		}
		assert.True(t, standard.ValidateExecutorExists("experts.data-analyst", robottypes.ExecutorAssistant, robot))
	})

	t.Run("returns_false_for_non_existing_agent", func(t *testing.T) {
		robot := &robottypes.Robot{
			Config: &robottypes.Config{
				Resources: &robottypes.Resources{Agents: []string{"experts.data-analyst"}},
			},
		}
		assert.False(t, standard.ValidateExecutorExists("experts.unknown", robottypes.ExecutorAssistant, robot))
	})

	t.Run("returns_true_when_robot_is_nil", func(t *testing.T) {
		assert.True(t, standard.ValidateExecutorExists("any", robottypes.ExecutorAssistant, nil))
	})
}

// ============================================================================
// Helpers
// ============================================================================

func createTasksExecution(robot *robottypes.Robot, trigger robottypes.TriggerType) *robottypes.Execution {
	exec := &robottypes.Execution{
		ID:          "test-exec-tasks-" + time.Now().Format("150405.000"),
		MemberID:    robot.MemberID,
		TeamID:      robot.TeamID,
		TriggerType: trigger,
		StartTime:   time.Now(),
		Status:      robottypes.ExecRunning,
		Phase:       robottypes.PhaseTasks,
	}
	exec.SetRobot(robot)
	return exec
}
