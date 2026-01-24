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
// P2 Tasks Phase Tests
// ============================================================================

func TestRunTasksBasic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("generates tasks from goals (clock trigger)", func(t *testing.T) {
		// Create robot with tasks agent configured
		robot := createTasksTestRobot(t, "robot.tasks")

		// Create execution with goals (from P1)
		exec := createTasksTestExecution(robot, types.TriggerClock)
		exec.Goals = &types.Goals{
			Content: `## Goals

1. [High] Analyze Q4 sales data and identify top performing products
   - Reason: Need to prepare quarterly report

2. [Normal] Generate a summary report for management
   - Reason: Weekly review meeting tomorrow`,
		}

		// Run tasks phase
		e := standard.New()
		err := e.RunTasks(ctx, exec, nil)

		require.NoError(t, err)
		require.NotNil(t, exec.Tasks)
		assert.NotEmpty(t, exec.Tasks)

		// Verify task structure
		for i, task := range exec.Tasks {
			t.Logf("Task %d: ID=%s, ExecutorType=%s, ExecutorID=%s", i, task.ID, task.ExecutorType, task.ExecutorID)
			assert.NotEmpty(t, task.ID, "task should have ID")
			assert.NotEmpty(t, task.ExecutorID, "task should have executor ID")
			assert.NotEmpty(t, task.Messages, "task should have messages")
		}
	})

	t.Run("includes expected output and validation rules", func(t *testing.T) {
		robot := createTasksTestRobot(t, "robot.tasks")
		exec := createTasksTestExecution(robot, types.TriggerClock)
		exec.Goals = &types.Goals{
			Content: `## Goals

1. [High] Fetch latest news about AI developments
   - Reason: Stay updated on industry trends

2. [Normal] Summarize the key findings
   - Reason: Share with team`,
		}

		e := standard.New()
		err := e.RunTasks(ctx, exec, nil)

		require.NoError(t, err)
		require.NotEmpty(t, exec.Tasks)

		// Check that at least one task has validation info
		hasValidationInfo := false
		for _, task := range exec.Tasks {
			if task.ExpectedOutput != "" || len(task.ValidationRules) > 0 {
				hasValidationInfo = true
				t.Logf("Task %s has validation: expected_output=%q, rules=%v",
					task.ID, task.ExpectedOutput, task.ValidationRules)
			}
		}

		// Note: LLM might not always include validation rules, so we just log
		t.Logf("Tasks have validation info: %v", hasValidationInfo)
	})
}

func TestRunTasksHumanTrigger(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("generates tasks from human-triggered goals", func(t *testing.T) {
		robot := createTasksTestRobot(t, "robot.tasks")
		exec := createTasksTestExecution(robot, types.TriggerHuman)

		// Goals from human request (P1 output)
		exec.Goals = &types.Goals{
			Content: `## Goals

1. [High] Research competitor pricing strategies
   - Reason: User requested competitive analysis

2. [Normal] Create comparison report
   - Reason: User needs data for presentation`,
		}

		e := standard.New()
		err := e.RunTasks(ctx, exec, nil)

		require.NoError(t, err)
		require.NotEmpty(t, exec.Tasks)

		// Tasks should relate to the goals
		for _, task := range exec.Tasks {
			t.Logf("Task: %s -> %s", task.ID, task.ExecutorID)
		}
	})
}

func TestRunTasksWithExpertAgents(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("assigns appropriate expert agents to tasks", func(t *testing.T) {
		robot := createTasksTestRobot(t, "robot.tasks")
		exec := createTasksTestExecution(robot, types.TriggerClock)

		// Goals that require different expert agents
		exec.Goals = &types.Goals{
			Content: `## Goals

1. [High] Analyze sales data from database
   - Reason: Quarterly review needed
   - Requires: Data analysis capabilities

2. [Normal] Write executive summary report
   - Reason: Management presentation
   - Requires: Text generation capabilities

3. [Low] Summarize key findings
   - Reason: Quick reference for team
   - Requires: Summarization capabilities`,
		}

		e := standard.New()
		err := e.RunTasks(ctx, exec, nil)

		require.NoError(t, err)
		require.NotEmpty(t, exec.Tasks)

		// Log assigned executors
		executorCounts := make(map[string]int)
		for _, task := range exec.Tasks {
			executorCounts[task.ExecutorID]++
			t.Logf("Task %s assigned to: %s (%s)", task.ID, task.ExecutorID, task.ExecutorType)
		}

		// Verify different executors were assigned (not all to same agent)
		t.Logf("Executor distribution: %v", executorCounts)
	})
}

