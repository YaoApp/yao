package trace

import (
	"github.com/yaoapp/yao/trace/types"
)

// Subscription Operations

// addUpdate adds an update to history and broadcasts to subscribers
func (m *manager) addUpdate(update *types.TraceUpdate) {
	// Add to history
	m.updatesMu.Lock()
	m.updates = append(m.updates, update)
	m.updatesMu.Unlock()

	// Only broadcast if there are subscribers
	m.subMu.RLock()
	hasSubscribers := len(m.subscribers) > 0
	m.subMu.RUnlock()

	if hasSubscribers {
		// Broadcast to real-time subscribers (non-blocking, in goroutine)
		go m.broadcast(update)
	}
}

// broadcast sends update to all active subscribers (non-blocking)
func (m *manager) broadcast(update *types.TraceUpdate) {
	m.subMu.RLock()
	defer m.subMu.RUnlock()

	for _, ch := range m.subscribers {
		// Use recover to handle closed channels safely
		func() {
			defer func() {
				if r := recover(); r != nil {
					// Channel was closed, ignore (subscriber cleanup race condition)
				}
			}()

			select {
			case ch <- update:
				// Sent successfully
			default:
				// Channel full, skip (or could log warning)
			}
		}()
	}
}

// Subscribe subscribes to all trace updates (replay history + real-time)
func (m *manager) Subscribe() (<-chan *types.TraceUpdate, error) {
	return m.SubscribeFrom(0)
}

// SubscribeFrom subscribes from a specific timestamp (for resume)
func (m *manager) SubscribeFrom(since int64) (<-chan *types.TraceUpdate, error) {
	// Create subscriber channel with buffer
	ch := make(chan *types.TraceUpdate, 100)
	subID := genNodeID()

	// Register subscriber
	m.subMu.Lock()
	m.subscribers[subID] = ch
	m.subMu.Unlock()

	// Start replay and streaming goroutine
	go m.replayAndStream(ch, subID, since)

	return ch, nil
}

// replayAndStream replays history then streams real-time updates
func (m *manager) replayAndStream(ch chan *types.TraceUpdate, subID string, since int64) {
	defer func() {
		// Close channel and cleanup subscriber
		close(ch)
		m.subMu.Lock()
		delete(m.subscribers, subID)
		m.subMu.Unlock()
	}()

	// Step 1: Replay history
	m.updatesMu.RLock()
	history := make([]*types.TraceUpdate, 0)
	for _, update := range m.updates {
		if update.Timestamp >= since {
			history = append(history, update)
		}
	}
	isCompleted := m.completed
	m.updatesMu.RUnlock()

	// Send history in order
	for _, update := range history {
		select {
		case ch <- update:
			// Sent successfully
			// Optional: add small delay to control replay speed
			// time.Sleep(10 * time.Millisecond)
		case <-m.ctx.Done():
			// Context cancelled, stop
			return
		}
	}

	// Step 2: If already completed, exit
	if isCompleted {
		return
	}

	// Step 3: Wait for completion or context cancellation
	// Real-time updates are sent by broadcast() method
	<-m.ctx.Done()
}

// IsComplete checks if the trace is completed
func (m *manager) IsComplete() bool {
	m.updatesMu.RLock()
	defer m.updatesMu.RUnlock()
	return m.completed
}
