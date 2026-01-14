package dedup

import (
	"sync"
	"time"

	"github.com/yaoapp/yao/agent/robot/types"
)

// Dedup implements types.Dedup interface
// This is a stub implementation for Phase 2
type Dedup struct {
	marks map[string]time.Time // key -> expiry time
	mu    sync.RWMutex
}

// New creates a new dedup instance
func New() *Dedup {
	return &Dedup{
		marks: make(map[string]time.Time),
	}
}

// Check checks if execution should be deduplicated
// Stub: always returns proceed (will be implemented in Phase 3)
func (d *Dedup) Check(ctx *types.Context, memberID string, trigger types.TriggerType) (types.DedupResult, error) {
	return types.DedupProceed, nil
}

// Mark marks an execution to prevent duplicates within window
// Stub: does nothing (will be implemented in Phase 3)
func (d *Dedup) Mark(memberID string, trigger types.TriggerType, window time.Duration) {
	// Stub: no-op
}
