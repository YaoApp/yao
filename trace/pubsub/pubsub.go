package pubsub

import (
	"sync"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/trace/types"
)

// PubSub is an independent publish-subscribe service for trace updates
// It acts as a message broker between trace writers and readers
type PubSub struct {
	eventBus    chan *types.TraceUpdate            // Event bus for incoming events
	subscribers map[string]chan *types.TraceUpdate // Active subscribers
	mu          sync.RWMutex                       // Protects subscribers map
	stopCh      chan struct{}                      // Signal to stop the service
	stopped     bool                               // Whether service is stopped
}

// New creates a new PubSub service
func New() *PubSub {
	ps := &PubSub{
		eventBus:    make(chan *types.TraceUpdate, 1000), // Buffered event bus
		subscribers: make(map[string]chan *types.TraceUpdate),
		stopCh:      make(chan struct{}),
		stopped:     false,
	}

	// Start forwarding service
	go ps.forward()

	return ps
}

// forward continuously forwards events from eventBus to all subscribers
// This runs in a dedicated goroutine
func (ps *PubSub) forward() {
	for {
		select {
		case event := <-ps.eventBus:
			ps.mu.RLock()
			subscriberCount := len(ps.subscribers)

			if subscriberCount == 0 {
				// No subscribers, discard event
				ps.mu.RUnlock()
				continue
			}

			// Forward to all subscribers (non-blocking)
			for subID, ch := range ps.subscribers {
				select {
				case ch <- event:
					// Sent successfully
				default:
					// Subscriber is slow or channel full, skip
					log.Trace("[PUBSUB] Subscriber %s is slow, skipping event type=%s", subID, event.Type)
				}
			}
			ps.mu.RUnlock()

		case <-ps.stopCh:
			return
		}
	}
}

// Publish sends an event to the event bus
// This is called by trace writers (e.g., manager.addUpdateAndBroadcast)
func (ps *PubSub) Publish(event *types.TraceUpdate) {
	if ps.stopped {
		return
	}

	select {
	case ps.eventBus <- event:
		// Event published successfully
	default:
		// Event bus full, this shouldn't happen with large buffer (log as warning)
		log.Warn("[PUBSUB] Event bus full, discarding event type=%s", event.Type)
	}
}

// Subscribe creates a new subscription and returns a channel for receiving updates
// The caller is responsible for reading from the channel and closing it when done
func (ps *PubSub) Subscribe(bufferSize int) (<-chan *types.TraceUpdate, string) {
	// Generate unique subscriber ID
	subID, _ := gonanoid.Generate("0123456789abcdefghijklmnopqrstuvwxyz", 12)

	// Create subscriber channel
	ch := make(chan *types.TraceUpdate, bufferSize)

	// Register subscriber
	ps.mu.Lock()
	ps.subscribers[subID] = ch
	ps.mu.Unlock()

	return ch, subID
}

// Unsubscribe removes a subscriber and closes its channel
func (ps *PubSub) Unsubscribe(subID string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ch, exists := ps.subscribers[subID]
	if !exists {
		return
	}

	// Remove from map
	delete(ps.subscribers, subID)

	// Close channel
	close(ch)
}

// SubscriberCount returns the number of active subscribers
func (ps *PubSub) SubscriberCount() int {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return len(ps.subscribers)
}

// Stop stops the forwarding service and closes all subscriber channels
func (ps *PubSub) Stop() {
	if ps.stopped {
		return
	}

	ps.stopped = true
	close(ps.stopCh)

	// Close all subscriber channels
	ps.mu.Lock()
	for _, ch := range ps.subscribers {
		close(ch)
	}
	ps.subscribers = make(map[string]chan *types.TraceUpdate)
	ps.mu.Unlock()
}