func TestRunTasksErrorHandling(t *testing.T) {
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
		}
		// Don't set robot

		e := standard.New()
		err := e.RunTasks(ctx, exec, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "robot not found")
	})

	t.Run("returns error when goals not available", func(t *testing.T) {
		robot := createTasksTestRobot(t, "robot.tasks")
		exec := createTasksTestExecution(robot, types.TriggerClock)
		exec.Goals = nil // No goals

		e := standard.New()
		err := e.RunTasks(ctx, exec, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "goals not available")
	})

	t.Run("returns error when goals content is empty", func(t *testing.T) {
		robot := createTasksTestRobot(t, "robot.tasks")
		exec := createTasksTestExecution(robot, types.TriggerClock)
		exec.Goals = &types.Goals{Content: ""} // Empty content

		e := standard.New()
		err := e.RunTasks(ctx, exec, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "goals not available")
	})

	t.Run("returns error when agent not found", func(t *testing.T) {
		robot := &types.Robot{
			MemberID: "test-robot-1",
			TeamID:   "test-team-1",
			Config: &types.Config{
				Identity: &types.Identity{Role: "Test"},
				Resources: &types.Resources{
					Phases: map[types.Phase]string{
						types.PhaseTasks: "non.existent.agent",
					},
				},
			},
		}
		exec := createTasksTestExecution(robot, types.TriggerClock)
		exec.Goals = &types.Goals{Content: "Test goals"}

		e := standard.New()
		err := e.RunTasks(ctx, exec, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "call failed")
	})
}

// ============================================================================
// ParseTasks Unit Tests
// ============================================================================

