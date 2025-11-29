package assistant

import (
	"fmt"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/agent/assistant/handlers"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/i18n"
	"github.com/yaoapp/yao/agent/output/message"
)

// Stream stream the agent
// handler is optional, if not provided, a default handler will be used
func (ast *Assistant) Stream(ctx *context.Context, inputMessages []context.Message, handler ...message.StreamFunc) (*context.Response, error) {

	log.Trace("[AGENT] Stream started: assistant=%s, contextID=%s", ast.ID, ctx.ID)
	defer log.Trace("[AGENT] Stream ended: assistant=%s, contextID=%s", ast.ID, ctx.ID)

	var err error
	streamStartTime := time.Now()

	// Set up interrupt handler if interrupt controller is available
	// InterruptController handles user interrupt signals (stop button) for appending messages
	// HTTP context cancellation is handled naturally by LLM/Agent layers
	if ctx.Interrupt != nil {
		ctx.Interrupt.SetHandler(func(c *context.Context, signal *context.InterruptSignal) error {
			return ast.handleInterrupt(c, signal)
		})
	}

	// Initialize stack and auto-handle completion/failure/restore
	_, _, done := context.EnterStack(ctx, ast.ID, ctx.Referer)
	defer done()

	// Determine stream handler
	streamHandler := ast.getStreamHandler(ctx, handler...)

	// Get connector and capabilities early (before sending stream_start)
	// so that output adapters can use them when converting stream_start event
	err = ast.initializeCapabilities(ctx)
	if err != nil {
		ast.sendStreamEndOnError(ctx, streamHandler, streamStartTime, err)
		return nil, err
	}

	// Send ChunkStreamStart only for root stack (agent-level stream start)
	// Now ctx.Capabilities is set, so output adapters can use it
	ast.sendAgentStreamStart(ctx, streamHandler, streamStartTime)

	// Initialize agent trace node
	agentNode := ast.initAgentTraceNode(ctx, inputMessages)

	// Full input messages with chat history
	fullMessages, err := ast.WithHistory(ctx, inputMessages)
	if err != nil {
		ast.traceAgentFail(agentNode, err)
		// Send error stream_end for root stack
		ast.sendStreamEndOnError(ctx, streamHandler, streamStartTime, err)
		return nil, err
	}

	// Log the chat history
	ast.traceAgentHistory(ctx, agentNode, fullMessages)

	// Request Create hook ( Optional )
	var createResponse *context.HookCreateResponse
	if ast.Script != nil {
		var err error
		createResponse, err = ast.Script.Create(ctx, fullMessages)
		if err != nil {
			ast.traceAgentFail(agentNode, err)
			// Send error stream_end for root stack
			ast.sendStreamEndOnError(ctx, streamHandler, streamStartTime, err)
			return nil, err
		}

		// Log the create response
		ast.traceCreateHook(agentNode, createResponse)
	}

	// LLM Call Stream ( Optional )
	var completionResponse *context.CompletionResponse
	if ast.Prompts != nil || ast.MCP != nil {
		// Build the LLM request first
		completionMessages, completionOptions, err := ast.BuildRequest(ctx, inputMessages, createResponse)
		if err != nil {
			ast.traceAgentFail(agentNode, err)
			// Send error stream_end for root stack
			ast.sendStreamEndOnError(ctx, streamHandler, streamStartTime, err)
			return nil, err
		}

		// Execute the LLM streaming call
		completionResponse, err = ast.executeLLMStream(ctx, completionMessages, completionOptions, agentNode, streamHandler)
		if err != nil {
			// Send error stream_end for root stack
			ast.sendStreamEndOnError(ctx, streamHandler, streamStartTime, err)
			return nil, err
		}
	}

	// Request MCP hook ( Optional )
	var mcpResponse *context.ResponseHookMCP
	if ast.MCP != nil {
		_ = mcpResponse // mcpResponse is available for further processing

		// MCP Execution Loop
	}

	// Request Done hook ( Optional )
	var doneResponse *context.ResponseHookDone
	if ast.Script != nil {
		var err error
		doneResponse, err = ast.Script.Done(ctx, fullMessages, completionResponse, mcpResponse)
		if err != nil {
			// Send error stream_end for root stack
			ast.sendStreamEndOnError(ctx, streamHandler, streamStartTime, err)
			return nil, err
		}
	}

	_ = doneResponse // doneResponse is available for further processing

	// Set the output of the agent node
	ast.traceAgentOutput(agentNode, createResponse, doneResponse, completionResponse)

	// Only close output and send stream_end if this is the root call (entry point)
	// Nested calls (from MCP, hooks, etc.) should not close the output or send stream_end
	// Note: Flush is already handled by the stream handler (handleStreamEnd)
	if ctx.Stack != nil && ctx.Stack.IsRoot() {
		// Log closing output for root call
		if trace, _ := ctx.Trace(); trace != nil {
			trace.Debug("Agent: Closing output (root call)", map[string]any{
				"stack_id":     ctx.Stack.ID,
				"depth":        ctx.Stack.Depth,
				"assistant_id": ctx.Stack.AssistantID,
			})
		}

		// Send ChunkStreamEnd (agent-level stream completion)
		ast.sendAgentStreamEnd(ctx, streamHandler, streamStartTime, "completed", nil, completionResponse)

		// Close the output writer to send [DONE] marker
		if err := ctx.CloseOutput(); err != nil {
			if trace, _ := ctx.Trace(); trace != nil {
				trace.Error(i18n.Tr(ast.ID, ctx.Locale, "assistant.agent.stream.close_error"), map[string]any{"error": err.Error()}) // "Failed to close output"
			}
		}
	} else {
		// Log skipping close for nested call
		if trace, _ := ctx.Trace(); trace != nil && ctx.Stack != nil {
			trace.Debug("Agent: Skipping output close (nested call)", map[string]any{
				"stack_id":     ctx.Stack.ID,
				"depth":        ctx.Stack.Depth,
				"parent_id":    ctx.Stack.ParentID,
				"assistant_id": ctx.Stack.AssistantID,
			})
		}
	}

	return &context.Response{
		ContextID:   ctx.ID,
		RequestID:   ctx.RequestID(),
		ChatID:      ctx.ChatID,
		AssistantID: ast.ID,
		Create:      createResponse, Done: doneResponse, Completion: completionResponse}, nil
}

