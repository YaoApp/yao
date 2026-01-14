package robot

import (
	"github.com/yaoapp/yao/agent/robot/cache"
	"github.com/yaoapp/yao/agent/robot/dedup"
	"github.com/yaoapp/yao/agent/robot/executor"
	"github.com/yaoapp/yao/agent/robot/manager"
	"github.com/yaoapp/yao/agent/robot/plan"
	"github.com/yaoapp/yao/agent/robot/pool"
	"github.com/yaoapp/yao/agent/robot/store"
	"github.com/yaoapp/yao/agent/robot/trigger"
)

var (
	// Global instances (will be initialized in Init)
	globalManager  *manager.Manager
	globalCache    *cache.Cache
	globalPool     *pool.Pool
	globalDedup    *dedup.Dedup
	globalStore    *store.Store
	globalTrigger  *trigger.Trigger
	globalExecutor *executor.Executor
	globalPlan     *plan.Plan
)

// Init initializes the robot agent system
// Stub: placeholder (will be implemented in Phase 3)
func Init() error {
	// Initialize global instances
	globalCache = cache.New()
	globalDedup = dedup.New()
	globalStore = store.New()
	globalPool = pool.New() // Default pool size
	globalTrigger = trigger.New()
	globalExecutor = executor.New()
	globalManager = manager.New()
	globalPlan = plan.New()

	// TODO Phase 3: Start manager and pool
	// return globalManager.Start()

	return nil
}

// Shutdown gracefully shuts down the robot agent system
// Stub: placeholder (will be implemented in Phase 3)
func Shutdown() error {
	// TODO Phase 3: Stop manager and pool
	// if globalManager != nil {
	//     return globalManager.Stop()
	// }
	return nil
}