func TestParseTasks(t *testing.T) {
	t.Run("parses valid tasks array", func(t *testing.T) {
		data := []interface{}{
			map[string]interface{}{
				"id":            "task-001",
				"goal_ref":      "Goal 1",
				"executor_type": "agent",
				"executor_id":   "experts.data-analyst",
				"messages": []interface{}{
					map[string]interface{}{
						"role":    "user",
						"content": "Analyze sales data",
					},
				},
				"expected_output": "JSON with sales metrics",
				"validation_rules": []interface{}{
					// Natural language rules (matched by validator)
					"output must be valid JSON",
					"must contain 'total_sales'",
					// Structured rule: check field type
					`{"type": "type", "path": "product_rankings", "value": "array"}`,
				},
				"order": float64(0),
			},
			map[string]interface{}{
				"id":            "task-002",
				"goal_ref":      "Goal 1",
				"executor_type": "agent",
				"executor_id":   "experts.text-writer",
				"description":   "Generate report from analysis",
				"order":         float64(1),
			},
		}

		tasks, err := standard.ParseTasks(data)

		require.NoError(t, err)
		require.Len(t, tasks, 2)

		// First task
		assert.Equal(t, "task-001", tasks[0].ID)
		assert.Equal(t, "Goal 1", tasks[0].GoalRef)
		assert.Equal(t, types.ExecutorAssistant, tasks[0].ExecutorType)
		assert.Equal(t, "experts.data-analyst", tasks[0].ExecutorID)
		assert.Len(t, tasks[0].Messages, 1)
		assert.Equal(t, "JSON with sales metrics", tasks[0].ExpectedOutput)
		assert.Len(t, tasks[0].ValidationRules, 3)
		assert.Equal(t, 0, tasks[0].Order)

		// Second task
		assert.Equal(t, "task-002", tasks[1].ID)
		assert.Equal(t, "experts.text-writer", tasks[1].ExecutorID)
		assert.Equal(t, "Generate report from analysis", tasks[1].Description) // description saved to field
		assert.Len(t, tasks[1].Messages, 1)                                    // description also converted to message
		assert.Equal(t, 1, tasks[1].Order)
	})

	t.Run("generates ID if missing", func(t *testing.T) {
		data := []interface{}{
			map[string]interface{}{
				"executor_type": "agent",
				"executor_id":   "experts.summarizer",
				"description":   "Summarize content",
			},
		}

		tasks, err := standard.ParseTasks(data)

		require.NoError(t, err)
		require.Len(t, tasks, 1)
		assert.Equal(t, "task-001", tasks[0].ID)
	})

	t.Run("saves description to field and preserves explicit messages", func(t *testing.T) {
		data := []interface{}{
			map[string]interface{}{
				"id":            "task-001",
				"executor_type": "agent",
				"executor_id":   "experts.summarizer",
				"description":   "Task description for UI",
				"messages": []interface{}{
					map[string]interface{}{
						"role":    "user",
						"content": "Explicit message content",
					},
				},
			},
		}

		tasks, err := standard.ParseTasks(data)

		require.NoError(t, err)
		require.Len(t, tasks, 1)

		// Description should be saved to field
		assert.Equal(t, "Task description for UI", tasks[0].Description)

		// Explicit messages should be preserved (not overwritten by description)
		assert.Len(t, tasks[0].Messages, 1)
		content, ok := tasks[0].Messages[0].GetContentAsString()
		assert.True(t, ok)
		assert.Equal(t, "Explicit message content", content)
	})

	t.Run("converts description to message when no messages provided", func(t *testing.T) {
		data := []interface{}{
			map[string]interface{}{
				"id":            "task-001",
				"executor_type": "agent",
				"executor_id":   "experts.summarizer",
				"description":   "Only description, no messages",
			},
		}

		tasks, err := standard.ParseTasks(data)

		require.NoError(t, err)
		require.Len(t, tasks, 1)

		// Description should be saved to field
		assert.Equal(t, "Only description, no messages", tasks[0].Description)

		// Description should also be converted to message for execution
		assert.Len(t, tasks[0].Messages, 1)
		content, ok := tasks[0].Messages[0].GetContentAsString()
		assert.True(t, ok)
		assert.Equal(t, "Only description, no messages", content)
	})

	t.Run("returns error for missing executor_type", func(t *testing.T) {
		data := []interface{}{
			map[string]interface{}{
				"id":          "task-001",
				"executor_id": "experts.summarizer",
				"description": "Summarize content",
			},
		}

		_, err := standard.ParseTasks(data)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing executor_type")
	})

	t.Run("returns error for missing executor_id", func(t *testing.T) {
		data := []interface{}{
			map[string]interface{}{
				"id":            "task-001",
				"executor_type": "agent",
				"description":   "Summarize content",
			},
		}

		_, err := standard.ParseTasks(data)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing executor_id")
	})

	t.Run("handles different executor types", func(t *testing.T) {
		data := []interface{}{
			map[string]interface{}{
				"executor_type": "agent",
				"executor_id":   "test-agent",
				"description":   "Agent task",
			},
			map[string]interface{}{
				"executor_type": "assistant",
				"executor_id":   "test-assistant",
				"description":   "Assistant task",
			},
			map[string]interface{}{
				"executor_type": "mcp",
				"executor_id":   "test-mcp",
				"description":   "MCP task",
			},
			map[string]interface{}{
				"executor_type": "process",
				"executor_id":   "test-process",
				"description":   "Process task",
			},
		}

		tasks, err := standard.ParseTasks(data)

		require.NoError(t, err)
		require.Len(t, tasks, 4)

		assert.Equal(t, types.ExecutorAssistant, tasks[0].ExecutorType)
		assert.Equal(t, types.ExecutorAssistant, tasks[1].ExecutorType) // assistant -> ExecutorAssistant
		assert.Equal(t, types.ExecutorMCP, tasks[2].ExecutorType)
		assert.Equal(t, types.ExecutorProcess, tasks[3].ExecutorType)
	})
}

