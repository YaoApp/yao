package context

import (
	"context"
	"fmt"
	"time"

	"github.com/yaoapp/kun/log"
)

// NewInterruptController creates a new interrupt controller
func NewInterruptController() *InterruptController {
	ctrl := &InterruptController{
		queue:   make(chan *InterruptSignal, 10), // Buffer for 10 interrupts
		pending: make([]*InterruptSignal, 0),
	}
	ctrl.ctx, ctrl.cancel = context.WithCancel(context.Background())
	return ctrl
}

// Start starts the interrupt listener goroutine
func (ic *InterruptController) Start(contextID string) {
	if ic.listenerStarted {
		return
	}

	ic.mutex.Lock()
	ic.listenerStarted = true
	ic.contextID = contextID
	ic.mutex.Unlock()

	go ic.listen()
}

// SetHandler sets the handler for interrupt signals
func (ic *InterruptController) SetHandler(handler InterruptHandler) {
	if ic == nil {
		return
	}
	ic.handler = handler
}

// listen is the main listener goroutine that processes interrupt signals
func (ic *InterruptController) listen() {
	for {
		select {
		case signal := <-ic.queue:
			// Handle user interrupt signal (stop button, for appending messages)
			ic.handleSignal(signal)

		case <-ic.ctx.Done():
			// Internal context cancelled, stop listening
			return
		}
	}
}

// handleSignal processes an interrupt signal
func (ic *InterruptController) handleSignal(signal *InterruptSignal) {
	if signal == nil {
		return
	}

	log.Trace("[INTERRUPT] Signal received: type=%s, messages=%d, timestamp=%d", signal.Type, len(signal.Messages), signal.Timestamp)

	ic.mutex.Lock()

	// If no current interrupt, set it as current
	if ic.current == nil {
		ic.current = signal
	} else {
		// If there's already a current interrupt, add to pending queue
		ic.pending = append(ic.pending, signal)
	}

	// For force interrupt with no messages (pure cancellation), cancel the interrupt context
	// This allows LLM streaming and other operations to check and stop
	if signal.Type == InterruptForce && len(signal.Messages) == 0 {
		if ic.cancel != nil {
			ic.cancel()
			// Create a new context for potential future operations
			ic.ctx, ic.cancel = context.WithCancel(context.Background())
		}
	}

	ic.mutex.Unlock()

	// Call the registered handler if available (outside lock to avoid deadlock)
	if ic.handler != nil && ic.contextID != "" {
		go func() {
			// Retrieve the parent context from global registry
			ctx, err := Get(ic.contextID)
			if err != nil {
				fmt.Printf("Failed to get context for interrupt handler: %v\n", err)
				return
			}

			// Call the handler
			if err := ic.handler(ctx, signal); err != nil {
				fmt.Printf("Interrupt handler error: %v\n", err)
			}
		}()
	}
}

// Check checks for current interrupt signal (non-blocking)
// Returns the current interrupt and moves to next one if available
func (ic *InterruptController) Check() *InterruptSignal {
	if ic == nil {
		return nil
	}

	ic.mutex.Lock()
	defer ic.mutex.Unlock()

	if ic.current == nil {
		return nil
	}

	// Get current interrupt
	signal := ic.current

	// Move to next interrupt in queue
	if len(ic.pending) > 0 {
		ic.current = ic.pending[0]
		ic.pending = ic.pending[1:]
	} else {
		ic.current = nil
	}

	return signal
}

// CheckWithMerge checks for interrupts and merges all pending messages
// This is useful when multiple interrupts should be handled together
func (ic *InterruptController) CheckWithMerge() *InterruptSignal {
	if ic == nil {
		return nil
	}

	ic.mutex.Lock()
	defer ic.mutex.Unlock()

	if ic.current == nil {
		return nil
	}

	// If there are pending interrupts, merge all messages
	if len(ic.pending) > 0 {
		// Collect all messages
		allMessages := append([]Message{}, ic.current.Messages...)
		for _, pending := range ic.pending {
			allMessages = append(allMessages, pending.Messages...)
		}

		// Create merged signal
		mergedSignal := &InterruptSignal{
			Type:      ic.current.Type, // Use first signal's type
			Messages:  allMessages,
			Timestamp: time.Now().UnixMilli(),
			Metadata: map[string]interface{}{
				"merged":        true,
				"merged_count":  len(ic.pending) + 1,
				"original_time": ic.current.Timestamp,
			},
		}

		// Clear all interrupts
		ic.current = nil
		ic.pending = make([]*InterruptSignal, 0)

		return mergedSignal
	}

	// No pending interrupts, return current
	signal := ic.current
	ic.current = nil
	return signal
}

// Peek returns the current interrupt without removing it
func (ic *InterruptController) Peek() *InterruptSignal {
	if ic == nil {
		return nil
	}

	ic.mutex.RLock()
	defer ic.mutex.RUnlock()

	return ic.current
}

// IsInterrupted checks if interrupt context is cancelled (force interrupt)
func (ic *InterruptController) IsInterrupted() bool {
	if ic == nil || ic.ctx == nil {
		return false
	}

	select {
	case <-ic.ctx.Done():
		return true
	default:
		return false
	}
}

// Context returns the interrupt control context
// This can be used in select statements to check for force interrupts
func (ic *InterruptController) Context() context.Context {
	if ic == nil {
		return context.Background()
	}
	return ic.ctx
}

// GetPendingCount returns the number of pending interrupts
func (ic *InterruptController) GetPendingCount() int {
	if ic == nil {
		return 0
	}

	ic.mutex.RLock()
	defer ic.mutex.RUnlock()

	count := len(ic.pending)
	if ic.current != nil {
		count++
	}
	return count
}

// Clear clears all interrupts (current and pending)
func (ic *InterruptController) Clear() {
	if ic == nil {
		return
	}

	ic.mutex.Lock()
	defer ic.mutex.Unlock()

	ic.current = nil
	ic.pending = make([]*InterruptSignal, 0)
}

// Stop stops the interrupt controller and cleans up resources
func (ic *InterruptController) Stop() {
	if ic == nil {
		return
	}

	// Cancel context to stop listener
	if ic.cancel != nil {
		ic.cancel()
	}

	// Close channel
	if ic.queue != nil {
		close(ic.queue)
	}

	// Clear interrupts
	ic.Clear()
}

// SendSignal sends an interrupt signal to the controller
// This is called from external sources (e.g., another HTTP request)
func (ic *InterruptController) SendSignal(signal *InterruptSignal) error {
	if ic == nil {
		return fmt.Errorf("interrupt controller is nil")
	}

	if ic.queue == nil {
		return fmt.Errorf("interrupt queue is not initialized")
	}

	// Non-blocking send
	select {
	case ic.queue <- signal:
		return nil
	case <-time.After(500 * time.Millisecond):
		return fmt.Errorf("failed to send interrupt: timeout")
	}
}
