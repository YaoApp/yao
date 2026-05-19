//go:build integration

package standard_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	agentcontext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/robot/executor/standard"
	"github.com/yaoapp/yao/agent/robot/executor/types"
	"github.com/yaoapp/yao/agent/robot/store"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func newTestRobot(t *testing.T, identity *testprepare.TestIdentity) *robottypes.Robot {
	t.Helper()
	name := t.Name()
	if len(name) > 40 {
		name = name[len(name)-40:]
	}
	return &robottypes.Robot{
		MemberID:       "rt-" + name,
		TeamID:         identity.AlphaTeamID,
		DisplayName:    "Test Robot",
		Status:         robottypes.RobotIdle,
		AutonomousMode: true,
		Config: &robottypes.Config{
			Identity: &robottypes.Identity{Role: "Test Assistant", Duties: []string{"Testing"}},
			Quota:    &robottypes.Quota{Max: 5, Queue: 10},
			Resources: &robottypes.Resources{
				Phases: map[robottypes.Phase]string{
					robottypes.PhaseInspiration: "tests.robot-inspiration",
					robottypes.PhaseGoals:       "tests.robot-goals",
					robottypes.PhaseTasks:       "tests.robot-tasks",
					robottypes.PhaseRun:         "tests.robot-validation",
					"validation":                "tests.robot-validation",
					robottypes.PhaseDelivery:    "tests.robot-delivery",
					robottypes.PhaseLearning:    "tests.robot-learning",
					robottypes.PhaseHost:        "tests.robot-host",
				},
				Agents: []string{"experts.text-writer", "experts.data-analyst"},
			},
		},
	}
}

func testAuth(identity *testprepare.TestIdentity) *oauthtypes.AuthorizedInfo {
	return &oauthtypes.AuthorizedInfo{
		UserID: identity.AlphaOwnerUserID,
		TeamID: identity.AlphaTeamID,
	}
}

func testCtx(identity *testprepare.TestIdentity) *robottypes.Context {
	return robottypes.NewContext(context.Background(), testAuth(identity))
}

// ============================================================================
// Executor Persistence Integration Tests
// ============================================================================

func TestExecutorPersistence(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)

	t.Run("persists_execution_record_on_start", func(t *testing.T) {
		ctx := testCtx(identity)
		robot := newTestRobot(t, identity)

		e := standard.NewWithConfig(types.Config{SkipPersistence: false})

		exec, err := e.Execute(ctx, robot, robottypes.TriggerHuman, "simulate_failure")
		require.NoError(t, err)
		require.NotNil(t, exec)

		s := store.NewExecutionStore()
		record, err := s.Get(context.Background(), exec.ID)
		require.NoError(t, err)
		require.NotNil(t, record)

		assert.Equal(t, exec.ID, record.ExecutionID)
		assert.Equal(t, robot.MemberID, record.MemberID)
		assert.Equal(t, identity.AlphaTeamID, record.TeamID)
		assert.Equal(t, robottypes.TriggerHuman, record.TriggerType)
		assert.Equal(t, robottypes.ExecFailed, record.Status)
		assert.Equal(t, "simulated failure", record.Error)

		_ = s.Delete(context.Background(), exec.ID)
	})

	t.Run("persists_failed_status_with_error", func(t *testing.T) {
		ctx := testCtx(identity)
		robot := newTestRobot(t, identity)

		e := standard.NewWithConfig(types.Config{SkipPersistence: false})
		exec, err := e.Execute(ctx, robot, robottypes.TriggerHuman, "simulate_failure")
		require.NoError(t, err)
		require.NotNil(t, exec)

		s := store.NewExecutionStore()
		record, err := s.Get(context.Background(), exec.ID)
		require.NoError(t, err)
		require.NotNil(t, record)

		assert.Equal(t, robottypes.ExecFailed, record.Status)
		assert.Equal(t, "simulated failure", record.Error)
		assert.NotNil(t, record.StartTime)

		_ = s.Delete(context.Background(), exec.ID)
	})

	t.Run("skips_persistence_when_disabled", func(t *testing.T) {
		ctx := testCtx(identity)
		robot := newTestRobot(t, identity)

		e := standard.NewWithConfig(types.Config{SkipPersistence: true})
		exec, err := e.Execute(ctx, robot, robottypes.TriggerHuman, "simulate_failure")
		require.NoError(t, err)
		require.NotNil(t, exec)

		s := store.NewExecutionStore()
		record, err := s.Get(context.Background(), exec.ID)
		require.NoError(t, err)
		assert.Nil(t, record)
	})
}

// ============================================================================
// Goals Injection Tests (Host Agent confirmed goals)
// ============================================================================

