package manager_test

// Integration tests for Human intervention triggers
// Tests Manager.Intervene() with various actions and scenarios

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/xun/capsule"
	agentcontext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/robot/manager"
	"github.com/yaoapp/yao/agent/robot/pool"
	"github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/testutils"
)

// ==================== Human Intervention Tests ====================

// TestIntegrationHumanIntervention tests human intervention trigger flow
func TestIntegrationHumanIntervention(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupIntegrationRobots(t)
	defer cleanupIntegrationRobots(t)

	t.Run("task.add action success", func(t *testing.T) {
		setupInterveneTestRobot(t, "robot_integ_human_add", "team_integ_human")

		config := &manager.Config{
			TickInterval: 100 * time.Millisecond,
			PoolConfig:   &pool.Config{WorkerSize: 3, QueueSize: 20},
		}
		m := manager.NewWithConfig(config)

		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		// Verify robot is loaded into cache
		robot := m.Cache().Get("robot_integ_human_add")
		require.NotNil(t, robot, "Robot should be loaded into cache")

		ctx := types.NewContext(context.Background(), nil)
		req := &types.InterveneRequest{
			TeamID:   "team_integ_human",
			MemberID: "robot_integ_human_add",
			Action:   types.ActionTaskAdd,
			Messages: []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: "Add a new task: analyze sales data"},
			},
		}

		result, err := m.Intervene(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.ExecutionID)
		assert.Equal(t, types.ExecPending, result.Status)
		assert.Contains(t, result.Message, "task.add")

		// Wait for execution
		time.Sleep(500 * time.Millisecond)

		// Verify execution completed
		assert.GreaterOrEqual(t, m.Executor().ExecCount(), 1)
	})

	t.Run("goal.adjust action success", func(t *testing.T) {
		setupInterveneTestRobot(t, "robot_integ_human_goal", "team_integ_human")

		m := manager.New()
		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)
		req := &types.InterveneRequest{
			TeamID:   "team_integ_human",
			MemberID: "robot_integ_human_goal",
			Action:   types.ActionGoalAdjust,
			Messages: []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: "Focus on high-priority customers only"},
			},
		}

		result, err := m.Intervene(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.ExecutionID)
	})

	t.Run("instruct action success", func(t *testing.T) {
		setupInterveneTestRobot(t, "robot_integ_human_instruct", "team_integ_human")

		m := manager.New()
		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)
		req := &types.InterveneRequest{
			TeamID:   "team_integ_human",
			MemberID: "robot_integ_human_instruct",
			Action:   types.ActionInstruct,
			Messages: []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: "Generate a weekly report"},
			},
		}

		result, err := m.Intervene(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.ExecutionID)
	})
}

// TestIntegrationHumanInterventionErrors tests error cases for human intervention
func TestIntegrationHumanInterventionErrors(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupIntegrationRobots(t)
	defer cleanupIntegrationRobots(t)

	t.Run("robot not found", func(t *testing.T) {
		m := manager.New()
		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)
		req := &types.InterveneRequest{
			MemberID: "robot_nonexistent",
			Action:   types.ActionTaskAdd,
			Messages: []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: "Test"},
			},
		}

		_, err = m.Intervene(ctx, req)
		assert.Error(t, err)
		assert.Equal(t, types.ErrRobotNotFound, err)
	})

	t.Run("robot paused", func(t *testing.T) {
		setupInterveneTestRobotPaused(t, "robot_integ_human_paused", "team_integ_human")

		m := manager.New()
		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)
		req := &types.InterveneRequest{
			MemberID: "robot_integ_human_paused",
			Action:   types.ActionTaskAdd,
			Messages: []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: "Test"},
			},
		}

		_, err = m.Intervene(ctx, req)
		assert.Error(t, err)
		assert.Equal(t, types.ErrRobotPaused, err)
	})

	t.Run("intervene trigger disabled", func(t *testing.T) {
		setupInterveneTestRobotDisabled(t, "robot_integ_human_disabled", "team_integ_human")

		m := manager.New()
		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)
		req := &types.InterveneRequest{
			MemberID: "robot_integ_human_disabled",
			Action:   types.ActionTaskAdd,
			Messages: []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: "Test"},
			},
		}

		_, err = m.Intervene(ctx, req)
		assert.Error(t, err)
		assert.Equal(t, types.ErrTriggerDisabled, err)
	})

	t.Run("invalid request - empty member_id", func(t *testing.T) {
		m := manager.New()
		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)
		req := &types.InterveneRequest{
			MemberID: "", // Empty
			Action:   types.ActionTaskAdd,
		}

		_, err = m.Intervene(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "member_id")
	})

	t.Run("invalid request - empty action", func(t *testing.T) {
		setupInterveneTestRobot(t, "robot_integ_human_noaction", "team_integ_human")

		m := manager.New()
		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)
		req := &types.InterveneRequest{
			MemberID: "robot_integ_human_noaction",
			Action:   "", // Empty action
		}

		_, err = m.Intervene(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "action")
	})

	t.Run("manager not started", func(t *testing.T) {
		m := manager.New()
		// Don't start

		ctx := types.NewContext(context.Background(), nil)
		req := &types.InterveneRequest{
			MemberID: "robot_test",
			Action:   types.ActionTaskAdd,
		}

		_, err := m.Intervene(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not started")
	})
}

