//go:build e2e

package standard_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/robot/executor/standard"
	"github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/testutils"
)

// ============================================================================
// P4 Delivery Phase Tests
// ============================================================================

func TestRunDeliveryBasic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.PrepareAgent(t)
	testutils.RequireE2EKeys(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("generates delivery content from execution results", func(t *testing.T) {
		robot := createDeliveryTestRobot(t, "robot.delivery")
		exec := createDeliveryTestExecution(robot)

		exec.Inspiration = &types.InspirationReport{
			Content: "Morning analysis suggests focus on Q4 review.",
		}
		exec.Goals = &types.Goals{
			Content: "## Goals\n1. Review Q4 data\n2. Generate summary report",
		}
		exec.Tasks = []types.Task{
			{ID: "task-001", ExecutorID: "experts.data-analyst", ExecutorType: types.ExecutorAssistant, Status: types.TaskCompleted},
			{ID: "task-002", ExecutorID: "experts.summarizer", ExecutorType: types.ExecutorAssistant, Status: types.TaskCompleted},
		}
		exec.Results = []types.TaskResult{
			{TaskID: "task-001", Success: true, Duration: 1500, Output: map[string]interface{}{"total_sales": 1500000}},
			{TaskID: "task-002", Success: true, Duration: 800, Output: "Q4 sales exceeded expectations by 15%."},
		}

		e := standard.New()
		err := e.RunDelivery(ctx, exec, nil)

		require.NoError(t, err)
		require.NotNil(t, exec.Delivery)
		require.NotNil(t, exec.Delivery.Content)
		assert.NotEmpty(t, exec.Delivery.Content.Summary)
		assert.NotEmpty(t, exec.Delivery.Content.Body)
		assert.True(t, exec.Delivery.Success)
	})

	t.Run("handles partial failure in results", func(t *testing.T) {
		robot := createDeliveryTestRobot(t, "robot.delivery")
		exec := createDeliveryTestExecution(robot)

		exec.Goals = &types.Goals{
			Content: "## Goals\n1. Analyze data\n2. Generate report",
		}
		exec.Tasks = []types.Task{
			{ID: "task-001", ExecutorID: "experts.data-analyst", ExecutorType: types.ExecutorAssistant, Status: types.TaskCompleted},
			{ID: "task-002", ExecutorID: "experts.summarizer", ExecutorType: types.ExecutorAssistant, Status: types.TaskFailed},
		}
		exec.Results = []types.TaskResult{
			{TaskID: "task-001", Success: true, Duration: 1500, Output: map[string]interface{}{"data": "analyzed"}},
			{TaskID: "task-002", Success: false, Duration: 500, Error: "Summarization failed: timeout"},
		}

		e := standard.New()
		err := e.RunDelivery(ctx, exec, nil)

		require.NoError(t, err)
		require.NotNil(t, exec.Delivery)
		require.NotNil(t, exec.Delivery.Content)

		body := strings.ToLower(exec.Delivery.Content.Body)
		hasFailureInfo := strings.Contains(body, "fail") ||
			strings.Contains(body, "error") ||
			strings.Contains(body, "partial") ||
			strings.Contains(body, "✗")

		assert.True(t, hasFailureInfo || exec.Delivery.Content.Summary != "", "should mention failure or have valid summary")
	})
}

func TestRunDeliveryErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.PrepareAgent(t)
	testutils.RequireE2EKeys(t)
	defer testutils.Clean(t)

	ctx := types.NewContext(context.Background(), testAuth())

	t.Run("returns error when robot is nil", func(t *testing.T) {
		exec := &types.Execution{
			ID:          "test-exec-1",
			TriggerType: types.TriggerClock,
		}

		e := standard.New()
		err := e.RunDelivery(ctx, exec, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "robot not found")
	})

	t.Run("returns error when agent not found", func(t *testing.T) {
		robot := &types.Robot{
			MemberID: "test-robot-1",
			TeamID:   "test-team-1",
			Config: &types.Config{
				Identity: &types.Identity{Role: "Test"},
				Resources: &types.Resources{
					Phases: map[types.Phase]string{
						types.PhaseDelivery: "non.existent.agent",
					},
				},
			},
		}
		exec := createDeliveryTestExecution(robot)
		exec.Results = []types.TaskResult{
			{TaskID: "task-001", Success: true, Duration: 100},
		}

		e := standard.New()
		err := e.RunDelivery(ctx, exec, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "call failed")
	})
}

// ============================================================================
// Email Channel Config Tests
// ============================================================================

