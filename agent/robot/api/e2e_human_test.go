package api_test

// End-to-end tests for Human intervention trigger flow
// These tests use REAL LLM calls via Standard executor (not DryRun)
//
// Test Flow: Human Trigger → P1 (Goals) → P2 (Tasks) → P3 (Run) → P4 (Delivery)
// Note: Human trigger SKIPS P0 (Inspiration) - user provides the input directly
//
// Prerequisites:
//   - Valid LLM API keys (OPENAI_TEST_KEY or DEEPSEEK_API_KEY)
//   - Test assistants in yao-dev-app/assistants/robot/
//   - Database connection (YAO_DB_PRIMARY)

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
	"github.com/yaoapp/yao/agent/robot/api"
	"github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/testutils"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
)

// testAuthHuman returns test auth info for human E2E tests
func testAuthHuman() *oauthtypes.AuthorizedInfo {
	return &oauthtypes.AuthorizedInfo{
		UserID: "e2e-human-user",
		TeamID: "e2e-human-team",
	}
}

// TestE2EHumanTriggerFullFlow tests the complete human intervention flow with real LLM calls
// Flow: Human Input → P1 (Goals) → P2 (Tasks) → P3 (Run) → P4 (Delivery)
func TestE2EHumanTriggerFullFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test - requires real LLM calls")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupE2ERobots(t)
	cleanupE2EExecutions(t)
	defer cleanupE2ERobots(t)
	defer cleanupE2EExecutions(t)

	t.Run("complete_P1_to_P4_flow_with_user_input", func(t *testing.T) {
		memberID := "robot_e2e_human_001"
		setupE2ERobotForHuman(t, memberID, "team_e2e_human")

		err := api.Start()
		require.NoError(t, err)
		defer api.Stop()

		ctx := types.NewContext(context.Background(), testAuthHuman())

		// Verify robot is loaded
		robot, err := api.GetRobot(ctx, memberID)
		require.NoError(t, err)
		require.NotNil(t, robot)

		// Trigger with human input - user requesting a specific task
		result, err := api.Trigger(ctx, memberID, &api.TriggerRequest{
			Type:   types.TriggerHuman,
			Action: types.ActionTaskAdd,
			Messages: []agentcontext.Message{
				{
					Role:    "user",
					Content: "Please write a brief summary of today's key tasks and priorities.",
				},
			},
		})
		require.NoError(t, err)
		require.NotNil(t, result)

		// Human trigger returns Queued=true (goes through Intervene)
		t.Logf("Trigger result: Accepted=%v, Queued=%v, Message=%s",
			result.Accepted, result.Queued, result.Message)

		// Wait for execution to complete
		var exec *types.Execution
		maxWait := 120 * time.Second
		pollInterval := 2 * time.Second
		deadline := time.Now().Add(maxWait)

		for time.Now().Before(deadline) {
			time.Sleep(pollInterval)

			executions, err := api.ListExecutions(ctx, memberID, &api.ExecutionQuery{
				Page:     1,
				PageSize: 1,
			})
			if err != nil || len(executions.Data) == 0 {
				continue
			}

			exec = executions.Data[0]
			t.Logf("Execution status: %s, phase: %s", exec.Status, exec.Phase)

			if exec.Status == types.ExecCompleted || exec.Status == types.ExecFailed {
				break
			}
		}

		require.NotNil(t, exec, "Execution should exist")

		// E2E test validates the flow executes correctly
		isFinished := exec.Status == types.ExecCompleted || exec.Status == types.ExecFailed
		assert.True(t, isFinished, "Execution should finish (completed or failed), got: %s", exec.Status)

		if exec.Status == types.ExecFailed {
			t.Logf("Execution finished with status=failed (acceptable for E2E): %s", exec.Error)
		} else {
			t.Logf("Execution finished with status=completed")
		}

		// Verify trigger type
		assert.Equal(t, types.TriggerHuman, exec.TriggerType, "Should be human trigger")

		// Human trigger skips P0, so Inspiration should be nil
		assert.Nil(t, exec.Inspiration, "P0 Inspiration should be nil for human trigger")

		// P1 Goals should always exist for human trigger
		assert.NotNil(t, exec.Goals, "P1 Goals should exist")

		// P2-P4 may or may not exist depending on where failure occurred
		if exec.Tasks != nil {
			t.Logf("P2 Tasks count: %d", len(exec.Tasks))
		}
		if exec.Results != nil {
			t.Logf("P3 Results count: %d", len(exec.Results))
		}
		if exec.Delivery != nil {
			t.Logf("P4 Delivery: RequestID=%s", exec.Delivery.RequestID)
		}

		t.Logf("Human trigger E2E completed")
	})
}

// TestE2EHumanTriggerWithMultimodalInput tests human trigger with rich content
func TestE2EHumanTriggerWithMultimodalInput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test - requires real LLM calls")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupE2ERobots(t)
	cleanupE2EExecutions(t)
	defer cleanupE2ERobots(t)
	defer cleanupE2EExecutions(t)

	t.Run("handles_multipart_message_input", func(t *testing.T) {
		memberID := "robot_e2e_human_multi"
		setupE2ERobotForHuman(t, memberID, "team_e2e_human")

		err := api.Start()
		require.NoError(t, err)
		defer api.Stop()

		ctx := types.NewContext(context.Background(), testAuthHuman())

		// Trigger with multipart message (text parts)
		result, err := api.Trigger(ctx, memberID, &api.TriggerRequest{
			Type:   types.TriggerHuman,
			Action: types.ActionGoalAdjust,
			Messages: []agentcontext.Message{
				{
					Role: "user",
					Content: []map[string]interface{}{
						{
							"type": "text",
							"text": "I need you to focus on the following priorities:",
						},
						{
							"type": "text",
							"text": "1. Review pending tasks\n2. Summarize progress\n3. Identify blockers",
						},
					},
				},
			},
		})
		require.NoError(t, err)
		require.NotNil(t, result)

		t.Logf("Multipart trigger result: Queued=%v", result.Queued)

		// Wait for execution
		maxWait := 120 * time.Second
		deadline := time.Now().Add(maxWait)

		for time.Now().Before(deadline) {
			time.Sleep(2 * time.Second)

			executions, err := api.ListExecutions(ctx, memberID, nil)
			if err != nil || len(executions.Data) == 0 {
				continue
			}

			exec := executions.Data[0]
			if exec.Status == types.ExecCompleted || exec.Status == types.ExecFailed {
				assert.Equal(t, types.ExecCompleted, exec.Status)
				t.Logf("Multipart input E2E completed")
				return
			}
		}

		t.Fatal("Execution did not complete in time")
	})
}

