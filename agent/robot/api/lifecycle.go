package api

import (
	"context"
	"fmt"
	"sync"

	robotevents "github.com/yaoapp/yao/agent/robot/events"
	"github.com/yaoapp/yao/agent/robot/events/integrations"
	dtadapter "github.com/yaoapp/yao/agent/robot/events/integrations/dingtalk"
	dcadapter "github.com/yaoapp/yao/agent/robot/events/integrations/discord"
	fsadapter "github.com/yaoapp/yao/agent/robot/events/integrations/feishu"
	"github.com/yaoapp/yao/agent/robot/events/integrations/telegram"
	weixinadapter "github.com/yaoapp/yao/agent/robot/events/integrations/weixin"
	"github.com/yaoapp/yao/agent/robot/logger"
	"github.com/yaoapp/yao/agent/robot/manager"
	"github.com/yaoapp/yao/agent/robot/types"
)

var log = logger.New("robot")

func init() {
	robotevents.RegisterTriggerFunc(func(ctx *types.Context, memberID string, triggerType types.TriggerType, data interface{}) (string, bool, error) {
		result, err := TriggerManual(ctx, memberID, triggerType, data)
		if err != nil {
			return "", false, err
		}
		return result.ExecutionID, result.Accepted, nil
	})
}

// ==================== Lifecycle API ====================
// These functions manage the robot agent system lifecycle

var (
	globalManager    *manager.Manager
	globalDispatcher *integrations.Dispatcher
	managerMu        sync.RWMutex
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

	if err := globalManager.Start(); err != nil {
		return err
	}

	// Start integration dispatcher (Telegram polling, webhook subscriptions, etc.)
	adapters := map[string]integrations.Adapter{
		"telegram": telegram.NewAdapter(),
		"feishu":   fsadapter.NewAdapter(),
		"dingtalk": dtadapter.NewAdapter(),
		"discord":  dcadapter.NewAdapter(),
		"weixin":   weixinadapter.NewAdapter(),
	}
	globalDispatcher = integrations.NewDispatcher(globalManager.Cache(), adapters)
	if err := globalDispatcher.Start(context.Background()); err != nil {
		log.Error("failed to start integration dispatcher: %v", err)
	}

	return nil
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

	if globalDispatcher != nil {
		globalDispatcher.Stop()
		globalDispatcher = nil
	}

	err := globalManager.Stop()
	if err != nil {
		return err
	}

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

// GetManager returns the global manager instance, or nil if not started.
func GetManager() *manager.Manager {
	managerMu.RLock()
	defer managerMu.RUnlock()
	if globalManager == nil || !globalManager.IsStarted() {
		return nil
	}
	return globalManager
}

// SetManager sets the global manager instance (for testing)
func SetManager(m *manager.Manager) {
	managerMu.Lock()
	defer managerMu.Unlock()
	globalManager = m
}
