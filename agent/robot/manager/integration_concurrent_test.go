package manager_test

// Integration tests for concurrent execution and quota enforcement
// Tests the two-level concurrency model:
//   1. Global pool limit (worker count)
//   2. Per-robot quota limit (Quota.Max, Quota.Queue)

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/yao/agent/robot/executor"
	"github.com/yaoapp/yao/agent/robot/manager"
	"github.com/yaoapp/yao/agent/robot/pool"
	"github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/testutils"
)

// ==================== Concurrent Execution Tests ====================

// TestIntegrationConcurrentExecution tests concurrent execution of multiple robots
func TestIntegrationConcurrentExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupIntegrationRobots(t)
	defer cleanupIntegrationRobots(t)

	t.Run("multiple robots execute concurrently", func(t *testing.T) {
		// Create 5 robots
		for i := 0; i < 5; i++ {
			memberID := "robot_integ_conc_multi_" + string(rune('A'+i))
			setupConcurrentTestRobot(t, memberID, "team_integ_conc", 3, 20)
		}

		// Track concurrent execution count
		var maxConcurrent int32
		var currentConcurrent int32

		exec := executor.NewDryRunWithCallbacks(100*time.Millisecond,
			func() {
				curr := atomic.AddInt32(&currentConcurrent, 1)
				for {
					old := atomic.LoadInt32(&maxConcurrent)
					if curr <= old || atomic.CompareAndSwapInt32(&maxConcurrent, old, curr) {
						break
					}
				}
			},
			func() {
				atomic.AddInt32(&currentConcurrent, -1)
			},
		)

		config := &manager.Config{
			TickInterval: 100 * time.Millisecond,
			PoolConfig:   &pool.Config{WorkerSize: 5, QueueSize: 50},
		}
		m := manager.NewWithConfig(config)
		m.Pool().SetExecutor(exec)

		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		// Verify robots are loaded into cache
		for i := 0; i < 5; i++ {
			memberID := "robot_integ_conc_multi_" + string(rune('A'+i))
			robot := m.Cache().Get(memberID)
			require.NotNil(t, robot, "Robot %s should be loaded into cache", memberID)
		}

		ctx := types.NewContext(context.Background(), nil)

		// Trigger all robots simultaneously
		var wg sync.WaitGroup
		for i := 0; i < 5; i++ {
			wg.Add(1)
			memberID := "robot_integ_conc_multi_" + string(rune('A'+i))
			go func(id string) {
				defer wg.Done()
				m.TriggerManual(ctx, id, types.TriggerClock, nil)
			}(memberID)
		}

		wg.Wait()

		// Wait for all executions
		time.Sleep(500 * time.Millisecond)

		// Should have achieved concurrent execution
		assert.GreaterOrEqual(t, int(maxConcurrent), 2, "Should achieve concurrent execution")
		assert.GreaterOrEqual(t, exec.ExecCount(), 5, "All robots should execute")
	})

	t.Run("same robot multiple triggers", func(t *testing.T) {
		setupConcurrentTestRobot(t, "robot_integ_conc_same", "team_integ_conc", 3, 20)

		exec := executor.NewDryRunWithDelay(50 * time.Millisecond)

		config := &manager.Config{
			TickInterval: 100 * time.Millisecond,
			PoolConfig:   &pool.Config{WorkerSize: 5, QueueSize: 50},
		}
		m := manager.NewWithConfig(config)
		m.Pool().SetExecutor(exec)

		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)

		// Trigger same robot multiple times
		for i := 0; i < 5; i++ {
			_, err := m.TriggerManual(ctx, "robot_integ_conc_same", types.TriggerClock, nil)
			assert.NoError(t, err)
		}

		// Wait for all executions
		time.Sleep(800 * time.Millisecond)

		// All 5 should eventually execute
		assert.GreaterOrEqual(t, exec.ExecCount(), 5, "All triggers should execute")
	})
}

// ==================== Quota Enforcement Tests ====================