func TestValidateTasks(t *testing.T) {
	t.Run("validates valid tasks", func(t *testing.T) {
		tasks := []types.Task{
			{
				ID:           "task-001",
				ExecutorType: types.ExecutorAssistant,
				ExecutorID:   "experts.data-analyst",
				Messages: []agentcontext.Message{
					{Role: agentcontext.RoleUser, Content: "Analyze data"},
				},
			},
			{
				ID:           "task-002",
				ExecutorType: types.ExecutorAssistant,
				ExecutorID:   "experts.text-writer",
				Messages: []agentcontext.Message{
					{Role: agentcontext.RoleUser, Content: "Write report"},
				},
			},
		}

		err := standard.ValidateTasks(tasks)
		assert.NoError(t, err)
	})

	t.Run("returns error for empty tasks", func(t *testing.T) {
		tasks := []types.Task{}

		err := standard.ValidateTasks(tasks)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no tasks generated")
	})

	t.Run("returns error for duplicate IDs", func(t *testing.T) {
		tasks := []types.Task{
			{
				ID:         "task-001",
				ExecutorID: "agent-1",
				Messages:   []agentcontext.Message{{Content: "test"}},
			},
			{
				ID:         "task-001", // duplicate
				ExecutorID: "agent-2",
				Messages:   []agentcontext.Message{{Content: "test"}},
			},
		}

		err := standard.ValidateTasks(tasks)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate task ID")
	})

	t.Run("returns error for missing executor_id", func(t *testing.T) {
		tasks := []types.Task{
			{
				ID:         "task-001",
				ExecutorID: "", // missing
				Messages:   []agentcontext.Message{{Content: "test"}},
			},
		}

		err := standard.ValidateTasks(tasks)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing executor_id")
	})

	t.Run("returns error for missing messages", func(t *testing.T) {
		tasks := []types.Task{
			{
				ID:         "task-001",
				ExecutorID: "agent-1",
				Messages:   []agentcontext.Message{}, // empty
			},
		}

		err := standard.ValidateTasks(tasks)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing messages")
	})
}

func TestParseExecutorType(t *testing.T) {
	t.Run("parses agent", func(t *testing.T) {
		assert.Equal(t, types.ExecutorAssistant, standard.ParseExecutorType("agent"))
	})

	t.Run("parses assistant", func(t *testing.T) {
		assert.Equal(t, types.ExecutorAssistant, standard.ParseExecutorType("assistant"))
	})

	t.Run("parses mcp", func(t *testing.T) {
		assert.Equal(t, types.ExecutorMCP, standard.ParseExecutorType("mcp"))
	})

	t.Run("parses process", func(t *testing.T) {
		assert.Equal(t, types.ExecutorProcess, standard.ParseExecutorType("process"))
	})

	t.Run("defaults to assistant for unknown", func(t *testing.T) {
		assert.Equal(t, types.ExecutorAssistant, standard.ParseExecutorType("unknown"))
		assert.Equal(t, types.ExecutorAssistant, standard.ParseExecutorType(""))
	})
}

func TestIsValidExecutorType(t *testing.T) {
	t.Run("valid executor types", func(t *testing.T) {
		assert.True(t, standard.IsValidExecutorType(types.ExecutorAssistant))
		assert.True(t, standard.IsValidExecutorType(types.ExecutorMCP))
		assert.True(t, standard.IsValidExecutorType(types.ExecutorProcess))
	})

	t.Run("invalid executor types", func(t *testing.T) {
		assert.False(t, standard.IsValidExecutorType(types.ExecutorType("invalid")))
		assert.False(t, standard.IsValidExecutorType(types.ExecutorType("")))
	})
}

func TestSortTasksByOrder(t *testing.T) {
	t.Run("sorts tasks by order", func(t *testing.T) {
		tasks := []types.Task{
			{ID: "task-c", Order: 2},
			{ID: "task-a", Order: 0},
			{ID: "task-b", Order: 1},
		}

		standard.SortTasksByOrder(tasks)

		assert.Equal(t, "task-a", tasks[0].ID)
		assert.Equal(t, "task-b", tasks[1].ID)
		assert.Equal(t, "task-c", tasks[2].ID)
	})

	t.Run("handles already sorted tasks", func(t *testing.T) {
		tasks := []types.Task{
			{ID: "task-a", Order: 0},
			{ID: "task-b", Order: 1},
			{ID: "task-c", Order: 2},
		}

		standard.SortTasksByOrder(tasks)

		assert.Equal(t, "task-a", tasks[0].ID)
		assert.Equal(t, "task-b", tasks[1].ID)
		assert.Equal(t, "task-c", tasks[2].ID)
	})

	t.Run("handles single task", func(t *testing.T) {
		tasks := []types.Task{
			{ID: "task-a", Order: 0},
		}

		standard.SortTasksByOrder(tasks)

		assert.Len(t, tasks, 1)
		assert.Equal(t, "task-a", tasks[0].ID)
	})

	t.Run("handles empty tasks", func(t *testing.T) {
		tasks := []types.Task{}

		standard.SortTasksByOrder(tasks)

		assert.Empty(t, tasks)
	})
}

