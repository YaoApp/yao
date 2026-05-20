//go:build unit

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

	p.Submit(ctx, robot, types.TriggerClock, nil)

	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, 1, exec.ExecCount())
}

func TestWorkerMultipleJobs(t *testing.T) {
	exec := executor.NewDryRunWithDelay(20 * time.Millisecond)
	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 1,
		QueueSize:  10,
	})
	p.SetExecutor(exec)
	p.Start()
	defer p.Stop()

	ctx := createTestContext()
	robot := createTestRobot("robot_1", "team_1", 10, 10, 5)

	for i := 0; i < 3; i++ {
		p.Submit(ctx, robot, types.TriggerClock, nil)
	}

	assert.Eventually(t, func() bool {
		return exec.ExecCount() >= 3
	}, 1*time.Second, 50*time.Millisecond, "All 3 jobs should complete")
}

// ==================== Worker Quota Check Tests ====================

func TestWorkerRespectsRobotQuota(t *testing.T) {
	exec := executor.NewDryRunWithDelay(100 * time.Millisecond)

	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 5,
		QueueSize:  20,
	})
	p.SetExecutor(exec)
	p.Start()
	defer p.Stop()

	ctx := createTestContext()
	robot := createTestRobot("robot_limited", "team_1", 2, 10, 5)

	for i := 0; i < 5; i++ {
		p.Submit(ctx, robot, types.TriggerClock, nil)
	}

	time.Sleep(1000 * time.Millisecond)

	assert.GreaterOrEqual(t, exec.ExecCount(), 5, "All jobs should eventually execute")
}

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
	robot := createTestRobot("robot_1", "team_1", 1, 50, 5)

	for i := 0; i < 5; i++ {
		p.Submit(ctx, robot, types.TriggerClock, nil)
	}

	assert.Eventually(t, func() bool {
		return exec.ExecCount() >= 5
	}, 2*time.Second, 100*time.Millisecond, "All 5 jobs should complete")
}

// ==================== Worker Concurrency Tests ====================

func TestWorkersConcurrentExecution(t *testing.T) {
	var maxConcurrent int32
	var currentConcurrent int32

	exec := executor.NewDryRunWithCallbacks(100*time.Millisecond, func() {
		current := atomic.AddInt32(&currentConcurrent, 1)
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
		WorkerSize: 5,
		QueueSize:  100,
	})
	p.SetExecutor(exec)
	p.Start()
	defer p.Stop()

	ctx := createTestContext()

	for i := 0; i < 10; i++ {
		robot := createTestRobot("robot_"+string(rune('A'+i)), "team_1", 5, 10, 5)
		p.Submit(ctx, robot, types.TriggerClock, nil)
	}

	time.Sleep(400 * time.Millisecond)

	assert.GreaterOrEqual(t, atomic.LoadInt32(&maxConcurrent), int32(2), "Should have concurrent execution")
}

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
		WorkerSize: 3,
		QueueSize:  100,
	})
	p.SetExecutor(exec)
	p.Start()
	defer p.Stop()

	ctx := createTestContext()

	for i := 0; i < 20; i++ {
		robot := createTestRobot("robot_"+string(rune('A'+i)), "team_1", 5, 10, 5)
		p.Submit(ctx, robot, types.TriggerClock, nil)
	}

	time.Sleep(500 * time.Millisecond)

	assert.LessOrEqual(t, maxConcurrent, int32(3), "Should not exceed worker size")
}

// ==================== Worker Stop Tests ====================

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

	p.Submit(ctx, robot, types.TriggerClock, nil)
	p.Submit(ctx, robot, types.TriggerClock, nil)

	time.Sleep(150 * time.Millisecond)

	err := p.Stop()
	assert.NoError(t, err)

	assert.False(t, p.IsStarted())
}

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

	p.Submit(ctx, robot, types.TriggerClock, nil)

	time.Sleep(150 * time.Millisecond)

	p.Stop()

	assert.GreaterOrEqual(t, exec.ExecCount(), 1)
}

// ==================== Worker Error Handling Tests ====================

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

	p.Submit(ctx, robot, types.TriggerClock, "simulate_failure")
	p.Submit(ctx, robot, types.TriggerClock, nil)

	time.Sleep(300 * time.Millisecond)

	assert.GreaterOrEqual(t, exec.ExecCount(), 2)
}

// ==================== Worker Running Counter Tests ====================

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

	for i := 0; i < 3; i++ {
		robot := createTestRobot("robot_"+string(rune('A'+i)), "team_1", 5, 10, 5)
		p.Submit(ctx, robot, types.TriggerClock, nil)
	}

	time.Sleep(200 * time.Millisecond)

	assert.Eventually(t, func() bool {
		return p.Running() == 0
	}, 1*time.Second, 50*time.Millisecond, "Running should be 0 after all jobs complete")
}

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

	p.Submit(ctx, robot, types.TriggerClock, "simulate_failure")

	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, 0, p.Running())
}

// ==================== Worker with Different Trigger Types ====================

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

	p.Submit(ctx, robot, types.TriggerClock, nil)
	p.Submit(ctx, robot, types.TriggerHuman, nil)
	p.Submit(ctx, robot, types.TriggerEvent, nil)

	assert.Eventually(t, func() bool {
		return exec.ExecCount() >= 3
	}, 1*time.Second, 50*time.Millisecond, "All 3 trigger types should execute")
}

// ==================== Worker Polling Behavior Tests ====================

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

	time.Sleep(50 * time.Millisecond)
	p.Submit(ctx, robot, types.TriggerClock, nil)

	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, 1, exec.ExecCount())
}

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

	time.Sleep(200 * time.Millisecond)

	p.Submit(ctx, robot, types.TriggerClock, nil)

	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, 1, exec.ExecCount())
}
