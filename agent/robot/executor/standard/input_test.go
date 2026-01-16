package standard_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	agentcontext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/robot/executor/standard"
	"github.com/yaoapp/yao/agent/robot/types"
)

// ============================================================================
// InputFormatter Tests
// ============================================================================

func TestInputFormatterFormatClockContext(t *testing.T) {
	formatter := standard.NewInputFormatter()

	t.Run("formats clock context with all fields", func(t *testing.T) {
		now := time.Date(2024, 1, 15, 9, 30, 0, 0, time.UTC)
		clock := types.NewClockContext(now, "UTC")

		result := formatter.FormatClockContext(clock, nil)

		assert.Contains(t, result, "## Current Time Context")
		assert.Contains(t, result, "2024-01-15 09:30:00")
		assert.Contains(t, result, "Monday")
		assert.Contains(t, result, "UTC")
		assert.Contains(t, result, "### Time Markers")
	})

	t.Run("includes robot identity when provided", func(t *testing.T) {
		now := time.Now()
		clock := types.NewClockContext(now, "UTC")
		robot := &types.Robot{
			MemberID: "test-robot",
			Config: &types.Config{
				Identity: &types.Identity{
					Role:   "Sales Analyst",
					Duties: []string{"Analyze sales data", "Generate reports"},
					Rules:  []string{"Be accurate", "Be concise"},
				},
			},
		}

		result := formatter.FormatClockContext(clock, robot)

		assert.Contains(t, result, "## Robot Identity")
		assert.Contains(t, result, "Sales Analyst")
		assert.Contains(t, result, "Analyze sales data")
		assert.Contains(t, result, "Be accurate")
	})

	t.Run("returns empty for nil clock", func(t *testing.T) {
		result := formatter.FormatClockContext(nil, nil)
		assert.Empty(t, result)
	})

	t.Run("marks weekend correctly", func(t *testing.T) {
		// Saturday
		saturday := time.Date(2024, 1, 13, 10, 0, 0, 0, time.UTC)
		clock := types.NewClockContext(saturday, "UTC")

		result := formatter.FormatClockContext(clock, nil)

		assert.Contains(t, result, "✓ Weekend")
	})

	t.Run("marks month start correctly", func(t *testing.T) {
		// 2nd of month
		monthStart := time.Date(2024, 1, 2, 10, 0, 0, 0, time.UTC)
		clock := types.NewClockContext(monthStart, "UTC")

		result := formatter.FormatClockContext(clock, nil)

		assert.Contains(t, result, "✓ Month Start")
	})

	t.Run("marks month end correctly", func(t *testing.T) {
		// 30th of January (last 3 days)
		monthEnd := time.Date(2024, 1, 30, 10, 0, 0, 0, time.UTC)
		clock := types.NewClockContext(monthEnd, "UTC")

		result := formatter.FormatClockContext(clock, nil)

		assert.Contains(t, result, "✓ Month End")
	})
}

func TestInputFormatterFormatInspirationReport(t *testing.T) {
	formatter := standard.NewInputFormatter()

	t.Run("formats inspiration report with clock", func(t *testing.T) {
		now := time.Date(2024, 1, 15, 9, 30, 0, 0, time.UTC)
		clock := types.NewClockContext(now, "UTC")
		report := &types.InspirationReport{
			Clock:   clock,
			Content: "Today is a good day to analyze sales data.",
		}

		result := formatter.FormatInspirationReport(report)

		assert.Contains(t, result, "## Time Context")
		assert.Contains(t, result, "Monday")
		assert.Contains(t, result, "## Inspiration Report")
		assert.Contains(t, result, "analyze sales data")
	})

	t.Run("formats inspiration report without clock", func(t *testing.T) {
		report := &types.InspirationReport{
			Content: "Focus on quarterly review.",
		}

		result := formatter.FormatInspirationReport(report)

		assert.NotContains(t, result, "## Time Context")
		assert.Contains(t, result, "## Inspiration Report")
		assert.Contains(t, result, "quarterly review")
	})

	t.Run("returns empty for nil report", func(t *testing.T) {
		result := formatter.FormatInspirationReport(nil)
		assert.Empty(t, result)
	})
}

