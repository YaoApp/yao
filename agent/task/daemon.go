package task

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/agent/output/message"
)

// DaemonStatus represents the current state of a daemon execution
type DaemonStatus string

const (
	DaemonRunning  DaemonStatus = "running"
	DaemonStopping DaemonStatus = "stopping"
	DaemonStopped  DaemonStatus = "stopped"
)

// DaemonContext manages the lifecycle of a single task execution,
// decoupled from HTTP connections. Subscribers receive messages via channels.
type DaemonContext struct {
	ChatID      string
	Context     context.Context
	Cancel      context.CancelFunc // graceful stop
	ForceCancel context.CancelFunc // force interrupt

	mu          sync.Mutex
	subscribers map[string]chan<- *message.Message
	sequence    int64
	status      DaemonStatus
	idleTimer   *time.Timer
	ringBuffer  []*message.Message
}

var daemonRegistry = &sync.Map{}

// GetDaemon retrieves a running DaemonContext by chat_id
func GetDaemon(chatID string) (*DaemonContext, bool) {
	v, ok := daemonRegistry.Load(chatID)
	if !ok {
		return nil, false
	}
	return v.(*DaemonContext), true
}

// UnregisterDaemon removes a DaemonContext from the registry
func UnregisterDaemon(chatID string) {
	daemonRegistry.Delete(chatID)
}

// newDaemonContext creates a DaemonContext with three-level context hierarchy:
//
//	globalShutdown (server shutdown)
//	  └→ gracefulCtx (Cancel: graceful stop)
//	      └→ forceCtx (ForceCancel: immediate interrupt)
func newDaemonContext(chatID string) *DaemonContext {
	gracefulCtx, gracefulCancel := context.WithCancel(globalShutdown)
	forceCtx, forceCancel := context.WithCancel(gracefulCtx)
	dc := &DaemonContext{
		ChatID:      chatID,
		Context:     forceCtx,
		Cancel:      gracefulCancel,
		ForceCancel: forceCancel,
		subscribers: make(map[string]chan<- *message.Message),
		status:      DaemonRunning,
	}
	dc.idleTimer = time.AfterFunc(30*time.Minute, func() {
		dc.Cancel()
	})
	return dc
}

// NextSequence returns the next monotonically increasing sequence number
func (dc *DaemonContext) NextSequence() int64 {
	return atomic.AddInt64(&dc.sequence, 1)
}

// Status returns the current daemon status (thread-safe)
func (dc *DaemonContext) Status() DaemonStatus {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	return dc.status
}

// SetStatus updates the daemon status (thread-safe)
func (dc *DaemonContext) SetStatus(s DaemonStatus) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.status = s
}

// Broadcast sends a message to all subscribers and appends to ringBuffer
func (dc *DaemonContext) Broadcast(msg *message.Message) {
	seq := dc.NextSequence()
	if msg.Metadata == nil {
		msg.Metadata = &message.Metadata{}
	}
	msg.Metadata.Sequence = int(seq)
	if msg.Metadata.Timestamp == 0 {
		msg.Metadata.Timestamp = time.Now().UnixMilli()
	}

	dc.mu.Lock()
	dc.ringBuffer = append(dc.ringBuffer, msg)
	subs := make(map[string]chan<- *message.Message, len(dc.subscribers))
	for k, v := range dc.subscribers {
		subs[k] = v
	}
	dc.mu.Unlock()

	for id, ch := range subs {
		select {
		case ch <- msg:
		default:
			log.Warn("task daemon: subscriber %s channel full, message seq=%d dropped", id, seq)
		}
	}

	dc.resetIdleTimer()
}

// CloseSubscribers closes all subscriber channels (call after final Broadcast)
func (dc *DaemonContext) CloseSubscribers() {
	dc.mu.Lock()
	subs := dc.subscribers
	dc.subscribers = nil
	dc.mu.Unlock()
	for _, ch := range subs {
		close(ch)
	}
}

