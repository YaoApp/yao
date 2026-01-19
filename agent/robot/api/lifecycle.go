package api

import (
	"fmt"
	"sync"

	"github.com/yaoapp/yao/agent/robot/manager"
)

// ==================== Lifecycle API ====================
// These functions manage the robot agent system lifecycle

var (
	globalManager *manager.Manager
	managerMu     sync.RWMutex
)

// Start starts the robot agent system
// This initializes and starts the manager which handles:
// - Robot cache loading
// - Worker pool
// - Clock ticker for scheduled triggers
func Start() error {
	managerMu.Lock()
	defer managerMu.Unlock()

	if globalManager != nil && globalManager.IsStarted() {
		return fmt.Errorf("robot agent system already started")
	}

	// Create new manager if not exists
	if globalManager == nil {
		globalManager = manager.New()
	}

	return globalManager.Start()
}

// StartWithConfig starts the robot agent system with custom configuration
func StartWithConfig(config *manager.Config) error {
	managerMu.Lock()
	defer managerMu.Unlock()

	if globalManager != nil && globalManager.IsStarted() {
		return fmt.Errorf("robot agent system already started")
	}

	globalManager = manager.NewWithConfig(config)
	return globalManager.Start()
}

// Stop stops the robot agent system gracefully
// This will:
// - Stop the clock ticker
// - Stop cache auto-refresh
// - Wait for running jobs to complete
// - Stop the worker pool
func Stop() error {
	managerMu.Lock()
	defer managerMu.Unlock()

	if globalManager == nil {
		return nil
	}

	err := globalManager.Stop()
	if err != nil {
		return err
	}

	// Reset global manager
	globalManager = nil
	return nil
}

// IsRunning returns true if the robot agent system is running
func IsRunning() bool {
	managerMu.RLock()
	defer managerMu.RUnlock()

	return globalManager != nil && globalManager.IsStarted()
}

// getManager returns the global manager instance
// Returns error if manager is not started
func getManager() (*manager.Manager, error) {
	managerMu.RLock()
	defer managerMu.RUnlock()

	if globalManager == nil || !globalManager.IsStarted() {
		return nil, fmt.Errorf("robot agent system not started")
	}
	return globalManager, nil
}

// SetManager sets the global manager instance (for testing)
func SetManager(m *manager.Manager) {
	managerMu.Lock()
	defer managerMu.Unlock()
	globalManager = m
}
