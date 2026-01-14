package plan

import (
	"time"

	"github.com/yaoapp/yao/agent/robot/types"
)

// Plan manages planned tasks/goals for later execution
// This is a stub implementation for Phase 2
type Plan struct{}

// New creates a new plan instance
func New() *Plan {
	return &Plan{}
}

// Add adds a task or goal to plan queue
// Stub: returns nil (will be implemented in Phase 11)
func (p *Plan) Add(ctx *types.Context, memberID string, item interface{}, executeAt time.Time) error {
	return nil
}

// Remove removes an item from plan queue
// Stub: returns nil (will be implemented in Phase 11)
func (p *Plan) Remove(ctx *types.Context, memberID string, itemID string) error {
	return nil
}

// List lists all planned items for a robot
// Stub: returns empty slice (will be implemented in Phase 11)
func (p *Plan) List(ctx *types.Context, memberID string) ([]interface{}, error) {
	return []interface{}{}, nil
}

// GetDue returns items that are due for execution
// Stub: returns empty slice (will be implemented in Phase 11)
func (p *Plan) GetDue(ctx *types.Context, now time.Time) ([]interface{}, error) {
	return []interface{}{}, nil
}
