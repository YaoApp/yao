package registry

import (
	"log/slog"
)

// NewForTest creates a standalone Registry for use in tests.
// Not intended for production use.
func NewForTest() *Registry {
	return &Registry{
		nodes:  make(map[string]*TaiNode),
		logger: slog.Default(),
	}
}

// SetGlobalForTest replaces the global registry singleton for testing.
// Not intended for production use.
func SetGlobalForTest(r *Registry) {
	global = r
}

// SetNodeModeForTest updates the mode of a registered node for testing.
// Not intended for production use.
func SetNodeModeForTest(taiID, mode string) bool {
	if global == nil {
		return false
	}
	global.mu.Lock()
	defer global.mu.Unlock()
	n, ok := global.nodes[taiID]
	if !ok {
		return false
	}
	n.Mode = mode
	return true
}
