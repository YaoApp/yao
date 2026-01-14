package manager

import (
	"time"

	"github.com/yaoapp/yao/agent/robot/types"
)

// Manager implements types.Manager interface
// This is a stub implementation for Phase 2
type Manager struct{}

// New creates a new manager instance
func New() *Manager {
	return &Manager{}
}

// Start starts the manager and clock ticker
// Stub: returns nil (will be implemented in Phase 3)
func (m *Manager) Start() error {
	return nil
}

// Stop stops the manager gracefully
// Stub: returns nil (will be implemented in Phase 3)
func (m *Manager) Stop() error {
	return nil
}

// Tick processes a clock tick
// Stub: returns nil (will be implemented in Phase 3)
func (m *Manager) Tick(ctx *types.Context, now time.Time) error {
	return nil
}
