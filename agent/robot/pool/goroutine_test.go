package pool_test

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent/robot/executor"
	"github.com/yaoapp/yao/agent/robot/pool"
	"github.com/yaoapp/yao/agent/robot/types"
)

// ==================== Goroutine Leak Detection Tests ====================

// getGoroutineCount returns current number of goroutines
func getGoroutineCount() int {
	return runtime.NumGoroutine()
}

// waitForGoroutineCount waits for goroutine count to stabilize
func waitForGoroutineCount(target int, timeout time.Duration) int {
	deadline := time.Now().Add(timeout)
	var count int
	for time.Now().Before(deadline) {
		count = getGoroutineCount()
		if count <= target {
			return count
		}
		runtime.Gosched()
		time.Sleep(10 * time.Millisecond)
	}
	return count
}

// TestPoolNoGoroutineLeak tests that pool doesn't leak goroutines after stop
func TestPoolNoGoroutineLeak(t *testing.T) {
	// Get baseline goroutine count
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	baseline := getGoroutineCount()

	// Create and start pool
	exec := executor.NewWithDelay(10 * time.Millisecond)
	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 5,
		QueueSize:  100,
	})
	p.SetExecutor(exec)
	p.Start()

	// Verify workers are running
	afterStart := getGoroutineCount()
	assert.Greater(t, afterStart, baseline, "Should have more goroutines after start")

	// Submit some jobs
	ctx := types.NewContext(context.Background(), nil)
	robot := createTestRobot("robot_1", "team_1", 5, 10, 5)
	for i := 0; i < 10; i++ {
		p.Submit(ctx, robot, types.TriggerClock, nil)
	}

	// Wait for jobs to complete
	time.Sleep(300 * time.Millisecond)

	// Stop pool
	p.Stop()

	// Wait for goroutines to clean up
	finalCount := waitForGoroutineCount(baseline+2, 500*time.Millisecond)

	// Allow small variance (test framework goroutines)
	assert.LessOrEqual(t, finalCount, baseline+2,
		"Goroutine count should return to near baseline after stop (baseline=%d, final=%d)", baseline, finalCount)
}

// TestPoolMultipleStartStop tests no leak with multiple start/stop cycles
func TestPoolMultipleStartStop(t *testing.T) {
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	baseline := getGoroutineCount()

	exec := executor.NewWithDelay(5 * time.Millisecond)

	for i := 0; i < 5; i++ {
		p := pool.NewWithConfig(&pool.Config{
			WorkerSize: 3,
			QueueSize:  50,
		})
		p.SetExecutor(exec)
		p.Start()

		// Submit a few jobs
		ctx := types.NewContext(context.Background(), nil)
		robot := createTestRobot("robot_1", "team_1", 5, 10, 5)
		for j := 0; j < 5; j++ {
			p.Submit(ctx, robot, types.TriggerClock, nil)
		}

		time.Sleep(100 * time.Millisecond)
		p.Stop()
	}

	// Wait for cleanup
	finalCount := waitForGoroutineCount(baseline+2, 500*time.Millisecond)

	assert.LessOrEqual(t, finalCount, baseline+2,
		"Goroutine count should return to near baseline after multiple cycles (baseline=%d, final=%d)", baseline, finalCount)
}

// TestPoolStopWithoutJobs tests no leak when stopping pool with no jobs submitted
func TestPoolStopWithoutJobs(t *testing.T) {
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	baseline := getGoroutineCount()

	exec := executor.New()
	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 10,
		QueueSize:  100,
	})
	p.SetExecutor(exec)
	p.Start()

	// Immediately stop without submitting any jobs
	p.Stop()

	finalCount := waitForGoroutineCount(baseline+2, 500*time.Millisecond)

	assert.LessOrEqual(t, finalCount, baseline+2,
		"Goroutine count should return to near baseline (baseline=%d, final=%d)", baseline, finalCount)
}

// TestPoolStopWithPendingJobs tests no leak when stopping with jobs in queue
func TestPoolStopWithPendingJobs(t *testing.T) {
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	baseline := getGoroutineCount()

	// Use slow executor so jobs stay in queue
	exec := executor.NewWithDelay(500 * time.Millisecond)
	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 1, // only 1 worker
		QueueSize:  100,
	})
	p.SetExecutor(exec)
	p.Start()

	// Submit many jobs (most will be queued)
	ctx := types.NewContext(context.Background(), nil)
	robot := createTestRobot("robot_1", "team_1", 5, 50, 5)
	for i := 0; i < 20; i++ {
		p.Submit(ctx, robot, types.TriggerClock, nil)
	}

	// Stop immediately (some jobs still in queue)
	time.Sleep(50 * time.Millisecond)
	p.Stop()

	finalCount := waitForGoroutineCount(baseline+2, 500*time.Millisecond)

	assert.LessOrEqual(t, finalCount, baseline+2,
		"Goroutine count should return to near baseline even with pending jobs (baseline=%d, final=%d)", baseline, finalCount)
}

