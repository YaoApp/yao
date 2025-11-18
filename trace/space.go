package trace

import (
	"context"

	"github.com/yaoapp/yao/trace/types"
)

// space implements the Space interface for custom space operations
type space struct {
	ctx     context.Context
	traceID string
	data    *types.TraceSpace
	driver  types.Driver
}

// NewSpace creates a new space instance
func NewSpace(ctx context.Context, traceID string, data *types.TraceSpace, driver types.Driver) types.Space {
	return &space{
		ctx:     ctx,
		traceID: traceID,
		data:    data,
		driver:  driver,
	}
}

// ID returns the space identifier
func (s *space) ID() string {
	return s.data.ID
}

// Set stores a value by key
func (s *space) Set(key string, value any) error {
	return s.driver.SetSpaceKey(s.ctx, s.traceID, s.data.ID, key, value)
}

// Get retrieves a value by key
func (s *space) Get(key string) (any, error) {
	return s.driver.GetSpaceKey(s.ctx, s.traceID, s.data.ID, key)
}

// Has checks if a key exists
func (s *space) Has(key string) bool {
	return s.driver.HasSpaceKey(s.ctx, s.traceID, s.data.ID, key)
}

// Delete removes a key-value pair
func (s *space) Delete(key string) error {
	return s.driver.DeleteSpaceKey(s.ctx, s.traceID, s.data.ID, key)
}

// Clear removes all key-value pairs
func (s *space) Clear() error {
	return s.driver.ClearSpaceKeys(s.ctx, s.traceID, s.data.ID)
}

// Keys returns all keys in the space
func (s *space) Keys() []string {
	keys, err := s.driver.ListSpaceKeys(s.ctx, s.traceID, s.data.ID)
	if err != nil {
		return nil
	}
	return keys
}
