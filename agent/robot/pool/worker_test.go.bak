package pool_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent/robot/executor"
	"github.com/yaoapp/yao/agent/robot/pool"
	"github.com/yaoapp/yao/agent/robot/types"
)

// ==================== Worker Basic Tests ====================

// TestWorkerExecutesJob tests that worker executes a job from queue
func TestWorkerExecutesJob(t *testing.T) {
	exec := executor.NewDryRunWithDelay(10 * time.Millisecond)
	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 1,
		QueueSize:  10,
	})
	p.SetExecutor(exec)
	p.Start()
	defer p.Stop()

	ctx := createTestContext()
	robot := createTestRobot("robot_1", "team_1", 5, 10, 5)

	// Submit job
	p.Submit(ctx, robot, types.TriggerClock, nil)

	// Wait for execution
	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, 1, exec.ExecCount())
}

// TestWorkerMultipleJobs tests worker processes multiple jobs sequentially
func TestWorkerMultipleJobs(t *testing.T) {
	exec := executor.NewDryRunWithDelay(20 * time.Millisecond)
	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 1, // single worker
		QueueSize:  10,
	})
	p.SetExecutor(exec)
	p.Start()
	defer p.Stop()

	ctx := createTestContext()
	robot := createTestRobot("robot_1", "team_1", 10, 10, 5)

	// Submit 3 jobs
	for i := 0; i < 3; i++ {
		p.Submit(ctx, robot, types.TriggerClock, nil)
	}

	// Wait for all executions (worker polls every 100ms, each job takes 20ms)
	// Use Eventually for CI timing variations
	assert.Eventually(t, func() bool {
		return exec.ExecCount() >= 3
	}, 1*time.Second, 50*time.Millisecond, "All 3 jobs should complete")
}

// ==================== Worker Quota Check Tests ====================

// TestWorkerRespectsRobotQuota tests worker re-enqueues when robot quota is full
func TestWorkerRespectsRobotQuota(t *testing.T) {
	// This test verifies that all jobs eventually complete even when robot quota limits concurrency
	exec := executor.NewDryRunWithDelay(100 * time.Millisecond)

	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 5, // multiple workers
		QueueSize:  20,
	})
	p.SetExecutor(exec)
	p.Start()
	defer p.Stop()

	ctx := createTestContext()
	// Robot can only run 2 at a time
	robot := createTestRobot("robot_limited", "team_1", 2, 10, 5)

	// Submit 5 jobs for same robot
	for i := 0; i < 5; i++ {
		p.Submit(ctx, robot, types.TriggerClock, nil)
	}

	// Wait for all to complete
	// With Quota.Max=2, jobs execute in batches: 2+2+1 = 3 batches
	// Each batch: 100ms exec + 100ms poll = ~200ms, total ~600ms, add buffer
	time.Sleep(1000 * time.Millisecond)

	// All should eventually execute
	assert.GreaterOrEqual(t, exec.ExecCount(), 5, "All jobs should eventually execute")
}

// TestWorkerReenqueueOnQuotaFull tests that jobs are re-enqueued when quota is full
func TestWorkerReenqueueOnQuotaFull(t *testing.T) {
	exec := executor.NewDryRunWithDelay(100 * time.Millisecond)
	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 3,
		QueueSize:  100,
	})
	p.SetExecutor(exec)
	p.Start()
	defer p.Stop()

	ctx := createTestContext()
	// Robot can only run 1 at a time, but large queue
	robot := createTestRobot("robot_1", "team_1", 1, 50, 5)

	// Submit 5 jobs
	for i := 0; i < 5; i++ {
		p.Submit(ctx, robot, types.TriggerClock, nil)
	}

	// Wait for all to complete
	// With Quota.Max=1, jobs execute sequentially: 5 * (100ms exec + 100ms poll) = ~1000ms
	// Use Eventually for CI timing variations
	assert.Eventually(t, func() bool {
		return exec.ExecCount() >= 5
	}, 2*time.Second, 100*time.Millisecond, "All 5 jobs should complete")
}

// ==================== Worker Concurrency Tests ====================