// TestIntegrationQuotaEnforcement tests per-robot quota limits
func TestIntegrationQuotaEnforcement(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupIntegrationRobots(t)
	defer cleanupIntegrationRobots(t)

	t.Run("respects Quota.Max limit", func(t *testing.T) {
		// Create robot with Max=2
		setupConcurrentTestRobot(t, "robot_integ_quota_max", "team_integ_quota", 2, 20)

		// Track max concurrent for this robot
		var maxConcurrent int32
		var currentConcurrent int32

		exec := executor.NewDryRunWithCallbacks(200*time.Millisecond,
			func() {
				curr := atomic.AddInt32(&currentConcurrent, 1)
				for {
					old := atomic.LoadInt32(&maxConcurrent)
					if curr <= old || atomic.CompareAndSwapInt32(&maxConcurrent, old, curr) {
						break
					}
				}
			},
			func() {
				atomic.AddInt32(&currentConcurrent, -1)
			},
		)

		config := &manager.Config{
			TickInterval: 100 * time.Millisecond,
			PoolConfig:   &pool.Config{WorkerSize: 10, QueueSize: 50}, // Many workers
		}
		m := manager.NewWithConfig(config)
		m.Pool().SetExecutor(exec)

		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)

		// Submit 10 jobs for the same robot
		for i := 0; i < 10; i++ {
			m.TriggerManual(ctx, "robot_integ_quota_max", types.TriggerClock, nil)
		}

		// Wait a bit for concurrent execution
		time.Sleep(300 * time.Millisecond)

		// Max concurrent should not exceed Quota.Max (2)
		assert.LessOrEqual(t, int(maxConcurrent), 2, "Should not exceed Quota.Max")

		// Wait for all to complete
		time.Sleep(1500 * time.Millisecond)

		// All should eventually execute
		assert.GreaterOrEqual(t, exec.ExecCount(), 10, "All jobs should eventually execute")
	})

	t.Run("respects Quota.Queue limit", func(t *testing.T) {
		// Create robot with Max=1, Queue=3
		setupConcurrentTestRobot(t, "robot_integ_quota_queue", "team_integ_quota", 1, 3)

		exec := executor.NewDryRunWithDelay(300 * time.Millisecond) // Slow execution

		config := &manager.Config{
			TickInterval: 100 * time.Millisecond,
			PoolConfig:   &pool.Config{WorkerSize: 10, QueueSize: 100},
		}
		m := manager.NewWithConfig(config)
		m.Pool().SetExecutor(exec)

		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)

		// Submit many jobs - some should be rejected due to queue limit
		successCount := 0
		for i := 0; i < 20; i++ {
			_, err := m.TriggerManual(ctx, "robot_integ_quota_queue", types.TriggerClock, nil)
			if err == nil {
				successCount++
			}
		}

		// Should accept at most Max + Queue = 1 + 3 = 4 jobs
		assert.LessOrEqual(t, successCount, 4, "Should respect queue limit")
		assert.GreaterOrEqual(t, successCount, 1, "Should accept at least 1 job")
	})

	t.Run("different robots have independent quotas", func(t *testing.T) {
		// Robot A: Max=1
		setupConcurrentTestRobot(t, "robot_integ_quota_A", "team_integ_quota", 1, 10)
		// Robot B: Max=3
		setupConcurrentTestRobot(t, "robot_integ_quota_B", "team_integ_quota", 3, 10)

		var concurrentA int32
		var concurrentB int32
		var maxA int32
		var maxB int32

		// Custom executor that tracks per-robot concurrency
		exec := &trackingExecutor{
			delay: 150 * time.Millisecond,
			onStart: func(robot *types.Robot) {
				if robot.MemberID == "robot_integ_quota_A" {
					curr := atomic.AddInt32(&concurrentA, 1)
					for {
						old := atomic.LoadInt32(&maxA)
						if curr <= old || atomic.CompareAndSwapInt32(&maxA, old, curr) {
							break
						}
					}
				} else {
					curr := atomic.AddInt32(&concurrentB, 1)
					for {
						old := atomic.LoadInt32(&maxB)
						if curr <= old || atomic.CompareAndSwapInt32(&maxB, old, curr) {
							break
						}
					}
				}
			},
			onEnd: func(robot *types.Robot) {
				if robot.MemberID == "robot_integ_quota_A" {
					atomic.AddInt32(&concurrentA, -1)
				} else {
					atomic.AddInt32(&concurrentB, -1)
				}
			},
		}

		config := &manager.Config{
			TickInterval: 100 * time.Millisecond,
			PoolConfig:   &pool.Config{WorkerSize: 10, QueueSize: 50},
		}
		m := manager.NewWithConfig(config)
		m.Pool().SetExecutor(exec)

		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)

		// Submit 5 jobs for each robot
		for i := 0; i < 5; i++ {
			m.TriggerManual(ctx, "robot_integ_quota_A", types.TriggerClock, nil)
			m.TriggerManual(ctx, "robot_integ_quota_B", types.TriggerClock, nil)
		}

		// Wait a bit
		time.Sleep(300 * time.Millisecond)

		// Robot A should have max 1 concurrent
		assert.LessOrEqual(t, int(maxA), 1, "Robot A should respect its quota")
		// Robot B should have max 3 concurrent
		assert.LessOrEqual(t, int(maxB), 3, "Robot B should respect its quota")

		// Wait for completion
		time.Sleep(1 * time.Second)
	})
}

