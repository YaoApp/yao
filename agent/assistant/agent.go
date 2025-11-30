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
func (ast *Assistant) Stream(ctx *context.Context, inputMessages []context.Message, handler ...message.StreamFunc) (interface{}, error) {

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

	// ================================================
	// Initialize
	// ================================================

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

	// ================================================
	// Get Full Messages with chat history
	// ================================================
	fullMessages, err := ast.WithHistory(ctx, inputMessages, agentNode)
	if err != nil {
		ast.traceAgentFail(agentNode, err)
		ast.sendStreamEndOnError(ctx, streamHandler, streamStartTime, err)
		return nil, err
	}

	// ================================================
	//  Execute Create Hook
	// ================================================
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

	// ================================================
	// Execute LLM Call Stream
	// ================================================
	// LLM Call Stream ( Optional )
	var completionResponse *context.CompletionResponse
	var completionMessages []context.Message
	var completionOptions *context.CompletionOptions
	if ast.Prompts != nil || ast.MCP != nil {
		// Build the LLM request first
		completionMessages, completionOptions, err = ast.BuildRequest(ctx, inputMessages, createResponse)
		if err != nil {
			ast.traceAgentFail(agentNode, err)
			// Send error stream_end for root stack
			ast.sendStreamEndOnError(ctx, streamHandler, streamStartTime, err)
			return nil, err
		}

		// Execute the LLM streaming call
		completionResponse, err = ast.executeLLMStream(ctx, completionMessages, completionOptions, agentNode, streamHandler)
		if err != nil {
			ast.traceAgentFail(agentNode, err)
			// Send error stream_end for root stack
			ast.sendStreamEndOnError(ctx, streamHandler, streamStartTime, err)
			return nil, err
		}
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

			// Execute all tool calls
			toolResults, hasErrors := ast.executeToolCalls(ctx, currentResponse.ToolCalls, attempt)

			// Convert toolResults to toolCallResponses
			toolCallResponses = make([]context.ToolCallResponse, len(toolResults))
			for i, result := range toolResults {
				parsedContent, _ := result.ParsedContent()
				toolCallResponses[i] = context.ToolCallResponse{
					ToolCallID: result.ToolCallID,
					Server:     result.Server(),
					Tool:       result.Tool(),
					Arguments:  nil,
					Result:     parsedContent,
					Error:      "",
				}
				if result.Error != nil {
					toolCallResponses[i].Error = result.Error.Error()
				}
			}

			// If all successful, break out
			if !hasErrors {
				log.Trace("[AGENT] All tool calls succeeded (attempt %d)", attempt)
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
				log.Error("[AGENT] %v", err)
				ast.traceAgentFail(agentNode, err)
				ast.sendStreamEndOnError(ctx, streamHandler, streamStartTime, err)
				return nil, err
			}

			// If it's the last attempt, return error
			if attempt == maxToolRetries-1 {
				err := fmt.Errorf("tool calls failed after %d attempts", maxToolRetries)
				log.Error("[AGENT] %v", err)
				ast.traceAgentFail(agentNode, err)
				ast.sendStreamEndOnError(ctx, streamHandler, streamStartTime, err)
				return nil, err
			}

			// Build retry messages with tool call results (including errors)
			retryMessages := ast.buildToolRetryMessages(currentMessages, currentResponse, toolResults)

			// Retry LLM call (streaming to keep user informed)
			log.Trace("[AGENT] Retrying LLM for tool call correction (attempt %d/%d)", attempt+1, maxToolRetries-1)
			currentResponse, err = ast.executeLLMForToolRetry(ctx, retryMessages, completionOptions, agentNode, streamHandler)
			if err != nil {
				log.Error("[AGENT] LLM retry failed: %v", err)
				ast.traceAgentFail(agentNode, err)
				ast.sendStreamEndOnError(ctx, streamHandler, streamStartTime, err)
				return nil, err
			}

			// If LLM didn't return tool calls, it might have given up
			if currentResponse.ToolCalls == nil {
				err := fmt.Errorf("LLM did not return tool calls in retry attempt %d", attempt+1)
				log.Error("[AGENT] %v", err)
				ast.traceAgentFail(agentNode, err)
				ast.sendStreamEndOnError(ctx, streamHandler, streamStartTime, err)
				return nil, err
			}

			// Update messages for next iteration
			currentMessages = retryMessages
		}

		// Update completionResponse with the final successful response
		completionResponse = currentResponse
	}

	// ================================================
	// Execute Next Hook and Process Response
	// ================================================
	var finalResponse interface{}
	var nextResponse *context.NextHookResponse = nil

	if ast.Script != nil {
		var err error
		nextResponse, err = ast.Script.Next(ctx, &context.NextHookPayload{
			Messages:   fullMessages,
			Completion: completionResponse,
			Tools:      toolCallResponses,
		})
		if err != nil {
			ast.traceAgentFail(agentNode, err)
			ast.sendStreamEndOnError(ctx, streamHandler, streamStartTime, err)
			return nil, err
		}

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
	// Handle based on interrupt type
	switch signal.Type {
	case context.InterruptForce:
		// Force interrupt: context is already cancelled in handleSignal
		// LLM streaming will detect ctx.Interrupt.Context().Done() and stop
		log.Trace("[AGENT] Force interrupt: stopping current operations immediately")

	case context.InterruptGraceful:
		log.Trace("[AGENT] Graceful interrupt: will process after current step completes")
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