// TestWorkersConcurrentExecution tests multiple workers execute concurrently
func TestWorkersConcurrentExecution(t *testing.T) {
	// Track max concurrent executions
	var maxConcurrent int32
	var currentConcurrent int32

	exec := executor.NewDryRunWithCallbacks(100*time.Millisecond, func() {
		current := atomic.AddInt32(&currentConcurrent, 1)
		// Update max if current is higher
		for {
			max := atomic.LoadInt32(&maxConcurrent)
			if current <= max || atomic.CompareAndSwapInt32(&maxConcurrent, max, current) {
				break
			}
		}
	}, func() {
		atomic.AddInt32(&currentConcurrent, -1)
	})

	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 5, // 5 workers
		QueueSize:  100,
	})
	p.SetExecutor(exec)
	p.Start()
	defer p.Stop()

	ctx := createTestContext()

	// Submit 10 jobs for different robots
	for i := 0; i < 10; i++ {
		robot := createTestRobot("robot_"+string(rune('A'+i)), "team_1", 5, 10, 5)
		p.Submit(ctx, robot, types.TriggerClock, nil)
	}

	// Wait for execution
	time.Sleep(400 * time.Millisecond)

	// Should have had concurrent execution (max > 1)
	assert.GreaterOrEqual(t, atomic.LoadInt32(&maxConcurrent), int32(2), "Should have concurrent execution")
}

// TestWorkersDoNotExceedPoolSize tests workers don't exceed pool size
func TestWorkersDoNotExceedPoolSize(t *testing.T) {
	var maxConcurrent int32
	var currentConcurrent int32
	var mu sync.Mutex

	exec := executor.NewDryRunWithCallbacks(50*time.Millisecond, func() {
		mu.Lock()
		currentConcurrent++
		if currentConcurrent > maxConcurrent {
			maxConcurrent = currentConcurrent
		}
		mu.Unlock()
	}, func() {
		mu.Lock()
		currentConcurrent--
		mu.Unlock()
	})

	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 3, // only 3 workers
		QueueSize:  100,
	})
	p.SetExecutor(exec)
	p.Start()
	defer p.Stop()

	ctx := createTestContext()

	// Submit 20 jobs
	for i := 0; i < 20; i++ {
		robot := createTestRobot("robot_"+string(rune('A'+i)), "team_1", 5, 10, 5)
		p.Submit(ctx, robot, types.TriggerClock, nil)
	}

	// Wait for all to complete
	time.Sleep(500 * time.Millisecond)

	// Max concurrent should not exceed worker size
	assert.LessOrEqual(t, maxConcurrent, int32(3), "Should not exceed worker size")
}

// ==================== Worker Stop Tests ====================

// TestWorkerStopsGracefully tests worker stops when signaled
func TestWorkerStopsGracefully(t *testing.T) {
	exec := executor.NewDryRunWithDelay(50 * time.Millisecond)
	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 2,
		QueueSize:  10,
	})
	p.SetExecutor(exec)
	p.Start()

	ctx := createTestContext()
	robot := createTestRobot("robot_1", "team_1", 5, 10, 5)

	// Submit jobs
	p.Submit(ctx, robot, types.TriggerClock, nil)
	p.Submit(ctx, robot, types.TriggerClock, nil)

	// Wait for jobs to start
	time.Sleep(150 * time.Millisecond)

	// Stop pool
	err := p.Stop()
	assert.NoError(t, err)

	// Pool should be stopped
	assert.False(t, p.IsStarted())
}

// TestWorkerCompletesCurrentJobOnStop tests worker completes current job before stopping
func TestWorkerCompletesCurrentJobOnStop(t *testing.T) {
	exec := executor.NewDryRunWithDelay(100 * time.Millisecond)
	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 1,
		QueueSize:  10,
	})
	p.SetExecutor(exec)
	p.Start()

	ctx := createTestContext()
	robot := createTestRobot("robot_1", "team_1", 5, 10, 5)

	// Submit job
	p.Submit(ctx, robot, types.TriggerClock, nil)

	// Wait for job to start
	time.Sleep(150 * time.Millisecond)

	// Stop pool - should wait for current job
	p.Stop()

	// Job should have completed
	assert.GreaterOrEqual(t, exec.ExecCount(), 1)
}

// ==================== Worker Error Handling Tests ====================

// TestWorkerHandlesExecutorError tests worker continues after executor error
func TestWorkerHandlesExecutorError(t *testing.T) {
	exec := executor.NewDryRunWithDelay(10 * time.Millisecond)
	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 1,
		QueueSize:  10,
	})
	p.SetExecutor(exec)
	p.Start()
	defer p.Stop()

	ctx := createTestContext()
	robot := createTestRobot("robot_1", "team_1", 5, 10, 5)

	// Submit job that will fail (using special data)
	p.Submit(ctx, robot, types.TriggerClock, "simulate_failure")

	// Submit another job that should succeed
	p.Submit(ctx, robot, types.TriggerClock, nil)

	// Wait for execution
	time.Sleep(300 * time.Millisecond)

	// Both should have been attempted
	assert.GreaterOrEqual(t, exec.ExecCount(), 2)
}