func TestInputFormatterFormatAvailableResources(t *testing.T) {
	formatter := standard.NewInputFormatter()

	t.Run("formats all resource types", func(t *testing.T) {
		robot := &types.Robot{
			MemberID: "test-robot",
			Config: &types.Config{
				Resources: &types.Resources{
					Agents: []string{"data-analyst", "chart-gen", "report-writer"},
					MCP: []types.MCPConfig{
						{ID: "database", Tools: []string{"query", "insert"}},
						{ID: "email", Tools: []string{}}, // all tools
					},
				},
				KB: &types.KB{
					Collections: []string{"sales-policies", "products"},
				},
				DB: &types.DB{
					Models: []string{"sales", "customers", "orders"},
				},
			},
		}

		result := formatter.FormatAvailableResources(robot)

		// Check structure
		assert.Contains(t, result, "## Available Resources")

		// Check agents
		assert.Contains(t, result, "### Agents")
		assert.Contains(t, result, "data-analyst")
		assert.Contains(t, result, "chart-gen")
		assert.Contains(t, result, "report-writer")

		// Check MCP tools
		assert.Contains(t, result, "### MCP Tools")
		assert.Contains(t, result, "database")
		assert.Contains(t, result, "query, insert")
		assert.Contains(t, result, "email")
		assert.Contains(t, result, "all tools available")

		// Check KB
		assert.Contains(t, result, "### Knowledge Base")
		assert.Contains(t, result, "sales-policies")
		assert.Contains(t, result, "products")

		// Check DB
		assert.Contains(t, result, "### Database")
		assert.Contains(t, result, "sales")
		assert.Contains(t, result, "customers")
		assert.Contains(t, result, "orders")

		// Check important note
		assert.Contains(t, result, "Only plan goals and tasks that can be accomplished")
	})

	t.Run("returns empty for nil robot", func(t *testing.T) {
		result := formatter.FormatAvailableResources(nil)
		assert.Empty(t, result)
	})

	t.Run("returns empty for robot without config", func(t *testing.T) {
		robot := &types.Robot{MemberID: "test"}
		result := formatter.FormatAvailableResources(robot)
		assert.Empty(t, result)
	})

	t.Run("returns empty for robot without resources", func(t *testing.T) {
		robot := &types.Robot{
			MemberID: "test",
			Config:   &types.Config{},
		}
		result := formatter.FormatAvailableResources(robot)
		assert.Empty(t, result)
	})

	t.Run("handles partial resources", func(t *testing.T) {
		robot := &types.Robot{
			MemberID: "test",
			Config: &types.Config{
				Resources: &types.Resources{
					Agents: []string{"single-agent"},
				},
			},
		}

		result := formatter.FormatAvailableResources(robot)

		assert.Contains(t, result, "## Available Resources")
		assert.Contains(t, result, "### Agents")
		assert.Contains(t, result, "single-agent")
		assert.NotContains(t, result, "### MCP Tools")
		assert.NotContains(t, result, "### Knowledge Base")
		assert.NotContains(t, result, "### Database")
	})
}

func TestInputFormatterFormatTriggerInput(t *testing.T) {
	formatter := standard.NewInputFormatter()

	t.Run("formats human intervention", func(t *testing.T) {
		input := &types.TriggerInput{
			Action: "task.add",
			UserID: "user-123",
			Messages: []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: "Please add a task to review Q4 sales"},
			},
		}

		result := formatter.FormatTriggerInput(input)

		assert.Contains(t, result, "## Human Intervention")
		assert.Contains(t, result, "task.add")
		assert.Contains(t, result, "user-123")
		assert.Contains(t, result, "### User Input")
		assert.Contains(t, result, "review Q4 sales")
	})

	t.Run("formats event trigger", func(t *testing.T) {
		input := &types.TriggerInput{
			Source:    "webhook",
			EventType: "order.created",
			Data: map[string]interface{}{
				"order_id": "12345",
				"amount":   99.99,
			},
		}

		result := formatter.FormatTriggerInput(input)

		assert.Contains(t, result, "## Event Trigger")
		assert.Contains(t, result, "webhook")
		assert.Contains(t, result, "order.created")
		assert.Contains(t, result, "### Event Data")
		assert.Contains(t, result, "order_id")
		assert.Contains(t, result, "12345")
	})

	t.Run("returns empty for nil input", func(t *testing.T) {
		result := formatter.FormatTriggerInput(nil)
		assert.Empty(t, result)
	})

	t.Run("returns empty for empty input", func(t *testing.T) {
		input := &types.TriggerInput{}
		result := formatter.FormatTriggerInput(input)
		assert.Empty(t, result)
	})
}