// GetConnector get the connector object, capabilities, and error with priority: createResponse > ctx > ast
// Note: createResponse.Connector is already applied to ctx.Connector by applyContextAdjustments in create.go
// Returns: (connector, capabilities, error)
func (ast *Assistant) GetConnector(ctx *context.Context) (connector.Connector, *context.ModelCapabilities, error) {
	// Determine connector ID with priority
	connectorID := ast.Connector
	if ctx.Connector != "" {
		connectorID = ctx.Connector
	}

	// If empty, return error
	if connectorID == "" {
		return nil, nil, fmt.Errorf("connector not specified")
	}

	// Load gou connector
	conn, err := connector.Select(connectorID)
	if err != nil {
		return nil, nil, err
	}

	// Get connector capabilities from settings
	capabilities := ast.getConnectorCapabilities(connectorID)

	return conn, capabilities, nil
}

// getConnectorCapabilities get the capabilities of a connector from settings
func (ast *Assistant) getConnectorCapabilities(connectorID string) *context.ModelCapabilities {
	// Initialize with default capabilities (all disabled)
	falseVal := false
	capabilities := &context.ModelCapabilities{
		Vision:    falseVal,
		ToolCalls: &falseVal,
		Audio:     &falseVal,
		Reasoning: &falseVal,
		Streaming: &falseVal,
	}

	// Get model capabilities from global configuration
	modelCaps, exists := modelCapabilities[connectorID]
	if !exists {
		// Return default capabilities if model not found in configuration
		return capabilities
	}

	// Update capabilities based on model configuration
	// Vision can be bool or string (VisionFormat)
	if modelCaps.Vision != nil {
		capabilities.Vision = modelCaps.Vision
	}

	// Handle both Tools (deprecated) and ToolCalls
	if modelCaps.ToolCalls || modelCaps.Tools {
		v := true
		capabilities.ToolCalls = &v
	}

	if modelCaps.Audio {
		v := true
		capabilities.Audio = &v
	}

	if modelCaps.Reasoning {
		v := true
		capabilities.Reasoning = &v
	}

	if modelCaps.Streaming {
		v := true
		capabilities.Streaming = &v
	}

	if modelCaps.JSON {
		v := true
		capabilities.JSON = &v
	}

	if modelCaps.Multimodal {
		v := true
		capabilities.Multimodal = &v
	}

	return capabilities
}

// Info get the assistant information
func (ast *Assistant) Info(locale ...string) *message.AssistantInfo {
	lc := "en"
	if len(locale) > 0 {
		lc = locale[0]
	}
	return &message.AssistantInfo{
		ID:          ast.ID,
		Type:        ast.Type,
		Name:        i18n.Tr(ast.ID, lc, ast.Name),
		Avatar:      ast.Avatar,
		Description: i18n.Tr(ast.ID, lc, ast.Description),
	}
}

// WithHistory with the history messages
func (ast *Assistant) WithHistory(ctx *context.Context, messages []context.Message) ([]context.Message, error) {
	return messages, nil
}