// TestIntegrationHumanInterventionMultimodal tests multimodal input support
func TestIntegrationHumanInterventionMultimodal(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupIntegrationRobots(t)
	defer cleanupIntegrationRobots(t)

	t.Run("text message", func(t *testing.T) {
		setupInterveneTestRobot(t, "robot_integ_human_text", "team_integ_human")

		m := manager.New()
		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)
		req := &types.InterveneRequest{
			MemberID: "robot_integ_human_text",
			Action:   types.ActionTaskAdd,
			Messages: []agentcontext.Message{
				{
					Role:    agentcontext.RoleUser,
					Content: "Analyze the quarterly sales report",
				},
			},
		}

		result, err := m.Intervene(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.ExecutionID)
	})

	t.Run("message with image reference", func(t *testing.T) {
		setupInterveneTestRobot(t, "robot_integ_human_image", "team_integ_human")

		m := manager.New()
		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)
		req := &types.InterveneRequest{
			MemberID: "robot_integ_human_image",
			Action:   types.ActionTaskAdd,
			Messages: []agentcontext.Message{
				{
					Role: agentcontext.RoleUser,
					Content: []interface{}{
						map[string]interface{}{
							"type": "text",
							"text": "Analyze this chart",
						},
						map[string]interface{}{
							"type": "image_url",
							"image_url": map[string]interface{}{
								"url": "https://example.com/chart.png",
							},
						},
					},
				},
			},
		}

		result, err := m.Intervene(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.ExecutionID)
	})

	t.Run("multiple messages", func(t *testing.T) {
		setupInterveneTestRobot(t, "robot_integ_human_multi", "team_integ_human")

		m := manager.New()
		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)
		req := &types.InterveneRequest{
			MemberID: "robot_integ_human_multi",
			Action:   types.ActionTaskAdd,
			Messages: []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: "First, check the sales data"},
				{Role: agentcontext.RoleUser, Content: "Then, prepare a summary report"},
			},
		}

		result, err := m.Intervene(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.ExecutionID)
	})
}

// TestIntegrationHumanInterventionAllActions tests all intervention actions
func TestIntegrationHumanInterventionAllActions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupIntegrationRobots(t)
	defer cleanupIntegrationRobots(t)

	// Test all defined actions
	actions := []types.InterventionAction{
		types.ActionTaskAdd,
		types.ActionTaskCancel,
		types.ActionTaskUpdate,
		types.ActionGoalAdjust,
		types.ActionGoalAdd,
		types.ActionGoalComplete,
		types.ActionGoalCancel,
		types.ActionInstruct,
		// Note: plan.add, plan.remove, plan.update are handled differently
	}

	for _, action := range actions {
		t.Run(string(action), func(t *testing.T) {
			memberID := "robot_integ_action_" + string(action)
			setupInterveneTestRobot(t, memberID, "team_integ_human")

			m := manager.New()
			err := m.Start()
			require.NoError(t, err)
			defer m.Stop()

			ctx := types.NewContext(context.Background(), nil)
			req := &types.InterveneRequest{
				MemberID: memberID,
				Action:   action,
				Messages: []agentcontext.Message{
					{Role: agentcontext.RoleUser, Content: "Test action: " + string(action)},
				},
			}

			result, err := m.Intervene(ctx, req)
			assert.NoError(t, err, "Action %s should succeed", action)
			assert.NotNil(t, result)
			assert.NotEmpty(t, result.ExecutionID)
			assert.Equal(t, types.ExecPending, result.Status)
		})
	}
}