func TestInputFormatterFormatGoals(t *testing.T) {
	formatter := standard.NewInputFormatter()

	t.Run("formats goals with resources", func(t *testing.T) {
		goals := &types.Goals{
			Content: "1. Analyze sales data\n2. Generate report\n3. Send to stakeholders",
		}
		robot := &types.Robot{
			MemberID: "test-robot",
			Config: &types.Config{
				Resources: &types.Resources{
					Agents: []string{"data-analyzer", "report-generator"},
					MCP: []types.MCPConfig{
						{ID: "database", Tools: []string{"query", "insert"}},
						{ID: "email"},
					},
				},
			},
		}

		result := formatter.FormatGoals(goals, robot)

		assert.Contains(t, result, "## Goals")
		assert.Contains(t, result, "Analyze sales data")
		assert.Contains(t, result, "## Available Resources")
		assert.Contains(t, result, "### Agents")
		assert.Contains(t, result, "data-analyzer")
		assert.Contains(t, result, "### MCP Tools")
		assert.Contains(t, result, "database")
		assert.Contains(t, result, "query, insert")
		assert.Contains(t, result, "email")
		assert.Contains(t, result, "all tools available")
	})

	t.Run("formats goals without robot", func(t *testing.T) {
		goals := &types.Goals{
			Content: "Complete the task.",
		}

		result := formatter.FormatGoals(goals, nil)

		assert.Contains(t, result, "## Goals")
		assert.Contains(t, result, "Complete the task")
		assert.NotContains(t, result, "## Available Resources")
	})

	t.Run("returns empty for nil goals", func(t *testing.T) {
		result := formatter.FormatGoals(nil, nil)
		assert.Empty(t, result)
	})
}

func TestInputFormatterFormatTasks(t *testing.T) {
	formatter := standard.NewInputFormatter()

	t.Run("formats multiple tasks", func(t *testing.T) {
		tasks := []types.Task{
			{
				ID:           "task-1",
				GoalRef:      "goal-1",
				Source:       types.TaskSourceAuto,
				ExecutorType: types.ExecutorMCP,
				ExecutorID:   "database.query",
				Messages: []agentcontext.Message{
					{Role: agentcontext.RoleUser, Content: "Query sales data for Q4"},
				},
				Args: []any{"sales", "Q4"},
			},
			{
				ID:           "task-2",
				GoalRef:      "goal-1",
				Source:       types.TaskSourceAuto,
				ExecutorType: types.ExecutorAssistant,
				ExecutorID:   "report-generator",
			},
		}

		result := formatter.FormatTasks(tasks)

		assert.Contains(t, result, "## Tasks to Execute")
		assert.Contains(t, result, "### Task 1: task-1")
		assert.Contains(t, result, "goal-1")
		assert.Contains(t, result, "database.query")
		assert.Contains(t, result, "**Instructions**")
		assert.Contains(t, result, "Query sales data")
		assert.Contains(t, result, "**Arguments**")
		assert.Contains(t, result, "### Task 2: task-2")
		assert.Contains(t, result, "report-generator")
	})

	t.Run("returns message for empty tasks", func(t *testing.T) {
		result := formatter.FormatTasks(nil)
		assert.Equal(t, "No tasks to execute.", result)

		result = formatter.FormatTasks([]types.Task{})
		assert.Equal(t, "No tasks to execute.", result)
	})
}

func TestInputFormatterFormatTaskResults(t *testing.T) {
	formatter := standard.NewInputFormatter()

	t.Run("formats task results with summary", func(t *testing.T) {
		results := []types.TaskResult{
			{
				TaskID:   "task-1",
				Success:  true,
				Duration: 150,
				Validation: &types.ValidationResult{
					Passed: true,
					Score:  0.95,
				},
				Output: map[string]interface{}{"rows": 100},
			},
			{
				TaskID:   "task-2",
				Success:  false,
				Duration: 50,
				Validation: &types.ValidationResult{
					Passed: false,
					Issues: []string{"Connection timeout"},
				},
				Error: "Connection timeout",
			},
		}

		result := formatter.FormatTaskResults(results)

		assert.Contains(t, result, "## Task Results")
		assert.Contains(t, result, "### Task: task-1")
		assert.Contains(t, result, "✓ Success")
		assert.Contains(t, result, "150ms")
		assert.Contains(t, result, "**Validation**: ✓ Passed")
		assert.Contains(t, result, "score: 0.95")
		assert.Contains(t, result, "**Output**")
		assert.Contains(t, result, "### Task: task-2")
		assert.Contains(t, result, "✗ Failed")
		assert.Contains(t, result, "**Validation**: ✗ Failed")
		assert.Contains(t, result, "Connection timeout")
		assert.Contains(t, result, "## Summary")
		assert.Contains(t, result, "Total: 2 tasks")
		assert.Contains(t, result, "Success: 1")
		assert.Contains(t, result, "Failed: 1")
		assert.Contains(t, result, "Validated: 1/2")
	})

	t.Run("returns message for empty results", func(t *testing.T) {
		result := formatter.FormatTaskResults(nil)
		assert.Equal(t, "No task results.", result)
	})
}

