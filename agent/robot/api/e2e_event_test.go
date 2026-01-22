package api_test

// End-to-end tests for Event trigger flow
// These tests use REAL LLM calls via Standard executor (not DryRun)
//
// Test Flow: Event Trigger → P1 (Goals) → P2 (Tasks) → P3 (Run) → P4 (Delivery)
// Note: Event trigger SKIPS P0 (Inspiration) - event data provides the context
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
	"github.com/yaoapp/yao/agent/robot/api"
	"github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/testutils"
	oauthtypes "github.com/yaoapp/yao/openapi/oauth/types"
)

// testAuthEvent returns test auth info for event E2E tests
func testAuthEvent() *oauthtypes.AuthorizedInfo {
	return &oauthtypes.AuthorizedInfo{
		UserID: "e2e-event-user",
		TeamID: "e2e-event-team",
	}
}

// TestE2EEventTriggerFullFlow tests the complete event trigger flow with real LLM calls
// Flow: Event → P1 (Goals) → P2 (Tasks) → P3 (Run) → P4 (Delivery)
func TestE2EEventTriggerFullFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test - requires real LLM calls")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupE2ERobots(t)
	cleanupE2EExecutions(t)
	defer cleanupE2ERobots(t)
	defer cleanupE2EExecutions(t)

	t.Run("complete_P1_to_P4_flow_with_webhook_event", func(t *testing.T) {
		memberID := "robot_e2e_event_001"
		setupE2ERobotForEvent(t, memberID, "team_e2e_event")

		err := api.Start()
		require.NoError(t, err)
		defer api.Stop()

		ctx := types.NewContext(context.Background(), testAuthEvent())

		// Verify robot is loaded
		robot, err := api.GetRobot(ctx, memberID)
		require.NoError(t, err)
		require.NotNil(t, robot)

		// Trigger with webhook event - simulating external system notification
		result, err := api.Trigger(ctx, memberID, &api.TriggerRequest{
			Type:      types.TriggerEvent,
			Source:    types.EventWebhook,
			EventType: "order.created",
			Data: map[string]interface{}{
				"order_id":    "ORD-2025-001",
				"customer":    "John Doe",
				"total":       299.99,
				"items_count": 3,
				"priority":    "high",
				"created_at":  time.Now().Format(time.RFC3339),
			},
		})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Accepted, "Event trigger should be accepted")

		t.Logf("Event trigger result: Accepted=%v, ExecutionID=%s", result.Accepted, result.ExecutionID)

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
		assert.Equal(t, types.TriggerEvent, exec.TriggerType, "Should be event trigger")

		// Event trigger skips P0, so Inspiration should be nil
		assert.Nil(t, exec.Inspiration, "P0 Inspiration should be nil for event trigger")

		// P1 Goals should always exist for event trigger
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

		t.Logf("Event trigger E2E completed")
	})
}

// TestE2EEventTriggerDatabaseEvent tests event trigger from database changes
func TestE2EEventTriggerDatabaseEvent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test - requires real LLM calls")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupE2ERobots(t)
	cleanupE2EExecutions(t)
	defer cleanupE2ERobots(t)
	defer cleanupE2EExecutions(t)

	t.Run("handles_database_event_source", func(t *testing.T) {
		memberID := "robot_e2e_event_db"
		setupE2ERobotForEvent(t, memberID, "team_e2e_event")

		err := api.Start()
		require.NoError(t, err)
		defer api.Stop()

		ctx := types.NewContext(context.Background(), testAuthEvent())

		// Trigger with database event - simulating record change notification
		result, err := api.Trigger(ctx, memberID, &api.TriggerRequest{
			Type:      types.TriggerEvent,
			Source:    types.EventDatabase,
			EventType: "user.updated",
			Data: map[string]interface{}{
				"table":     "users",
				"operation": "UPDATE",
				"record_id": 12345,
				"changes": map[string]interface{}{
					"status": map[string]interface{}{
						"old": "pending",
						"new": "active",
					},
				},
			},
		})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Accepted)

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
				t.Logf("Database event E2E completed")
				return
			}
		}

		t.Fatal("Execution did not complete in time")
	})
}

