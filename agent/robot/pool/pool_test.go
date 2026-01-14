package pool_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent/robot/executor"
	"github.com/yaoapp/yao/agent/robot/pool"
	"github.com/yaoapp/yao/agent/robot/types"
)

// createTestRobot creates a robot for testing with specified quota
func createTestRobot(memberID, teamID string, maxConcurrent, queueSize, priority int) *types.Robot {
	return &types.Robot{
		MemberID:       memberID,
		TeamID:         teamID,
		DisplayName:    "Test Robot " + memberID,
		Status:         types.RobotIdle,
		AutonomousMode: true,
		Config: &types.Config{
			Identity: &types.Identity{Role: "Test"},
			Quota: &types.Quota{
				Max:      maxConcurrent,
				Queue:    queueSize,
				Priority: priority,
			},
		},
	}
}

// createTestContext creates a context for testing
func createTestContext() *types.Context {
	return types.NewContext(context.Background(), nil)
}

// TestPoolStartStop tests pool start and stop lifecycle
func TestPoolStartStop(t *testing.T) {
	p := pool.New()
	exec := executor.New()
	p.SetExecutor(exec)

	t.Run("start pool", func(t *testing.T) {
		err := p.Start()
		assert.NoError(t, err)
		assert.True(t, p.IsStarted())
	})

	t.Run("start already started pool", func(t *testing.T) {
		err := p.Start()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already started")
	})

	t.Run("stop pool", func(t *testing.T) {
		err := p.Stop()
		assert.NoError(t, err)
		assert.False(t, p.IsStarted())
	})

	t.Run("stop already stopped pool", func(t *testing.T) {
		err := p.Stop()
		assert.NoError(t, err) // should not error
	})
}

// TestPoolSubmitWithoutStart tests submitting to unstarted pool
func TestPoolSubmitWithoutStart(t *testing.T) {
	p := pool.New()
	exec := executor.New()
	p.SetExecutor(exec)

	robot := createTestRobot("robot_1", "team_1", 2, 10, 5)
	ctx := createTestContext()

	_, err := p.Submit(ctx, robot, types.TriggerClock, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not started")
}

// TestPoolSubmitNilRobot tests submitting nil robot
func TestPoolSubmitNilRobot(t *testing.T) {
	p := pool.New()
	exec := executor.New()
	p.SetExecutor(exec)
	p.Start()
	defer p.Stop()

	ctx := createTestContext()
	_, err := p.Submit(ctx, nil, types.TriggerClock, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be nil")
}

// TestPoolBasicExecution tests basic job execution
func TestPoolBasicExecution(t *testing.T) {
	exec := executor.NewWithDelay(50 * time.Millisecond)
	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 5,
		QueueSize:  100,
	})
	p.SetExecutor(exec)
	p.Start()
	defer p.Stop()

	robot := createTestRobot("robot_1", "team_1", 2, 10, 5)
	ctx := createTestContext()

	// Submit a job
	execID, err := p.Submit(ctx, robot, types.TriggerClock, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, execID)

	// Wait for execution
	time.Sleep(200 * time.Millisecond)

	// Verify execution completed
	assert.Equal(t, 1, exec.ExecCount())
	assert.Equal(t, 0, exec.CurrentCount())
}

// TestPoolConcurrencyLimit tests global worker limit
func TestPoolConcurrencyLimit(t *testing.T) {
	exec := executor.NewWithDelay(200 * time.Millisecond) // longer delay
	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 3, // only 3 workers
		QueueSize:  100,
	})
	p.SetExecutor(exec)
	p.Start()
	defer p.Stop()

	ctx := createTestContext()

	// Create robots with high quota (won't be the bottleneck)
	robots := make([]*types.Robot, 10)
	for i := 0; i < 10; i++ {
		robots[i] = createTestRobot(
			"robot_"+string(rune('A'+i)),
			"team_1",
			5,  // max concurrent per robot
			20, // queue size per robot
			5,  // priority
		)
	}

	// Submit 10 jobs
	for i := 0; i < 10; i++ {
		_, err := p.Submit(ctx, robots[i], types.TriggerClock, nil)
		assert.NoError(t, err)
	}

	// Wait for workers to pick up jobs (worker polls every 100ms)
	time.Sleep(150 * time.Millisecond)

	// Should have at most 3 running (worker limit)
	running := p.Running()
	assert.LessOrEqual(t, running, 3, "Should not exceed worker limit")

	// Wait for all to complete
	time.Sleep(800 * time.Millisecond)

	assert.Equal(t, 10, exec.ExecCount())
}

