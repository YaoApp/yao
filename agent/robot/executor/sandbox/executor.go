package sandbox

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/yaoapp/yao/agent/robot/executor/types"
	robottypes "github.com/yaoapp/yao/agent/robot/types"
)

// Executor implements a sandboxed executor placeholder.
//
// ⚠️ NOT IMPLEMENTED: True sandbox mode requires container-level isolation
// (Docker/gVisor/Firecracker) for security. This placeholder currently
// behaves like DryRun mode and does NOT provide real security isolation.
//
// Future Implementation:
// - Container isolation: Each execution in separate container
// - Resource limits: CPU, memory, disk enforced by container runtime
// - Network isolation: Restricted network via container networking
// - File system isolation: Read-only root, limited writable paths
// - Process isolation: Separate PID namespace
//
// Current behavior: Simulates execution with mock data (same as DryRun)
type Executor struct {
	config       types.SandboxConfig
	execCount    atomic.Int32
	currentCount atomic.Int32
}

// New creates a new sandbox executor with default settings
func New() *Executor {
	return &Executor{
		config: types.SandboxConfig{
			MaxDuration:   30 * time.Minute,
			NetworkAccess: true,
			FileAccess:    false,
		},
	}
}

// NewWithConfig creates a sandbox executor with custom configuration
func NewWithConfig(config types.SandboxConfig) *Executor {
	return &Executor{
		config: config,
	}
}

// Execute runs robot execution within sandbox constraints (auto-generates ID)
func (e *Executor) Execute(ctx *robottypes.Context, robot *robottypes.Robot, trigger robottypes.TriggerType, data interface{}) (*robottypes.Execution, error) {
	return e.ExecuteWithControl(ctx, robot, trigger, data, "", nil)
}

// ExecuteWithID runs robot execution within sandbox constraints with a pre-generated execution ID (no control)
func (e *Executor) ExecuteWithID(ctx *robottypes.Context, robot *robottypes.Robot, trigger robottypes.TriggerType, data interface{}, execID string) (*robottypes.Execution, error) {
	return e.ExecuteWithControl(ctx, robot, trigger, data, execID, nil)
}

// ExecuteWithControl runs robot execution within sandbox constraints with execution control
func (e *Executor) ExecuteWithControl(ctx *robottypes.Context, robot *robottypes.Robot, trigger robottypes.TriggerType, data interface{}, execID string, control robottypes.ExecutionControl) (*robottypes.Execution, error) {
	if robot == nil {
		return nil, fmt.Errorf("robot cannot be nil")
	}

	// Create timeout context
	execCtx, cancel := context.WithTimeout(ctx.Context, e.config.MaxDuration)
	defer cancel()

	// Create new context with timeout
	sandboxCtx := robottypes.NewContext(execCtx, ctx.Auth)

	// Determine starting phase
	startPhaseIndex := 0
	if trigger == robottypes.TriggerHuman || trigger == robottypes.TriggerEvent {
		startPhaseIndex = 1
	}

	// Use provided execID or generate new one
	if execID == "" {
		execID = fmt.Sprintf("sandbox_%d", time.Now().UnixNano())
	}

	// Create execution record
	exec := &robottypes.Execution{
		ID:          execID,
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

	// Update status
	exec.Status = robottypes.ExecRunning

	// Execute phases with sandbox constraints
	phases := robottypes.AllPhases[startPhaseIndex:]
	for _, phase := range phases {
		// Check timeout or cancellation
		select {
		case <-execCtx.Done():
			exec.Status = robottypes.ExecFailed
			exec.Error = "execution timeout exceeded"
			return exec, nil
		default:
		}

		// Wait if paused
		if control != nil {
			if err := control.WaitIfPaused(); err != nil {
				exec.Status = robottypes.ExecCancelled
				exec.Error = "execution cancelled while paused"
				return exec, nil
			}
		}

		exec.Phase = phase

		if e.config.OnPhaseStart != nil {
			e.config.OnPhaseStart(phase)
		}

		// Execute phase with sandbox constraints
		if err := e.runSandboxedPhase(sandboxCtx, exec, phase, data); err != nil {
			exec.Status = robottypes.ExecFailed
			exec.Error = err.Error()
			return exec, nil
		}

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

// runSandboxedPhase executes a phase with sandbox constraints
func (e *Executor) runSandboxedPhase(ctx *robottypes.Context, exec *robottypes.Execution, phase robottypes.Phase, data interface{}) error {
	// Validate agent is allowed (if whitelist is set)
	if len(e.config.AllowedAgents) > 0 {
		robot := exec.GetRobot()
		if robot != nil && robot.Config != nil && robot.Config.Resources != nil {
			agentID := robot.Config.Resources.GetPhaseAgent(phase)
			if !e.isAgentAllowed(agentID) {
				return fmt.Errorf("agent %s is not allowed in sandbox", agentID)
			}
		}
	}

	// For now, generate mock output (real implementation would call agents with restrictions)
	e.mockPhaseOutput(exec, phase)

	return nil
}

// isAgentAllowed checks if an agent is in the whitelist
func (e *Executor) isAgentAllowed(agentID string) bool {
	for _, allowed := range e.config.AllowedAgents {
		if allowed == agentID || allowed == "*" {
			return true
		}
	}
	return false
}

// mockPhaseOutput generates mock output for each phase
func (e *Executor) mockPhaseOutput(exec *robottypes.Execution, phase robottypes.Phase) {
	switch phase {
	case robottypes.PhaseInspiration:
		exec.Inspiration = &robottypes.InspirationReport{
			Clock:   robottypes.NewClockContext(time.Now(), ""),
			Content: "## Sandbox Inspiration\n\nExecuted in isolated sandbox environment.",
		}
	case robottypes.PhaseGoals:
		exec.Goals = &robottypes.Goals{
			Content: "## Sandbox Goals\n\n1. [High] Sandboxed goal execution",
		}
	case robottypes.PhaseTasks:
		exec.Tasks = []robottypes.Task{
			{
				ID:           "sandbox-task-1",
				GoalRef:      "Goal 1",
				Source:       robottypes.TaskSourceAuto,
				ExecutorType: robottypes.ExecutorAssistant,
				ExecutorID:   "sandbox-agent",
				Status:       robottypes.TaskPending,
			},
		}
	case robottypes.PhaseRun:
		exec.Results = []robottypes.TaskResult{
			{
				TaskID:   "sandbox-task-1",
				Success:  true,
				Output:   map[string]interface{}{"mode": "sandbox", "isolated": true},
				Duration: 50,
				Validation: &robottypes.ValidationResult{
					Passed: true,
					Score:  1.0,
				},
			},
		}
	case robottypes.PhaseDelivery:
		exec.Delivery = &robottypes.DeliveryResult{
			RequestID: "sandbox-" + exec.ID,
			Content: &robottypes.DeliveryContent{
				Summary: "Sandbox delivery completed",
				Body:    "# Sandbox Delivery\n\nThis is a simulated sandbox delivery result.",
			},
			Success: true,
		}
	case robottypes.PhaseLearning:
		exec.Learning = []robottypes.LearningEntry{
			{
				Type:    robottypes.LearnExecution,
				Content: "Sandbox execution completed within constraints",
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