// TestIntegrationHumanInterventionPlanAdd tests plan.add action (deferred execution)
func TestIntegrationHumanInterventionPlanAdd(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupIntegrationRobots(t)
	defer cleanupIntegrationRobots(t)

	t.Run("plan.add with future time", func(t *testing.T) {
		setupInterveneTestRobot(t, "robot_integ_human_plan", "team_integ_human")

		m := manager.New()
		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)
		planTime := time.Now().Add(1 * time.Hour)
		req := &types.InterveneRequest{
			MemberID: "robot_integ_human_plan",
			Action:   types.ActionPlanAdd,
			Messages: []agentcontext.Message{
				{Role: agentcontext.RoleUser, Content: "Send weekly report"},
			},
			PlanTime: &planTime,
		}

		result, err := m.Intervene(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, types.ExecPending, result.Status)
		assert.Contains(t, result.Message, "Planned")
		// Note: Plan queue not implemented yet, so execution is deferred
	})
}

// ==================== Test Data Setup Helpers ====================

// setupInterveneTestRobot creates a robot with intervene trigger enabled
func setupInterveneTestRobot(t *testing.T, memberID, teamID string) {
	m := model.Select("__yao.member")
	tableName := m.MetaData.Table.Name
	qb := capsule.Query()

	robotConfig := map[string]interface{}{
		"identity": map[string]interface{}{
			"role":   "Intervene Test Robot",
			"duties": []string{"Handle human interventions"},
		},
		"quota": map[string]interface{}{
			"max":      5,
			"queue":    20,
			"priority": 5,
		},
		"triggers": map[string]interface{}{
			"clock":     map[string]interface{}{"enabled": false},
			"intervene": map[string]interface{}{"enabled": true},
			"event":     map[string]interface{}{"enabled": false},
		},
	}
	configJSON, _ := json.Marshal(robotConfig)

	err := qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       memberID,
			"team_id":         teamID,
			"member_type":     "robot",
			"display_name":    "Intervene Test Robot " + memberID,
			"status":          "active",
			"role_id":         "member",
			"autonomous_mode": true,
			"robot_status":    "idle",
			"robot_config":    string(configJSON),
		},
	})
	if err != nil {
		t.Fatalf("Failed to insert %s: %v", memberID, err)
	}
}

// setupInterveneTestRobotPaused creates a paused robot
func setupInterveneTestRobotPaused(t *testing.T, memberID, teamID string) {
	m := model.Select("__yao.member")
	tableName := m.MetaData.Table.Name
	qb := capsule.Query()

	robotConfig := map[string]interface{}{
		"identity": map[string]interface{}{"role": "Paused Robot"},
		"triggers": map[string]interface{}{
			"intervene": map[string]interface{}{"enabled": true},
		},
	}
	configJSON, _ := json.Marshal(robotConfig)

	err := qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       memberID,
			"team_id":         teamID,
			"member_type":     "robot",
			"display_name":    "Paused Robot " + memberID,
			"status":          "active",
			"role_id":         "member",
			"autonomous_mode": true,
			"robot_status":    "paused", // Paused
			"robot_config":    string(configJSON),
		},
	})
	if err != nil {
		t.Fatalf("Failed to insert %s: %v", memberID, err)
	}
}

// setupInterveneTestRobotDisabled creates a robot with intervene trigger disabled
func setupInterveneTestRobotDisabled(t *testing.T, memberID, teamID string) {
	m := model.Select("__yao.member")
	tableName := m.MetaData.Table.Name
	qb := capsule.Query()

	robotConfig := map[string]interface{}{
		"identity": map[string]interface{}{"role": "Intervene Disabled Robot"},
		"triggers": map[string]interface{}{
			"intervene": map[string]interface{}{"enabled": false}, // Disabled
		},
	}
	configJSON, _ := json.Marshal(robotConfig)

	err := qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       memberID,
			"team_id":         teamID,
			"member_type":     "robot",
			"display_name":    "Intervene Disabled Robot " + memberID,
			"status":          "active",
			"role_id":         "member",
			"autonomous_mode": true,
			"robot_status":    "idle",
			"robot_config":    string(configJSON),
		},
	})
	if err != nil {
		t.Fatalf("Failed to insert %s: %v", memberID, err)
	}
}
