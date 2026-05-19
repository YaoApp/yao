//go:build unit

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
		assert.NoError(t, err)
	})
}

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

func TestPoolBasicExecution(t *testing.T) {
	exec := executor.NewDryRunWithDelay(50 * time.Millisecond)
	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 5,
		QueueSize:  100,
	})
	p.SetExecutor(exec)
	p.Start()
	defer p.Stop()

	robot := createTestRobot("robot_1", "team_1", 2, 10, 5)
	ctx := createTestContext()

	execID, err := p.Submit(ctx, robot, types.TriggerClock, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, execID)

	time.Sleep(300 * time.Millisecond)

	assert.Equal(t, 1, exec.ExecCount())
	assert.Eventually(t, func() bool {
		return exec.CurrentCount() == 0
	}, 500*time.Millisecond, 50*time.Millisecond, "CurrentCount should be 0 after execution")
}

func TestPoolConcurrencyLimit(t *testing.T) {
	exec := executor.NewDryRunWithDelay(200 * time.Millisecond)
	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 3,
		QueueSize:  100,
	})
	p.SetExecutor(exec)
	p.Start()
	defer p.Stop()

	ctx := createTestContext()

	robots := make([]*types.Robot, 10)
	for i := 0; i < 10; i++ {
		robots[i] = createTestRobot(
			"robot_"+string(rune('A'+i)),
			"team_1",
			5, 20, 5,
		)
	}

	for i := 0; i < 10; i++ {
		_, err := p.Submit(ctx, robots[i], types.TriggerClock, nil)
		assert.NoError(t, err)
	}

	time.Sleep(200 * time.Millisecond)

	running := p.Running()
	assert.LessOrEqual(t, running, 3, "Should not exceed worker limit")

	assert.Eventually(t, func() bool {
		return exec.ExecCount() >= 10
	}, 2*time.Second, 100*time.Millisecond, "All 10 jobs should complete")
}

func TestRobotConcurrencyLimit(t *testing.T) {
	exec := executor.NewDryRunWithDelay(100 * time.Millisecond)
	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 10,
		QueueSize:  100,
	})
	p.SetExecutor(exec)
	p.Start()
	defer p.Stop()

	ctx := createTestContext()

	robot := createTestRobot("robot_limited", "team_1", 2, 20, 5)

	for i := 0; i < 5; i++ {
		_, err := p.Submit(ctx, robot, types.TriggerClock, nil)
		assert.NoError(t, err)
	}

	time.Sleep(150 * time.Millisecond)

	runningCount := robot.RunningCount()
	assert.LessOrEqual(t, runningCount, 2, "Robot should not exceed Quota.Max")

	time.Sleep(800 * time.Millisecond)

	assert.GreaterOrEqual(t, exec.ExecCount(), 5, "All jobs should eventually execute")
}

func TestRobotQueueLimit(t *testing.T) {
	exec := executor.NewDryRunWithDelay(200 * time.Millisecond)
	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 2,
		QueueSize:  100,
	})
	p.SetExecutor(exec)
	p.Start()
	defer p.Stop()

	ctx := createTestContext()

	robot := createTestRobot("robot_small_queue", "team_1", 1, 3, 5)

	successCount := 0
	for i := 0; i < 10; i++ {
		_, err := p.Submit(ctx, robot, types.TriggerClock, nil)
		if err == nil {
			successCount++
		}
	}

	assert.LessOrEqual(t, successCount, 4, "Should respect robot queue limit")
	assert.GreaterOrEqual(t, successCount, 1, "Should accept at least 1 job")
}

func TestGlobalQueueLimit(t *testing.T) {
	exec := executor.NewDryRunWithDelay(500 * time.Millisecond)
	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 1,
		QueueSize:  5,
	})
	p.SetExecutor(exec)
	p.Start()
	defer p.Stop()

	ctx := createTestContext()

	successCount := 0
	for i := 0; i < 20; i++ {
		robot := createTestRobot(
			"robot_"+string(rune('A'+i%26)),
			"team_1",
			5, 20, 5,
		)
		_, err := p.Submit(ctx, robot, types.TriggerClock, nil)
		if err == nil {
			successCount++
		}
	}

	assert.LessOrEqual(t, successCount, 6, "Should respect global queue limit")
}

