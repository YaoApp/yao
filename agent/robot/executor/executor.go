package executor

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/yaoapp/yao/agent/robot/job"
	"github.com/yaoapp/yao/agent/robot/types"
)

// Config holds executor configuration
type Config struct {
	// SkipJobIntegration skips job system integration (for unit tests)
	SkipJobIntegration bool

	// OnPhaseStart callback when a phase starts (for testing)
	OnPhaseStart func(phase types.Phase)

	// OnPhaseEnd callback when a phase ends (for testing)
	OnPhaseEnd func(phase types.Phase)
}

// Executor implements types.Executor interface
// This is a stub implementation that simulates full execution with Job integration
//
// Phase Implementation Strategy:
// Each phase has a dedicated file and method:
//   - inspiration.go: RunInspiration() - P0
//   - goals.go:       RunGoals()       - P1
//   - tasks.go:       RunTasks()       - P2
//   - run.go:         RunExecution()   - P3
//   - delivery.go:    RunDelivery()    - P4
//   - learning.go:    RunLearning()    - P5
//
// Currently all methods return mock data with simulated delay.
// When implementing real phases (Phase 4+), replace the method body with
// actual Agent Stream calls (Assistant.Stream()).
type Executor struct {
	config       Config
	execCount    atomic.Int32 // total execution count
	currentCount atomic.Int32 // currently running count
	onStart      func()       // callback on execution start (for testing)
	onEnd        func()       // callback on execution end (for testing)
}

// New creates a new executor instance
func New() *Executor {
	return &Executor{}
}

// NewWithConfig creates a new executor with custom configuration
func NewWithConfig(config Config) *Executor {
	return &Executor{
		config: config,
	}
}

// NewWithDelay creates a new executor with simulated delay (for testing)
// Note: delay parameter is kept for API compatibility but not used internally
// Real delay comes from simulateStreamDelay() which simulates Agent Stream latency
func NewWithDelay(_ time.Duration) *Executor {
	return &Executor{
		config: Config{
			SkipJobIntegration: true, // Skip job integration for simple delay tests
		},
	}
}

// NewWithCallback creates a new executor with callbacks (for testing concurrency)
func NewWithCallback(_ time.Duration, onStart, onEnd func()) *Executor {
	return &Executor{
		config: Config{
			SkipJobIntegration: true, // Skip job integration for callback tests
		},
		onStart: onStart,
		onEnd:   onEnd,
	}
}