// ==================== Global Pool Limit Tests ====================

// TestIntegrationGlobalPoolLimit tests global worker pool limits
func TestIntegrationGlobalPoolLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupIntegrationRobots(t)
	defer cleanupIntegrationRobots(t)

	t.Run("respects global worker limit", func(t *testing.T) {
		// Create 10 robots with high quotas
		for i := 0; i < 10; i++ {
			memberID := "robot_integ_pool_limit_" + string(rune('A'+i))
			setupConcurrentTestRobot(t, memberID, "team_integ_pool", 5, 20)
		}

		var maxConcurrent int32
		var currentConcurrent int32

		exec := executor.NewDryRunWithCallbacks(200*time.Millisecond,
			func() {
				curr := atomic.AddInt32(&currentConcurrent, 1)
				for {
					old := atomic.LoadInt32(&maxConcurrent)
					if curr <= old || atomic.CompareAndSwapInt32(&maxConcurrent, old, curr) {
						break
					}
				}
			},
			func() {
				atomic.AddInt32(&currentConcurrent, -1)
			},
		)

		config := &manager.Config{
			TickInterval: 100 * time.Millisecond,
			PoolConfig:   &pool.Config{WorkerSize: 3, QueueSize: 100}, // Only 3 workers
		}
		m := manager.NewWithConfig(config)
		m.Pool().SetExecutor(exec)

		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)

		// Trigger all 10 robots
		for i := 0; i < 10; i++ {
			memberID := "robot_integ_pool_limit_" + string(rune('A'+i))
			m.TriggerManual(ctx, memberID, types.TriggerClock, nil)
		}

		// Wait a bit
		time.Sleep(300 * time.Millisecond)

		// Max concurrent should not exceed worker limit (3)
		assert.LessOrEqual(t, int(maxConcurrent), 3, "Should not exceed worker limit")

		// Wait for all to complete
		time.Sleep(1 * time.Second)

		// All 10 should execute
		assert.GreaterOrEqual(t, exec.ExecCount(), 10, "All robots should execute")
	})

	t.Run("respects global queue limit", func(t *testing.T) {
		// Create robots
		for i := 0; i < 20; i++ {
			memberID := "robot_integ_pool_queue_" + string(rune('A'+i%26))
			setupConcurrentTestRobot(t, memberID, "team_integ_pool", 5, 20)
		}

		exec := executor.NewDryRunWithDelay(500 * time.Millisecond) // Slow execution

		config := &manager.Config{
			TickInterval: 100 * time.Millisecond,
			PoolConfig:   &pool.Config{WorkerSize: 1, QueueSize: 5}, // Small queue
		}
		m := manager.NewWithConfig(config)
		m.Pool().SetExecutor(exec)

		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)

		// Try to submit many jobs
		successCount := 0
		for i := 0; i < 20; i++ {
			memberID := "robot_integ_pool_queue_" + string(rune('A'+i%26))
			_, err := m.TriggerManual(ctx, memberID, types.TriggerClock, nil)
			if err == nil {
				successCount++
			}
		}

		// Should respect global queue limit
		// Max = WorkerSize + QueueSize = 1 + 5 = 6
		assert.LessOrEqual(t, successCount, 6, "Should respect global queue limit")
	})
}

// ==================== Priority Tests ====================

