package store

import "github.com/yaoapp/yao/agent/robot/types"

// Store implements types.Store interface
// This is a stub implementation for Phase 2
type Store struct{}

// New creates a new store instance
func New() *Store {
	return &Store{}
}

// SaveLearning saves learning entries to private KB
// Stub: returns nil (will be implemented in Phase 9)
func (s *Store) SaveLearning(ctx *types.Context, memberID string, entries []types.LearningEntry) error {
	return nil
}

// GetHistory retrieves learning history from private KB
// Stub: returns empty slice (will be implemented in Phase 9)
func (s *Store) GetHistory(ctx *types.Context, memberID string, limit int) ([]types.LearningEntry, error) {
	return []types.LearningEntry{}, nil
}

// SearchKB searches knowledge base collections
// Stub: returns empty slice (will be implemented in Phase 4+)
func (s *Store) SearchKB(ctx *types.Context, collections []string, query string) ([]interface{}, error) {
	return []interface{}{}, nil
}

// QueryDB queries database models
// Stub: returns empty slice (will be implemented in Phase 4+)
func (s *Store) QueryDB(ctx *types.Context, models []string, query interface{}) ([]interface{}, error) {
	return []interface{}{}, nil
}
