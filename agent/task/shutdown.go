package task

import (
	"context"
	"sync"
)

var (
	globalShutdown       context.Context
	globalShutdownCancel context.CancelFunc
	daemonWg             sync.WaitGroup
)

func init() {
	globalShutdown, globalShutdownCancel = context.WithCancel(context.Background())
}

// Shutdown gracefully stops all task execution:
// 1. Stop schedule engine (no new triggers)
// 2. Cancel all DaemonContexts via global context
// 3. Wait for all daemon goroutines to exit
func Shutdown() {
	GlobalScheduleEngine.Stop()
	globalShutdownCancel()
	daemonWg.Wait()
}
