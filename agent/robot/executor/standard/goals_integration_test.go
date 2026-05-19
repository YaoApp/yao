//go:build integration

package standard_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/robot/executor/standard"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

// ============================================================================
// P1 Goals Phase Tests
// ============================================================================

// TODO: TestRunGoalsBasic requires mock-llm fixtures for structured phase responses
// TODO: TestRunGoalsHumanTrigger requires mock-llm fixtures for structured phase responses
// TODO: TestRunGoalsFallbackBehavior requires mock-llm fixtures for structured phase responses

func TestRunGoalsPrePopulated(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	ctx := testCtx(identity)

	t.Run("succeeds_when_goals_content_pre_populated", func(t *testing.T) {
		robot := newTestRobot(t, identity)
		exec := createGoalsExecution(robot, robottypes.TriggerClock)
		exec.Goals = &robottypes.Goals{
			Content: "## Goals\n1. [High] Review Q4 data\n2. [Normal] Generate summary report",
		}

		e := standard.New()
		err := e.RunGoals(ctx, exec, nil)

		require.NoError(t, err)
		assert.Equal(t, "## Goals\n1. [High] Review Q4 data\n2. [Normal] Generate summary report", exec.Goals.Content)
	})
}

func TestRunGoalsErrorHandling(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	ctx := testCtx(identity)

	t.Run("returns_error_when_robot_is_nil", func(t *testing.T) {
		exec := &robottypes.Execution{
			ID:          "test-exec-goals-norobot",
			TriggerType: robottypes.TriggerClock,
		}

		e := standard.New()
		err := e.RunGoals(ctx, exec, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "robot not found")
	})

	t.Run("returns_error_when_no_input_and_no_identity", func(t *testing.T) {
		robot := &robottypes.Robot{
			MemberID: "test-robot-no-identity",
			TeamID:   identity.AlphaTeamID,
			Config: &robottypes.Config{
				Resources: &robottypes.Resources{
					Phases: map[robottypes.Phase]string{
						robottypes.PhaseGoals: "tests.robot-goals",
					},
				},
			},
		}
		exec := createGoalsExecution(robot, robottypes.TriggerHuman)
		exec.Input = nil

		e := standard.New()
		err := e.RunGoals(ctx, exec, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no input available")
	})

	t.Run("skips_regeneration_when_goals_pre_populated", func(t *testing.T) {
		robot := newTestRobot(t, identity)
		exec := createGoalsExecution(robot, robottypes.TriggerHuman)
		exec.Goals = &robottypes.Goals{Content: "Pre-confirmed goals content"}

		e := standard.New()
		err := e.RunGoals(ctx, exec, nil)

		require.NoError(t, err)
		assert.Equal(t, "Pre-confirmed goals content", exec.Goals.Content)
	})
}

// ============================================================================
// ParseDelivery Tests
// ============================================================================

func TestParseDelivery(t *testing.T) {
	_ = testprepare.PrepareSandbox(t)

	t.Run("parses_valid_delivery_with_all_fields", func(t *testing.T) {
		data := map[string]interface{}{
			"type":       "email",
			"recipients": []interface{}{"user@example.com", "team@example.com"},
			"format":     "markdown",
			"template":   "weekly-report",
			"options":    map[string]interface{}{"subject": "Weekly Report"},
		}

		result := standard.ParseDelivery(data)

		require.NotNil(t, result)
		assert.Equal(t, robottypes.DeliveryEmail, result.Type)
		assert.Equal(t, []string{"user@example.com", "team@example.com"}, result.Recipients)
		assert.Equal(t, "markdown", result.Format)
		assert.Equal(t, "weekly-report", result.Template)
		assert.Equal(t, "Weekly Report", result.Options["subject"])
	})

	t.Run("returns_nil_for_nil_data", func(t *testing.T) {
		assert.Nil(t, standard.ParseDelivery(nil))
	})

	t.Run("returns_nil_for_missing_type", func(t *testing.T) {
		data := map[string]interface{}{"recipients": []interface{}{"user@example.com"}}
		assert.Nil(t, standard.ParseDelivery(data))
	})

	t.Run("returns_nil_for_invalid_type", func(t *testing.T) {
		data := map[string]interface{}{"type": "sms", "recipients": []interface{}{"user@example.com"}}
		assert.Nil(t, standard.ParseDelivery(data))
	})
}

func TestDeliveryTypeValidation(t *testing.T) {
	_ = testprepare.PrepareSandbox(t)

	t.Run("valid_delivery_types", func(t *testing.T) {
		for _, dt := range []robottypes.DeliveryType{
			robottypes.DeliveryEmail, robottypes.DeliveryWebhook,
			robottypes.DeliveryProcess, robottypes.DeliveryNotify,
		} {
			assert.True(t, standard.IsValidDeliveryType(dt))
		}
	})

	t.Run("invalid_delivery_types", func(t *testing.T) {
		for _, dt := range []robottypes.DeliveryType{"invalid", "sms", ""} {
			assert.False(t, standard.IsValidDeliveryType(dt))
		}
	})
}

// ============================================================================
// InputFormatter Tests for P1
// ============================================================================

func TestInputFormatterFormatRobotIdentity(t *testing.T) {
	_ = testprepare.PrepareSandbox(t)
	formatter := standard.NewInputFormatter()

	t.Run("formats_robot_identity_correctly", func(t *testing.T) {
		robot := &robottypes.Robot{
			MemberID: "test-robot",
			Config: &robottypes.Config{
				Identity: &robottypes.Identity{
					Role:   "Sales Analyst",
					Duties: []string{"Analyze sales data", "Generate reports"},
					Rules:  []string{"Be accurate", "Be concise"},
				},
			},
		}

		content := formatter.FormatRobotIdentity(robot)

		assert.Contains(t, content, "## Robot Identity")
		assert.Contains(t, content, "Sales Analyst")
		assert.Contains(t, content, "Analyze sales data")
		assert.Contains(t, content, "Be accurate")
	})

	t.Run("returns_empty_for_nil_robot", func(t *testing.T) {
		assert.Empty(t, formatter.FormatRobotIdentity(nil))
	})

	t.Run("returns_empty_for_robot_without_identity", func(t *testing.T) {
		robot := &robottypes.Robot{MemberID: "test", Config: &robottypes.Config{}}
		assert.Empty(t, formatter.FormatRobotIdentity(robot))
	})
}

// ============================================================================
// Helpers
// ============================================================================

func createGoalsExecution(robot *robottypes.Robot, trigger robottypes.TriggerType) *robottypes.Execution {
	exec := &robottypes.Execution{
		ID:          "test-exec-goals-" + time.Now().Format("150405.000"),
		MemberID:    robot.MemberID,
		TeamID:      robot.TeamID,
		TriggerType: trigger,
		StartTime:   time.Now(),
		Status:      robottypes.ExecRunning,
		Phase:       robottypes.PhaseGoals,
	}
	exec.SetRobot(robot)
	return exec
}
