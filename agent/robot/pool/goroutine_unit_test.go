//go:build unit

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

func getGoroutineCount() int {
	return runtime.NumGoroutine()
}

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

func TestPoolNoGoroutineLeak(t *testing.T) {
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	baseline := getGoroutineCount()

	exec := executor.NewDryRunWithDelay(10 * time.Millisecond)
	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 5,
		QueueSize:  100,
	})
	p.SetExecutor(exec)
	p.Start()

	afterStart := getGoroutineCount()
	assert.Greater(t, afterStart, baseline, "Should have more goroutines after start")

	ctx := types.NewContext(context.Background(), nil)
	robot := createTestRobot("robot_1", "team_1", 5, 10, 5)
	for i := 0; i < 10; i++ {
		p.Submit(ctx, robot, types.TriggerClock, nil)
	}

	time.Sleep(300 * time.Millisecond)

	p.Stop()

	finalCount := waitForGoroutineCount(baseline+2, 500*time.Millisecond)

	assert.LessOrEqual(t, finalCount, baseline+2,
		"Goroutine count should return to near baseline after stop (baseline=%d, final=%d)", baseline, finalCount)
}

func TestPoolMultipleStartStop(t *testing.T) {
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	baseline := getGoroutineCount()

	exec := executor.NewDryRunWithDelay(5 * time.Millisecond)

	for i := 0; i < 5; i++ {
		p := pool.NewWithConfig(&pool.Config{
			WorkerSize: 3,
			QueueSize:  50,
		})
		p.SetExecutor(exec)
		p.Start()

		ctx := types.NewContext(context.Background(), nil)
		robot := createTestRobot("robot_1", "team_1", 5, 10, 5)
		for j := 0; j < 5; j++ {
			p.Submit(ctx, robot, types.TriggerClock, nil)
		}

		time.Sleep(100 * time.Millisecond)
		p.Stop()
	}

	finalCount := waitForGoroutineCount(baseline+2, 500*time.Millisecond)

	assert.LessOrEqual(t, finalCount, baseline+2,
		"Goroutine count should return to near baseline after multiple cycles (baseline=%d, final=%d)", baseline, finalCount)
}

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

	p.Stop()

	finalCount := waitForGoroutineCount(baseline+2, 500*time.Millisecond)

	assert.LessOrEqual(t, finalCount, baseline+2,
		"Goroutine count should return to near baseline (baseline=%d, final=%d)", baseline, finalCount)
}

func TestPoolStopWithPendingJobs(t *testing.T) {
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	baseline := getGoroutineCount()

	exec := executor.NewDryRunWithDelay(500 * time.Millisecond)
	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 1,
		QueueSize:  100,
	})
	p.SetExecutor(exec)
	p.Start()

	ctx := types.NewContext(context.Background(), nil)
	robot := createTestRobot("robot_1", "team_1", 5, 50, 5)
	for i := 0; i < 20; i++ {
		p.Submit(ctx, robot, types.TriggerClock, nil)
	}

	time.Sleep(50 * time.Millisecond)
	p.Stop()

	finalCount := waitForGoroutineCount(baseline+2, 500*time.Millisecond)

	assert.LessOrEqual(t, finalCount, baseline+2,
		"Goroutine count should return to near baseline even with pending jobs (baseline=%d, final=%d)", baseline, finalCount)
}

func TestPoolConcurrentStartStop(t *testing.T) {
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	baseline := getGoroutineCount()

	exec := executor.NewDryRunWithDelay(10 * time.Millisecond)
	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 5,
		QueueSize:  100,
	})
	p.SetExecutor(exec)

	p.Start()

	done := make(chan bool, 3)

	go func() {
		ctx := types.NewContext(context.Background(), nil)
		robot := createTestRobot("robot_1", "team_1", 5, 10, 5)
		for i := 0; i < 20; i++ {
			p.Submit(ctx, robot, types.TriggerClock, nil)
			time.Sleep(5 * time.Millisecond)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 20; i++ {
			_ = p.Running()
			_ = p.Queued()
			time.Sleep(5 * time.Millisecond)
		}
		done <- true
	}()

	<-done
	<-done

	p.Stop()

	finalCount := waitForGoroutineCount(baseline+2, 500*time.Millisecond)

	assert.LessOrEqual(t, finalCount, baseline+2,
		"Goroutine count should return to near baseline after concurrent ops (baseline=%d, final=%d)", baseline, finalCount)
}

func TestWorkerGoroutinesCleanup(t *testing.T) {
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	baseline := getGoroutineCount()

	exec := executor.NewDryRunWithDelay(10 * time.Millisecond)

	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 20,
		QueueSize:  100,
	})
	p.SetExecutor(exec)
	p.Start()

	afterStart := getGoroutineCount()
	assert.GreaterOrEqual(t, afterStart, baseline+20, "Should have at least 20 worker goroutines")

	p.Stop()

	finalCount := waitForGoroutineCount(baseline+2, 500*time.Millisecond)

	assert.LessOrEqual(t, finalCount, baseline+2,
		"All worker goroutines should be cleaned up (baseline=%d, final=%d)", baseline, finalCount)
}

func TestPoolLongRunningJobsNoLeak(t *testing.T) {
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	baseline := getGoroutineCount()

	exec := executor.NewDryRunWithDelay(200 * time.Millisecond)
	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 3,
		QueueSize:  100,
	})
	p.SetExecutor(exec)
	p.Start()

	ctx := types.NewContext(context.Background(), nil)
	robot := createTestRobot("robot_1", "team_1", 5, 10, 5)
	for i := 0; i < 5; i++ {
		p.Submit(ctx, robot, types.TriggerClock, nil)
	}

	time.Sleep(500 * time.Millisecond)

	p.Stop()

	finalCount := waitForGoroutineCount(baseline+2, 500*time.Millisecond)

	assert.LessOrEqual(t, finalCount, baseline+2,
		"No goroutine leak after long-running jobs (baseline=%d, final=%d)", baseline, finalCount)
}

func TestQueueNoGoroutineLeak(t *testing.T) {
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	baseline := getGoroutineCount()

	pq := pool.NewPriorityQueue(1000)

	for i := 0; i < 500; i++ {
		robot := createTestRobot("robot_"+string(rune('A'+i%26)), "team_1", 5, 100, 5)
		pq.Enqueue(&pool.QueueItem{
			Robot:   robot,
			Trigger: types.TriggerClock,
		})
	}

	for pq.Size() > 0 {
		pq.Dequeue()
	}

	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	finalCount := getGoroutineCount()

	assert.LessOrEqual(t, finalCount, baseline+2,
		"Queue operations should not leak goroutines (baseline=%d, final=%d)", baseline, finalCount)
}
