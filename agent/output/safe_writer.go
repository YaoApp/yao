package output

import (
	"context"
	"net/http"
	"sync"
)

// SafeWriter wraps http.ResponseWriter with a channel-based queue
// to serialize concurrent SSE writes and prevent "short write" errors.
//
// When multiple goroutines (e.g., concurrent sub-agents via ctx.agent.All)
// write to the same SSE stream, direct writes can cause data corruption
// or "short write" errors. SafeWriter solves this by:
//
// 1. Accepting write requests via a buffered channel
// 2. Processing writes sequentially in a dedicated goroutine
// 3. Providing non-blocking writes with overflow protection
// 4. Automatic cleanup when context is cancelled (client disconnect)
type SafeWriter struct {
	ch        chan writeRequest
	writer    http.ResponseWriter
	done      chan struct{}
	ctx       context.Context    // For detecting client disconnection
	cancel    context.CancelFunc // To signal run() to stop
	closeOnce sync.Once
	closed    bool
	mu        sync.RWMutex
}

// writeRequest represents a single write request
type writeRequest struct {
	data []byte
}

// QueueCapacity is the default buffer size for the write queue
// Large enough to handle high concurrency without blocking
const QueueCapacity = 10000

// NewSafeWriter creates a new SafeWriter that wraps an http.ResponseWriter
// and starts a background goroutine to process writes sequentially.
// The context should be the HTTP request context to detect client disconnection.
func NewSafeWriter(w http.ResponseWriter) *SafeWriter {
	// Create internal context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	sw := &SafeWriter{
		ch:     make(chan writeRequest, QueueCapacity),
		writer: w,
		done:   make(chan struct{}),
		ctx:    ctx,
		cancel: cancel,
	}
	go sw.run()
	return sw
}

// NewSafeWriterWithContext creates a SafeWriter that respects the given context.
// When the context is cancelled (e.g., client disconnects), the run() goroutine exits.
// This prevents goroutine leaks in enterprise applications with many concurrent requests.
func NewSafeWriterWithContext(ctx context.Context, w http.ResponseWriter) *SafeWriter {
	// Derive a cancellable context from the parent
	childCtx, cancel := context.WithCancel(ctx)
	sw := &SafeWriter{
		ch:     make(chan writeRequest, QueueCapacity),
		writer: w,
		done:   make(chan struct{}),
		ctx:    childCtx,
		cancel: cancel,
	}
	go sw.run()
	return sw
}

// run processes write requests from the channel sequentially
// Exits when channel is closed OR context is cancelled (client disconnect)
func (sw *SafeWriter) run() {
	defer close(sw.done)

	for {
		select {
		case req, ok := <-sw.ch:
			if !ok {
				// Channel closed, exit gracefully
				return
			}
			if sw.writer != nil {
				sw.writer.Write(req.data)
				// Flush after each write to ensure SSE data is sent immediately
				if flusher, ok := sw.writer.(http.Flusher); ok {
					flusher.Flush()
				}
			}
		case <-sw.ctx.Done():
			// Context cancelled (client disconnected or explicit close)
			// Continue reading from channel until it's closed to avoid blocking senders
			// and to process any remaining messages that were already queued
			sw.drainUntilClosed()
			return
		}
	}
}

// drainUntilClosed reads from channel until it's closed
// This prevents senders from blocking after context cancellation
func (sw *SafeWriter) drainUntilClosed() {
	for range sw.ch {
		// Discard messages - context is cancelled so we don't write them
	}
}

// Write implements io.Writer interface
// Queues the data for sequential writing by the background goroutine
func (sw *SafeWriter) Write(data []byte) (int, error) {
	sw.mu.RLock()
	if sw.closed {
		sw.mu.RUnlock()
		return 0, nil // Silently ignore writes after close
	}
	sw.mu.RUnlock()

	// Make a copy of data since the caller may reuse the buffer
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)

	// Non-blocking send with overflow protection
	select {
	case sw.ch <- writeRequest{data: dataCopy}:
		return len(data), nil
	default:
		// Channel full - this shouldn't happen with 10000 capacity
		// but if it does, drop the message rather than block
		// Note: In production, this indicates either:
		// 1. Extremely high concurrency (>10000 pending writes)
		// 2. The underlying writer is blocked/slow
		// Consider increasing QueueCapacity if this occurs frequently
		return len(data), nil
	}
}

// Header returns the header map from the underlying ResponseWriter
func (sw *SafeWriter) Header() http.Header {
	if sw.writer == nil {
		return http.Header{}
	}
	return sw.writer.Header()
}

// WriteHeader sends an HTTP response header with the provided status code
func (sw *SafeWriter) WriteHeader(statusCode int) {
	if sw.writer != nil {
		sw.writer.WriteHeader(statusCode)
	}
}

// Flush implements http.Flusher interface
// Note: Actual flushing happens in the run() goroutine after each write
func (sw *SafeWriter) Flush() {
	// Flushing is handled automatically in run() after each write
	// This method exists to satisfy the http.Flusher interface
}

// Close closes the write channel and waits for all pending writes to complete
// This is safe to call multiple times (idempotent via sync.Once)
func (sw *SafeWriter) Close() error {
	sw.closeOnce.Do(func() {
		// First close channel to signal run() to stop and process remaining messages
		close(sw.ch)

		// Wait for run() to finish processing all queued messages
		<-sw.done

		// Then mark as closed and cancel context
		sw.mu.Lock()
		sw.closed = true
		sw.mu.Unlock()

		sw.cancel()
	})
	return nil
}

// IsClosed returns whether the SafeWriter has been closed
func (sw *SafeWriter) IsClosed() bool {
	sw.mu.RLock()
	defer sw.mu.RUnlock()
	return sw.closed
}

// Underlying returns the underlying http.ResponseWriter
// Use with caution - direct writes bypass the queue
func (sw *SafeWriter) Underlying() http.ResponseWriter {
	return sw.writer
}
