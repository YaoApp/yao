package dryrun

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/yaoapp/yao/agent/robot/executor/types"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
)

// Executor implements a dry-run executor that simulates execution
// without making real Agent calls. Useful for:
// - Testing scheduling and concurrency logic
// - Demo and preview modes
// - Debugging execution flow
// - Performance testing
type Executor struct {
	config       types.DryRunConfig
	execCount    atomic.Int32
	currentCount atomic.Int32
}

// New creates a new dry-run executor with default settings
func New() *Executor {
	return &Executor{}
}

// NewWithDelay creates a dry-run executor with specified delay
func NewWithDelay(delay time.Duration) *Executor {
	return &Executor{
		config: types.DryRunConfig{
			Delay: delay,
		},
	}
}

// NewWithConfig creates a dry-run executor with full configuration
func NewWithConfig(config types.DryRunConfig) *Executor {
	return &Executor{
		config: config,
	}
}

// Execute simulates robot execution without real Agent calls
func (e *Executor) Execute(ctx *robottypes.Context, robot *robottypes.Robot, trigger robottypes.TriggerType, data interface{}) (*robottypes.Execution, error) {
	if robot == nil {
		return nil, fmt.Errorf("robot cannot be nil")
	}

	// Determine starting phase
	startPhaseIndex := 0
	if trigger == robottypes.TriggerHuman || trigger == robottypes.TriggerEvent {
		startPhaseIndex = 1 // Skip P0
	}

	// Create execution record
	exec := &robottypes.Execution{
		ID:          fmt.Sprintf("dryrun_%d", time.Now().UnixNano()),
		MemberID:    robot.MemberID,
		TeamID:      robot.TeamID,
		TriggerType: trigger,
		StartTime:   time.Now(),
		Status:      robottypes.ExecPending,
		Phase:       robottypes.AllPhases[startPhaseIndex],
		Input:       types.BuildTriggerInput(trigger, data),
	}

	// Set robot reference
	exec.SetRobot(robot)

	// Acquire slot
	if !robot.TryAcquireSlot(exec) {
		return nil, robottypes.ErrQuotaExceeded
	}
	defer robot.RemoveExecution(exec.ID)

	// Track counts
	e.execCount.Add(1)
	e.currentCount.Add(1)
	defer e.currentCount.Add(-1)

	// Start callback
	if e.config.OnStart != nil {
		e.config.OnStart()
	}
	if e.config.OnEnd != nil {
		defer e.config.OnEnd()
	}

	// Update status
	exec.Status = robottypes.ExecRunning

	// Simulate execution delay (once for entire execution, not per-phase)
	if e.config.Delay > 0 {
		time.Sleep(e.config.Delay)
	}

	// Check for simulated failure
	if dataStr, ok := data.(string); ok && dataStr == "simulate_failure" {
		exec.Status = robottypes.ExecFailed
		exec.Error = "simulated failure"
		return exec, nil
	}

	// Execute phases with mock data
	phases := robottypes.AllPhases[startPhaseIndex:]
	for _, phase := range phases {
		exec.Phase = phase

		// Phase start callback
		if e.config.OnPhaseStart != nil {
			e.config.OnPhaseStart(phase)
		}

		// Generate mock output
		e.mockPhaseOutput(exec, phase)

		// Phase end callback
		if e.config.OnPhaseEnd != nil {
			e.config.OnPhaseEnd(phase)
		}
	}

	// Mark completed
	exec.Status = robottypes.ExecCompleted
	now := time.Now()
	exec.EndTime = &now

	return exec, nil
}

// mockPhaseOutput generates mock output for each phase
func (e *Executor) mockPhaseOutput(exec *robottypes.Execution, phase robottypes.Phase) {
	switch phase {
	case robottypes.PhaseInspiration:
		exec.Inspiration = &robottypes.InspirationReport{
			Clock:   robottypes.NewClockContext(time.Now(), ""),
			Content: "## Dry-Run Inspiration\n\nThis is a simulated inspiration report for testing.",
		}
	case robottypes.PhaseGoals:
		exec.Goals = &robottypes.Goals{
			Content: "## Dry-Run Goals\n\n1. [High] Simulated goal for testing",
		}
	case robottypes.PhaseTasks:
		exec.Tasks = []robottypes.Task{
			{
				ID:           "dryrun-task-1",
				GoalRef:      "Goal 1",
				Source:       robottypes.TaskSourceAuto,
				ExecutorType: robottypes.ExecutorAssistant,
				ExecutorID:   "mock-agent",
				Status:       robottypes.TaskPending,
			},
		}
	case robottypes.PhaseRun:
		exec.Results = []robottypes.TaskResult{
			{
				TaskID:   "dryrun-task-1",
				Success:  true,
				Output:   map[string]interface{}{"mode": "dryrun", "result": "simulated"},
				Duration: 100,
				Validation: &robottypes.ValidationResult{
					Passed: true,
					Score:  1.0,
				},
			},
		}
	case robottypes.PhaseDelivery:
		exec.Delivery = &robottypes.DeliveryResult{
			Type:    robottypes.DeliveryNotify,
			Success: true,
		}
	case robottypes.PhaseLearning:
		exec.Learning = []robottypes.LearningEntry{
			{
				Type:    robottypes.LearnExecution,
				Content: "Dry-run execution completed successfully",
			},
		}
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

// Verify Executor implements types.Executor
var _ types.Executor = (*Executor)(nil)
