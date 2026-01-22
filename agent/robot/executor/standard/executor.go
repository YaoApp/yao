package standard

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/agent/robot/executor/types"
	"github.com/yaoapp/yao/agent/robot/store"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/robot/utils"
)

// Executor implements the standard executor with real Agent calls
// This is the production executor that:
// - Persists execution history to database
// - Calls real Agents via Assistant.Stream()
// - Logs phase transitions and errors using kun/log
type Executor struct {
	config       types.Config
	store        *store.ExecutionStore
	execCount    atomic.Int32
	currentCount atomic.Int32
	onStart      func()
	onEnd        func()
}

// New creates a new standard executor
func New() *Executor {
	return &Executor{
		store: store.NewExecutionStore(),
	}
}

// NewWithConfig creates a new standard executor with configuration
func NewWithConfig(config types.Config) *Executor {
	return &Executor{
		config: config,
		store:  store.NewExecutionStore(),
	}
}

// Execute runs a robot through all applicable phases with real Agent calls
func (e *Executor) Execute(ctx *robottypes.Context, robot *robottypes.Robot, trigger robottypes.TriggerType, data interface{}) (*robottypes.Execution, error) {
	if robot == nil {
		return nil, fmt.Errorf("robot cannot be nil")
	}

	// Determine starting phase based on trigger type
	startPhaseIndex := 0
	if trigger == robottypes.TriggerHuman || trigger == robottypes.TriggerEvent {
		startPhaseIndex = 1 // Skip P0 (Inspiration)
	}

	// Create execution (Job system removed, using ExecutionStore only)
	exec := &robottypes.Execution{
		ID:          utils.NewID(),
		MemberID:    robot.MemberID,
		TeamID:      robot.TeamID,
		TriggerType: trigger,
		StartTime:   time.Now(),
		Status:      robottypes.ExecPending,
		Phase:       robottypes.AllPhases[startPhaseIndex],
		Input:       types.BuildTriggerInput(trigger, data),
	}

	// Set robot reference for phase methods
	exec.SetRobot(robot)

	// Persist execution record to database
	// Robot is identified by member_id (globally unique in __yao.member table)
	if !e.config.SkipPersistence && e.store != nil {
		record := store.FromExecution(exec)
		if err := e.store.Save(ctx.Context, record); err != nil {
			// Log warning but don't fail execution
			log.With(log.F{
				"execution_id": exec.ID,
				"member_id":    exec.MemberID,
				"error":        err,
			}).Warn("Failed to persist execution record: %v", err)
		}
	}

	// Acquire execution slot
	if !robot.TryAcquireSlot(exec) {
		log.With(log.F{
			"execution_id": exec.ID,
			"member_id":    exec.MemberID,
		}).Warn("Execution quota exceeded")
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
	log.With(log.F{
		"execution_id": exec.ID,
		"member_id":    exec.MemberID,
		"trigger_type": string(exec.TriggerType),
	}).Info("Execution started")

	// Persist running status
	if !e.config.SkipPersistence && e.store != nil {
		if err := e.store.UpdateStatus(ctx.Context, exec.ID, robottypes.ExecRunning, ""); err != nil {
			log.With(log.F{
				"execution_id": exec.ID,
				"error":        err,
			}).Warn("Failed to persist running status: %v", err)
		}
	}

	// Check for simulated failure (for testing)
	if dataStr, ok := data.(string); ok && dataStr == "simulate_failure" {
		exec.Status = robottypes.ExecFailed
		exec.Error = "simulated failure"
		log.With(log.F{
			"execution_id": exec.ID,
			"member_id":    exec.MemberID,
		}).Warn("Simulated failure triggered")
		// Persist failed status
		if !e.config.SkipPersistence && e.store != nil {
			_ = e.store.UpdateStatus(ctx.Context, exec.ID, robottypes.ExecFailed, "simulated failure")
		}
		return exec, nil
	}

	// Execute phases
	phases := robottypes.AllPhases[startPhaseIndex:]
	for _, phase := range phases {
		if err := e.runPhase(ctx, exec, phase, data); err != nil {
			exec.Status = robottypes.ExecFailed
			exec.Error = err.Error()
			log.With(log.F{
				"execution_id": exec.ID,
				"member_id":    exec.MemberID,
				"phase":        string(phase),
				"error":        err.Error(),
			}).Error("Phase execution failed: %v", err)
			// Persist failed status
			if !e.config.SkipPersistence && e.store != nil {
				_ = e.store.UpdateStatus(ctx.Context, exec.ID, robottypes.ExecFailed, err.Error())
			}
			return exec, nil
		}
	}

	// Mark completed
	exec.Status = robottypes.ExecCompleted
	now := time.Now()
	exec.EndTime = &now

	duration := now.Sub(exec.StartTime)
	log.With(log.F{
		"execution_id": exec.ID,
		"member_id":    exec.MemberID,
		"duration_ms":  duration.Milliseconds(),
	}).Info("Execution completed successfully")

	// Persist completed status
	if !e.config.SkipPersistence && e.store != nil {
		if err := e.store.UpdateStatus(ctx.Context, exec.ID, robottypes.ExecCompleted, ""); err != nil {
			log.With(log.F{
				"execution_id": exec.ID,
				"error":        err,
			}).Warn("Failed to persist completed status: %v", err)
		}
	}

	return exec, nil
}

// runPhase executes a single phase
func (e *Executor) runPhase(ctx *robottypes.Context, exec *robottypes.Execution, phase robottypes.Phase, data interface{}) error {
	exec.Phase = phase

	log.With(log.F{
		"execution_id": exec.ID,
		"member_id":    exec.MemberID,
		"phase":        string(phase),
	}).Info("Phase started: %s", phase)

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
		log.With(log.F{
			"execution_id": exec.ID,
			"member_id":    exec.MemberID,
			"phase":        string(phase),
			"error":        err.Error(),
		}).Error("Phase failed: %s - %v", phase, err)
		return err
	}

	// Persist phase output to database
	if !e.config.SkipPersistence && e.store != nil {
		phaseData := e.getPhaseData(exec, phase)
		if phaseData != nil {
			if err := e.store.UpdatePhase(ctx.Context, exec.ID, phase, phaseData); err != nil {
				// Log warning but don't fail execution
				log.With(log.F{
					"execution_id": exec.ID,
					"phase":        string(phase),
					"error":        err,
				}).Warn("Failed to persist phase %s data: %v", phase, err)
			}
		}
	}

	if e.config.OnPhaseEnd != nil {
		e.config.OnPhaseEnd(phase)
	}

	phaseDuration := time.Since(phaseStart).Milliseconds()
	log.With(log.F{
		"execution_id": exec.ID,
		"member_id":    exec.MemberID,
		"phase":        string(phase),
		"duration_ms":  phaseDuration,
	}).Info("Phase completed: %s (took %dms)", phase, phaseDuration)

	return nil
}

// getPhaseData extracts the output data for a specific phase from execution
func (e *Executor) getPhaseData(exec *robottypes.Execution, phase robottypes.Phase) interface{} {
	switch phase {
	case robottypes.PhaseInspiration:
		return exec.Inspiration
	case robottypes.PhaseGoals:
		return exec.Goals
	case robottypes.PhaseTasks:
		return exec.Tasks
	case robottypes.PhaseRun:
		return exec.Results
	case robottypes.PhaseDelivery:
		return exec.Delivery
	case robottypes.PhaseLearning:
		return exec.Learning
	default:
		return nil
	}
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
