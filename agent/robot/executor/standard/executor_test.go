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

	testutils.Prepare(t)
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
