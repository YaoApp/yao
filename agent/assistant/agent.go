package assistant

import (
	"fmt"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/connector/openai"
	"github.com/yaoapp/yao/agent/assistant/handlers"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/i18n"
	"github.com/yaoapp/yao/agent/llm"
	"github.com/yaoapp/yao/agent/output/message"
)

// Stream stream the agent
// handler is optional, if not provided, a default handler will be used
func (ast *Assistant) Stream(ctx *context.Context, inputMessages []context.Message, options ...*context.Options) (*context.Response, error) {

	// Update logger with assistant ID and start logging
	ctx.Logger.SetAssistantID(ast.ID)
	ctx.Logger.Start()

	// Validate user permissions
	var err error
	err = ast.checkPermissions(ctx)
	if err != nil {
		return nil, err
	}

	// Start stream time
	streamStartTime := time.Now()

	// Set up interrupt handler if interrupt controller is available
	// InterruptController handles user interrupt signals (stop button) for appending messages
	// HTTP context cancellation is handled naturally by LLM/Agent layers
	if ctx.Interrupt != nil {
		ctx.Interrupt.SetHandler(func(c *context.Context, signal *context.InterruptSignal) error {
			return ast.handleInterrupt(c, signal)
		})
	}

	// ================================================
	// Initialize
	// ================================================
	ctx.Logger.Phase("Initialize")

	// Get or create options
	var opts *context.Options
	if len(options) > 0 && options[0] != nil {
		opts = options[0]
	} else {
		opts = &context.Options{}
	}

	// Initialize stack and auto-handle completion/failure/restore
	_, _, done := context.EnterStack(ctx, ast.ID, opts)
	defer done()

	// Auto-skip history for forked Agent-to-Agent calls (ctx.agent.Call/All/Any/Race)
	// This ensures forked A2A messages don't pollute chat history.
	// Delegate calls (RefererAgent) still save history as they are part of the main conversation flow.
	// Note: Output is NOT skipped - sub-agents output normally with ThreadID for UI separation.
	if ctx.IsForkedA2ACall() {
		if opts == nil {
			opts = &context.Options{}
		}
		opts.ForceA2A()
	}

	// ================================================
	// Initialize Chat Buffer (for root stack only)
	// Buffer is flushed in defer block at the end
	// ================================================
	ast.InitBuffer(ctx)

	// Track final status for buffer flush
	var finalStatus = context.StepStatusCompleted
	var finalError error

	// Defer buffer flush - always executes on exit (success, error, interrupt, panic)
	defer func() {
		// Handle panic recovery for status tracking
		if r := recover(); r != nil {
			finalStatus = context.ResumeStatusFailed
			if e, ok := r.(error); ok {
				finalError = e
			} else {
				finalError = fmt.Errorf("panic: %v", r)
			}
			ctx.Logger.Error("Panic recovered in Stream: %v", r)
			// Re-panic after flush to preserve original behavior
			defer panic(r)
		}

		// Flush buffer to database
		ast.FlushBuffer(ctx, finalStatus, finalError)

		// Log end of request
		ctx.Logger.End(finalStatus == context.StepStatusCompleted, finalError)
	}()

	// Determine stream handler
	streamHandler := ast.getStreamHandler(ctx, opts)

	// Get connector and capabilities early (before sending stream_start)
	// so that output adapters can use them when converting stream_start event
	err = ast.initializeCapabilities(ctx, opts)
	if err != nil {
		finalStatus = context.ResumeStatusFailed
		finalError = err
		ast.sendStreamEndOnError(ctx, streamHandler, streamStartTime, err)
		return nil, err
	}

	// Send ChunkStreamStart only for root stack (agent-level stream start)
	// Now ctx.Capabilities is set, so output adapters can use it
	ast.sendAgentStreamStart(ctx, streamHandler, streamStartTime)

	// Initialize chat, prepare kb collection (optional) etc.
	// Use async version to not block the main flow
	ast.InitializeConversationAsync(ctx, opts)

	ctx.Logger.PhaseComplete("Initialize")

	// Ensure chat session exists
	ast.EnsureChat(ctx)

	// Initialize agent trace node
	agentNode := ast.initAgentTraceNode(ctx, inputMessages)

	// ================================================
	// Get Full Messages with chat history
	// ================================================
	ctx.Logger.Phase("History")
	historyResult, err := ast.WithHistory(ctx, inputMessages, agentNode, opts)
	if err != nil {
		ast.traceAgentFail(agentNode, err)
		ast.sendStreamEndOnError(ctx, streamHandler, streamStartTime, err)
		return nil, err
	}
	fullMessages := historyResult.FullMessages

	// Buffer user input messages (use cleaned input without overlap)
	// Skip if History is disabled in options (for internal calls like needsearch)
	// Note: For A2A calls, ForceA2A() sets skip.history = true, so this will be skipped
	if opts == nil || opts.Skip == nil || !opts.Skip.History {
		ast.BufferUserInput(ctx, historyResult.InputMessages)
	}
	ctx.Logger.PhaseComplete("History")

	// ================================================
	//  Execute Create Hook
	// ================================================
	// Request Create hook ( Optional )
	var createResponse *context.HookCreateResponse
	if ast.HookScript != nil {
		ctx.Logger.HookStart("Create")
		// Begin step tracking for hook_create
		ast.BeginStep(ctx, context.StepTypeHookCreate, map[string]interface{}{
			"messages": fullMessages,
		})

		var err error
		createResponse, opts, err = ast.HookScript.Create(ctx, fullMessages, opts)
		if err != nil {
			finalStatus = context.ResumeStatusFailed
			finalError = err
			ast.traceAgentFail(agentNode, err)
			// Send error stream_end for root stack
			ast.sendStreamEndOnError(ctx, streamHandler, streamStartTime, err)
			return nil, err
		}

		// Complete step
		ast.CompleteStep(ctx, map[string]interface{}{
			"response": createResponse,
		})

		// Log the create response
		ast.traceCreateHook(agentNode, createResponse)
		ctx.Logger.HookComplete("Create")

		// Check if Create hook wants to delegate to another agent
		// This allows early routing to sub-agents without LLM call
		if createResponse != nil && createResponse.Delegate != nil {
			ctx.Logger.Debug("Create hook delegating to agent: %s", createResponse.Delegate.AgentID)

			// Delegate to target agent (reuse existing delegation logic from next.go)
			// Note: User input is already buffered by root agent, delegated agent will skip buffering
			delegateResponse, err := ast.handleDelegation(ctx, createResponse.Delegate, streamHandler)
			if err != nil {
				finalStatus = context.ResumeStatusFailed
				finalError = err
				ast.traceAgentFail(agentNode, err)
				ast.sendStreamEndOnError(ctx, streamHandler, streamStartTime, err)
				return nil, err
			}

			// For root stack, send stream_end and close output
			// (delegated agent handles its own stream events, but root needs to close)
			if ctx.Stack != nil && ctx.Stack.IsRoot() {
				ast.sendAgentStreamEnd(ctx, streamHandler, streamStartTime, "completed", nil, nil)
				if err := ctx.CloseOutput(); err != nil {
					if trace, _ := ctx.Trace(); trace != nil {
						trace.Error(i18n.Tr(ast.ID, ctx.Locale, "assistant.agent.stream.close_error"), map[string]any{"error": err.Error()})
					}
				}
			}

			// Return delegated response directly (skip LLM call and Next hook)
			return delegateResponse, nil
		}
	}

	// ================================================
	// Execute LLM Call Stream
	// ================================================
	// LLM Call Stream ( Optional )
	var completionResponse *context.CompletionResponse
	var completionMessages []context.Message
	var completionOptions *context.CompletionOptions
	if ast.Prompts != nil || ast.MCP != nil {
		ctx.Logger.Phase("LLM")

		// Build the LLM request first (use fullMessages which includes history)
		// Note: completionMessages here are still in original format (with __yao.attachment:// URLs)
		// Content conversion (BuildContent) happens inside executeLLMStream, right before LLM call
		// This ensures autoSearch and delegate receive original messages, not converted ones
		completionMessages, completionOptions, err = ast.BuildRequest(ctx, fullMessages, createResponse)
		if err != nil {
			finalStatus = context.ResumeStatusFailed
			finalError = err
			ast.traceAgentFail(agentNode, err)
			// Send error stream_end for root stack
			ast.sendStreamEndOnError(ctx, streamHandler, streamStartTime, err)
			return nil, err
		}

		// ================================================
		// Execute Auto Search (if enabled)
		// ================================================
		if intent := ast.shouldAutoSearch(ctx, completionMessages, createResponse, opts); intent != nil {
			refCtx := ast.executeAutoSearch(ctx, completionMessages, createResponse, intent, opts)
			if refCtx != nil && len(refCtx.References) > 0 {
				completionMessages = ast.injectSearchContext(completionMessages, refCtx)
			}
		}

		// Begin step tracking for LLM call
		ast.BeginStep(ctx, context.StepTypeLLM, map[string]interface{}{
			"messages": completionMessages,
		})

		// Execute the LLM streaming call
		completionResponse, err = ast.executeLLMStream(ctx, completionMessages, completionOptions, agentNode, streamHandler, opts)
		if err != nil {
			finalStatus = context.ResumeStatusFailed
			finalError = err
			ast.traceAgentFail(agentNode, err)
			// Send error stream_end for root stack
			ast.sendStreamEndOnError(ctx, streamHandler, streamStartTime, err)
			return nil, err
		}

		// Complete LLM step
		ast.CompleteStep(ctx, map[string]interface{}{
			"content":    completionResponse.Content,
			"tool_calls": completionResponse.ToolCalls,
		})

		hasToolCalls := completionResponse != nil && completionResponse.ToolCalls != nil && len(completionResponse.ToolCalls) > 0
		tokens := 0
		if completionResponse != nil && completionResponse.Usage != nil {
			tokens = completionResponse.Usage.TotalTokens
		}
		ctx.Logger.LLMComplete(tokens, hasToolCalls)
		ctx.Logger.PhaseComplete("LLM")
	}

	// ================================================
	// Execute tool calls with retry
	// ================================================
	var toolCallResponses []context.ToolCallResponse = nil
	if completionResponse != nil && completionResponse.ToolCalls != nil {

		maxToolRetries := 3
		currentMessages := completionMessages
		currentResponse := completionResponse

		for attempt := 0; attempt < maxToolRetries; attempt++ {

			// Begin step tracking for tool calls
			ast.BeginStep(ctx, context.StepTypeTool, map[string]interface{}{
				"tool_calls": currentResponse.ToolCalls,
				"attempt":    attempt,
			})

			// Execute all tool calls
			toolResults, hasErrors := ast.executeToolCalls(ctx, currentResponse.ToolCalls, attempt)

			// Build a map of tool call ID to arguments for quick lookup
			toolCallArgsMap := make(map[string]interface{})
			for _, tc := range currentResponse.ToolCalls {
				toolCallArgsMap[tc.ID] = tc.Function.Arguments
			}

			// Convert toolResults to toolCallResponses
			toolCallResponses = make([]context.ToolCallResponse, len(toolResults))
			for i, result := range toolResults {
				parsedContent, _ := result.ParsedContent()
				toolCallResponses[i] = context.ToolCallResponse{
					ToolCallID: result.ToolCallID,
					Server:     result.Server(),
					Tool:       result.Tool(),
					Arguments:  toolCallArgsMap[result.ToolCallID],
					Result:     parsedContent,
					Error:      "",
				}
				if result.Error != nil {
					toolCallResponses[i].Error = result.Error.Error()
				}
			}

			// If all successful, complete step and break out
			if !hasErrors {
				ast.CompleteStep(ctx, map[string]interface{}{
					"results": toolCallResponses,
				})
				ctx.Logger.Debug("All tool calls succeeded (attempt %d)", attempt)
				break
			}

			// Check if any errors are retryable (parameter/validation issues)
			hasRetryableErrors := false
			for _, result := range toolResults {
				if result.Error != nil && result.IsRetryableError {
					hasRetryableErrors = true
					break
				}
			}

			// If no retryable errors, don't retry (MCP internal issues)
			if !hasRetryableErrors {
				err := fmt.Errorf("tool calls failed with non-retryable errors (MCP internal issues)")
				finalStatus = context.ResumeStatusFailed
				finalError = err
				ctx.Logger.Error("Tool calls failed: %v", err)
				ast.traceAgentFail(agentNode, err)
				ast.sendStreamEndOnError(ctx, streamHandler, streamStartTime, err)
				return nil, err
			}

			// If it's the last attempt, return error
			if attempt == maxToolRetries-1 {
				err := fmt.Errorf("tool calls failed after %d attempts", maxToolRetries)
				finalStatus = context.ResumeStatusFailed
				finalError = err
				ctx.Logger.Error("Tool calls failed: %v", err)
				ast.traceAgentFail(agentNode, err)
				ast.sendStreamEndOnError(ctx, streamHandler, streamStartTime, err)
				return nil, err
			}

			// Complete current step (with partial results)
			ast.CompleteStep(ctx, map[string]interface{}{
				"results":    toolCallResponses,
				"has_errors": true,
			})

			// Build retry messages with tool call results (including errors)
			retryMessages := ast.buildToolRetryMessages(currentMessages, currentResponse, toolResults)

			// Begin LLM retry step
			ast.BeginStep(ctx, context.StepTypeLLM, map[string]interface{}{
				"messages":      retryMessages,
				"retry_attempt": attempt + 1,
			})

			// Retry LLM call (streaming to keep user informed)
			ctx.Logger.Debug("Retrying LLM for tool call correction (attempt %d/%d)", attempt+1, maxToolRetries-1)
			currentResponse, err = ast.executeLLMForToolRetry(ctx, retryMessages, completionOptions, agentNode, streamHandler, opts)
			if err != nil {
				finalStatus = context.ResumeStatusFailed
				finalError = err
				ctx.Logger.Error("LLM retry failed: %v", err)
				ast.traceAgentFail(agentNode, err)
				ast.sendStreamEndOnError(ctx, streamHandler, streamStartTime, err)
				return nil, err
			}

			// If LLM didn't return tool calls, it might have given up
			if currentResponse.ToolCalls == nil {
				err := fmt.Errorf("LLM did not return tool calls in retry attempt %d", attempt+1)
				finalStatus = context.ResumeStatusFailed
				finalError = err
				ctx.Logger.Error("LLM did not return tool calls: %v", err)
				ast.traceAgentFail(agentNode, err)
				ast.sendStreamEndOnError(ctx, streamHandler, streamStartTime, err)
				return nil, err
			}

			// Complete LLM retry step
			ast.CompleteStep(ctx, map[string]interface{}{
				"content":    currentResponse.Content,
				"tool_calls": currentResponse.ToolCalls,
			})

			// Update messages for next iteration
			currentMessages = retryMessages
		}

		// Update completionResponse with the final successful response
		completionResponse = currentResponse
	}

	// ================================================
	// Execute Next Hook and Process Response
	// ================================================
	var finalResponse *context.Response
	var nextResponse *context.NextHookResponse = nil

	if ast.HookScript != nil {
		ctx.Logger.HookStart("Next")

		// Begin step tracking for hook_next
		ast.BeginStep(ctx, context.StepTypeHookNext, map[string]interface{}{
			"messages":   fullMessages,
			"completion": completionResponse,
			"tools":      toolCallResponses,
		})

		var err error
		nextResponse, opts, err = ast.HookScript.Next(ctx, &context.NextHookPayload{
			Messages:   fullMessages,
			Completion: completionResponse,
			Tools:      toolCallResponses,
		}, opts)
		if err != nil {
			finalStatus = context.ResumeStatusFailed
			finalError = err
			ast.traceAgentFail(agentNode, err)
			ast.sendStreamEndOnError(ctx, streamHandler, streamStartTime, err)
			return nil, err
		}

		// Complete hook_next step
		ast.CompleteStep(ctx, map[string]interface{}{
			"response": nextResponse,
		})

		ctx.Logger.HookComplete("Next")

		// Process Next hook response
		finalResponse, err = ast.processNextResponse(&NextProcessContext{
			Context:            ctx,
			NextResponse:       nextResponse,
			CompletionResponse: completionResponse,
			FullMessages:       fullMessages,
			ToolCallResponses:  toolCallResponses,
			StreamHandler:      streamHandler,
			CreateResponse:     createResponse,
		})
		if err != nil {
			finalStatus = context.ResumeStatusFailed
			finalError = err
			ast.traceAgentFail(agentNode, err)
			ast.sendStreamEndOnError(ctx, streamHandler, streamStartTime, err)
			return nil, err
		}
	} else {
		// No Next hook: use standard response
		finalResponse = ast.buildStandardResponse(&NextProcessContext{
			Context:            ctx,
			NextResponse:       nil,
			CompletionResponse: completionResponse,
			FullMessages:       fullMessages,
			ToolCallResponses:  toolCallResponses,
			StreamHandler:      streamHandler,
			CreateResponse:     createResponse,
		})
	}

	// Create completion node to report final output
	ast.traceAgentCompletion(ctx, createResponse, nextResponse, completionResponse, finalResponse)

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

	// Return finalResponse which could be:
	// 1. Result from delegated agent call (already a Response)
	// 2. Custom data from Next hook (wrapped in standard Response)
	// 3. Standard response
	return finalResponse, nil
}