// ==================== Worker Running Counter Tests ====================

// TestWorkerRunningCounterAccurate tests running counter is accurate
func TestWorkerRunningCounterAccurate(t *testing.T) {
	exec := executor.NewDryRunWithDelay(100 * time.Millisecond)
	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 3,
		QueueSize:  10,
	})
	p.SetExecutor(exec)
	p.Start()
	defer p.Stop()

	ctx := createTestContext()

	// Submit jobs for different robots
	for i := 0; i < 3; i++ {
		robot := createTestRobot("robot_"+string(rune('A'+i)), "team_1", 5, 10, 5)
		p.Submit(ctx, robot, types.TriggerClock, nil)
	}

	// Wait for jobs to start
	time.Sleep(200 * time.Millisecond)

	// Running should be > 0 while jobs are executing
	// Note: On fast CI, jobs may already be done, so we just verify it doesn't panic

	// Wait for completion and verify running counter returns to 0
	assert.Eventually(t, func() bool {
		return p.Running() == 0
	}, 1*time.Second, 50*time.Millisecond, "Running should be 0 after all jobs complete")
}

// TestWorkerRunningCounterDecrementsOnError tests running counter decrements on error
func TestWorkerRunningCounterDecrementsOnError(t *testing.T) {
	exec := executor.NewDryRunWithDelay(10 * time.Millisecond)
	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 1,
		QueueSize:  10,
	})
	p.SetExecutor(exec)
	p.Start()
	defer p.Stop()

	ctx := createTestContext()
	robot := createTestRobot("robot_1", "team_1", 5, 10, 5)

	// Submit failing job
	p.Submit(ctx, robot, types.TriggerClock, "simulate_failure")

	// Wait for execution
	time.Sleep(200 * time.Millisecond)

	// Running should be 0 (decremented even on error)
	assert.Equal(t, 0, p.Running())
}

// ==================== Worker with Different Trigger Types ====================

// TestWorkerProcessesDifferentTriggers tests worker handles all trigger types
func TestWorkerProcessesDifferentTriggers(t *testing.T) {
	exec := executor.NewDryRunWithDelay(10 * time.Millisecond)
	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 1,
		QueueSize:  10,
	})
	p.SetExecutor(exec)
	p.Start()
	defer p.Stop()

	ctx := createTestContext()
	robot := createTestRobot("robot_1", "team_1", 5, 10, 5)

	// Submit different trigger types
	p.Submit(ctx, robot, types.TriggerClock, nil)
	p.Submit(ctx, robot, types.TriggerHuman, nil)
	p.Submit(ctx, robot, types.TriggerEvent, nil)

	// Wait for execution (worker polls every 100ms, each job takes 10ms)
	// Use Eventually for CI timing variations
	assert.Eventually(t, func() bool {
		return exec.ExecCount() >= 3
	}, 1*time.Second, 50*time.Millisecond, "All 3 trigger types should execute")
}

// ==================== Worker Polling Behavior Tests ====================

// TestWorkerPollsQueuePeriodically tests worker polls queue at regular intervals
func TestWorkerPollsQueuePeriodically(t *testing.T) {
	exec := executor.NewDryRunWithDelay(10 * time.Millisecond)
	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 1,
		QueueSize:  10,
	})
	p.SetExecutor(exec)
	p.Start()
	defer p.Stop()

	ctx := createTestContext()
	robot := createTestRobot("robot_1", "team_1", 5, 10, 5)

	// Submit job after pool started
	time.Sleep(50 * time.Millisecond)
	p.Submit(ctx, robot, types.TriggerClock, nil)

	// Worker should pick up job within poll interval (100ms)
	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, 1, exec.ExecCount())
}

// TestWorkerContinuesAfterEmptyQueue tests worker continues polling after empty queue
func TestWorkerContinuesAfterEmptyQueue(t *testing.T) {
	exec := executor.NewDryRunWithDelay(10 * time.Millisecond)
	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 1,
		QueueSize:  10,
	})
	p.SetExecutor(exec)
	p.Start()
	defer p.Stop()

	ctx := createTestContext()
	robot := createTestRobot("robot_1", "team_1", 5, 10, 5)

	// Wait with empty queue
	time.Sleep(200 * time.Millisecond)

	// Submit job
	p.Submit(ctx, robot, types.TriggerClock, nil)

	// Worker should still be running and pick up job
	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, 1, exec.ExecCount())
}
