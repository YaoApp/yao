package pubsub

import (
	"time"

	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/trace/types"
)

// Subscriber represents a subscription to trace updates
type Subscriber struct {
	ID      string
	Channel <-chan *types.TraceUpdate
	pubsub  *PubSub
}

// Unsubscribe removes the subscription and closes the channel
func (s *Subscriber) Unsubscribe() {
	s.pubsub.Unsubscribe(s.ID)
}

// SubscribeWithHistory creates a subscription and replays historical updates first
// historicalUpdates: updates to replay before starting live stream
// bufferSize: size of the subscription channel buffer
func (ps *PubSub) SubscribeWithHistory(historicalUpdates []*types.TraceUpdate, bufferSize int) *Subscriber {
	// Create subscription
	ch, subID := ps.Subscribe(bufferSize)

	// Create subscriber
	sub := &Subscriber{
		ID:      subID,
		Channel: ch,
		pubsub:  ps,
	}

	// Replay historical updates in a goroutine
	// This allows the subscription to start immediately
	go func() {
		// Get writable channel for replay
		ps.mu.RLock()
		writeCh, exists := ps.subscribers[subID]
		ps.mu.RUnlock()

		if !exists {
			return
		}

		// Replay all historical updates (blocking send to ensure delivery)
		for i, update := range historicalUpdates {
			select {
			case writeCh <- update:
				// Sent successfully
			case <-time.After(5 * time.Second):
				// Timeout - subscriber is too slow or disconnected
				log.Trace("[PUBSUB] Subscriber %s timed out during replay at update %d/%d", subID, i, len(historicalUpdates))
				return
			}
		}
	}()

	return sub
}
