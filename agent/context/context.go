package context

import (
	"context"
	"fmt"
	"sync"

	"github.com/yaoapp/yao/agent/memory"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/trace"
	traceTypes "github.com/yaoapp/yao/trace/types"
)

// Global context registry for interrupt management
var (
	contextRegistry = &sync.Map{} // map[contextID]*Context
)

// New create a new context with basic initialization
func New(parent context.Context, authorized *types.AuthorizedInfo, chatID string) *Context {
	if parent == nil {
		parent = context.Background()
	}

	contextID := generateContextID()

	// Extract user and team IDs from authorized info
	var userID, teamID string
	if authorized != nil {
		userID = authorized.UserID
		teamID = authorized.TeamID
	}

	// Create memory instance using global manager
	mem, _ := memory.GetMemory(userID, teamID, chatID, contextID)

	ctx := &Context{
		Context:         parent,
		ID:              contextID,  // Generate unique ID for the context
		Authorized:      authorized, // Set authorized info
		Memory:          mem,
		ChatID:          chatID,
		IDGenerator:     message.NewIDGenerator(),                // Initialize ID generator for this context
		messageMetadata: newMessageMetadataStore(),               // Initialize message metadata store
		Logger:          NewRequestLogger("", chatID, contextID), // Initialize logger (assistantID set later)
	}

	return ctx
}

// Release the context and clean up all resources including stacks and trace
func (ctx *Context) Release() {
	if ctx.Logger != nil {
		ctx.Logger.Release()
	}

	// Unregister from global registry
	if ctx.ID != "" {
		Unregister(ctx.ID)
	}

	// Stop interrupt controller
	if ctx.Interrupt != nil {
		if ctx.Logger != nil {
			ctx.Logger.Cleanup("Interrupt controller")
		}
		ctx.Interrupt.Stop()
		ctx.Interrupt = nil
	}

	// Complete and release trace if exists
	if ctx.trace != nil && ctx.Stack != nil && ctx.Stack.TraceID != "" {
		if ctx.Logger != nil {
			ctx.Logger.Cleanup("Trace: " + ctx.Stack.TraceID)
		}

		// Check if context is cancelled - if so, mark as cancelled instead of complete
		if ctx.Context != nil && ctx.Context.Err() != nil {
			trace.MarkCancelled(ctx.Stack.TraceID, ctx.Context.Err().Error())
			trace.Release(ctx.Stack.TraceID)
		} else {
			ctx.trace.MarkComplete()
			trace.Release(ctx.Stack.TraceID)
		}
		ctx.trace = nil
	}

	// Clear context-level memory only (request-scoped temporary data)
	// User, Team, Chat level memory is persistent and should NOT be cleared
	if ctx.Memory != nil && ctx.Memory.Context != nil {
		if ctx.Logger != nil {
			ctx.Logger.Cleanup("Memory.Context")
		}
		ctx.Memory.Context.Clear()
	}
	ctx.Memory = nil

	// Clear stacks
	if ctx.Stacks != nil {
		if ctx.Logger != nil {
			ctx.Logger.Cleanup(fmt.Sprintf("Stacks (%d)", len(ctx.Stacks)))
		}
		for k := range ctx.Stacks {
			delete(ctx.Stacks, k)
		}
		ctx.Stacks = nil
	}

	// Clear current stack reference
	ctx.Stack = nil

	// Close SafeWriter if exists (must be before setting Writer to nil)
	// This ensures the background goroutine is properly stopped
	ctx.CloseSafeWriter()

	// Clear writer reference
	ctx.Writer = nil

	// Close logger (MUST be last)
	if ctx.Logger != nil {
		ctx.Logger.Close()
		ctx.Logger = nil
	}
}

// GetAuthorizedMap returns the authorized information as a map
// This implements the AuthorizedProvider interface for MCP process calls
// Allows MCP tools to receive authorization context when called via Process transport
func (ctx *Context) GetAuthorizedMap() map[string]interface{} {
	if ctx.Authorized == nil {
		return nil
	}
	return ctx.Authorized.AuthorizedToMap()
}

