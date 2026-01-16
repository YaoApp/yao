package standard

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/yaoapp/yao/agent/robot/executor/types"
	"github.com/yaoapp/yao/agent/robot/job"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
)

// Executor implements the standard executor with real Agent calls
// This is the production executor that:
// - Creates Job records for tracking
// - Calls real Agents via Assistant.Stream()
// - Logs phase transitions and errors
type Executor struct {
	config       types.Config
	execCount    atomic.Int32
	currentCount atomic.Int32
	onStart      func()
	onEnd        func()
}

// New creates a new standard executor
func New() *Executor {
	return &Executor{}
}

// NewWithConfig creates a new standard executor with configuration
func NewWithConfig(config types.Config) *Executor {
	return &Executor{
		config: config,
	}
}

// Execute runs a robot through all applicable phases with real Agent calls
func (e *Executor) Execute(ctx *robottypes.Context, robot *robottypes.Robot, trigger robottypes.TriggerType, data interface{}) (*robottypes.Execution, error) {
	if robot == nil {
		return nil, fmt.Errorf("robot cannot be nil")
	}

	var exec *robottypes.Execution
	var err error

	// Determine starting phase based on trigger type
	startPhaseIndex := 0
	if trigger == robottypes.TriggerHuman || trigger == robottypes.TriggerEvent {
		startPhaseIndex = 1 // Skip P0 (Inspiration)
	}

	// Create execution with Job integration
	if !e.config.SkipJobIntegration {
		exec, err = job.CreateExecution(ctx, &job.CreateOptions{
			Robot:       robot,
			TriggerType: trigger,
			Input:       types.BuildTriggerInput(trigger, data),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create execution: %w", err)
		}
	} else {
		exec = &robottypes.Execution{
			ID:          fmt.Sprintf("exec_%d", time.Now().UnixNano()),
			MemberID:    robot.MemberID,
			TeamID:      robot.TeamID,
			TriggerType: trigger,
			StartTime:   time.Now(),
			Status:      robottypes.ExecPending,
			Phase:       robottypes.AllPhases[startPhaseIndex],
			Input:       types.BuildTriggerInput(trigger, data),
		}
	}

	// Set robot reference for phase methods
	exec.SetRobot(robot)

	// Acquire execution slot
	if !robot.TryAcquireSlot(exec) {
		if !e.config.SkipJobIntegration && exec.JobID != "" {
			_ = job.FailExecution(ctx, exec, robottypes.ErrQuotaExceeded)
		}
		return nil, robottypes.ErrQuotaExceeded
	}
	defer robot.RemoveExecution(exec.ID)

	// Track execution count
	e.execCount.Add(1)
	e.currentCount.Add(1)
	defer e.currentCount.Add(-1)

	// Callbacks
	if e.onStart != nil {
		e.onStart()
	}
	if e.onEnd != nil {
		defer e.onEnd()
	}

	// Update status to running
	exec.Status = robottypes.ExecRunning
	if !e.config.SkipJobIntegration {
		if err := job.UpdateStatus(ctx, exec, robottypes.ExecRunning); err != nil {
			_ = job.LogWarn(ctx, exec, fmt.Sprintf("Failed to update status to running: %v", err))
		}
	}

	// Check for simulated failure (for testing)
	if dataStr, ok := data.(string); ok && dataStr == "simulate_failure" {
		exec.Status = robottypes.ExecFailed
		exec.Error = "simulated failure"
		if !e.config.SkipJobIntegration {
			_ = job.FailExecution(ctx, exec, fmt.Errorf("simulated failure"))
		}
		return exec, nil
	}

	// Execute phases
	phases := robottypes.AllPhases[startPhaseIndex:]
	for _, phase := range phases {
		if err := e.runPhase(ctx, exec, phase, data); err != nil {
			exec.Status = robottypes.ExecFailed
			exec.Error = err.Error()
			if !e.config.SkipJobIntegration {
				_ = job.FailExecution(ctx, exec, err)
			}
			return exec, nil
		}
	}

	// Mark completed
	exec.Status = robottypes.ExecCompleted
	now := time.Now()
	exec.EndTime = &now

	if !e.config.SkipJobIntegration {
		if err := job.CompleteExecution(ctx, exec); err != nil {
			_ = job.LogWarn(ctx, exec, fmt.Sprintf("Failed to mark execution as completed: %v", err))
		}
	}

	return exec, nil
}

// runPhase executes a single phase
func (e *Executor) runPhase(ctx *robottypes.Context, exec *robottypes.Execution, phase robottypes.Phase, data interface{}) error {
	exec.Phase = phase

	if !e.config.SkipJobIntegration {
		if err := job.UpdatePhase(ctx, exec, phase); err != nil {
			_ = job.LogWarn(ctx, exec, fmt.Sprintf("Failed to update phase to %s: %v", phase, err))
		}
	}

	if e.config.OnPhaseStart != nil {
		e.config.OnPhaseStart(phase)
	}

	phaseStart := time.Now()

	// Execute phase-specific logic
	var err error
	switch phase {
	case robottypes.PhaseInspiration:
		err = e.RunInspiration(ctx, exec, data)
	case robottypes.PhaseGoals:
		err = e.RunGoals(ctx, exec, data)
	case robottypes.PhaseTasks:
		err = e.RunTasks(ctx, exec, data)
	case robottypes.PhaseRun:
		err = e.RunExecution(ctx, exec, data)
	case robottypes.PhaseDelivery:
		err = e.RunDelivery(ctx, exec, data)
	case robottypes.PhaseLearning:
		err = e.RunLearning(ctx, exec, data)
	}

	if err != nil {
		if !e.config.SkipJobIntegration {
			_ = job.LogPhaseError(ctx, exec, phase, err)
		}
		return err
	}

	if e.config.OnPhaseEnd != nil {
		e.config.OnPhaseEnd(phase)
	}

	if !e.config.SkipJobIntegration {
		phaseDuration := time.Since(phaseStart).Milliseconds()
		_ = job.LogPhaseEnd(ctx, exec, phase, phaseDuration)
	}

	return nil
}

// ExecCount returns total execution count
func (e *Executor) ExecCount() int {
	return int(e.execCount.Load())
}

// CurrentCount returns currently running execution count
func (e *Executor) CurrentCount() int {
	return int(e.currentCount.Load())
}

// Reset resets the executor counters
func (e *Executor) Reset() {
	e.execCount.Store(0)
	e.currentCount.Store(0)
}

// DefaultStreamDelay is the simulated delay for Agent Stream calls
// This will be removed when real Agent calls are implemented
const DefaultStreamDelay = 50 * time.Millisecond

// simulateStreamDelay simulates the delay of an Agent Stream call
func (e *Executor) simulateStreamDelay() {
	time.Sleep(DefaultStreamDelay)
}

// Verify Executor implements types.Executor
var _ types.Executor = (*Executor)(nil)