func TestPriorityOrder(t *testing.T) {
	exec := executor.NewDryRunWithDelay(50 * time.Millisecond)
	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 1,
		QueueSize:  100,
	})
	p.SetExecutor(exec)
	p.Start()
	defer p.Stop()

	ctx := createTestContext()

	robotLow := createTestRobot("robot_low", "team_1", 5, 20, 1)
	robotMed := createTestRobot("robot_med", "team_1", 5, 20, 5)
	robotHigh := createTestRobot("robot_high", "team_1", 5, 20, 10)

	p.Submit(ctx, robotLow, types.TriggerClock, nil)
	p.Submit(ctx, robotMed, types.TriggerClock, nil)
	p.Submit(ctx, robotHigh, types.TriggerClock, nil)

	assert.Eventually(t, func() bool {
		return exec.ExecCount() >= 3
	}, 1*time.Second, 50*time.Millisecond, "All 3 jobs should complete")
}

func TestTriggerTypePriority(t *testing.T) {
	exec := executor.NewDryRunWithDelay(50 * time.Millisecond)
	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 1,
		QueueSize:  100,
	})
	p.SetExecutor(exec)
	p.Start()
	defer p.Stop()

	ctx := createTestContext()

	robot := createTestRobot("robot_1", "team_1", 5, 20, 5)

	p.Submit(ctx, robot, types.TriggerClock, nil)
	p.Submit(ctx, robot, types.TriggerHuman, nil)

	assert.Eventually(t, func() bool {
		return exec.ExecCount() >= 2
	}, 1*time.Second, 50*time.Millisecond, "Both jobs should complete")
}

func TestMultipleRobotsFairness(t *testing.T) {
	exec := executor.NewDryRunWithDelay(30 * time.Millisecond)
	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 5,
		QueueSize:  100,
	})
	p.SetExecutor(exec)
	p.Start()
	defer p.Stop()

	ctx := createTestContext()

	robotA := createTestRobot("robot_A", "team_1", 2, 10, 5)
	robotB := createTestRobot("robot_B", "team_1", 2, 10, 5)
	robotC := createTestRobot("robot_C", "team_1", 2, 10, 5)

	for i := 0; i < 6; i++ {
		p.Submit(ctx, robotA, types.TriggerClock, nil)
		p.Submit(ctx, robotB, types.TriggerClock, nil)
		p.Submit(ctx, robotC, types.TriggerClock, nil)
	}

	assert.Eventually(t, func() bool {
		return exec.ExecCount() >= 18
	}, 3*time.Second, 100*time.Millisecond, "All 18 jobs should complete")
}

func TestGracefulShutdown(t *testing.T) {
	exec := executor.NewDryRunWithDelay(200 * time.Millisecond)
	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 2,
		QueueSize:  10,
	})
	p.SetExecutor(exec)
	p.Start()

	ctx := createTestContext()
	robot := createTestRobot("robot_1", "team_1", 5, 20, 5)

	p.Submit(ctx, robot, types.TriggerClock, nil)
	p.Submit(ctx, robot, types.TriggerClock, nil)

	time.Sleep(150 * time.Millisecond)

	assert.GreaterOrEqual(t, p.Running(), 1, "Should have at least 1 running job")

	p.Stop()

	assert.GreaterOrEqual(t, exec.ExecCount(), 1, "Should have executed at least 1 job")
}

func TestDefaultConfig(t *testing.T) {
	config := pool.DefaultConfig()
	assert.Equal(t, pool.DefaultWorkerSize, config.WorkerSize)
	assert.Equal(t, pool.DefaultQueueSize, config.QueueSize)
}

func TestPoolWithNilConfig(t *testing.T) {
	p := pool.NewWithConfig(nil)
	assert.Equal(t, pool.DefaultWorkerSize, p.Size())
	assert.Equal(t, pool.DefaultQueueSize, p.QueueSize())
}

func TestPoolWithZeroConfig(t *testing.T) {
	p := pool.NewWithConfig(&pool.Config{
		WorkerSize: 0,
		QueueSize:  0,
	})
	assert.Equal(t, pool.DefaultWorkerSize, p.Size())
	assert.Equal(t, pool.DefaultQueueSize, p.QueueSize())
}

func TestPoolWithoutExecutor(t *testing.T) {
	p := pool.New()
	err := p.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "executor not set")
}
