//go:build e2e

package standard_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/robot/executor/standard"
	"github.com/yaoapp/yao/agent/robot/executor/types"
	"github.com/yaoapp/yao/agent/robot/store"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/testutils"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
)

// ============================================================================
// Executor Persistence Integration Tests
// ============================================================================

func TestExecutorPersistence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.PrepareAgent(t)
	testutils.RequireE2EKeys(t)
	defer testutils.Clean(t)

	t.Run("persists_execution_record_on_start", func(t *testing.T) {
		ctx := robottypes.NewContext(context.Background(), &oauthTypes.AuthorizedInfo{
			UserID: "user_persist_001",
			TeamID: "team_persist_001",
		})

		robot := createPersistenceTestRobot("member_persist_001", "team_persist_001")

		// Create executor with persistence enabled
		e := standard.NewWithConfig(types.Config{
			SkipPersistence: false,
		})

		// Execute with simulated failure to ensure we get a result
		exec, err := e.Execute(ctx, robot, robottypes.TriggerHuman, "simulate_failure")
		require.NoError(t, err)
		require.NotNil(t, exec)

		// Verify execution record was persisted
		s := store.NewExecutionStore()
		record, err := s.Get(context.Background(), exec.ID)
		require.NoError(t, err)
		require.NotNil(t, record)

		assert.Equal(t, exec.ID, record.ExecutionID)
		assert.Equal(t, "member_persist_001", record.MemberID)
		assert.Equal(t, "team_persist_001", record.TeamID)
		assert.Equal(t, robottypes.TriggerHuman, record.TriggerType)
		assert.Equal(t, robottypes.ExecFailed, record.Status)
		assert.Equal(t, "simulated failure", record.Error)

		// Cleanup
		_ = s.Delete(context.Background(), exec.ID)
	})

	t.Run("persists_failed_status_with_error", func(t *testing.T) {
		ctx := robottypes.NewContext(context.Background(), &oauthTypes.AuthorizedInfo{
			UserID: "user_persist_002",
			TeamID: "team_persist_002",
		})

		robot := createPersistenceTestRobot("member_persist_002", "team_persist_002")

		e := standard.NewWithConfig(types.Config{
			SkipPersistence: false,
		})

		// Execute with simulated failure
		exec, err := e.Execute(ctx, robot, robottypes.TriggerHuman, "simulate_failure")
		require.NoError(t, err)
		require.NotNil(t, exec)

		// Verify the record has failed status with error message
		s := store.NewExecutionStore()
		record, err := s.Get(context.Background(), exec.ID)
		require.NoError(t, err)
		require.NotNil(t, record)

		assert.Equal(t, robottypes.ExecFailed, record.Status)
		assert.Equal(t, "simulated failure", record.Error)
		assert.NotNil(t, record.StartTime)

		// Cleanup
		_ = s.Delete(context.Background(), exec.ID)
	})

	t.Run("skips_persistence_when_disabled", func(t *testing.T) {
		ctx := robottypes.NewContext(context.Background(), &oauthTypes.AuthorizedInfo{
			UserID: "user_persist_003",
			TeamID: "team_persist_003",
		})

		robot := createPersistenceTestRobot("member_persist_003", "team_persist_003")

		// Create executor with persistence disabled
		e := standard.NewWithConfig(types.Config{
			SkipPersistence: true,
		})

		exec, err := e.Execute(ctx, robot, robottypes.TriggerHuman, "simulate_failure")
		require.NoError(t, err)
		require.NotNil(t, exec)

		// Verify no record was created
		s := store.NewExecutionStore()
		record, err := s.Get(context.Background(), exec.ID)
		require.NoError(t, err)
		assert.Nil(t, record) // Should not exist
	})
}

// ============================================================================
// Goals Injection Tests (Host Agent confirmed goals)
// ============================================================================

