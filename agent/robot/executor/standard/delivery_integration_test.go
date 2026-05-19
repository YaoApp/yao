//go:build integration

package standard_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent/robot/executor/standard"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

// ============================================================================
// P4 Delivery Phase Tests
// ============================================================================

// TODO: TestRunDeliveryBasic requires mock-llm fixtures for structured delivery content (summary/body JSON)

func TestRunDeliveryErrorHandling(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	ctx := testCtx(identity)

	t.Run("returns_error_when_robot_is_nil", func(t *testing.T) {
		exec := &robottypes.Execution{ID: "test-exec-delivery-norobot", TriggerType: robottypes.TriggerClock}

		e := standard.New()
		err := e.RunDelivery(ctx, exec, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "robot not found")
	})

	t.Run("returns_error_when_agent_not_found", func(t *testing.T) {
		robot := &robottypes.Robot{
			MemberID: "test-robot-bad-delivery",
			TeamID:   identity.AlphaTeamID,
			Config: &robottypes.Config{
				Identity: &robottypes.Identity{Role: "Test"},
				Resources: &robottypes.Resources{
					Phases: map[robottypes.Phase]string{
						robottypes.PhaseDelivery: "non.existent.agent",
					},
				},
			},
		}
		exec := createDeliveryExecution(robot)
		exec.Results = []robottypes.TaskResult{
			{TaskID: "task-001", Success: true, Duration: 100},
		}

		e := standard.New()
		err := e.RunDelivery(ctx, exec, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "call failed")
	})
}

// ============================================================================
// FormatDeliveryInput Tests
// ============================================================================

func TestFormatDeliveryInput(t *testing.T) {
	_ = testprepare.PrepareSandbox(t)
	formatter := standard.NewInputFormatter()

	t.Run("formats_complete_execution_context", func(t *testing.T) {
		robot := &robottypes.Robot{
			MemberID: "test-robot",
			Config: &robottypes.Config{
				Identity: &robottypes.Identity{
					Role:   "Sales Analyst",
					Duties: []string{"Analyze data", "Generate reports"},
				},
			},
		}

		startTime := time.Now().Add(-5 * time.Minute)
		endTime := time.Now()
		exec := &robottypes.Execution{
			ID: "exec-123", TriggerType: robottypes.TriggerClock,
			Status: robottypes.ExecCompleted, StartTime: startTime, EndTime: &endTime,
			Inspiration: &robottypes.InspirationReport{Content: "Morning analysis suggests focus on Q4."},
			Goals:       &robottypes.Goals{Content: "## Goals\n1. Review Q4 data"},
			Tasks: []robottypes.Task{
				{ID: "task-001", ExecutorID: "data-analyst", ExecutorType: robottypes.ExecutorAssistant, Status: robottypes.TaskCompleted},
			},
			Results: []robottypes.TaskResult{
				{TaskID: "task-001", Success: true, Duration: 1500, Output: map[string]interface{}{"sales": 1000000}},
			},
		}

		result := formatter.FormatDeliveryInput(exec, robot)

		assert.Contains(t, result, "## Robot Identity")
		assert.Contains(t, result, "Sales Analyst")
		assert.Contains(t, result, "## Execution Context")
		assert.Contains(t, result, "## Inspiration (P0)")
		assert.Contains(t, result, "## Goals (P1)")
		assert.Contains(t, result, "## Tasks (P2)")
		assert.Contains(t, result, "## Results (P3)")
	})

	t.Run("handles_nil_execution", func(t *testing.T) {
		result := formatter.FormatDeliveryInput(nil, nil)
		assert.Empty(t, result)
	})
}

// ============================================================================
// Helpers
// ============================================================================

func createDeliveryExecution(robot *robottypes.Robot) *robottypes.Execution {
	exec := &robottypes.Execution{
		ID:          "test-exec-delivery-" + time.Now().Format("150405.000"),
		MemberID:    robot.MemberID,
		TeamID:      robot.TeamID,
		TriggerType: robottypes.TriggerClock,
		StartTime:   time.Now(),
		Status:      robottypes.ExecRunning,
		Phase:       robottypes.PhaseDelivery,
	}
	exec.SetRobot(robot)
	return exec
}