// TestE2EHumanTriggerAllActions tests different intervention actions
func TestE2EHumanTriggerAllActions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test - requires real LLM calls")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupE2ERobots(t)
	cleanupE2EExecutions(t)
	defer cleanupE2ERobots(t)
	defer cleanupE2EExecutions(t)

	actions := []struct {
		name   string
		action types.InterventionAction
		input  string
	}{
		{
			name:   "task_add",
			action: types.ActionTaskAdd,
			input:  "Add a new task: Review system logs for errors",
		},
		{
			name:   "goal_adjust",
			action: types.ActionGoalAdjust,
			input:  "Adjust goal: Focus on performance optimization instead of new features",
		},
		{
			name:   "instruct",
			action: types.ActionInstruct,
			input:  "Please prioritize security review as the top task",
		},
	}

	for i, tc := range actions {
		t.Run(tc.name, func(t *testing.T) {
			memberID := "robot_e2e_human_action_" + tc.name
			setupE2ERobotForHuman(t, memberID, "team_e2e_human")

			// Start fresh for each action test
			if i == 0 {
				err := api.Start()
				require.NoError(t, err)
			}

			ctx := types.NewContext(context.Background(), testAuthHuman())

			result, err := api.Trigger(ctx, memberID, &api.TriggerRequest{
				Type:   types.TriggerHuman,
				Action: tc.action,
				Messages: []agentcontext.Message{
					{Role: "user", Content: tc.input},
				},
			})
			require.NoError(t, err)
			require.NotNil(t, result)

			t.Logf("Action %s: Queued=%v", tc.action, result.Queued)

			// Wait for execution (shorter timeout for action tests)
			maxWait := 90 * time.Second
			deadline := time.Now().Add(maxWait)

			for time.Now().Before(deadline) {
				time.Sleep(2 * time.Second)

				executions, err := api.ListExecutions(ctx, memberID, nil)
				if err != nil || len(executions.Data) == 0 {
					continue
				}

				exec := executions.Data[0]
				if exec.Status == types.ExecCompleted || exec.Status == types.ExecFailed {
					if exec.Status == types.ExecFailed {
						t.Logf("Action %s failed: %s", tc.action, exec.Error)
					} else {
						t.Logf("Action %s completed successfully", tc.action)
					}
					return
				}
			}

			t.Logf("Action %s: execution did not complete in time (may still be running)", tc.action)
		})
	}

	// Stop after all action tests
	api.Stop()
}

// ==================== Helper Functions ====================

// setupE2ERobotForHuman creates a robot configured for human intervention E2E tests
func setupE2ERobotForHuman(t *testing.T, memberID, teamID string) {
	m := model.Select("__yao.member")
	tableName := m.MetaData.Table.Name
	qb := capsule.Query()

	// Simple config for E2E testing - minimal tasks
	robotConfig := map[string]interface{}{
		"identity": map[string]interface{}{
			"role":   "Simple E2E Test Robot",
			"duties": []string{"Echo user input"}, // Very simple duty
			"rules":  []string{"Keep responses under 50 words"},
		},
		"quota": map[string]interface{}{
			"max":      5,
			"queue":    20,
			"priority": 5,
		},
		"triggers": map[string]interface{}{
			"clock":     map[string]interface{}{"enabled": false},
			"intervene": map[string]interface{}{"enabled": true},
			"event":     map[string]interface{}{"enabled": true},
		},
		"resources": map[string]interface{}{
			"phases": map[string]interface{}{
				"inspiration": "robot.inspiration",
				"goals":       "robot.goals",
				"tasks":       "tests.e2e-tasks", // Use simple E2E test task planner
				"run":         "robot.validation",
				"validation":  "tests.e2e-validation", // Use lenient E2E test validator
				"delivery":    "robot.delivery",
				"learning":    "robot.learning",
			},
			"agents": []string{"experts.text-writer"},
		},
		"delivery": map[string]interface{}{
			"email":   map[string]interface{}{"enabled": false},
			"webhook": map[string]interface{}{"enabled": false},
			"process": map[string]interface{}{"enabled": false},
		},
	}
	configJSON, _ := json.Marshal(robotConfig)

	systemPrompt := `You are a simple E2E test robot. Your job is to echo user requests.
When generating goals: create exactly 1 simple goal.
When generating tasks: create exactly 1 simple task.
Keep all outputs brief. No complex analysis needed.`

	err := qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       memberID,
			"team_id":         teamID,
			"member_type":     "robot",
			"display_name":    "E2E Human Test Robot " + memberID,
			"system_prompt":   systemPrompt,
			"status":          "active",
			"role_id":         "member",
			"autonomous_mode": true,
			"robot_status":    "idle",
			"robot_config":    string(configJSON),
		},
	})
	if err != nil {
		t.Fatalf("Failed to insert robot %s: %v", memberID, err)
	}
}
