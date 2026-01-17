package executor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent/robot/types"
)

// Smoke tests to verify basic flow works
// These tests use SkipJobIntegration=true to avoid DB dependencies
// Real integration tests are in manager_test.go and job_test.go

func TestExecutorSmoke(t *testing.T) {
	exec := NewDryRunWithDelay(0)
	robot := &types.Robot{
		MemberID: "test-smoke",
		TeamID:   "team-1",
		Config:   &types.Config{Quota: &types.Quota{Max: 1}},
	}
	ctx := types.NewContext(context.Background(), nil)

	result, err := exec.Execute(ctx, robot, types.TriggerClock, nil)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, types.ExecCompleted, result.Status)
	assert.Equal(t, types.TriggerClock, result.TriggerType)

	// Clock trigger executes all phases (P0-P5)
	assert.NotNil(t, result.Inspiration, "P0 should be executed for clock trigger")
	assert.NotNil(t, result.Goals, "P1 should be executed")
	assert.NotEmpty(t, result.Tasks, "P2 should generate tasks")
	assert.NotEmpty(t, result.Results, "P3 should generate results")
	assert.NotNil(t, result.Delivery, "P4 should be executed")
	assert.NotEmpty(t, result.Learning, "P5 should be executed")
}

func TestExecutorHumanTriggerSkipsP0(t *testing.T) {
	exec := NewDryRunWithDelay(0)
	robot := &types.Robot{
		MemberID: "test-human",
		TeamID:   "team-1",
		Config:   &types.Config{Quota: &types.Quota{Max: 1}},
	}
	ctx := types.NewContext(context.Background(), nil)

	result, err := exec.Execute(ctx, robot, types.TriggerHuman, nil)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, types.ExecCompleted, result.Status)

	// Human trigger skips P0 (Inspiration)
	assert.Nil(t, result.Inspiration, "P0 should be skipped for human trigger")
	assert.NotNil(t, result.Goals, "P1 should be executed")
}

func TestExecutorEventTriggerSkipsP0(t *testing.T) {
	exec := NewDryRunWithDelay(0)
	robot := &types.Robot{
		MemberID: "test-event",
		TeamID:   "team-1",
		Config:   &types.Config{Quota: &types.Quota{Max: 1}},
	}
	ctx := types.NewContext(context.Background(), nil)

	result, err := exec.Execute(ctx, robot, types.TriggerEvent, nil)

	assert.NoError(t, err)
	assert.Nil(t, result.Inspiration, "P0 should be skipped for event trigger")
	assert.NotNil(t, result.Goals)
}

func TestExecutorNilRobot(t *testing.T) {
	exec := NewDryRunWithDelay(0)
	ctx := types.NewContext(context.Background(), nil)

	result, err := exec.Execute(ctx, nil, types.TriggerClock, nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "robot cannot be nil")
}

func TestExecutorSimulatedFailure(t *testing.T) {
	exec := NewDryRunWithDelay(0)
	robot := &types.Robot{
		MemberID: "test-fail",
		TeamID:   "team-1",
		Config:   &types.Config{Quota: &types.Quota{Max: 1}},
	}
	ctx := types.NewContext(context.Background(), nil)

	// Pass "simulate_failure" to trigger simulated failure
	result, err := exec.Execute(ctx, robot, types.TriggerClock, "simulate_failure")

	assert.NoError(t, err) // Execute returns nil error, failure is in result
	assert.NotNil(t, result)
	assert.Equal(t, types.ExecFailed, result.Status)
	assert.Equal(t, "simulated failure", result.Error)
}

func TestExecutorCounters(t *testing.T) {
	exec := NewDryRunWithDelay(0)
	robot := &types.Robot{
		MemberID: "test-counter",
		TeamID:   "team-1",
		Config:   &types.Config{Quota: &types.Quota{Max: 10}},
	}
	ctx := types.NewContext(context.Background(), nil)

	assert.Equal(t, 0, exec.ExecCount())
	assert.Equal(t, 0, exec.CurrentCount())

	_, _ = exec.Execute(ctx, robot, types.TriggerClock, nil)
	assert.Equal(t, 1, exec.ExecCount())
	assert.Equal(t, 0, exec.CurrentCount()) // Completed, so 0

	_, _ = exec.Execute(ctx, robot, types.TriggerClock, nil)
	assert.Equal(t, 2, exec.ExecCount())

	exec.Reset()
	assert.Equal(t, 0, exec.ExecCount())
}