func TestExecutorGoalsInjection(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)

	t.Run("goals_injected_from_trigger_input_data", func(t *testing.T) {
		ctx := testCtx(identity)
		robot := newTestRobot(t, identity)

		e := standard.NewWithConfig(types.Config{SkipPersistence: false})
		triggerInput := &robottypes.TriggerInput{
			Data: map[string]interface{}{
				"goals":   "Create a mecha image with sci-fi style",
				"chat_id": "robot_test_inject_001",
			},
		}

		exec, err := e.Execute(ctx, robot, robottypes.TriggerHuman, triggerInput)
		require.NoError(t, err)
		require.NotNil(t, exec)

		require.NotNil(t, exec.Goals)
		assert.Equal(t, "Create a mecha image with sci-fi style", exec.Goals.Content)
		assert.NotEmpty(t, exec.Name)
		assert.NotEqual(t, "Preparing...", exec.Name)

		s := store.NewExecutionStore()
		_ = s.Delete(context.Background(), exec.ID)
	})

	t.Run("no_trigger_input_uses_normal_flow", func(t *testing.T) {
		ctx := testCtx(identity)
		robot := newTestRobot(t, identity)

		e := standard.NewWithConfig(types.Config{SkipPersistence: true})
		exec, err := e.Execute(ctx, robot, robottypes.TriggerHuman, "simulate_failure")
		require.NoError(t, err)
		require.NotNil(t, exec)

		assert.Nil(t, exec.Goals)
	})
}

// ============================================================================
// Executor Counter Tests
// ============================================================================

func TestExecutorCounters(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)

	t.Run("exec_count_increments", func(t *testing.T) {
		ctx := testCtx(identity)
		robot := newTestRobot(t, identity)

		e := standard.New()
		e.Reset()

		assert.Equal(t, 0, e.ExecCount())
		assert.Equal(t, 0, e.CurrentCount())

		exec, err := e.Execute(ctx, robot, robottypes.TriggerHuman, "simulate_failure")
		require.NoError(t, err)
		require.NotNil(t, exec)

		assert.Equal(t, 1, e.ExecCount())
		assert.Equal(t, 0, e.CurrentCount())
	})
}

// ============================================================================
// Nil Robot Test
// ============================================================================

func TestExecutorNilRobot(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)
	ctx := testCtx(identity)

	e := standard.New()
	exec, err := e.Execute(ctx, nil, robottypes.TriggerHuman, "test")
	assert.Error(t, err)
	assert.Nil(t, exec)
	assert.Contains(t, err.Error(), "robot cannot be nil")
}

// ============================================================================
// UI Fields / i18n
// ============================================================================

func TestExecutorUIFields(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)

	t.Run("human_trigger_extracts_message_as_name", func(t *testing.T) {
		ctx := testCtx(identity)
		robot := newTestRobot(t, identity)

		e := standard.NewWithConfig(types.Config{SkipPersistence: true})
		triggerInput := &robottypes.TriggerInput{
			Messages: []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: "Analyze Q4 revenue data"},
			},
		}

		exec, err := e.Execute(ctx, robot, robottypes.TriggerHuman, triggerInput)
		require.NoError(t, err)
		require.NotNil(t, exec)
		assert.Contains(t, exec.Name, "Analyze Q4 revenue data")
	})

	t.Run("clock_trigger_uses_scheduled_execution_name", func(t *testing.T) {
		ctx := testCtx(identity)
		robot := newTestRobot(t, identity)

		e := standard.NewWithConfig(types.Config{SkipPersistence: true})

		exec, err := e.Execute(ctx, robot, robottypes.TriggerClock, "simulate_failure")
		require.NoError(t, err)
		require.NotNil(t, exec)
		assert.NotEmpty(t, exec.Name)
	})
}

// ============================================================================
// Trigger-Based Phase Skipping
// ============================================================================

func TestExecutorTriggerPhaseSkipping(t *testing.T) {
	identity := testprepare.PrepareSandbox(t)

	t.Run("human_trigger_skips_inspiration_phase", func(t *testing.T) {
		ctx := testCtx(identity)
		robot := newTestRobot(t, identity)

		phaseLog := []robottypes.Phase{}
		e := standard.NewWithConfig(types.Config{
			SkipPersistence: true,
			OnPhaseStart: func(phase robottypes.Phase) {
				phaseLog = append(phaseLog, phase)
			},
		})

		_, _ = e.Execute(ctx, robot, robottypes.TriggerHuman, "simulate_failure")

		for _, p := range phaseLog {
			assert.NotEqual(t, robottypes.PhaseInspiration, p, "human trigger should skip inspiration")
		}
	})
}

// ============================================================================
// Helpers for TriggerMessage (used only in this file)
// ============================================================================

// Ensure TriggerMessage is compatible with exec.Input.Messages
func init() {
	// This block is intentionally empty. TriggerMessage adapts via existing interfaces.
}

// createTestExecution creates a minimal test execution for direct phase calls
func createTestExecution(robot *robottypes.Robot, trigger robottypes.TriggerType) *robottypes.Execution {
	exec := &robottypes.Execution{
		ID:          "test-exec-" + time.Now().Format("150405"),
		MemberID:    robot.MemberID,
		TeamID:      robot.TeamID,
		TriggerType: trigger,
		StartTime:   time.Now(),
		Status:      robottypes.ExecRunning,
		Phase:       robottypes.PhaseInspiration,
		Input: &robottypes.TriggerInput{
			Clock: robottypes.NewClockContext(time.Now(), ""),
		},
	}
	exec.SetRobot(robot)
	return exec
}
