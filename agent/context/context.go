package context

import (
	"context"
	"fmt"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/plan"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/trace"
	traceTypes "github.com/yaoapp/yao/trace/types"
)

// New create a new context
func New(parent context.Context, authorized *types.AuthorizedInfo, chatID, payload string) Context {

	if parent == nil {
		parent = context.Background()
	}

	// Validate the client type
	ctx := Context{
		Context: parent,
		Space:   plan.NewMemorySharedSpace(),
		ChatID:  chatID,
	}

	if payload == "" {
		return ctx
	}

	err := jsoniter.Unmarshal([]byte(payload), &ctx)
	if err != nil {
		log.Error("%s", err.Error())
	}

	return ctx
}

// NewWithCancel create a new context with cancel
func NewWithCancel(parent context.Context, authorized *types.AuthorizedInfo, chatID, payload string) (Context, context.CancelFunc) {
	ctx := New(parent, authorized, chatID, payload)
	return WithCancel(ctx)
}

// NewWithTimeout create a new context with timeout
func NewWithTimeout(parent context.Context, authorized *types.AuthorizedInfo, chatID, payload string, timeout time.Duration) (Context, context.CancelFunc) {
	ctx := New(parent, authorized, chatID, payload)
	return WithTimeout(ctx, timeout)
}

// WithCancel create a new context
func WithCancel(parent Context) (Context, context.CancelFunc) {
	new, cancel := context.WithCancel(parent.Context)
	parent.Context = new
	return parent, cancel
}

// WithTimeout create a new context
func WithTimeout(parent Context, timeout time.Duration) (Context, context.CancelFunc) {
	new, cancel := context.WithTimeout(parent.Context, timeout)
	parent.Context = new
	return parent, cancel
}

// Release the context and clean up all resources including stacks and trace
func (ctx *Context) Release() {
	// Complete and release trace if exists
	if ctx.trace != nil && ctx.Stack != nil && ctx.Stack.TraceID != "" {
		// Mark trace as complete (sends final event)
		_ = ctx.trace.MarkComplete()

		// Release from global registry (removes from registry and closes resources)
		_ = trace.Release(ctx.Stack.TraceID)

		ctx.trace = nil
	}

	// Clear space
	if ctx.Space != nil {
		ctx.Space.Clear()
		ctx.Space = nil
	}

	// Clear stacks
	if ctx.Stacks != nil {
		for k := range ctx.Stacks {
			delete(ctx.Stacks, k)
		}
		ctx.Stacks = nil
	}

	// Clear current stack reference
	ctx.Stack = nil

	// Clear writer reference
	ctx.Writer = nil

	ctx = nil
}

// Send sends data to the context's writer
// This is used by the output module to send messages to the client
func (ctx *Context) Send(data []byte) error {
	if ctx.Writer == nil {
		return nil // No writer, silently ignore
	}

	_, err := ctx.Writer.Write(data)
	return err
}

// Trace returns the trace manager for this context, lazily initialized on first call
// Uses the TraceID from ctx.Stack if available, or generates a new one
func (ctx *Context) Trace() (traceTypes.Manager, error) {
	// Return trace if already initialized
	if ctx.trace != nil {
		return ctx.trace, nil
	}

	// Get TraceID from Stack or generate new one
	var traceID string
	if ctx.Stack != nil && ctx.Stack.TraceID != "" {
		traceID = ctx.Stack.TraceID

		// Try to load existing trace first
		manager, err := trace.Load(traceID)
		if err == nil {
			// Found in registry, reuse it
			ctx.trace = manager
			return manager, nil
		}
	}

	// Get trace configuration from global config
	cfg := config.Conf

	// Prepare driver options
	var driverOptions []any
	var driverType string

	switch cfg.Trace.Driver {
	case "store":
		driverType = trace.Store
		if cfg.Trace.Store == "" {
			return nil, fmt.Errorf("trace store ID not configured")
		}
		driverOptions = []any{cfg.Trace.Store, cfg.Trace.Prefix}

	case "local", "":
		driverType = trace.Local
		driverOptions = []any{cfg.Trace.Path}

	default:
		return nil, fmt.Errorf("unsupported trace driver: %s", cfg.Trace.Driver)
	}

	// Create trace using trace.New (handles registry)
	createdTraceID, manager, err := trace.New(ctx.Context, driverType, &traceTypes.TraceOption{
		ID: traceID, // Use existing ID from Stack or empty to generate new one
	}, driverOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace: %w", err)
	}

	// Update Stack with the created TraceID if needed
	if ctx.Stack != nil && ctx.Stack.TraceID == "" {
		ctx.Stack.TraceID = createdTraceID
	}

	// Store for future calls
	ctx.trace = manager

	return manager, nil
}

// Map the context to a map
func (ctx *Context) Map() map[string]interface{} {
	data := map[string]interface{}{}

	// Authorized information
	if ctx.Authorized != nil {
		data["authorized"] = ctx.Authorized
	}
	if ctx.ChatID != "" {
		data["chat_id"] = ctx.ChatID
	}
	if ctx.AssistantID != "" {
		data["assistant_id"] = ctx.AssistantID
	}
	if ctx.Connector != "" {
		data["connector"] = ctx.Connector
	}
	if ctx.Search != nil {
		data["search"] = *ctx.Search
	}

	// Arguments for call
	if len(ctx.Args) > 0 {
		data["args"] = ctx.Args
	}
	if ctx.Retry {
		data["retry"] = ctx.Retry
	}
	if ctx.RetryTimes > 0 {
		data["retry_times"] = ctx.RetryTimes
	}

	// Locale information
	if ctx.Locale != "" {
		data["locale"] = ctx.Locale
	}
	if ctx.Theme != "" {
		data["theme"] = ctx.Theme
	}

	// Request information
	if ctx.Client.Type != "" || ctx.Client.UserAgent != "" || ctx.Client.IP != "" {
		data["client"] = map[string]interface{}{
			"type":       ctx.Client.Type,
			"user_agent": ctx.Client.UserAgent,
			"ip":         ctx.Client.IP,
		}
	}
	if ctx.Referer != "" {
		data["referer"] = ctx.Referer
	}
	if ctx.Accept != "" {
		data["accept"] = ctx.Accept
	}

	// CUI Context information
	if ctx.Route != "" {
		data["route"] = ctx.Route
	}
	if len(ctx.Metadata) > 0 {
		data["metadata"] = ctx.Metadata
	}

	return data
}