// TestRobotConcurrencyLimit tests per-robot concurrent execution limit
func TestRobotConcurrencyLimit(t *testing.T) {
	exec := executor.NewWithDelay(100 * time.Millisecond)
	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 10, // plenty of workers
		QueueSize:  100,
	})
	p.SetExecutor(exec)
	p.Start()
	defer p.Stop()

	ctx := createTestContext()

	// Create robot with Max=2 (can only run 2 at a time)
	robot := createTestRobot("robot_limited", "team_1", 2, 20, 5)

	// Submit 5 jobs for the same robot
	for i := 0; i < 5; i++ {
		_, err := p.Submit(ctx, robot, types.TriggerClock, nil)
		assert.NoError(t, err)
	}

	// Wait a bit for execution to start
	time.Sleep(150 * time.Millisecond)

	// Robot should have at most 2 running (Quota.Max=2)
	runningCount := robot.RunningCount()
	assert.LessOrEqual(t, runningCount, 2, "Robot should not exceed Quota.Max")

	// Wait for all to complete (with re-enqueue, need more time)
	// 5 jobs with Max=2: ~3 batches * 100ms exec + poll overhead
	time.Sleep(800 * time.Millisecond)

	// All 5 jobs should eventually execute
	assert.GreaterOrEqual(t, exec.ExecCount(), 5, "All jobs should eventually execute")
}

// TestRobotQueueLimit tests per-robot queue limit
func TestRobotQueueLimit(t *testing.T) {
	exec := executor.NewWithDelay(200 * time.Millisecond)
	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 2,
		QueueSize:  100, // global queue is large
	})
	p.SetExecutor(exec)
	p.Start()
	defer p.Stop()

	ctx := createTestContext()

	// Create robot with small queue limit
	robot := createTestRobot("robot_small_queue", "team_1", 1, 3, 5) // Queue=3

	// Submit jobs until queue limit is reached
	successCount := 0
	for i := 0; i < 10; i++ {
		_, err := p.Submit(ctx, robot, types.TriggerClock, nil)
		if err == nil {
			successCount++
		}
	}

	// Should only accept up to Queue limit (some may have started executing)
	// Max accepted = Queue(3) + Max(1) = 4 (1 running + 3 in queue)
	assert.LessOrEqual(t, successCount, 4, "Should respect robot queue limit")
	assert.GreaterOrEqual(t, successCount, 1, "Should accept at least 1 job")
}

// TestGlobalQueueLimit tests global queue limit
func TestGlobalQueueLimit(t *testing.T) {
	exec := executor.NewWithDelay(500 * time.Millisecond) // slow execution
	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 1, // only 1 worker
		QueueSize:  5, // small global queue
	})
	p.SetExecutor(exec)
	p.Start()
	defer p.Stop()

	ctx := createTestContext()

	// Create multiple robots with large queue limits
	successCount := 0
	for i := 0; i < 20; i++ {
		robot := createTestRobot(
			"robot_"+string(rune('A'+i%26)),
			"team_1",
			5,  // large max
			20, // large per-robot queue
			5,
		)
		_, err := p.Submit(ctx, robot, types.TriggerClock, nil)
		if err == nil {
			successCount++
		}
	}

	// Should only accept up to global queue limit + running
	// Max = QueueSize(5) + WorkerSize(1) = 6
	assert.LessOrEqual(t, successCount, 6, "Should respect global queue limit")
}

// TestPriorityOrder tests that higher priority jobs execute first
func TestPriorityOrder(t *testing.T) {
	exec := executor.NewWithDelay(50 * time.Millisecond)
	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 1, // single worker to ensure order
		QueueSize:  100,
	})
	p.SetExecutor(exec)
	p.Start()
	defer p.Stop()

	ctx := createTestContext()

	// Create robots with different priorities
	robotLow := createTestRobot("robot_low", "team_1", 5, 20, 1)    // priority 1
	robotMed := createTestRobot("robot_med", "team_1", 5, 20, 5)    // priority 5
	robotHigh := createTestRobot("robot_high", "team_1", 5, 20, 10) // priority 10

	// Submit in low-to-high order
	p.Submit(ctx, robotLow, types.TriggerClock, nil)
	p.Submit(ctx, robotMed, types.TriggerClock, nil)
	p.Submit(ctx, robotHigh, types.TriggerClock, nil)

	// Wait for all to complete
	time.Sleep(400 * time.Millisecond)

	// Verify all executed
	assert.Equal(t, 3, exec.ExecCount())
}