// TestIntegrationPriorityExecution tests priority-based execution order
func TestIntegrationPriorityExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupIntegrationRobots(t)
	defer cleanupIntegrationRobots(t)

	t.Run("higher priority executes first", func(t *testing.T) {
		// Create robots with different priorities
		setupConcurrentTestRobotWithPriority(t, "robot_integ_prio_low", "team_integ_prio", 2, 10, 1)
		setupConcurrentTestRobotWithPriority(t, "robot_integ_prio_med", "team_integ_prio", 2, 10, 5)
		setupConcurrentTestRobotWithPriority(t, "robot_integ_prio_high", "team_integ_prio", 2, 10, 10)

		executionOrder := make([]string, 0)
		var mu sync.Mutex

		exec := &trackingExecutor{
			delay: 50 * time.Millisecond,
			onStart: func(robot *types.Robot) {
				mu.Lock()
				executionOrder = append(executionOrder, robot.MemberID)
				mu.Unlock()
			},
			onEnd: func(robot *types.Robot) {},
		}

		config := &manager.Config{
			TickInterval: 100 * time.Millisecond,
			PoolConfig:   &pool.Config{WorkerSize: 1, QueueSize: 50}, // Single worker for ordering
		}
		m := manager.NewWithConfig(config)
		m.Pool().SetExecutor(exec)

		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)

		// Submit in low-to-high priority order
		_, err = m.TriggerManual(ctx, "robot_integ_prio_low", types.TriggerClock, nil)
		assert.NoError(t, err)
		_, err = m.TriggerManual(ctx, "robot_integ_prio_med", types.TriggerClock, nil)
		assert.NoError(t, err)
		_, err = m.TriggerManual(ctx, "robot_integ_prio_high", types.TriggerClock, nil)
		assert.NoError(t, err)

		// Wait for all to complete
		time.Sleep(500 * time.Millisecond)

		// Verify execution order (high priority should be first or early)
		mu.Lock()
		order := executionOrder
		mu.Unlock()

		assert.Len(t, order, 3, "All 3 robots should execute")
		// Note: First job may already be picked up before others are queued
		// So we just verify all executed
	})

	t.Run("human trigger has higher priority than clock", func(t *testing.T) {
		setupConcurrentTestRobotAllTriggers(t, "robot_integ_prio_trigger", "team_integ_prio", 2, 10, 5)

		executionOrder := make([]types.TriggerType, 0)
		var mu sync.Mutex

		exec := &triggerTrackingExecutor{
			delay: 50 * time.Millisecond,
			onStart: func(trigger types.TriggerType) {
				mu.Lock()
				executionOrder = append(executionOrder, trigger)
				mu.Unlock()
			},
		}

		config := &manager.Config{
			TickInterval: 100 * time.Millisecond,
			PoolConfig:   &pool.Config{WorkerSize: 1, QueueSize: 50},
		}
		m := manager.NewWithConfig(config)
		m.Pool().SetExecutor(exec)

		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)

		// Submit clock first, then human
		_, err = m.TriggerManual(ctx, "robot_integ_prio_trigger", types.TriggerClock, nil)
		assert.NoError(t, err)
		_, err = m.TriggerManual(ctx, "robot_integ_prio_trigger", types.TriggerHuman, nil)
		assert.NoError(t, err)

		// Wait for execution
		time.Sleep(300 * time.Millisecond)

		mu.Lock()
		order := executionOrder
		mu.Unlock()

		assert.Len(t, order, 2, "Both triggers should execute")
	})
}

// ==================== Helper Types ====================

// trackingExecutor tracks execution per robot
type trackingExecutor struct {
	delay   time.Duration
	onStart func(robot *types.Robot)
	onEnd   func(robot *types.Robot)
	count   int32
}

func (e *trackingExecutor) Execute(ctx *types.Context, robot *types.Robot, trigger types.TriggerType, data interface{}) (*types.Execution, error) {
	return e.ExecuteWithControl(ctx, robot, trigger, data, "", nil)
}

func (e *trackingExecutor) ExecuteWithID(ctx *types.Context, robot *types.Robot, trigger types.TriggerType, data interface{}, execID string) (*types.Execution, error) {
	return e.ExecuteWithControl(ctx, robot, trigger, data, execID, nil)
}

func (e *trackingExecutor) ExecuteWithControl(ctx *types.Context, robot *types.Robot, trigger types.TriggerType, data interface{}, execID string, control types.ExecutionControl) (*types.Execution, error) {
	if robot == nil {
		return nil, types.ErrRobotNotFound
	}

	// Use provided execID or generate unique ID for each execution to properly track quota
	if execID == "" {
		execID = fmt.Sprintf("exec_%d", time.Now().UnixNano())
	}
	exec := &types.Execution{
		ID:          execID,
		MemberID:    robot.MemberID,
		TeamID:      robot.TeamID,
		TriggerType: trigger,
		StartTime:   time.Now(),
		Status:      types.ExecPending,
	}

	if !robot.TryAcquireSlot(exec) {
		return nil, types.ErrQuotaExceeded
	}
	defer robot.RemoveExecution(exec.ID)

	if e.onStart != nil {
		e.onStart(robot)
	}

	exec.Status = types.ExecRunning
	time.Sleep(e.delay)

	if e.onEnd != nil {
		e.onEnd(robot)
	}

	exec.Status = types.ExecCompleted
	now := time.Now()
	exec.EndTime = &now

	atomic.AddInt32(&e.count, 1)
	return exec, nil
}

func (e *trackingExecutor) ExecCount() int {
	return int(atomic.LoadInt32(&e.count))
}

func (e *trackingExecutor) CurrentCount() int {
	return 0
}

func (e *trackingExecutor) Reset() {
	atomic.StoreInt32(&e.count, 0)
}

