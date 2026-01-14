package trigger

import "github.com/yaoapp/yao/agent/robot/types"

// Trigger handles all trigger sources
// This is a stub implementation for Phase 2
type Trigger struct{}

// New creates a new trigger instance
func New() *Trigger {
	return &Trigger{}
}

// Clock processes clock trigger
// Stub: returns nil (will be implemented in Phase 3)
func (t *Trigger) Clock(ctx *types.Context, robot *types.Robot) error {
	return nil
}

// Intervene processes human intervention
// Stub: returns empty result (will be implemented in Phase 3)
func (t *Trigger) Intervene(ctx *types.Context, req *types.InterveneRequest) (*types.ExecutionResult, error) {
	return &types.ExecutionResult{}, nil
}

// Event processes event trigger
// Stub: returns empty result (will be implemented in Phase 3)
func (t *Trigger) Event(ctx *types.Context, req *types.EventRequest) (*types.ExecutionResult, error) {
	return &types.ExecutionResult{}, nil
}

// Pause pauses a running execution
// Stub: returns nil (will be implemented in Phase 3)
func (t *Trigger) Pause(ctx *types.Context, execID string) error {
	return nil
}

// Resume resumes a paused execution
// Stub: returns nil (will be implemented in Phase 3)
func (t *Trigger) Resume(ctx *types.Context, execID string) error {
	return nil
}

// Stop stops a running execution
// Stub: returns nil (will be implemented in Phase 3)
func (t *Trigger) Stop(ctx *types.Context, execID string) error {
	return nil
}
