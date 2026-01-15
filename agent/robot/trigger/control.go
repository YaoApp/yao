package trigger

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/yaoapp/yao/agent/robot/types"
)

// ExecutionController manages execution lifecycle (pause/resume/stop)
type ExecutionController struct {
	executions map[string]*ControlledExecution
	mu         sync.RWMutex
}

// ControlledExecution represents an execution that can be controlled
type ControlledExecution struct {
	ID        string
	MemberID  string
	TeamID    string
	Status    types.ExecStatus
	Phase     types.Phase
	StartTime time.Time
	PausedAt  *time.Time

	// Control channels
	ctx     context.Context
	cancel  context.CancelFunc
	paused  bool
	pauseMu sync.Mutex
	pauseCh chan struct{} // closed when paused, recreated on resume
}

// NewExecutionController creates a new execution controller
func NewExecutionController() *ExecutionController {
	return &ExecutionController{
		executions: make(map[string]*ControlledExecution),
	}
}

// Track starts tracking an execution
func (c *ExecutionController) Track(execID, memberID, teamID string) *ControlledExecution {
	c.mu.Lock()
	defer c.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	exec := &ControlledExecution{
		ID:        execID,
		MemberID:  memberID,
		TeamID:    teamID,
		Status:    types.ExecRunning,
		Phase:     types.PhaseInspiration,
		StartTime: time.Now(),
		ctx:       ctx,
		cancel:    cancel,
		paused:    false,
		pauseCh:   make(chan struct{}),
	}

	c.executions[execID] = exec
	return exec
}

// Untrack stops tracking an execution
func (c *ExecutionController) Untrack(execID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.executions, execID)
}

// Get returns a tracked execution
func (c *ExecutionController) Get(execID string) *ControlledExecution {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.executions[execID]
}

// List returns all tracked executions
func (c *ExecutionController) List() []*ControlledExecution {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]*ControlledExecution, 0, len(c.executions))
	for _, exec := range c.executions {
		result = append(result, exec)
	}
	return result
}

// ListByMember returns all executions for a specific member
func (c *ExecutionController) ListByMember(memberID string) []*ControlledExecution {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var result []*ControlledExecution
	for _, exec := range c.executions {
		if exec.MemberID == memberID {
			result = append(result, exec)
		}
	}
	return result
}

// Pause pauses an execution
func (c *ExecutionController) Pause(execID string) error {
	exec := c.Get(execID)
	if exec == nil {
		return fmt.Errorf("execution not found: %s", execID)
	}

	exec.pauseMu.Lock()
	defer exec.pauseMu.Unlock()

	if exec.paused {
		return fmt.Errorf("execution already paused: %s", execID)
	}

	exec.paused = true
	now := time.Now()
	exec.PausedAt = &now

	// Close the pause channel to signal pause
	close(exec.pauseCh)

	return nil
}

// Resume resumes a paused execution
func (c *ExecutionController) Resume(execID string) error {
	exec := c.Get(execID)
	if exec == nil {
		return fmt.Errorf("execution not found: %s", execID)
	}

	exec.pauseMu.Lock()
	defer exec.pauseMu.Unlock()

	if !exec.paused {
		return fmt.Errorf("execution not paused: %s", execID)
	}

	exec.paused = false
	exec.PausedAt = nil

	// Create new pause channel for future pauses
	exec.pauseCh = make(chan struct{})

	return nil
}

// Stop stops an execution
func (c *ExecutionController) Stop(execID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	exec, ok := c.executions[execID]
	if !ok {
		return fmt.Errorf("execution not found: %s", execID)
	}

	// Cancel the context to signal stop
	if exec.cancel != nil {
		exec.cancel()
	}

	exec.Status = types.ExecCancelled

	// Remove from tracking
	delete(c.executions, execID)

	return nil
}

// ==================== ControlledExecution methods ====================

// IsPaused returns true if the execution is paused
func (e *ControlledExecution) IsPaused() bool {
	e.pauseMu.Lock()
	defer e.pauseMu.Unlock()
	return e.paused
}

// IsCancelled returns true if the execution is cancelled
func (e *ControlledExecution) IsCancelled() bool {
	select {
	case <-e.ctx.Done():
		return true
	default:
		return false
	}
}

// Context returns the execution's context
func (e *ControlledExecution) Context() context.Context {
	return e.ctx
}

// WaitIfPaused blocks until the execution is resumed or cancelled
// Returns error if cancelled
func (e *ControlledExecution) WaitIfPaused() error {
	e.pauseMu.Lock()
	paused := e.paused
	pauseCh := e.pauseCh
	e.pauseMu.Unlock()

	if !paused {
		return nil
	}

	// Wait for resume (new pauseCh created) or cancel
	select {
	case <-e.ctx.Done():
		return types.ErrExecutionCancelled
	case <-pauseCh:
		// Pause channel closed, check if we're still paused
		// If still paused, this was the pause signal; wait for resume
		for {
			e.pauseMu.Lock()
			if !e.paused {
				e.pauseMu.Unlock()
				return nil
			}
			newPauseCh := e.pauseCh
			e.pauseMu.Unlock()

			select {
			case <-e.ctx.Done():
				return types.ErrExecutionCancelled
			case <-newPauseCh:
				// Channel closed again, loop to check state
			}
		}
	}
}

// CheckCancelled checks if the execution is cancelled and returns error if so
func (e *ControlledExecution) CheckCancelled() error {
	if e.IsCancelled() {
		return types.ErrExecutionCancelled
	}
	return nil
}

// UpdatePhase updates the current phase
func (e *ControlledExecution) UpdatePhase(phase types.Phase) {
	e.Phase = phase
}

// UpdateStatus updates the execution status
func (e *ControlledExecution) UpdateStatus(status types.ExecStatus) {
	e.Status = status
}