// SubscribeLive subscribes to the daemon's live stream.
// It replays the entire ringBuffer first, then pipes live messages.
// The caller should use message_id deduplication for messages that overlap with DB history.
func (dc *DaemonContext) SubscribeLive() (*WatchStream, error) {
	liveCh := make(chan *message.Message, 64)
	outputCh := make(chan *message.Message, 64)
	subID := uuid.New().String()

	dc.mu.Lock()
	if dc.subscribers == nil {
		dc.mu.Unlock()
		return nil, ErrDaemonStopping
	}

	replay := make([]*message.Message, len(dc.ringBuffer))
	copy(replay, dc.ringBuffer)
	dc.subscribers[subID] = liveCh
	fmt.Printf("  • [task.subscribeLive] chatID=%s subID=%s totalSubs=%d replay=%d\n", dc.ChatID, subID, len(dc.subscribers), len(replay))
	dc.mu.Unlock()

	doneCh := make(chan struct{})
	go func() {
		defer close(outputCh)
		// Phase 1: replay ringBuffer
		for _, m := range replay {
			select {
			case outputCh <- m:
			case <-doneCh:
				return
			}
		}
		// Phase 2: live pipe
		for {
			select {
			case m, ok := <-liveCh:
				if !ok {
					return
				}
				select {
				case outputCh <- m:
				case <-doneCh:
					return
				}
			case <-doneCh:
				return
			}
		}
	}()

	var cancelOnce sync.Once
	return &WatchStream{
		Ch: outputCh,
		Cancel: func() {
			cancelOnce.Do(func() {
				fmt.Printf("  • [task.subscribeLive.cancel] chatID=%s subID=%s\n", dc.ChatID, subID)
				dc.mu.Lock()
				delete(dc.subscribers, subID)
				dc.mu.Unlock()
				close(doneCh)
			})
		},
		LiveMode: true,
	}, nil
}

// resetIdleTimer resets the 30-minute idle timer
func (dc *DaemonContext) resetIdleTimer() {
	if dc.idleTimer != nil {
		dc.idleTimer.Reset(30 * time.Minute)
	}
}

// StopIdleTimer stops the idle timer (call on daemon exit to prevent orphan timer fire)
func (dc *DaemonContext) StopIdleTimer() {
	if dc.idleTimer != nil {
		dc.idleTimer.Stop()
	}
}

// DaemonResponseWriter implements http.ResponseWriter for DaemonContext.
// It intercepts SSE-formatted output from assistant.Stream() and broadcasts
// parsed messages to subscribers.
type DaemonResponseWriter struct {
	dc     *DaemonContext
	header http.Header
	buf    bytes.Buffer
}

// NewDaemonResponseWriter creates a new DaemonResponseWriter
func NewDaemonResponseWriter(dc *DaemonContext) *DaemonResponseWriter {
	return &DaemonResponseWriter{
		dc:     dc,
		header: http.Header{},
	}
}

func (w *DaemonResponseWriter) Header() http.Header { return w.header }
func (w *DaemonResponseWriter) WriteHeader(int)     {}
func (w *DaemonResponseWriter) Write(p []byte) (int, error) {
	w.buf.Write(p)
	w.processBuffer()
	return len(p), nil
}

// Flush implements http.Flusher (no-op for daemon writer)
func (w *DaemonResponseWriter) Flush() {}

// processBuffer extracts complete SSE "data:" lines and broadcasts parsed messages
func (w *DaemonResponseWriter) processBuffer() {
	for {
		line, err := w.buf.ReadString('\n')
		if err != nil {
			// Incomplete line — put it back for next Write call
			w.buf.WriteString(line)
			return
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		if payload == "[DONE]" {
			continue
		}
		var msg message.Message
		if err := json.Unmarshal([]byte(payload), &msg); err != nil {
			continue
		}
		w.dc.Broadcast(&msg)
	}
}