// TestExecutorGoalsInjection verifies that when TriggerHuman is used with
// pre-confirmed goals (from Host Agent via /v1/agent/robots/:id/execute),
// the goals are injected directly into exec.Goals before RunGoals runs,
// and are persisted (title updated) so the task list shows the correct title.
func TestExecutorGoalsInjection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.PrepareAgent(t)
	testutils.RequireE2EKeys(t)
	defer testutils.Clean(t)

	t.Run("goals_injected_from_trigger_input_data", func(t *testing.T) {
		ctx := robottypes.NewContext(context.Background(), &oauthTypes.AuthorizedInfo{
			UserID: "user_goals_inject_001",
			TeamID: "team_goals_inject_001",
		})

		robot := createPersistenceTestRobot("member_goals_inject_001", "team_goals_inject_001")

		e := standard.NewWithConfig(types.Config{
			SkipPersistence: false,
		})

		// Simulate Host Agent confirmed goals passed via TriggerInput.Data
		triggerInput := &robottypes.TriggerInput{
			Data: map[string]interface{}{
				"goals":   "Create a mecha image with sci-fi style",
				"chat_id": "robot_member_goals_inject_001_1234567890",
			},
		}

		exec, err := e.Execute(ctx, robot, robottypes.TriggerHuman, triggerInput)
		require.NoError(t, err)
		require.NotNil(t, exec)

		// Goals should be injected from TriggerInput.Data
		require.NotNil(t, exec.Goals, "Goals should be injected from TriggerInput.Data")
		assert.Equal(t, "Create a mecha image with sci-fi style", exec.Goals.Content,
			"Goals content should match the pre-confirmed goals")

		// Verify goals were persisted to the store
		s := store.NewExecutionStore()
		record, err := s.Get(context.Background(), exec.ID)
		require.NoError(t, err)
		require.NotNil(t, record)

		// Execution name should reflect the goals (not "Preparing...")
		assert.NotEmpty(t, exec.Name, "Execution name should be set from goals")
		assert.NotEqual(t, "Preparing...", exec.Name, "Name should not be the default placeholder")

		// Cleanup
		_ = s.Delete(context.Background(), exec.ID)
		t.Logf("✓ Goals injected from TriggerInput.Data: goals=%q, name=%q",
			exec.Goals.Content, exec.Name)
	})

	t.Run("empty_goals_falls_through_to_goals_agent", func(t *testing.T) {
		// When TriggerInput.Data["goals"] is an empty string, the executor
		// does NOT inject pre-confirmed goals and falls through to RunGoals.
		// RunGoals will call the Goals Agent (LLM), which may succeed or fail
		// depending on the environment. We only verify that the executor returns
		// without a panic and that no pre-confirmed goals were force-injected.
		//
		// This test requires a running AI environment; skip in short mode.
		if testing.Short() {
			t.Skip("Skipping: requires LLM for RunGoals fallback")
		}

		ctx := robottypes.NewContext(context.Background(), &oauthTypes.AuthorizedInfo{
			UserID: "user_goals_empty_002",
			TeamID: "team_goals_empty_002",
		})

		robot := createPersistenceTestRobot("member_goals_empty_002", "team_goals_empty_002")

		e := standard.NewWithConfig(types.Config{
			SkipPersistence: true,
		})

		triggerInput := &robottypes.TriggerInput{
			Data: map[string]interface{}{
				"goals": "", // empty — should not be injected as pre-confirmed
			},
		}

		exec, err := e.Execute(ctx, robot, robottypes.TriggerHuman, triggerInput)
		require.NoError(t, err)
		require.NotNil(t, exec)

		// If Goals was set, it came from the Goals Agent, NOT from the empty string injection.
		// Either nil (agent skipped) or non-nil (agent ran) is acceptable.
		if exec.Goals != nil {
			assert.NotEmpty(t, exec.Goals.Content,
				"If Goals Agent ran, content should be non-empty")
		}
		t.Logf("✓ Empty goals falls through to Goals Agent (goals=%v)", exec.Goals != nil)
	})

	t.Run("no_trigger_input_uses_normal_flow", func(t *testing.T) {
		ctx := robottypes.NewContext(context.Background(), &oauthTypes.AuthorizedInfo{
			UserID: "user_goals_normal_001",
			TeamID: "team_goals_normal_001",
		})

		robot := createPersistenceTestRobot("member_goals_normal_001", "team_goals_normal_001")

		e := standard.NewWithConfig(types.Config{
			SkipPersistence: true,
		})

		// No TriggerInput — simulate plain string fallback (old API usage)
		exec, err := e.Execute(ctx, robot, robottypes.TriggerHuman, "simulate_failure")
		require.NoError(t, err)
		require.NotNil(t, exec)

		// Goals nil is expected — RunGoals would normally call the LLM
		assert.Nil(t, exec.Goals, "Without pre-confirmed goals, Goals should remain nil")
		t.Logf("✓ Normal flow (no pre-confirmed goals) proceeds without injection")
	})
}

// ============================================================================
// Helper Functions
// ============================================================================

func createPersistenceTestRobot(memberID, teamID string) *robottypes.Robot {
	return &robottypes.Robot{
		MemberID:       memberID,
		TeamID:         teamID,
		DisplayName:    "Persistence Test Robot",
		Status:         robottypes.RobotIdle,
		AutonomousMode: true,
		Config: &robottypes.Config{
			Identity: &robottypes.Identity{
				Role:   "Test Robot",
				Duties: []string{"Testing persistence"},
			},
			Quota: &robottypes.Quota{
				Max:   5,
				Queue: 10,
			},
			Triggers: &robottypes.Triggers{
				Intervene: &robottypes.TriggerSwitch{Enabled: true},
			},
			Resources: &robottypes.Resources{
				Phases: map[robottypes.Phase]string{
					robottypes.PhaseInspiration: "robot.inspiration",
					robottypes.PhaseGoals:       "robot.goals",
					robottypes.PhaseTasks:       "robot.tasks",
					robottypes.PhaseRun:         "robot.validation",
					"validation":                "robot.validation",
				},
				Agents: []string{"experts.text-writer", "experts.data-analyst"},
			},
		},
	}
}