func TestInputFormatterFormatExecutionSummary(t *testing.T) {
	formatter := standard.NewInputFormatter()

	t.Run("formats complete execution summary", func(t *testing.T) {
		startTime := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)
		endTime := time.Date(2024, 1, 15, 9, 5, 0, 0, time.UTC)
		exec := &types.Execution{
			ID:          "exec-123",
			TriggerType: types.TriggerClock,
			Status:      types.ExecCompleted,
			StartTime:   startTime,
			EndTime:     &endTime,
			Inspiration: &types.InspirationReport{
				Content: "Morning analysis suggests high activity.",
			},
			Goals: &types.Goals{
				Content: "1. Review data\n2. Generate report",
			},
			Tasks: []types.Task{
				{ID: "t1", Status: types.TaskCompleted, ExecutorID: "db.query"},
				{ID: "t2", Status: types.TaskCompleted, ExecutorID: "report.gen"},
			},
			Results: []types.TaskResult{
				{TaskID: "t1", Success: true, Duration: 100},
				{TaskID: "t2", Success: true, Duration: 200},
			},
			Delivery: &types.DeliveryResult{
				Type:    types.DeliveryEmail,
				Success: true,
			},
		}

		result := formatter.FormatExecutionSummary(exec)

		assert.Contains(t, result, "## Execution Summary")
		assert.Contains(t, result, "exec-123")
		assert.Contains(t, result, "clock")
		assert.Contains(t, result, "completed")
		assert.Contains(t, result, "**Duration**:")
		assert.Contains(t, result, "## Inspiration (P0)")
		assert.Contains(t, result, "Morning analysis")
		assert.Contains(t, result, "## Goals (P1)")
		assert.Contains(t, result, "Review data")
		assert.Contains(t, result, "## Tasks (P2)")
		assert.Contains(t, result, "db.query")
		assert.Contains(t, result, "## Results (P3)")
		assert.Contains(t, result, "✓ t1")
		assert.Contains(t, result, "## Delivery (P4)")
		assert.Contains(t, result, "email")
	})

	t.Run("formats execution with error", func(t *testing.T) {
		startTime := time.Now()
		exec := &types.Execution{
			ID:          "exec-456",
			TriggerType: types.TriggerHuman,
			Status:      types.ExecFailed,
			StartTime:   startTime,
			Error:       "Task execution failed",
		}

		result := formatter.FormatExecutionSummary(exec)

		assert.Contains(t, result, "exec-456")
		assert.Contains(t, result, "failed")
		assert.Contains(t, result, "**Error**: Task execution failed")
	})

	t.Run("returns empty for nil execution", func(t *testing.T) {
		result := formatter.FormatExecutionSummary(nil)
		assert.Empty(t, result)
	})
}

func TestInputFormatterBuildMessages(t *testing.T) {
	formatter := standard.NewInputFormatter()

	t.Run("builds user message", func(t *testing.T) {
		msgs := formatter.BuildMessages("Hello, world!")

		require.Len(t, msgs, 1)
		assert.Equal(t, agentcontext.RoleUser, msgs[0].Role)
		assert.Equal(t, "Hello, world!", msgs[0].Content)
	})
}

func TestInputFormatterBuildMessagesWithSystem(t *testing.T) {
	formatter := standard.NewInputFormatter()

	t.Run("builds system and user messages", func(t *testing.T) {
		msgs := formatter.BuildMessagesWithSystem(
			"You are a helpful assistant.",
			"What is the weather?",
		)

		require.Len(t, msgs, 2)
		assert.Equal(t, agentcontext.RoleSystem, msgs[0].Role)
		assert.Equal(t, "You are a helpful assistant.", msgs[0].Content)
		assert.Equal(t, agentcontext.RoleUser, msgs[1].Role)
		assert.Equal(t, "What is the weather?", msgs[1].Content)
	})
}
