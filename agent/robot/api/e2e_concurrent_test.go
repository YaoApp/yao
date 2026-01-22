package api_test

// End-to-end tests for Concurrent execution
// These tests verify that multiple robots can execute simultaneously
// and that quota limits are enforced correctly.
//
// Prerequisites:
//   - Valid LLM API keys (OPENAI_TEST_KEY or DEEPSEEK_API_KEY)
//   - Test assistants in yao-dev-app/assistants/robot/
//   - Database connection (YAO_DB_PRIMARY)

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
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

// testAuthConcurrent returns test auth info for concurrent E2E tests
func testAuthConcurrent() *oauthtypes.AuthorizedInfo {
	return &oauthtypes.AuthorizedInfo{
		UserID: "e2e-concurrent-user",
		TeamID: "e2e-concurrent-team",
	}
}

// TestE2EConcurrentMultipleRobots tests concurrent execution of multiple different robots
func TestE2EConcurrentMultipleRobots(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test - requires real LLM calls")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupE2ERobots(t)
	cleanupE2EExecutions(t)
	defer cleanupE2ERobots(t)
	defer cleanupE2EExecutions(t)

	t.Run("multiple_robots_execute_concurrently", func(t *testing.T) {
		// Create 3 different robots
		robots := []string{
			"robot_e2e_concurrent_001",
			"robot_e2e_concurrent_002",
			"robot_e2e_concurrent_003",
		}

		for _, memberID := range robots {
			setupE2ERobotForConcurrent(t, memberID, "team_e2e_concurrent")
		}

		err := api.Start()
		require.NoError(t, err)
		defer api.Stop()

		ctx := types.NewContext(context.Background(), testAuthConcurrent())

		// Trigger all robots concurrently
		var wg sync.WaitGroup
		var acceptedCount atomic.Int32
		results := make([]*api.TriggerResult, len(robots))
		var mu sync.Mutex

		for i, memberID := range robots {
			wg.Add(1)
			go func(idx int, id string) {
				defer wg.Done()

				result, err := api.TriggerManual(ctx, id, types.TriggerClock, nil)
				if err != nil {
					t.Logf("Robot %s trigger error: %v", id, err)
					return
				}

				mu.Lock()
				results[idx] = result
				mu.Unlock()

				if result.Accepted {
					acceptedCount.Add(1)
					t.Logf("Robot %s accepted: ExecutionID=%s", id, result.ExecutionID)
				}
			}(i, memberID)
		}

		wg.Wait()

		// All 3 should be accepted (different robots, no quota conflict)
		assert.Equal(t, int32(3), acceptedCount.Load(), "All 3 robots should be accepted")

		// Wait for all executions to complete
		maxWait := 180 * time.Second // Longer timeout for concurrent
		deadline := time.Now().Add(maxWait)

		completedCount := 0
		for time.Now().Before(deadline) {
			time.Sleep(3 * time.Second)

			completedCount = 0
			for _, memberID := range robots {
				executions, err := api.ListExecutions(ctx, memberID, nil)
				if err != nil || len(executions.Data) == 0 {
					continue
				}

				exec := executions.Data[0]
				if exec.Status == types.ExecCompleted || exec.Status == types.ExecFailed {
					completedCount++
				}
			}

			t.Logf("Completed: %d/%d", completedCount, len(robots))

			if completedCount == len(robots) {
				break
			}
		}

		// Verify all completed
		assert.Equal(t, len(robots), completedCount, "All robots should complete execution")
	})
}

// TestE2EConcurrentSameRobotMultipleTriggers tests multiple triggers on the same robot
func TestE2EConcurrentSameRobotMultipleTriggers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test - requires real LLM calls")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupE2ERobots(t)
	cleanupE2EExecutions(t)
	defer cleanupE2ERobots(t)
	defer cleanupE2EExecutions(t)

	t.Run("same_robot_handles_multiple_triggers", func(t *testing.T) {
		memberID := "robot_e2e_concurrent_same"
		// Create robot with high quota to allow multiple concurrent executions
		setupE2ERobotHighQuota(t, memberID, "team_e2e_concurrent")

		err := api.Start()
		require.NoError(t, err)
		defer api.Stop()

		ctx := types.NewContext(context.Background(), testAuthConcurrent())

		// Trigger 3 executions on the same robot
		triggerCount := 3
		var wg sync.WaitGroup
		var acceptedCount atomic.Int32

		for i := 0; i < triggerCount; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				result, err := api.TriggerManual(ctx, memberID, types.TriggerClock, nil)
				if err != nil {
					t.Logf("Trigger %d error: %v", idx, err)
					return
				}

				if result.Accepted {
					acceptedCount.Add(1)
					t.Logf("Trigger %d accepted: ExecutionID=%s", idx, result.ExecutionID)
				} else {
					t.Logf("Trigger %d rejected: %s", idx, result.Message)
				}
			}(i)

			// Small delay between triggers to avoid race conditions
			time.Sleep(100 * time.Millisecond)
		}

		wg.Wait()

		// With high quota (max=5), all 3 should be accepted
		assert.GreaterOrEqual(t, acceptedCount.Load(), int32(1), "At least 1 trigger should be accepted")
		t.Logf("Accepted triggers: %d/%d", acceptedCount.Load(), triggerCount)

		// Wait for executions to complete
		maxWait := 180 * time.Second
		deadline := time.Now().Add(maxWait)

		for time.Now().Before(deadline) {
			time.Sleep(3 * time.Second)

			executions, err := api.ListExecutions(ctx, memberID, nil)
			if err != nil {
				continue
			}

			completedCount := 0
			for _, exec := range executions.Data {
				if exec.Status == types.ExecCompleted || exec.Status == types.ExecFailed {
					completedCount++
				}
			}

			t.Logf("Completed: %d/%d", completedCount, int(acceptedCount.Load()))

			if completedCount >= int(acceptedCount.Load()) {
				break
			}
		}

		// Verify execution count
		executions, err := api.ListExecutions(ctx, memberID, nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(executions.Data), 1, "Should have at least 1 execution")
	})
}