// Fork creates a child context for concurrent agent/LLM calls
// The forked context shares read-only resources (Authorized, Cache, Writer)
// but has its own independent Stack, Logger, and Memory.Context namespace
// to avoid race conditions and state sharing issues.
//
// This is essential for batch operations (All/Any/Race) where multiple goroutines
// need to execute concurrently without interfering with each other's state.
//
// Key behavior:
// - Memory.User, Memory.Team, Memory.Chat are shared (cross-request state)
// - Memory.Context is INDEPENDENT (request-scoped state, isolated per fork)
//
// The forked context does NOT need to be released separately - the parent context
// manages shared resources. However, the child's Stack will be collected in parent's Stacks map.
func (ctx *Context) Fork() *Context {
	childID := generateContextID()

	// Fork memory with independent Context namespace
	// This prevents parallel sub-agents from sharing ctx.memory.context state
	var forkedMemory *memory.Memory
	if ctx.Memory != nil {
		var err error
		forkedMemory, err = ctx.Memory.Fork(childID)
		if err != nil {
			// Fallback to shared memory if fork fails (log warning)
			forkedMemory = ctx.Memory
		}
	}

	child := &Context{
		// Inherit parent's standard context
		Context: ctx.Context,

		// New unique ID for this forked context
		ID: childID,

		// Memory with independent Context namespace (see above)
		Memory: forkedMemory,

		// Share read-only/thread-safe resources with parent
		Cache:        ctx.Cache,        // Cache store is thread-safe
		Writer:       ctx.Writer,       // Output writer is thread-safe (output module handles concurrency)
		Authorized:   ctx.Authorized,   // Read-only auth info
		Capabilities: ctx.Capabilities, // Read-only model capabilities

		// Share reference to parent's Stacks map for trace collection
		// Child stacks will be added here by EnterStack
		Stacks: ctx.Stacks,

		// Stack is nil for forked contexts - will be set by EnterStack
		// ForkParent stores parent stack info so EnterStack can create child stack
		Stack: nil,

		// Create independent resources to avoid race conditions
		IDGenerator:     message.NewIDGenerator(),
		Logger:          NewRequestLogger(ctx.AssistantID, ctx.ChatID, childID),
		messageMetadata: newMessageMetadataStore(),

		// Inherit context metadata
		ChatID:      ctx.ChatID,
		AssistantID: ctx.AssistantID,
		Locale:      ctx.Locale,
		Theme:       ctx.Theme,
		Client:      ctx.Client,
		Referer:     ctx.Referer,
		Accept:      ctx.Accept,
		Route:       ctx.Route,
		Metadata:    ctx.Metadata,

		// Don't inherit these - they are request-specific
		Buffer:    nil, // Buffer belongs to root context
		Interrupt: nil, // Interrupt controller belongs to root context
		trace:     nil, // Trace will be inherited via TraceID in Stack
	}

	// Set ForkParent info if parent has a Stack
	// This allows EnterStack to create a child stack instead of root stack
	if ctx.Stack != nil {
		child.ForkParent = &ForkParentInfo{
			StackID: ctx.Stack.ID,
			TraceID: ctx.Stack.TraceID,
			Depth:   ctx.Stack.Depth,
			Path:    append([]string{}, ctx.Stack.Path...), // Copy path slice
		}
	}

	return child
}

// Send sends data to the context's writer
// This is used by the output module to send messages to the client
// func (ctx *Context) Send(data []byte) error {
// 	if ctx.Writer == nil {
// 		return nil // No writer, silently ignore
// 	}

// 	_, err := ctx.Writer.Write(data)
// 	return err
// }

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

	// Prepare trace options
	traceOption := &traceTypes.TraceOption{ID: traceID, AutoArchive: config.Conf.Mode == "production"}

	// Set trace options from authorized information
	if ctx.Authorized != nil {
		traceOption.CreatedBy = ctx.Authorized.UserID
		traceOption.TeamID = ctx.Authorized.TeamID
		traceOption.TenantID = ctx.Authorized.TenantID
	}

	// Create trace using trace.New (handles registry)
	createdTraceID, manager, err := trace.New(ctx.Context, driverType, traceOption, driverOptions...)
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

// Global Registry Functions
// ===================================

// Register registers a context to the global registry
func Register(ctx *Context) error {
	if ctx == nil {
		return fmt.Errorf("context is nil")
	}

	if ctx.ID == "" {
		return fmt.Errorf("context ID is empty")
	}

	contextRegistry.Store(ctx.ID, ctx)
	return nil
}

// Unregister removes a context from the global registry
func Unregister(contextID string) {
	contextRegistry.Delete(contextID)
}

// Get retrieves a context from the global registry by ID
func Get(contextID string) (*Context, error) {
	value, ok := contextRegistry.Load(contextID)
	if !ok {
		return nil, fmt.Errorf("context not found: %s", contextID)
	}

	ctx, ok := value.(*Context)
	if !ok {
		return nil, fmt.Errorf("invalid context type")
	}

	return ctx, nil
}

// SendInterrupt sends an interrupt signal to a context by ID
// This is the main entry point for external interrupt requests
func SendInterrupt(contextID string, signal *InterruptSignal) error {
	ctx, err := Get(contextID)
	if err != nil {
		return err
	}

	if ctx.Interrupt == nil {
		return fmt.Errorf("interrupt controller not initialized for context: %s", contextID)
	}

	return ctx.Interrupt.SendSignal(signal)
}

