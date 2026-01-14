package executor

import (
	"sync/atomic"
	"time"

	"github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/robot/utils"
)

// Executor implements types.Executor interface
// This is a stub implementation for Phase 2
type Executor struct {
	delay        time.Duration // simulated execution delay
	execCount    atomic.Int32  // total execution count
	currentCount atomic.Int32  // currently running count
	onStart      func()        // callback on execution start (for testing)
	onEnd        func()        // callback on execution end (for testing)
}

// New creates a new executor instance
func New() *Executor {
	return &Executor{}
}

// NewWithDelay creates a new executor with simulated delay (for testing)
func NewWithDelay(delay time.Duration) *Executor {
	return &Executor{
		delay: delay,
	}
}

// NewWithCallback creates a new executor with callbacks (for testing concurrency)
func NewWithCallback(delay time.Duration, onStart, onEnd func()) *Executor {
	return &Executor{
		delay:   delay,
		onStart: onStart,
		onEnd:   onEnd,
	}
}

// Execute executes a robot through all phases
// Stub: returns empty execution (will be implemented in Phase 3+)
func (e *Executor) Execute(ctx *types.Context, robot *types.Robot, trigger types.TriggerType, data interface{}) (*types.Execution, error) {
	// Create execution record first
	execID := utils.NewID()
	exec := &types.Execution{
		ID:          execID,
		MemberID:    robot.MemberID,
		TeamID:      robot.TeamID,
		TriggerType: trigger,
		Status:      types.ExecRunning,
		Phase:       types.PhaseInspiration,
	}

	// Atomically check quota and acquire slot
	// This prevents race condition where multiple workers pass CanRun() check
	// but then all add executions, exceeding the quota
	if !robot.TryAcquireSlot(exec) {
		return nil, types.ErrQuotaExceeded
	}
	defer robot.RemoveExecution(execID)

	// Track execution count (after successful slot acquisition)
	e.execCount.Add(1)
	e.currentCount.Add(1)
	defer e.currentCount.Add(-1)

	// Call start callback if set
	if e.onStart != nil {
		e.onStart()
	}
	// Call end callback on return
	if e.onEnd != nil {
		defer e.onEnd()
	}

	// Simulate execution delay
	if e.delay > 0 {
		time.Sleep(e.delay)
	}

	// Check for simulated failure
	if dataStr, ok := data.(string); ok && dataStr == "simulate_failure" {
		exec.Status = types.ExecFailed
		return exec, nil // return error is optional, we track status
	}

	// Update execution status
	exec.Status = types.ExecCompleted
	exec.Phase = types.PhaseLearning

	return exec, nil
}

// ExecCount returns total execution count
func (e *Executor) ExecCount() int {
	return int(e.execCount.Load())
}

// CurrentCount returns currently running execution count
func (e *Executor) CurrentCount() int {
	return int(e.currentCount.Load())
}

// Reset resets the executor counters (for testing)
func (e *Executor) Reset() {
	e.execCount.Store(0)
	e.currentCount.Store(0)
}