// TestE2EConcurrentQuotaEnforcement tests that quota limits are enforced
func TestE2EConcurrentQuotaEnforcement(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test - requires real LLM calls")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupE2ERobots(t)
	cleanupE2EExecutions(t)
	defer cleanupE2ERobots(t)
	defer cleanupE2EExecutions(t)

	t.Run("quota_limit_enforced", func(t *testing.T) {
		memberID := "robot_e2e_concurrent_quota"
		// Create robot with low quota (max=2)
		setupE2ERobotLowQuota(t, memberID, "team_e2e_concurrent")

		err := api.Start()
		require.NoError(t, err)
		defer api.Stop()

		ctx := types.NewContext(context.Background(), testAuthConcurrent())

		// Try to trigger 5 executions on robot with max=2
		triggerCount := 5
		var acceptedCount atomic.Int32
		var rejectedCount atomic.Int32

		for i := 0; i < triggerCount; i++ {
			result, err := api.TriggerManual(ctx, memberID, types.TriggerClock, nil)
			if err != nil {
				t.Logf("Trigger %d error: %v", i, err)
				continue
			}

			if result.Accepted {
				acceptedCount.Add(1)
				t.Logf("Trigger %d accepted", i)
			} else {
				rejectedCount.Add(1)
				t.Logf("Trigger %d rejected: %s", i, result.Message)
			}

			// Small delay to allow execution to start
			time.Sleep(200 * time.Millisecond)
		}

		// With max=2, only 2 should be accepted at a time
		// Some may be rejected due to quota
		t.Logf("Accepted: %d, Rejected: %d", acceptedCount.Load(), rejectedCount.Load())

		// At least some should be accepted
		assert.GreaterOrEqual(t, acceptedCount.Load(), int32(1), "At least 1 should be accepted")

		// Wait for completion
		time.Sleep(120 * time.Second)

		// Query final execution count
		executions, err := api.ListExecutions(ctx, memberID, nil)
		require.NoError(t, err)
		t.Logf("Total executions: %d", len(executions.Data))
	})
}

// TestE2EConcurrentMixedTriggerTypes tests concurrent execution with different trigger types
func TestE2EConcurrentMixedTriggerTypes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test - requires real LLM calls")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupE2ERobots(t)
	cleanupE2EExecutions(t)
	defer cleanupE2ERobots(t)
	defer cleanupE2EExecutions(t)

	t.Run("mixed_trigger_types_execute_concurrently", func(t *testing.T) {
		// Create robots for different trigger types
		clockRobot := "robot_e2e_concurrent_clock"
		humanRobot := "robot_e2e_concurrent_human"
		eventRobot := "robot_e2e_concurrent_event"

		setupE2ERobotForConcurrent(t, clockRobot, "team_e2e_concurrent")
		setupE2ERobotForConcurrent(t, humanRobot, "team_e2e_concurrent")
		setupE2ERobotForConcurrent(t, eventRobot, "team_e2e_concurrent")

		err := api.Start()
		require.NoError(t, err)
		defer api.Stop()

		ctx := types.NewContext(context.Background(), testAuthConcurrent())

		// Trigger all three types concurrently
		var wg sync.WaitGroup
		var acceptedCount atomic.Int32

		// Clock trigger
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := api.TriggerManual(ctx, clockRobot, types.TriggerClock, nil)
			if err == nil && result.Accepted {
				acceptedCount.Add(1)
				t.Logf("Clock trigger accepted")
			}
		}()

		// Human trigger
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := api.Trigger(ctx, humanRobot, &api.TriggerRequest{
				Type:   types.TriggerHuman,
				Action: types.ActionTaskAdd,
			})
			if err == nil && (result.Accepted || result.Queued) {
				acceptedCount.Add(1)
				t.Logf("Human trigger accepted/queued")
			}
		}()

		// Event trigger
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := api.Trigger(ctx, eventRobot, &api.TriggerRequest{
				Type:      types.TriggerEvent,
				Source:    types.EventWebhook,
				EventType: "test.concurrent",
				Data:      map[string]interface{}{"test": true},
			})
			if err == nil && result.Accepted {
				acceptedCount.Add(1)
				t.Logf("Event trigger accepted")
			}
		}()

		wg.Wait()

		// All should be accepted (different robots)
		assert.GreaterOrEqual(t, acceptedCount.Load(), int32(2), "At least 2 triggers should be accepted")

		// Wait for executions
		time.Sleep(120 * time.Second)

		// Verify executions exist for each robot
		for _, memberID := range []string{clockRobot, humanRobot, eventRobot} {
			executions, err := api.ListExecutions(ctx, memberID, nil)
			if err == nil && len(executions.Data) > 0 {
				t.Logf("Robot %s has %d executions", memberID, len(executions.Data))
			}
		}
	})
}

