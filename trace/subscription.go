package trace

import (
	"fmt"

	"github.com/yaoapp/yao/trace/types"
)

// Subscribe creates a new subscription for trace updates (replays all historical events from the beginning)
func (m *manager) Subscribe() (<-chan *types.TraceUpdate, error) {
	return m.subscribe(0) // Subscribe from beginning to get all historical events
}

// SubscribeFrom creates a subscription starting from a specific timestamp
func (m *manager) SubscribeFrom(since int64) (<-chan *types.TraceUpdate, error) {
	return m.subscribe(since)
}

// subscribe is the internal implementation for subscriptions
func (m *manager) subscribe(since int64) (<-chan *types.TraceUpdate, error) {
	// Get historical updates
	updates := m.stateGetUpdates(since)

	// Use manager's pubsub reference (always available)
	if m.pubsub == nil {
		return nil, fmt.Errorf("pubsub service not initialized for trace: %s", m.traceID)
	}

	// Create subscription with historical replay
	// Buffer size should be large enough to hold historical updates plus some live updates
	// Using max of 1000 or len(updates)+100 to handle large traces
	bufferSize := 1000
	if len(updates)+100 > bufferSize {
		bufferSize = len(updates) + 100
	}
	sub := m.pubsub.SubscribeWithHistory(updates, bufferSize)

	return sub.Channel, nil
}