// generateContextID generates a unique context ID
func generateContextID() string {
	return message.GenerateNanoID()
}

// RequestID returns a unique request ID using NanoID
func (ctx *Context) RequestID() string {
	return message.GenerateNanoID()
}

// TraceID returns the trace ID for the context
func (ctx *Context) TraceID() string {
	if ctx.Stack != nil {
		return ctx.Stack.TraceID
	}
	return ""
}

// getMessageMetadata retrieves metadata for a message by ID
// Returns nil if message metadata is not found
func (ctx *Context) getMessageMetadata(messageID string) *MessageMetadata {
	if ctx.messageMetadata == nil {
		return nil
	}
	return ctx.messageMetadata.getMessage(messageID)
}

// GetMessageMetadata returns metadata for a message (public version)
func (ctx *Context) GetMessageMetadata(messageID string) *MessageMetadata {
	return ctx.getMessageMetadata(messageID)
}

// =============================================================================
// Chat Buffer Methods
// =============================================================================

// InitBuffer initializes the chat buffer for this context
// Should be called at the start of Stream() to begin buffering messages and steps
func (ctx *Context) InitBuffer(assistantID, connector, mode string) *ChatBuffer {
	ctx.Buffer = NewChatBuffer(ctx.ChatID, ctx.RequestID(), assistantID, connector, mode)
	return ctx.Buffer
}

// HasBuffer returns true if the buffer is initialized
func (ctx *Context) HasBuffer() bool {
	return ctx.Buffer != nil
}

// BufferUserInput adds user input to the buffer
// Should be called at the start of Stream() to buffer the user's input message
func (ctx *Context) BufferUserInput(messages []Message) {
	if ctx.Buffer == nil {
		return
	}

	for _, msg := range messages {
		if msg.Role == RoleUser {
			// Get name if available
			var name string
			if msg.Name != nil {
				name = *msg.Name
			}
			ctx.Buffer.AddUserInput(msg.Content, name)
		}
	}
}

// BufferAssistantMessage adds an assistant message to the buffer
// Called by ctx.Send() to buffer messages for batch saving
func (ctx *Context) BufferAssistantMessage(messageID, msgType string, props map[string]interface{}, blockID, threadID string, metadata map[string]interface{}) {
	if ctx.Buffer == nil {
		return
	}

	ctx.Buffer.AddAssistantMessage(messageID, msgType, props, blockID, threadID, ctx.AssistantID, metadata)
}

// BeginStep starts tracking a new execution step
// Returns the step for further updates
func (ctx *Context) BeginStep(stepType string, input map[string]interface{}) *BufferedStep {
	if ctx.Buffer == nil {
		return nil
	}

	// Update context memory snapshot before starting step (for recovery)
	if ctx.Memory != nil && ctx.Memory.Context != nil {
		ctx.Buffer.SetSpaceSnapshot(ctx.Memory.Context.Snapshot())
	}

	return ctx.Buffer.BeginStep(stepType, input, ctx.Stack)
}

// CompleteStep marks the current step as completed
func (ctx *Context) CompleteStep(output map[string]interface{}) {
	if ctx.Buffer == nil {
		return
	}
	ctx.Buffer.CompleteStep(output)
}

// FailCurrentStep marks the current step as failed or interrupted
func (ctx *Context) FailCurrentStep(status string, err error) {
	if ctx.Buffer == nil {
		return
	}
	ctx.Buffer.FailCurrentStep(status, err)
}

// shouldSkipHistory checks if history saving should be skipped
// Returns true if Skip.History is set in the current stack options
func (ctx *Context) shouldSkipHistory() bool {
	if ctx.Stack == nil || ctx.Stack.Options == nil || ctx.Stack.Options.Skip == nil {
		return false
	}
	return ctx.Stack.Options.Skip.History
}

// IsA2ACall returns true if this is any Agent-to-Agent call (delegate or fork)
// A2A calls are identified by Referer being "agent" or "agent_fork":
// - ctx.agent.Call/All/Any/Race uses RefererAgentFork (forked context, skips history)
// - delegate uses RefererAgent (same context flow, saves history)
func (ctx *Context) IsA2ACall() bool {
	return ctx.Referer == RefererAgent || ctx.Referer == RefererAgentFork
}

// IsForkedA2ACall returns true if this is a forked A2A call (ctx.agent.Call/All/Any/Race)
// Forked calls use RefererAgentFork, while delegate calls use RefererAgent.
// This is used to skip history saving for forked sub-agent calls,
// while allowing delegate calls to save history normally.
func (ctx *Context) IsForkedA2ACall() bool {
	return ctx.Referer == RefererAgentFork
}
