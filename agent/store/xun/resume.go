package xun

import (
	"github.com/yaoapp/yao/agent/store/types"
)

// =============================================================================
// Resume Management (only called on failure/interrupt)
// =============================================================================

// SaveResume batch saves resume records
// Only called when request is interrupted or failed
func (store *Xun) SaveResume(records []*types.Resume) error {
	// TODO: implement
	return nil
}

// GetResume retrieves all resume records for a chat
func (store *Xun) GetResume(chatID string) ([]*types.Resume, error) {
	// TODO: implement
	return nil, nil
}

// GetLastResume retrieves the last (most recent) resume record for a chat
func (store *Xun) GetLastResume(chatID string) (*types.Resume, error) {
	// TODO: implement
	return nil, nil
}

// GetResumeByStackID retrieves resume records for a specific stack
func (store *Xun) GetResumeByStackID(stackID string) ([]*types.Resume, error) {
	// TODO: implement
	return nil, nil
}

// GetStackPath returns the stack path from root to the given stack
// Returns: [root_stack_id, ..., current_stack_id]
func (store *Xun) GetStackPath(stackID string) ([]string, error) {
	// TODO: implement
	return nil, nil
}

// DeleteResume deletes all resume records for a chat
// Called after successful resume to clean up
func (store *Xun) DeleteResume(chatID string) error {
	// TODO: implement
	return nil
}