// TestPoolConcurrentStartStop tests no leak with concurrent start/stop
func TestPoolConcurrentStartStop(t *testing.T) {
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	baseline := getGoroutineCount()

	exec := executor.NewWithDelay(10 * time.Millisecond)
	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 5,
		QueueSize:  100,
	})
	p.SetExecutor(exec)

	// Start pool
	p.Start()

	// Concurrent operations
	done := make(chan bool, 3)

	// Goroutine 1: Submit jobs
	go func() {
		ctx := types.NewContext(context.Background(), nil)
		robot := createTestRobot("robot_1", "team_1", 5, 10, 5)
		for i := 0; i < 20; i++ {
			p.Submit(ctx, robot, types.TriggerClock, nil)
			time.Sleep(5 * time.Millisecond)
		}
		done <- true
	}()

	// Goroutine 2: Check status
	go func() {
		for i := 0; i < 20; i++ {
			_ = p.Running()
			_ = p.Queued()
			time.Sleep(5 * time.Millisecond)
		}
		done <- true
	}()

	// Wait for operations
	<-done
	<-done

	// Stop pool
	p.Stop()

	finalCount := waitForGoroutineCount(baseline+2, 500*time.Millisecond)

	assert.LessOrEqual(t, finalCount, baseline+2,
		"Goroutine count should return to near baseline after concurrent ops (baseline=%d, final=%d)", baseline, finalCount)
}

// TestWorkerGoroutinesCleanup tests that worker goroutines are properly cleaned up
func TestWorkerGoroutinesCleanup(t *testing.T) {
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	baseline := getGoroutineCount()

	exec := executor.NewWithDelay(10 * time.Millisecond)

	// Create pool with many workers
	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 20,
		QueueSize:  100,
	})
	p.SetExecutor(exec)
	p.Start()

	// Should have baseline + 20 workers
	afterStart := getGoroutineCount()
	assert.GreaterOrEqual(t, afterStart, baseline+20, "Should have at least 20 worker goroutines")

	// Stop pool
	p.Stop()

	// All worker goroutines should be cleaned up
	finalCount := waitForGoroutineCount(baseline+2, 500*time.Millisecond)

	assert.LessOrEqual(t, finalCount, baseline+2,
		"All worker goroutines should be cleaned up (baseline=%d, final=%d)", baseline, finalCount)
}

// TestPoolLongRunningJobsNoLeak tests no leak with long-running jobs
func TestPoolLongRunningJobsNoLeak(t *testing.T) {
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	baseline := getGoroutineCount()

	exec := executor.NewWithDelay(200 * time.Millisecond)
	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 3,
		QueueSize:  100,
	})
	p.SetExecutor(exec)
	p.Start()

	// Submit jobs
	ctx := types.NewContext(context.Background(), nil)
	robot := createTestRobot("robot_1", "team_1", 5, 10, 5)
	for i := 0; i < 5; i++ {
		p.Submit(ctx, robot, types.TriggerClock, nil)
	}

	// Wait for some jobs to complete
	time.Sleep(500 * time.Millisecond)

	// Stop pool
	p.Stop()

	finalCount := waitForGoroutineCount(baseline+2, 500*time.Millisecond)

	assert.LessOrEqual(t, finalCount, baseline+2,
		"No goroutine leak after long-running jobs (baseline=%d, final=%d)", baseline, finalCount)
}

// TestQueueNoGoroutineLeak tests that queue operations don't leak goroutines
func TestQueueNoGoroutineLeak(t *testing.T) {
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	baseline := getGoroutineCount()

	// Create queue and perform many operations
	pq := pool.NewPriorityQueue(1000)

	// Enqueue many items
	for i := 0; i < 500; i++ {
		robot := createTestRobot("robot_"+string(rune('A'+i%26)), "team_1", 5, 100, 5)
		pq.Enqueue(&pool.QueueItem{
			Robot:   robot,
			Trigger: types.TriggerClock,
		})
	}

	// Dequeue all items
	for pq.Size() > 0 {
		pq.Dequeue()
	}

	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	finalCount := getGoroutineCount()

	assert.LessOrEqual(t, finalCount, baseline+2,
		"Queue operations should not leak goroutines (baseline=%d, final=%d)", baseline, finalCount)
}