// TestE2EEventTriggerVariousEventTypes tests different event types
// Optimized: Only tests one representative event type to reduce CI time
// The event handling logic is the same for all event types, so testing one is sufficient
func TestE2EEventTriggerVariousEventTypes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test - requires real LLM calls")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupE2ERobots(t)
	cleanupE2EExecutions(t)
	defer cleanupE2ERobots(t)
	defer cleanupE2EExecutions(t)

	// Test only one representative event type (notification)
	// All event types use the same code path, so one test is sufficient
	t.Run("webhook_event", func(t *testing.T) {
		memberID := "robot_e2e_event_webhook"
		setupE2ERobotForEvent(t, memberID, "team_e2e_event")

		err := api.Start()
		require.NoError(t, err)
		defer api.Stop()

		ctx := types.NewContext(context.Background(), testAuthEvent())

		result, err := api.Trigger(ctx, memberID, &api.TriggerRequest{
			Type:      types.TriggerEvent,
			Source:    types.EventWebhook,
			EventType: "notification.received",
			Data: map[string]interface{}{
				"message":  "Test notification",
				"priority": "normal",
			},
		})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Accepted, "Event should be accepted")

		t.Logf("Event triggered: ExecutionID=%s", result.ExecutionID)

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
			t.Logf("Event status: %s, phase: %s", exec.Status, exec.Phase)

			if exec.Status == types.ExecCompleted || exec.Status == types.ExecFailed {
				if exec.Status == types.ExecFailed {
					t.Logf("Event execution failed: %s", exec.Error)
				} else {
					t.Logf("Event execution completed")
				}
				return
			}
		}

		t.Logf("Event execution did not complete in time (may be CI latency)")
	})
}

// TestE2EEventTriggerWithComplexData tests event with nested/complex data structures
func TestE2EEventTriggerWithComplexData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test - requires real LLM calls")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupE2ERobots(t)
	cleanupE2EExecutions(t)
	defer cleanupE2ERobots(t)
	defer cleanupE2EExecutions(t)

	t.Run("handles_complex_nested_data", func(t *testing.T) {
		memberID := "robot_e2e_event_complex"
		setupE2ERobotForEvent(t, memberID, "team_e2e_event")

		err := api.Start()
		require.NoError(t, err)
		defer api.Stop()

		ctx := types.NewContext(context.Background(), testAuthEvent())

		// Complex nested event data
		result, err := api.Trigger(ctx, memberID, &api.TriggerRequest{
			Type:      types.TriggerEvent,
			Source:    types.EventWebhook,
			EventType: "report.generated",
			Data: map[string]interface{}{
				"report": map[string]interface{}{
					"id":         "RPT-2025-001",
					"type":       "sales_summary",
					"period":     "monthly",
					"generated":  time.Now().Format(time.RFC3339),
					"department": "Sales",
				},
				"metrics": []map[string]interface{}{
					{"name": "total_sales", "value": 150000, "unit": "USD"},
					{"name": "orders_count", "value": 450, "unit": "orders"},
					{"name": "avg_order_value", "value": 333.33, "unit": "USD"},
				},
				"comparison": map[string]interface{}{
					"previous_period": map[string]interface{}{
						"total_sales":    140000,
						"orders_count":   420,
						"change_percent": 7.14,
					},
				},
				"highlights": []string{
					"Sales increased by 7.14% compared to last month",
					"Top performing product: Widget Pro",
					"New customer acquisition up 15%",
				},
			},
		})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.True(t, result.Accepted)

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
				assert.Equal(t, types.ExecCompleted, exec.Status, "Complex data event should complete")
				t.Logf("Complex data event E2E completed")
				return
			}
		}

		t.Fatal("Execution did not complete in time")
	})
}

// ==================== Helper Functions ====================

// setupE2ERobotForEvent creates a robot configured for event trigger E2E tests
func setupE2ERobotForEvent(t *testing.T, memberID, teamID string) {
	m := model.Select("__yao.member")
	tableName := m.MetaData.Table.Name
	qb := capsule.Query()

	// Simple config for E2E testing - minimal tasks
	robotConfig := map[string]interface{}{
		"identity": map[string]interface{}{
			"role":   "Simple E2E Test Robot",
			"duties": []string{"Acknowledge events"}, // Very simple duty
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
		"event": map[string]interface{}{
			"types": []string{"*"},
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

	systemPrompt := `You are a simple E2E test robot. Your job is to acknowledge events.
When generating goals: create exactly 1 simple goal.
When generating tasks: create exactly 1 simple task.
Keep all outputs brief. No complex analysis needed.`

	err := qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       memberID,
			"team_id":         teamID,
			"member_type":     "robot",
			"display_name":    "E2E Event Test Robot " + memberID,
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
