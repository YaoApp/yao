package registry

import (
	"log/slog"
	"net"
	"time"
)

// NewForTest creates a standalone Registry for use in tests.
// Not intended for production use.
func NewForTest() *Registry {
	return &Registry{
		nodes:   make(map[string]*TaiNode),
		pending: make(map[string]*pendingChannel),
		logger:  slog.Default(),
	}
}

// SetGlobalForTest replaces the global registry singleton for testing.
// Not intended for production use.
func SetGlobalForTest(r *Registry) {
	global = r
}

// SetPendingForTest injects a pending channel entry for testing.
// Not intended for production use.
func (r *Registry) SetPendingForTest(channelID, taiID string, result chan net.Conn, timer *time.Timer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pending[channelID] = &pendingChannel{taiID: taiID, result: result, timer: timer}
}