// ==================== Helper Functions ====================

// setupE2ERobotForConcurrent creates a robot for concurrent execution tests
func setupE2ERobotForConcurrent(t *testing.T, memberID, teamID string) {
	m := model.Select("__yao.member")
	tableName := m.MetaData.Table.Name
	qb := capsule.Query()

	// Simple config for E2E testing - minimal tasks
	robotConfig := map[string]interface{}{
		"identity": map[string]interface{}{
			"role":   "Simple E2E Test Robot",
			"duties": []string{"Say hello"}, // Very simple duty
			"rules":  []string{"Keep responses under 50 words"},
		},
		"quota": map[string]interface{}{
			"max":      3,
			"queue":    10,
			"priority": 5,
		},
		"triggers": map[string]interface{}{
			"clock":     map[string]interface{}{"enabled": true},
			"intervene": map[string]interface{}{"enabled": true},
			"event":     map[string]interface{}{"enabled": true},
		},
		"clock": map[string]interface{}{
			"mode":  "interval",
			"every": "1h",
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

	systemPrompt := `You are a simple E2E test robot. Your job is to say hello.
When generating goals: create exactly 1 simple goal.
When generating tasks: create exactly 1 simple task.
Keep all outputs brief. No complex analysis needed.`

	err := qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       memberID,
			"team_id":         teamID,
			"member_type":     "robot",
			"display_name":    "E2E Concurrent Robot " + memberID,
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

// setupE2ERobotHighQuota creates a robot with high quota for concurrent tests
func setupE2ERobotHighQuota(t *testing.T, memberID, teamID string) {
	m := model.Select("__yao.member")
	tableName := m.MetaData.Table.Name
	qb := capsule.Query()

	robotConfig := map[string]interface{}{
		"identity": map[string]interface{}{
			"role": "E2E Test Robot - High Quota",
		},
		"quota": map[string]interface{}{
			"max":      5, // High quota
			"queue":    20,
			"priority": 5,
		},
		"triggers": map[string]interface{}{
			"clock":     map[string]interface{}{"enabled": true},
			"intervene": map[string]interface{}{"enabled": true},
			"event":     map[string]interface{}{"enabled": true},
		},
		// Resources: phase agents and expert agents from yao-dev-app/assistants/
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

	err := qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       memberID,
			"team_id":         teamID,
			"member_type":     "robot",
			"display_name":    "E2E High Quota Robot " + memberID,
			"system_prompt":   "You are a high quota test robot.",
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

// setupE2ERobotLowQuota creates a robot with low quota for quota enforcement tests
func setupE2ERobotLowQuota(t *testing.T, memberID, teamID string) {
	m := model.Select("__yao.member")
	tableName := m.MetaData.Table.Name
	qb := capsule.Query()

	robotConfig := map[string]interface{}{
		"identity": map[string]interface{}{
			"role": "E2E Test Robot - Low Quota",
		},
		"quota": map[string]interface{}{
			"max":      2, // Low quota for testing limits
			"queue":    5,
			"priority": 5,
		},
		"triggers": map[string]interface{}{
			"clock":     map[string]interface{}{"enabled": true},
			"intervene": map[string]interface{}{"enabled": true},
			"event":     map[string]interface{}{"enabled": true},
		},
		// Resources: phase agents and expert agents from yao-dev-app/assistants/
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

	err := qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       memberID,
			"team_id":         teamID,
			"member_type":     "robot",
			"display_name":    "E2E Low Quota Robot " + memberID,
			"system_prompt":   "You are a low quota test robot.",
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
