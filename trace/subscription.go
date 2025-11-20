package trace

import (
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
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
	// Generate unique subscriber ID
	subID, _ := gonanoid.Generate("0123456789abcdefghijklmnopqrstuvwxyz", 12)

	// Create update channel
	updateCh := make(chan *types.TraceUpdate, 100)

	// Register subscriber
	m.stateAddSubscriber(subID, updateCh)

	// Start replay and stream goroutine (will auto-cleanup on completion)
	go m.replayAndStream(subID, updateCh, since)

	return updateCh, nil
}

// replayAndStream replays historical updates and streams new ones
func (m *manager) replayAndStream(subID string, ch chan *types.TraceUpdate, since int64) {
	// Auto-cleanup on exit - MUST remove from map before closing channel
	defer func() {
		// Remove from subscribers map first to prevent new broadcasts
		m.stateRemoveSubscriber(subID)
		// Close channel (any in-flight broadcasts will be caught by recover)
		close(ch)
	}()

	// Get historical updates
	updates := m.stateGetUpdates(since)

	// Replay historical updates and check if trace was already completed
	traceWasCompleted := false
	for _, update := range updates {
		select {
		case ch <- update:
			// Check if this is a trace complete event
			if update.Type == types.UpdateTypeComplete {
				traceWasCompleted = true
			}
		case <-m.ctx.Done():
			return
		}
	}

	// If trace was already completed in historical events, exit immediately
	if traceWasCompleted {
		return
	}

	// Continue streaming new updates
	// The channel will receive updates via broadcast from addUpdate
	// Monitor completion to know when to exit
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if m.stateIsCompleted() {
				return
			}
		case <-m.ctx.Done():
			return
		}
	}
}