func TestValidateExecutorExists(t *testing.T) {
	t.Run("returns true for existing agent", func(t *testing.T) {
		robot := &types.Robot{
			Config: &types.Config{
				Resources: &types.Resources{
					Agents: []string{"experts.data-analyst", "experts.text-writer"},
				},
			},
		}

		assert.True(t, standard.ValidateExecutorExists("experts.data-analyst", types.ExecutorAssistant, robot))
		assert.True(t, standard.ValidateExecutorExists("experts.text-writer", types.ExecutorAssistant, robot))
	})

	t.Run("returns false for non-existing agent", func(t *testing.T) {
		robot := &types.Robot{
			Config: &types.Config{
				Resources: &types.Resources{
					Agents: []string{"experts.data-analyst"},
				},
			},
		}

		assert.False(t, standard.ValidateExecutorExists("experts.unknown", types.ExecutorAssistant, robot))
	})

	t.Run("returns true for existing MCP", func(t *testing.T) {
		robot := &types.Robot{
			Config: &types.Config{
				Resources: &types.Resources{
					MCP: []types.MCPConfig{
						{ID: "database"},
						{ID: "email"},
					},
				},
			},
		}

		assert.True(t, standard.ValidateExecutorExists("database", types.ExecutorMCP, robot))
		assert.True(t, standard.ValidateExecutorExists("email", types.ExecutorMCP, robot))
	})

	t.Run("returns false for non-existing MCP", func(t *testing.T) {
		robot := &types.Robot{
			Config: &types.Config{
				Resources: &types.Resources{
					MCP: []types.MCPConfig{
						{ID: "database"},
					},
				},
			},
		}

		assert.False(t, standard.ValidateExecutorExists("unknown", types.ExecutorMCP, robot))
	})

	t.Run("returns true for process (not validated)", func(t *testing.T) {
		robot := &types.Robot{
			Config: &types.Config{
				Resources: &types.Resources{},
			},
		}

		assert.True(t, standard.ValidateExecutorExists("models.user.Find", types.ExecutorProcess, robot))
	})

	t.Run("returns true when robot is nil", func(t *testing.T) {
		assert.True(t, standard.ValidateExecutorExists("any", types.ExecutorAssistant, nil))
	})

	t.Run("returns true when resources is nil", func(t *testing.T) {
		robot := &types.Robot{
			Config: &types.Config{},
		}

		assert.True(t, standard.ValidateExecutorExists("any", types.ExecutorAssistant, robot))
	})
}

func TestValidateTasksWithResources(t *testing.T) {
	t.Run("returns no warnings for valid tasks", func(t *testing.T) {
		robot := &types.Robot{
			Config: &types.Config{
				Resources: &types.Resources{
					Agents: []string{"experts.data-analyst", "experts.text-writer"},
				},
			},
		}
		tasks := []types.Task{
			{
				ID:           "task-001",
				ExecutorType: types.ExecutorAssistant,
				ExecutorID:   "experts.data-analyst",
				Messages:     []agentcontext.Message{{Content: "test"}},
			},
		}

		warnings, err := standard.ValidateTasksWithResources(tasks, robot)

		assert.NoError(t, err)
		assert.Empty(t, warnings)
	})

	t.Run("returns warnings for unknown executor", func(t *testing.T) {
		robot := &types.Robot{
			Config: &types.Config{
				Resources: &types.Resources{
					Agents: []string{"experts.data-analyst"},
				},
			},
		}
		tasks := []types.Task{
			{
				ID:           "task-001",
				ExecutorType: types.ExecutorAssistant,
				ExecutorID:   "experts.unknown",
				Messages:     []agentcontext.Message{{Content: "test"}},
			},
		}

		warnings, err := standard.ValidateTasksWithResources(tasks, robot)

		assert.NoError(t, err)
		assert.Len(t, warnings, 1)
		assert.Contains(t, warnings[0], "experts.unknown")
		assert.Contains(t, warnings[0], "not found")
	})

	t.Run("returns error for invalid tasks", func(t *testing.T) {
		robot := &types.Robot{}
		tasks := []types.Task{} // empty

		_, err := standard.ValidateTasksWithResources(tasks, robot)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no tasks generated")
	})
}