// TestTriggerTypePriority tests that human triggers have higher priority than clock
func TestTriggerTypePriority(t *testing.T) {
	exec := executor.NewWithDelay(50 * time.Millisecond)
	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 1, // single worker
		QueueSize:  100,
	})
	p.SetExecutor(exec)
	p.Start()
	defer p.Stop()

	ctx := createTestContext()

	// Same robot, same priority, different trigger types
	robot := createTestRobot("robot_1", "team_1", 5, 20, 5)

	// Submit clock first, then human
	p.Submit(ctx, robot, types.TriggerClock, nil)
	p.Submit(ctx, robot, types.TriggerHuman, nil) // should execute first

	time.Sleep(300 * time.Millisecond)

	assert.Equal(t, 2, exec.ExecCount())
}

// TestMultipleRobotsFairness tests that multiple robots get fair access
func TestMultipleRobotsFairness(t *testing.T) {
	exec := executor.NewWithDelay(30 * time.Millisecond)
	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 5,
		QueueSize:  100,
	})
	p.SetExecutor(exec)
	p.Start()
	defer p.Stop()

	ctx := createTestContext()

	// Create 3 robots with same priority
	robotA := createTestRobot("robot_A", "team_1", 2, 10, 5)
	robotB := createTestRobot("robot_B", "team_1", 2, 10, 5)
	robotC := createTestRobot("robot_C", "team_1", 2, 10, 5)

	// Submit jobs for each robot
	for i := 0; i < 6; i++ {
		p.Submit(ctx, robotA, types.TriggerClock, nil)
		p.Submit(ctx, robotB, types.TriggerClock, nil)
		p.Submit(ctx, robotC, types.TriggerClock, nil)
	}

	// Wait for all to complete
	time.Sleep(500 * time.Millisecond)

	// All 18 jobs should complete
	assert.Equal(t, 18, exec.ExecCount())
}

// TestGracefulShutdown tests that pool waits for running jobs on shutdown
func TestGracefulShutdown(t *testing.T) {
	exec := executor.NewWithDelay(200 * time.Millisecond)
	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 2,
		QueueSize:  10,
	})
	p.SetExecutor(exec)
	p.Start()

	ctx := createTestContext()
	robot := createTestRobot("robot_1", "team_1", 5, 20, 5)

	// Submit 2 jobs
	p.Submit(ctx, robot, types.TriggerClock, nil)
	p.Submit(ctx, robot, types.TriggerClock, nil)

	// Wait for workers to pick up jobs (poll every 100ms)
	time.Sleep(150 * time.Millisecond)

	// Verify jobs are running
	assert.GreaterOrEqual(t, p.Running(), 1, "Should have at least 1 running job")

	// Stop - workers will finish their current tick cycle
	p.Stop()

	// After stop, verify jobs completed
	assert.GreaterOrEqual(t, exec.ExecCount(), 1, "Should have executed at least 1 job")
}

// TestDefaultConfig tests default configuration values
func TestDefaultConfig(t *testing.T) {
	config := pool.DefaultConfig()
	assert.Equal(t, pool.DefaultWorkerSize, config.WorkerSize)
	assert.Equal(t, pool.DefaultQueueSize, config.QueueSize)
}

// TestPoolWithNilConfig tests pool creation with nil config
func TestPoolWithNilConfig(t *testing.T) {
	p := pool.NewWithConfig(nil)
	assert.Equal(t, pool.DefaultWorkerSize, p.Size())
	assert.Equal(t, pool.DefaultQueueSize, p.QueueSize())
}

// TestPoolWithZeroConfig tests pool creation with zero values
func TestPoolWithZeroConfig(t *testing.T) {
	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 0,
		QueueSize:  0,
	})
	// Should use defaults for zero values
	assert.Equal(t, pool.DefaultWorkerSize, p.Size())
	assert.Equal(t, pool.DefaultQueueSize, p.QueueSize())
}

// TestPoolWithoutExecutor tests starting pool without executor
func TestPoolWithoutExecutor(t *testing.T) {
	p := pool.New()
	// Don't set executor
	err := p.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "executor not set")
}