// GetConnector get the connector object, capabilities, and error with priority: opts.Connector > ast.Connector
// Note: opts.Connector may be set by Create hook's applyOptionsAdjustments
// Returns: (connector, capabilities, error)
func (ast *Assistant) GetConnector(ctx *context.Context, opts ...*context.Options) (connector.Connector, *openai.Capabilities, error) {
	// Determine connector ID with priority: opts.Connector > ast.Connector
	connectorID := ast.Connector
	if len(opts) > 0 && opts[0] != nil && opts[0].Connector != "" {
		connectorID = opts[0].Connector
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
	// Uses unified capability getter: 1. User-defined models.yml, 2. connector's Setting()["capabilities"], 3. default
	capabilities := llm.GetCapabilitiesFromConn(conn, modelCapabilities)

	return conn, capabilities, nil
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

// getStreamHandler returns the stream handler from options or a default one
func (ast *Assistant) getStreamHandler(ctx *context.Context, opts ...*context.Options) message.StreamFunc {
	// Check if handler is provided in options
	if len(opts) > 0 && opts[0] != nil && opts[0].Writer != nil {
		return handlers.DefaultStreamHandler(ctx)
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
		ctx.Logger.Debug("Context cancelled, skipping sendAgentStreamEnd handler call")
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
	// Handle based on interrupt type
	switch signal.Type {
	case context.InterruptForce:
		// Force interrupt: context is already cancelled in handleSignal
		// LLM streaming will detect ctx.Interrupt.Context().Done() and stop
		ctx.Logger.Debug("Force interrupt: stopping current operations immediately")

	case context.InterruptGraceful:
		ctx.Logger.Debug("Graceful interrupt: will process after current step completes")
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
func (ast *Assistant) initializeCapabilities(ctx *context.Context, opts *context.Options) error {
	if ast.Prompts == nil && ast.MCP == nil {
		return nil
	}

	_, capabilities, err := ast.GetConnector(ctx, opts)
	if err != nil {
		return err
	}

	// Set capabilities in context for output adapters to use
	if capabilities != nil {
		ctx.Capabilities = capabilities
	}

	return nil
}

// buildToolRetryMessages builds messages for LLM retry with tool call results
// Format follows OpenAI's tool call response pattern:
// 1. Assistant message with tool calls
// 2. Tool messages with results (one per tool call)
// 3. System message explaining the retry
func (ast *Assistant) buildToolRetryMessages(
	previousMessages []context.Message,
	completionResponse *context.CompletionResponse,
	toolResults []ToolCallResult,
) []context.Message {
	retryMessages := make([]context.Message, 0, len(previousMessages)+len(toolResults)+2)

	// Add all previous messages
	retryMessages = append(retryMessages, previousMessages...)

	// Add assistant message with tool calls
	assistantMsg := context.Message{
		Role:      context.RoleAssistant,
		Content:   completionResponse.Content,
		ToolCalls: completionResponse.ToolCalls,
	}
	retryMessages = append(retryMessages, assistantMsg)

	// Add tool result messages (one per tool call)
	for _, result := range toolResults {
		toolMsg := context.Message{
			Role:       context.RoleTool,
			Content:    result.Content,
			ToolCallID: &result.ToolCallID,
		}
		// Add tool name if available
		if result.Name != "" {
			name := result.Name
			toolMsg.Name = &name
		}
		retryMessages = append(retryMessages, toolMsg)
	}

	// Add system message explaining the retry (optional, helps LLM understand context)
	systemMsg := context.Message{
		Role:    context.RoleSystem,
		Content: i18n.Tr(ast.ID, "en", "assistant.agent.tool_retry_prompt"),
	}
	retryMessages = append(retryMessages, systemMsg)

	return retryMessages
}