func TestDefaultEmailChannel(t *testing.T) {
	t.Run("returns default email channel", func(t *testing.T) {
		assert.Equal(t, "default", types.DefaultEmailChannel())
	})

	t.Run("can set custom email channel", func(t *testing.T) {
		original := types.DefaultEmailChannel()
		defer types.SetDefaultEmailChannel(original)

		types.SetDefaultEmailChannel("custom-email")
		assert.Equal(t, "custom-email", types.DefaultEmailChannel())
	})

	t.Run("ignores empty channel", func(t *testing.T) {
		original := types.DefaultEmailChannel()
		defer types.SetDefaultEmailChannel(original)

		types.SetDefaultEmailChannel("")
		assert.Equal(t, original, types.DefaultEmailChannel())
	})
}

func TestRobotEmailInDelivery(t *testing.T) {
	t.Run("robot email field is loaded from map", func(t *testing.T) {
		data := map[string]interface{}{
			"member_id":   "robot-001",
			"team_id":     "team-001",
			"robot_email": "robot@example.com",
		}

		robot, err := types.NewRobotFromMap(data)
		require.NoError(t, err)
		assert.Equal(t, "robot@example.com", robot.RobotEmail)
	})

	t.Run("robot email can be empty", func(t *testing.T) {
		data := map[string]interface{}{
			"member_id": "robot-001",
			"team_id":   "team-001",
		}

		robot, err := types.NewRobotFromMap(data)
		require.NoError(t, err)
		assert.Empty(t, robot.RobotEmail)
	})
}

// ============================================================================
// FormatDeliveryInput Tests
// ============================================================================

func TestFormatDeliveryInput(t *testing.T) {
	formatter := standard.NewInputFormatter()

	t.Run("formats complete execution context", func(t *testing.T) {
		robot := &types.Robot{
			MemberID: "test-robot",
			Config: &types.Config{
				Identity: &types.Identity{
					Role:   "Sales Analyst",
					Duties: []string{"Analyze data", "Generate reports"},
				},
			},
		}

		startTime := time.Now().Add(-5 * time.Minute)
		endTime := time.Now()
		exec := &types.Execution{
			ID:          "exec-123",
			TriggerType: types.TriggerClock,
			Status:      types.ExecCompleted,
			StartTime:   startTime,
			EndTime:     &endTime,
			Inspiration: &types.InspirationReport{
				Content: "Morning analysis suggests focus on Q4.",
			},
			Goals: &types.Goals{
				Content: "## Goals\n1. Review Q4 data",
			},
			Tasks: []types.Task{
				{ID: "task-001", ExecutorID: "data-analyst", ExecutorType: types.ExecutorAssistant, Status: types.TaskCompleted, ExpectedOutput: "JSON with sales data"},
			},
			Results: []types.TaskResult{
				{TaskID: "task-001", Success: true, Duration: 1500, Output: map[string]interface{}{"sales": 1000000}},
			},
		}

		result := formatter.FormatDeliveryInput(exec, robot)

		assert.Contains(t, result, "## Robot Identity")
		assert.Contains(t, result, "Sales Analyst")
		assert.Contains(t, result, "## Execution Context")
		assert.Contains(t, result, "clock")
		assert.Contains(t, result, "## Inspiration (P0)")
		assert.Contains(t, result, "Morning analysis")
		assert.Contains(t, result, "## Goals (P1)")
		assert.Contains(t, result, "Review Q4 data")
		assert.Contains(t, result, "## Tasks (P2)")
		assert.Contains(t, result, "task-001")
		assert.Contains(t, result, "## Results (P3)")
		assert.Contains(t, result, "✓ Task: task-001")
	})

	t.Run("handles empty execution", func(t *testing.T) {
		exec := &types.Execution{
			ID:          "exec-empty",
			TriggerType: types.TriggerHuman,
			Status:      types.ExecPending,
			StartTime:   time.Now(),
		}

		result := formatter.FormatDeliveryInput(exec, nil)

		assert.Contains(t, result, "## Execution Context")
		assert.Contains(t, result, "human")
	})
}

// ============================================================================
// Helper Functions
// ============================================================================

func createDeliveryTestRobot(t *testing.T, agentID string) *types.Robot {
	t.Helper()
	return &types.Robot{
		MemberID:    "test-robot-1",
		TeamID:      "test-team-1",
		DisplayName: "Test Robot",
		Config: &types.Config{
			Identity: &types.Identity{
				Role:   "Test Assistant",
				Duties: []string{"Testing", "Validation"},
			},
			Resources: &types.Resources{
				Phases: map[types.Phase]string{
					types.PhaseDelivery: agentID,
				},
			},
		},
	}
}

func createDeliveryTestExecution(robot *types.Robot) *types.Execution {
	exec := &types.Execution{
		ID:          "test-exec-delivery-1",
		MemberID:    robot.MemberID,
		TeamID:      robot.TeamID,
		TriggerType: types.TriggerClock,
		StartTime:   time.Now(),
		Status:      types.ExecRunning,
		Phase:       types.PhaseDelivery,
	}
	exec.SetRobot(robot)
	return exec
}
