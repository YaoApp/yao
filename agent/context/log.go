package context

import (
	"fmt"
	"strings"
	"sync"
	"time"

	kunlog "github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
)

// =============================================================================
// ANSI Color Codes
// =============================================================================

const (
	colorReset   = "\033[0m"
	colorRed     = "\033[31m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorBlue    = "\033[34m"
	colorMagenta = "\033[35m"
	colorCyan    = "\033[36m"
	colorWhite   = "\033[37m"
	colorGray    = "\033[90m"

	colorBoldRed     = "\033[1;31m"
	colorBoldGreen   = "\033[1;32m"
	colorBoldYellow  = "\033[1;33m"
	colorBoldBlue    = "\033[1;34m"
	colorBoldMagenta = "\033[1;35m"
	colorBoldCyan    = "\033[1;36m"
)

// =============================================================================
// Log Level
// =============================================================================

// LogLevel represents log severity
type LogLevel int

const (
	// LogLevelTrace represents the most verbose logging level for detailed tracing
	LogLevelTrace LogLevel = iota
	// LogLevelDebug represents debug level logging for development diagnostics
	LogLevelDebug
	// LogLevelInfo represents informational messages for normal operation
	LogLevelInfo
	// LogLevelWarn represents warning messages for potentially harmful situations
	LogLevelWarn
	// LogLevelError represents error messages for serious problems
	LogLevelError
)

// =============================================================================
// Log Entry
// =============================================================================

// LogEntry represents a single log entry
type LogEntry struct {
	Level     LogLevel
	Message   string
	Timestamp time.Time
	Phase     string // For phase logging
	Elapsed   time.Duration
}

// =============================================================================
// Request Logger
// =============================================================================

// RequestLogger provides request-scoped async logging
type RequestLogger struct {
	assistantID string
	chatID      string
	requestID   string
	shortID     string // Short version of requestID for display
	startTime   time.Time

	ch     chan LogEntry
	done   chan struct{}
	once   sync.Once
	closed bool
	noop   bool // noop logger does nothing (for nil safety)
	mu     sync.RWMutex
}

// noopLogger is a shared no-op logger instance
var noopLogger = &RequestLogger{noop: true}

// NewRequestLogger creates a new request-scoped logger with async processing
func NewRequestLogger(assistantID, chatID, requestID string) *RequestLogger {
	l := &RequestLogger{
		assistantID: assistantID,
		chatID:      chatID,
		requestID:   requestID,
		shortID:     shortID(requestID),
		startTime:   time.Now(),
		ch:          make(chan LogEntry, 100), // Buffered channel
		done:        make(chan struct{}),
	}

	// Start consumer goroutine
	go l.consume()

	return l
}

// Noop returns a no-op logger that does nothing (nil-safe)
func Noop() *RequestLogger {
	return noopLogger
}

// SetAssistantID sets the assistant ID (called when entering Stream)
func (l *RequestLogger) SetAssistantID(id string) {
	if l.noop {
		return
	}
	l.assistantID = id
}

// Close closes the logger and waits for all entries to be processed
func (l *RequestLogger) Close() {
	if l.noop {
		return
	}
	l.once.Do(func() {
		l.mu.Lock()
		l.closed = true
		l.mu.Unlock()

		close(l.ch)
		<-l.done // Wait for consumer to finish
	})
}

// consume processes log entries from the channel
func (l *RequestLogger) consume() {
	defer close(l.done)

	for entry := range l.ch {
		l.processEntry(entry)
	}
}

// processEntry handles a single log entry based on mode
func (l *RequestLogger) processEntry(entry LogEntry) {
	if config.IsDevelopment() {
		l.printDev(entry)
	} else {
		l.printProd(entry)
	}
}

// printDev prints colorful output for development mode
func (l *RequestLogger) printDev(entry LogEntry) {
	switch entry.Level {
	case LogLevelTrace:
		fmt.Printf("%s  → %s%s\n", colorGray, entry.Message, colorReset)
	case LogLevelDebug:
		fmt.Printf("%s  • %s%s\n", colorGray, entry.Message, colorReset)
	case LogLevelInfo:
		fmt.Printf("%s  ℹ %s%s\n", colorCyan, entry.Message, colorReset)
	case LogLevelWarn:
		fmt.Printf("%s  ⚠ %s%s\n", colorYellow, entry.Message, colorReset)
	case LogLevelError:
		fmt.Printf("%s  ✗ %s%s\n", colorRed, entry.Message, colorReset)
	}
}

// printProd logs to kun/log for production mode
func (l *RequestLogger) printProd(entry LogEntry) {
	prefix := fmt.Sprintf("[AGENT] %s ", l.shortID)

	switch entry.Level {
	case LogLevelTrace:
		kunlog.Trace("%s%s", prefix, entry.Message)
	case LogLevelDebug:
		// Skip debug in production
	case LogLevelInfo:
		kunlog.Info("%s%s", prefix, entry.Message)
	case LogLevelWarn:
		kunlog.Warn("%s%s", prefix, entry.Message)
	case LogLevelError:
		kunlog.Error("%s%s", prefix, entry.Message)
	}
}

// send sends an entry to the channel (non-blocking if closed)
func (l *RequestLogger) send(entry LogEntry) {
	if l.noop {
		return
	}

	l.mu.RLock()
	closed := l.closed
	l.mu.RUnlock()

	if closed {
		return
	}

	entry.Timestamp = time.Now()
	select {
	case l.ch <- entry:
	default:
		// Channel full, drop the log (shouldn't happen with buffered channel)
	}
}

// =============================================================================
// Standard Log Interface
// =============================================================================

// Trace logs a trace level message
func (l *RequestLogger) Trace(format string, args ...interface{}) {
	l.send(LogEntry{
		Level:   LogLevelTrace,
		Message: fmt.Sprintf(format, args...),
	})
}

// Debug logs a debug level message
func (l *RequestLogger) Debug(format string, args ...interface{}) {
	l.send(LogEntry{
		Level:   LogLevelDebug,
		Message: fmt.Sprintf(format, args...),
	})
}

// Info logs an info level message
func (l *RequestLogger) Info(format string, args ...interface{}) {
	l.send(LogEntry{
		Level:   LogLevelInfo,
		Message: fmt.Sprintf(format, args...),
	})
}

// Warn logs a warning level message
func (l *RequestLogger) Warn(format string, args ...interface{}) {
	l.send(LogEntry{
		Level:   LogLevelWarn,
		Message: fmt.Sprintf(format, args...),
	})
}

// Error logs an error level message
func (l *RequestLogger) Error(format string, args ...interface{}) {
	l.send(LogEntry{
		Level:   LogLevelError,
		Message: fmt.Sprintf(format, args...),
	})
}

// =============================================================================
// Business Quick Functions
// =============================================================================

// Start logs the start of a request with visual separator
func (l *RequestLogger) Start() {
	if l.noop {
		return
	}

	if !config.IsDevelopment() {
		kunlog.Trace("[AGENT] Request %s started: assistant=%s, chat=%s, request=%s",
			l.shortID, l.assistantID, shortID(l.chatID), shortID(l.requestID))
		return
	}

	// Development: colorful output (direct print, not through channel for immediate display)
	fmt.Println()
	fmt.Printf("%s%s%s\n", colorBoldCyan, strings.Repeat("═", 60), colorReset)
	fmt.Printf("%s  🚀 AGENT REQUEST %s%s\n", colorBoldCyan, l.shortID, colorReset)
	fmt.Printf("%s%s%s\n", colorBoldCyan, strings.Repeat("─", 60), colorReset)
	fmt.Printf("%s  Assistant: %s%s%s\n", colorGray, colorWhite, l.assistantID, colorReset)
	fmt.Printf("%s  Chat ID:   %s%s%s\n", colorGray, colorWhite, l.chatID, colorReset)
	fmt.Printf("%s  Request:   %s%s%s\n", colorGray, colorWhite, l.requestID, colorReset)
	fmt.Printf("%s  Time:      %s%s%s\n", colorGray, colorWhite, l.startTime.Format("15:04:05.000"), colorReset)
	fmt.Printf("%s%s%s\n", colorCyan, strings.Repeat("─", 60), colorReset)
}

// End logs the end of a request with summary
func (l *RequestLogger) End(success bool, err error) {
	if l.noop {
		return
	}

	duration := time.Since(l.startTime)

	if !config.IsDevelopment() {
		if success {
			kunlog.Trace("[AGENT] Request %s completed: assistant=%s, duration=%v",
				l.shortID, l.assistantID, duration.Round(time.Millisecond))
		} else {
			kunlog.Trace("[AGENT] Request %s failed: assistant=%s, duration=%v, error=%v",
				l.shortID, l.assistantID, duration.Round(time.Millisecond), err)
		}
		return
	}

	// Development: colorful output (direct print for immediate display)
	fmt.Printf("%s%s%s\n", colorCyan, strings.Repeat("─", 60), colorReset)
	if success {
		fmt.Printf("%s  ✅ REQUEST %s COMPLETED%s\n", colorBoldGreen, l.shortID, colorReset)
	} else {
		fmt.Printf("%s  ❌ REQUEST %s FAILED%s\n", colorBoldRed, l.shortID, colorReset)
		if err != nil {
			fmt.Printf("%s  Error: %s%v%s\n", colorGray, colorRed, err, colorReset)
		}
	}
	fmt.Printf("%s  Assistant: %s%s%s\n", colorGray, colorWhite, l.assistantID, colorReset)
	fmt.Printf("%s  Duration:  %s%v%s\n", colorGray, colorWhite, duration.Round(time.Millisecond), colorReset)
	fmt.Printf("%s%s%s\n", colorCyan, strings.Repeat("─", 60), colorReset)
	fmt.Println()
}

// Phase logs a major phase in the request lifecycle
func (l *RequestLogger) Phase(name string) {
	if l.noop {
		return
	}

	elapsed := time.Since(l.startTime).Round(time.Millisecond)

	if config.IsDevelopment() {
		fmt.Printf("%s  ▶ %s%s %s[+%v]%s\n", colorBoldBlue, name, colorReset, colorGray, elapsed, colorReset)
	} else {
		kunlog.Trace("[AGENT] %s Phase: %s (+%v)", l.shortID, name, elapsed)
	}
}

// PhaseComplete logs the completion of a phase
func (l *RequestLogger) PhaseComplete(name string) {
	if l.noop {
		return
	}

	elapsed := time.Since(l.startTime).Round(time.Millisecond)

	if config.IsDevelopment() {
		fmt.Printf("%s  ✓ %s%s %s[+%v]%s\n", colorGreen, name, colorReset, colorGray, elapsed, colorReset)
	} else {
		kunlog.Trace("[AGENT] %s Phase completed: %s (+%v)", l.shortID, name, elapsed)
	}
}

// PhaseSkip logs a skipped phase (development only)
func (l *RequestLogger) PhaseSkip(name, reason string) {
	if l.noop {
		return
	}
	if config.IsDevelopment() {
		fmt.Printf("%s  ⊘ %s (%s)%s\n", colorGray, name, reason, colorReset)
	}
}

// LLMStart logs the start of an LLM call
func (l *RequestLogger) LLMStart(connector, model string, messageCount int) {
	if l.noop {
		return
	}

	elapsed := time.Since(l.startTime).Round(time.Millisecond)

	if config.IsDevelopment() {
		fmt.Printf("%s  🤖 LLM Call%s %s[+%v]%s\n", colorBoldMagenta, colorReset, colorGray, elapsed, colorReset)
		fmt.Printf("%s    Connector: %s%s%s\n", colorGray, colorWhite, connector, colorReset)
		if model != "" {
			fmt.Printf("%s    Model: %s%s%s\n", colorGray, colorWhite, model, colorReset)
		}
		fmt.Printf("%s    Messages: %s%d%s\n", colorGray, colorWhite, messageCount, colorReset)
	} else {
		kunlog.Trace("[AGENT] %s LLM call: connector=%s, model=%s, messages=%d (+%v)", l.shortID, connector, model, messageCount, elapsed)
	}
}

// LLMComplete logs the completion of an LLM call
func (l *RequestLogger) LLMComplete(tokens int, hasToolCalls bool) {
	if l.noop {
		return
	}

	elapsed := time.Since(l.startTime).Round(time.Millisecond)
	status := "streaming"
	if hasToolCalls {
		status = "tool_calls"
	}

	if config.IsDevelopment() {
		fmt.Printf("%s  ✓ LLM Response (%s)%s", colorGreen, status, colorReset)
		if tokens > 0 {
			fmt.Printf(" %s[tokens: %d]%s", colorGray, tokens, colorReset)
		}
		fmt.Printf(" %s[+%v]%s\n", colorGray, elapsed, colorReset)
	} else {
		kunlog.Trace("[AGENT] %s LLM response: status=%s, tokens=%d (+%v)", l.shortID, status, tokens, elapsed)
	}
}

// ToolStart logs the start of tool execution
func (l *RequestLogger) ToolStart(toolName string) {
	if l.noop {
		return
	}

	if config.IsDevelopment() {
		fmt.Printf("%s  🔧 Tool: %s%s\n", colorYellow, toolName, colorReset)
	} else {
		kunlog.Trace("[AGENT] %s Tool call: %s", l.shortID, toolName)
	}
}

// ToolComplete logs the completion of tool execution
func (l *RequestLogger) ToolComplete(toolName string, success bool) {
	if l.noop {
		return
	}

	if config.IsDevelopment() {
		if success {
			fmt.Printf("%s    ✓ %s completed%s\n", colorGreen, toolName, colorReset)
		} else {
			fmt.Printf("%s    ✗ %s failed%s\n", colorRed, toolName, colorReset)
		}
	} else {
		if success {
			kunlog.Trace("[AGENT] %s Tool completed: %s", l.shortID, toolName)
		} else {
			kunlog.Trace("[AGENT] %s Tool failed: %s", l.shortID, toolName)
		}
	}
}

// HookStart logs the start of a hook execution
func (l *RequestLogger) HookStart(hookName string) {
	if l.noop {
		return
	}

	elapsed := time.Since(l.startTime).Round(time.Millisecond)

	if config.IsDevelopment() {
		fmt.Printf("%s  🪝 Hook: %s%s %s[+%v]%s\n", colorMagenta, hookName, colorReset, colorGray, elapsed, colorReset)
	} else {
		kunlog.Trace("[AGENT] %s Hook: %s (+%v)", l.shortID, hookName, elapsed)
	}
}

// HookComplete logs the completion of a hook
func (l *RequestLogger) HookComplete(hookName string) {
	if l.noop {
		return
	}

	if config.IsDevelopment() {
		fmt.Printf("%s    ✓ %s done%s\n", colorGreen, hookName, colorReset)
	} else {
		kunlog.Trace("[AGENT] %s Hook completed: %s", l.shortID, hookName)
	}
}

// Cleanup logs resource cleanup
func (l *RequestLogger) Cleanup(resource string) {
	if l.noop {
		return
	}

	if config.IsDevelopment() {
		fmt.Printf("%s    ✓ %s%s\n", colorGray, resource, colorReset)
	} else {
		kunlog.Trace("[AGENT] %s Cleanup: %s", l.shortID, resource)
	}
}

// HistoryLoad logs history loading
func (l *RequestLogger) HistoryLoad(count, maxSize int) {
	if l.noop {
		return
	}

	if config.IsDevelopment() {
		fmt.Printf("%s    Loaded %d/%d history messages%s\n", colorGray, count, maxSize, colorReset)
	} else {
		kunlog.Trace("[AGENT] %s History loaded: %d/%d messages", l.shortID, count, maxSize)
	}
}

// HistoryOverlap logs overlap detection
func (l *RequestLogger) HistoryOverlap(overlapCount int) {
	if l.noop {
		return
	}

	if overlapCount > 0 {
		if config.IsDevelopment() {
			fmt.Printf("%s    Removed %d overlapping messages%s\n", colorYellow, overlapCount, colorReset)
		} else {
			kunlog.Trace("[AGENT] %s History overlap removed: %d messages", l.shortID, overlapCount)
		}
	}
}

// Release logs the start of resource release phase
func (l *RequestLogger) Release() {
	if l.noop {
		return
	}

	if config.IsDevelopment() {
		fmt.Printf("%s  🧹 RELEASE %s%s %s(%s)%s\n", colorBoldYellow, l.shortID, colorReset, colorGray, l.assistantID, colorReset)
	} else {
		kunlog.Trace("[AGENT] %s Release started", l.shortID)
	}
}

// =============================================================================
// Helper
// =============================================================================

// shortID returns first 8 characters of an ID
func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}