// getStreamHandler returns the stream handler from the provided handlers or a default one
func (ast *Assistant) getStreamHandler(ctx *context.Context, handler ...message.StreamFunc) message.StreamFunc {
	if len(handler) > 0 && handler[0] != nil {
		return handler[0]
	}
	return handlers.DefaultStreamHandler(ctx)
}

// sendAgentStreamStart sends ChunkStreamStart for root stack only (agent-level stream start)
// This ensures only one stream_start per agent execution, even with multiple LLM calls
func (ast *Assistant) sendAgentStreamStart(ctx *context.Context, handler message.StreamFunc, startTime time.Time) {
	if ctx.Stack == nil || !ctx.Stack.IsRoot() || handler == nil {
		return
	}

	// Build the start data
	startData := message.EventStreamStartData{
		ContextID: ctx.ID,
		ChatID:    ctx.ChatID,
		TraceID:   ctx.TraceID(),
		RequestID: ctx.RequestID(),
		Timestamp: startTime.UnixMilli(),
		Assistant: ast.Info(ctx.Locale),
		Metadata:  ctx.Metadata,
	}

	if startJSON, err := jsoniter.Marshal(startData); err == nil {
		handler(message.ChunkStreamStart, startJSON)
	}
}

// sendAgentStreamEnd sends ChunkStreamEnd for root stack only (agent-level stream completion)
func (ast *Assistant) sendAgentStreamEnd(ctx *context.Context, handler message.StreamFunc, startTime time.Time, status string, err error, response *context.CompletionResponse) {
	if ctx.Stack == nil || !ctx.Stack.IsRoot() || handler == nil {
		return
	}

	// Check if context is cancelled - if so, skip handler call to avoid blocking
	if ctx.Context != nil && ctx.Context.Err() != nil {
		log.Trace("[AGENT] Context cancelled, skipping sendAgentStreamEnd handler call")
		return
	}

	endData := &message.EventStreamEndData{
		RequestID:  ctx.RequestID(),
		ContextID:  ctx.ID,
		Timestamp:  time.Now().UnixMilli(),
		DurationMs: time.Since(startTime).Milliseconds(),
		Status:     status,
		TraceID:    ctx.TraceID(),
		Metadata:   ctx.Metadata,
	}

	if err != nil {
		endData.Error = err.Error()
	}

	if response != nil && response.Usage != nil {
		endData.Usage = response.Usage
	}

	if endJSON, marshalErr := jsoniter.Marshal(endData); marshalErr == nil {
		handler(message.ChunkStreamEnd, endJSON)
	}
}

// sendStreamEndOnError sends ChunkStreamEnd with error status for root stack only
func (ast *Assistant) sendStreamEndOnError(ctx *context.Context, handler message.StreamFunc, startTime time.Time, err error) {
	ast.sendAgentStreamEnd(ctx, handler, startTime, "error", err, nil)
}

// handleInterrupt handles the interrupt signal
// This is called by the interrupt listener when a signal is received
func (ast *Assistant) handleInterrupt(ctx *context.Context, signal *context.InterruptSignal) error {
	fmt.Printf("=== Interrupt Received ===\n")
	fmt.Printf("Assistant: %s\n", ast.ID)
	fmt.Printf("Type: %s\n", signal.Type)
	fmt.Printf("Messages: %d\n", len(signal.Messages))
	fmt.Printf("Timestamp: %d\n", signal.Timestamp)

	// Handle based on interrupt type
	switch signal.Type {
	case context.InterruptForce:
		fmt.Println("Force interrupt: stopping current operations immediately...")
		// Force interrupt: context is already cancelled in handleSignal
		// LLM streaming will detect ctx.Interrupt.Context().Done() and stop

	case context.InterruptGraceful:
		fmt.Println("Graceful interrupt: will process after current step completes...")
		// Graceful interrupt: let current operation complete
		// The signal is stored in current/pending, can be checked at checkpoints
	}

	// TODO: Implement actual interrupt handling logic:
	// 1. For graceful: wait for current step, then merge messages and restart
	// 2. For force: immediately stop and restart with new messages
	// 3. Call Interrupted Hook if configured
	// 4. Decide whether to continue, restart, or abort based on Hook response

	return nil
}

// initializeCapabilities gets connector and capabilities, then sets them in context
// This should be called early (before sending stream_start) so that output adapters
// can use capabilities when converting stream_start event
func (ast *Assistant) initializeCapabilities(ctx *context.Context) error {
	if ast.Prompts == nil && ast.MCP == nil {
		return nil
	}

	_, capabilities, err := ast.GetConnector(ctx)
	if err != nil {
		return err
	}

	// Set capabilities in context for output adapters to use
	if capabilities != nil {
		ctx.Capabilities = capabilities
	}

	return nil
}