// Execute executes a robot through all phases
// This stub implementation:
// 1. Creates Execution record + Job (via job package)
// 2. Updates phase: P0 → P1 → P2 → P3 → P4 → P5
// 3. Logs phase transitions
// 4. Returns success with mock data
func (e *Executor) Execute(ctx *types.Context, robot *types.Robot, trigger types.TriggerType, data interface{}) (*types.Execution, error) {
	if robot == nil {
		return nil, fmt.Errorf("robot cannot be nil")
	}

	var exec *types.Execution
	var err error

	// Determine starting phase based on trigger type
	// Clock trigger starts from P0 (Inspiration)
	// Human/Event triggers skip P0 and start from P1 (Goals)
	startPhaseIndex := 0
	if trigger == types.TriggerHuman || trigger == types.TriggerEvent {
		startPhaseIndex = 1 // Skip P0 (Inspiration)
	}

	// Create execution with Job integration
	if !e.config.SkipJobIntegration {
		exec, err = job.CreateExecution(ctx, &job.CreateOptions{
			Robot:       robot,
			TriggerType: trigger,
			Input:       buildTriggerInput(trigger, data),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create execution: %w", err)
		}
	} else {
		// Simple execution for tests without job integration
		exec = &types.Execution{
			ID:          fmt.Sprintf("exec_%d", time.Now().UnixNano()),
			MemberID:    robot.MemberID,
			TeamID:      robot.TeamID,
			TriggerType: trigger,
			StartTime:   time.Now(),
			Status:      types.ExecPending,
			Phase:       types.AllPhases[startPhaseIndex],
			Input:       buildTriggerInput(trigger, data),
		}
	}

	// Atomically check quota and acquire slot
	// This prevents race condition where multiple workers pass CanRun() check
	// but then all add executions, exceeding the quota
	if !robot.TryAcquireSlot(exec) {
		// If job was created, mark it as failed
		if !e.config.SkipJobIntegration && exec.JobID != "" {
			_ = job.FailExecution(ctx, exec, types.ErrQuotaExceeded)
		}
		return nil, types.ErrQuotaExceeded
	}
	defer robot.RemoveExecution(exec.ID)

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

	// Update status to running
	exec.Status = types.ExecRunning
	if !e.config.SkipJobIntegration {
		if err := job.UpdateStatus(ctx, exec, types.ExecRunning); err != nil {
			// Log error but continue execution
			_ = job.LogWarn(ctx, exec, fmt.Sprintf("Failed to update status to running: %v", err))
		}
	}

	// Check for simulated failure
	if dataStr, ok := data.(string); ok && dataStr == "simulate_failure" {
		exec.Status = types.ExecFailed
		exec.Error = "simulated failure"
		if !e.config.SkipJobIntegration {
			_ = job.FailExecution(ctx, exec, fmt.Errorf("simulated failure"))
		}
		return exec, nil
	}

	// Execute phases
	phases := types.AllPhases[startPhaseIndex:]
	for _, phase := range phases {
		// Run phase with common pre/post processing
		if err := e.runPhase(ctx, exec, phase, data); err != nil {
			exec.Status = types.ExecFailed
			exec.Error = err.Error()
			if !e.config.SkipJobIntegration {
				_ = job.FailExecution(ctx, exec, err)
			}
			return exec, nil
		}
	}

	// Mark execution as completed
	exec.Status = types.ExecCompleted
	now := time.Now()
	exec.EndTime = &now

	if !e.config.SkipJobIntegration {
		if err := job.CompleteExecution(ctx, exec); err != nil {
			// Log error but return success since execution completed
			_ = job.LogWarn(ctx, exec, fmt.Sprintf("Failed to mark execution as completed: %v", err))
		}
	}

	return exec, nil
}

// runPhase executes a single phase with common pre/post processing
func (e *Executor) runPhase(ctx *types.Context, exec *types.Execution, phase types.Phase, data interface{}) error {
	// Update phase
	exec.Phase = phase

	// Log phase start
	if !e.config.SkipJobIntegration {
		if err := job.UpdatePhase(ctx, exec, phase); err != nil {
			// Log error but continue
			_ = job.LogWarn(ctx, exec, fmt.Sprintf("Failed to update phase to %s: %v", phase, err))
		}
	}

	// Call phase start callback
	if e.config.OnPhaseStart != nil {
		e.config.OnPhaseStart(phase)
	}

	phaseStart := time.Now()

	// Execute phase-specific logic
	// Each phase method calls the corresponding Agent via Assistant.Stream()
	// Currently returns mock data; replace with real Agent calls in Phase 4+
	var err error
	switch phase {
	case types.PhaseInspiration:
		err = e.RunInspiration(ctx, exec, data)
	case types.PhaseGoals:
		err = e.RunGoals(ctx, exec, data)
	case types.PhaseTasks:
		err = e.RunTasks(ctx, exec, data)
	case types.PhaseRun:
		err = e.RunExecution(ctx, exec, data)
	case types.PhaseDelivery:
		err = e.RunDelivery(ctx, exec, data)
	case types.PhaseLearning:
		err = e.RunLearning(ctx, exec, data)
	}

	if err != nil {
		// Log phase error
		if !e.config.SkipJobIntegration {
			_ = job.LogPhaseError(ctx, exec, phase, err)
		}
		return err
	}

	// Call phase end callback
	if e.config.OnPhaseEnd != nil {
		e.config.OnPhaseEnd(phase)
	}

	// Log phase end
	if !e.config.SkipJobIntegration {
		phaseDuration := time.Since(phaseStart).Milliseconds()
		_ = job.LogPhaseEnd(ctx, exec, phase, phaseDuration)
	}

	return nil
}

// buildTriggerInput builds TriggerInput from trigger data
func buildTriggerInput(trigger types.TriggerType, data interface{}) *types.TriggerInput {
	input := &types.TriggerInput{}

	switch trigger {
	case types.TriggerClock:
		input.Clock = types.NewClockContext(time.Now(), "")

	case types.TriggerHuman:
		if req, ok := data.(*types.InterveneRequest); ok {
			input.Action = req.Action
			input.Messages = req.Messages
		}

	case types.TriggerEvent:
		if req, ok := data.(*types.EventRequest); ok {
			input.Source = types.EventSource(req.Source)
			input.EventType = req.EventType
			input.Data = req.Data
		}
	}

	return input
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

// DefaultStreamDelay is the simulated delay for Agent Stream calls
// This will be removed when real Agent calls are implemented
const DefaultStreamDelay = 50 * time.Millisecond

// simulateStreamDelay simulates the delay of an Agent Stream call
// This will be removed when real Agent calls are implemented in Phase 4+
func (e *Executor) simulateStreamDelay() {
	time.Sleep(DefaultStreamDelay)
}
