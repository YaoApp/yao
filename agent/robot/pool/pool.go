package pool

import "github.com/yaoapp/yao/agent/robot/types"

// Pool implements types.Pool interface
// This is a stub implementation for Phase 2
type Pool struct {
	size int
}

// New creates a new pool instance
func New(size int) *Pool {
	return &Pool{
		size: size,
	}
}

// Start starts the worker pool
// Stub: returns nil (will be implemented in Phase 3)
func (p *Pool) Start() error {
	return nil
}

// Stop stops the worker pool gracefully
// Stub: returns nil (will be implemented in Phase 3)
func (p *Pool) Stop() error {
	return nil
}

// Submit submits a robot execution to the pool
// Stub: returns empty job ID (will be implemented in Phase 3)
func (p *Pool) Submit(ctx *types.Context, robot *types.Robot, trigger types.TriggerType, data interface{}) (string, error) {
	return "", nil
}

// Running returns number of currently running jobs
// Stub: returns 0 (will be implemented in Phase 3)
func (p *Pool) Running() int {
	return 0
}

// Queued returns number of queued jobs
// Stub: returns 0 (will be implemented in Phase 3)
func (p *Pool) Queued() int {
	return 0
}