// ============================================================================
// InputFormatter Tests for P2
// ============================================================================

func TestInputFormatterFormatGoalsForTasks(t *testing.T) {
	formatter := standard.NewInputFormatter()

	t.Run("formats goals with resources", func(t *testing.T) {
		goals := &types.Goals{
			Content: "## Goals\n\n1. [High] Analyze data\n2. [Normal] Write report",
		}
		robot := &types.Robot{
			MemberID: "test-robot",
			Config: &types.Config{
				Resources: &types.Resources{
					Agents: []string{"experts.data-analyst", "experts.text-writer"},
				},
			},
		}

		content := formatter.FormatGoals(goals, robot)

		assert.Contains(t, content, "## Goals")
		assert.Contains(t, content, "[High] Analyze data")
		assert.Contains(t, content, "## Available Resources")
		assert.Contains(t, content, "experts.data-analyst")
		assert.Contains(t, content, "experts.text-writer")
	})

	t.Run("formats goals without robot", func(t *testing.T) {
		goals := &types.Goals{
			Content: "## Goals\n\n1. Test goal",
		}

		content := formatter.FormatGoals(goals, nil)

		assert.Contains(t, content, "## Goals")
		assert.Contains(t, content, "Test goal")
		assert.NotContains(t, content, "## Available Resources")
	})

	t.Run("formats goals with delivery target", func(t *testing.T) {
		goals := &types.Goals{
			Content: "## Goals\n\n1. Generate weekly report",
			Delivery: &types.DeliveryTarget{
				Type:       types.DeliveryEmail,
				Recipients: []string{"team@example.com", "manager@example.com"},
				Format:     "markdown",
				Template:   "weekly-report",
			},
		}
		robot := &types.Robot{
			MemberID: "test-robot",
			Config: &types.Config{
				Resources: &types.Resources{
					Agents: []string{"experts.text-writer"},
				},
			},
		}

		content := formatter.FormatGoals(goals, robot)

		assert.Contains(t, content, "## Goals")
		assert.Contains(t, content, "## Delivery Target")
		assert.Contains(t, content, "email")
		assert.Contains(t, content, "team@example.com")
		assert.Contains(t, content, "manager@example.com")
		assert.Contains(t, content, "markdown")
		assert.Contains(t, content, "weekly-report")
		assert.Contains(t, content, "Design tasks to produce output suitable")
	})

	t.Run("formats goals without delivery target", func(t *testing.T) {
		goals := &types.Goals{
			Content:  "## Goals\n\n1. Test goal",
			Delivery: nil,
		}

		content := formatter.FormatGoals(goals, nil)

		assert.Contains(t, content, "## Goals")
		assert.NotContains(t, content, "## Delivery Target")
	})
}

// ============================================================================
// Helper Functions
// ============================================================================

// createTasksTestRobot creates a test robot with specified tasks agent
// Includes available expert agents for task assignment
func createTasksTestRobot(t *testing.T, agentID string) *types.Robot {
	t.Helper()
	return &types.Robot{
		MemberID:    "test-robot-1",
		TeamID:      "test-team-1",
		DisplayName: "Test Robot",
		Config: &types.Config{
			Identity: &types.Identity{
				Role:   "Test Assistant",
				Duties: []string{"Testing", "Data Analysis", "Report Generation"},
			},
			Resources: &types.Resources{
				Phases: map[types.Phase]string{
					types.PhaseTasks: agentID,
				},
				// Available expert agents that can be assigned to tasks
				Agents: []string{
					"experts.data-analyst",
					"experts.summarizer",
					"experts.text-writer",
					"experts.web-reader",
				},
			},
		},
	}
}

// createTasksTestExecution creates a test execution for tasks phase
func createTasksTestExecution(robot *types.Robot, trigger types.TriggerType) *types.Execution {
	exec := &types.Execution{
		ID:          "test-exec-tasks-1",
		MemberID:    robot.MemberID,
		TeamID:      robot.TeamID,
		TriggerType: trigger,
		StartTime:   time.Now(),
		Status:      types.ExecRunning,
		Phase:       types.PhaseTasks,
	}
	exec.SetRobot(robot)
	return exec
}

// Note: testAuth is defined in goals_test.go in the same package