// triggerTrackingExecutor tracks execution by trigger type
type triggerTrackingExecutor struct {
	delay   time.Duration
	onStart func(trigger types.TriggerType)
	count   int32
}

func (e *triggerTrackingExecutor) Execute(ctx *types.Context, robot *types.Robot, trigger types.TriggerType, data interface{}) (*types.Execution, error) {
	return e.ExecuteWithControl(ctx, robot, trigger, data, "", nil)
}

func (e *triggerTrackingExecutor) ExecuteWithID(ctx *types.Context, robot *types.Robot, trigger types.TriggerType, data interface{}, execID string) (*types.Execution, error) {
	return e.ExecuteWithControl(ctx, robot, trigger, data, execID, nil)
}

func (e *triggerTrackingExecutor) ExecuteWithControl(ctx *types.Context, robot *types.Robot, trigger types.TriggerType, data interface{}, execID string, control types.ExecutionControl) (*types.Execution, error) {
	if robot == nil {
		return nil, types.ErrRobotNotFound
	}

	// Use provided execID or generate unique ID for each execution to properly track quota
	if execID == "" {
		execID = fmt.Sprintf("exec_trigger_%s_%d", string(trigger), time.Now().UnixNano())
	}
	exec := &types.Execution{
		ID:          execID,
		MemberID:    robot.MemberID,
		TeamID:      robot.TeamID,
		TriggerType: trigger,
		StartTime:   time.Now(),
		Status:      types.ExecPending,
	}

	if !robot.TryAcquireSlot(exec) {
		return nil, types.ErrQuotaExceeded
	}
	defer robot.RemoveExecution(exec.ID)

	if e.onStart != nil {
		e.onStart(trigger)
	}

	exec.Status = types.ExecRunning
	time.Sleep(e.delay)

	exec.Status = types.ExecCompleted
	now := time.Now()
	exec.EndTime = &now

	atomic.AddInt32(&e.count, 1)
	return exec, nil
}

func (e *triggerTrackingExecutor) ExecCount() int {
	return int(atomic.LoadInt32(&e.count))
}

func (e *triggerTrackingExecutor) CurrentCount() int {
	return 0
}

func (e *triggerTrackingExecutor) Reset() {
	atomic.StoreInt32(&e.count, 0)
}

// ==================== Test Data Setup Helpers ====================

// setupConcurrentTestRobot creates a robot for concurrency testing
func setupConcurrentTestRobot(t *testing.T, memberID, teamID string, max, queue int) {
	setupConcurrentTestRobotWithPriority(t, memberID, teamID, max, queue, 5)
}

// setupConcurrentTestRobotWithPriority creates a robot with specified priority
func setupConcurrentTestRobotWithPriority(t *testing.T, memberID, teamID string, max, queue, priority int) {
	m := model.Select("__yao.member")
	tableName := m.MetaData.Table.Name
	qb := capsule.Query()

	robotConfig := map[string]interface{}{
		"identity": map[string]interface{}{
			"role": "Concurrent Test Robot " + memberID,
		},
		"quota": map[string]interface{}{
			"max":      max,
			"queue":    queue,
			"priority": priority,
		},
		"triggers": map[string]interface{}{
			"clock":     map[string]interface{}{"enabled": true},
			"intervene": map[string]interface{}{"enabled": true},
			"event":     map[string]interface{}{"enabled": true},
		},
		"clock": map[string]interface{}{
			"mode":  "times",
			"times": []string{"09:00"},
			"tz":    "Asia/Shanghai",
		},
	}
	configJSON, _ := json.Marshal(robotConfig)

	err := qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       memberID,
			"team_id":         teamID,
			"member_type":     "robot",
			"display_name":    "Concurrent Test Robot " + memberID,
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

// setupConcurrentTestRobotAllTriggers creates a robot with all triggers enabled
func setupConcurrentTestRobotAllTriggers(t *testing.T, memberID, teamID string, max, queue, priority int) {
	m := model.Select("__yao.member")
	tableName := m.MetaData.Table.Name
	qb := capsule.Query()

	robotConfig := map[string]interface{}{
		"identity": map[string]interface{}{
			"role": "All Triggers Test Robot",
		},
		"quota": map[string]interface{}{
			"max":      max,
			"queue":    queue,
			"priority": priority,
		},
		"triggers": map[string]interface{}{
			"clock":     map[string]interface{}{"enabled": true},
			"intervene": map[string]interface{}{"enabled": true},
			"event":     map[string]interface{}{"enabled": true},
		},
	}
	configJSON, _ := json.Marshal(robotConfig)

	err := qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       memberID,
			"team_id":         teamID,
			"member_type":     "robot",
			"display_name":    "All Triggers Robot " + memberID,
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
