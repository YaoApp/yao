package openai

import (
	gocontext "context"
	"fmt"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/connector"
	gouOpenAI "github.com/yaoapp/gou/connector/openai"
	"github.com/yaoapp/gou/http"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/i18n"
	"github.com/yaoapp/yao/agent/llm/adapters"
	"github.com/yaoapp/yao/agent/llm/providers/base"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/utils/jsonschema"
)

// startMessage starts a new message and sends group_start event
// Note: group_start/group_end events are used for backward compatibility
// but at LLM level they represent message boundaries, not Agent-level blocks
func (mt *messageTracker) startMessage(messageType message.StreamChunkType, handler message.StreamFunc) {
	if mt.active {
		// End previous message first
		mt.endMessage(handler)
	}

	mt.active = true
	// Generate message ID using context's ID generator
	if mt.idGenerator != nil {
		mt.messageID = mt.idGenerator.GenerateMessageID() // M1, M2, M3...
	} else {
		// Fallback to global generator if no context generator
		mt.messageID = message.GenerateNanoID()
	}
	mt.messageType = messageType
	mt.startTime = time.Now().UnixMilli()
	mt.chunkCount = 0
	mt.toolCallInfo = nil

	if handler != nil {
		startData := &message.EventMessageStartData{
			MessageID: mt.messageID,
			Type:      string(messageType),
			Timestamp: mt.startTime,
		}
		if startJSON, err := jsoniter.Marshal(startData); err == nil {
			handler(message.ChunkMessageStart, startJSON)
		}
	}
}

// startToolCallMessage starts a new tool call message with tool call info
func (mt *messageTracker) startToolCallMessage(toolCallInfo *message.EventToolCallInfo, handler message.StreamFunc) {
	if mt.active {
		mt.endMessage(handler)
	}

	mt.active = true
	// Generate message ID using context's ID generator
	if mt.idGenerator != nil {
		mt.messageID = mt.idGenerator.GenerateMessageID() // M1, M2, M3...
	} else {
		// Fallback to global generator if no context generator
		mt.messageID = message.GenerateNanoID()
	}
	mt.messageType = message.ChunkToolCall
	mt.startTime = time.Now().UnixMilli()
	mt.chunkCount = 0
	mt.toolCallInfo = toolCallInfo

	if handler != nil {
		startData := &message.EventMessageStartData{
			MessageID: mt.messageID,
			Type:      string(message.ChunkToolCall),
			Timestamp: mt.startTime,
			ToolCall:  toolCallInfo,
		}
		if startJSON, err := jsoniter.Marshal(startData); err == nil {
			handler(message.ChunkMessageStart, startJSON)
		}
	}
}

// incrementChunk increments the chunk count for the current message
func (mt *messageTracker) incrementChunk() {
	if mt.active {
		mt.chunkCount++
	}
}

// endMessage ends the current message and sends group_end event
// Note: group_end event is used for backward compatibility
// but at LLM level it represents message completion, not Agent-level block
func (mt *messageTracker) endMessage(handler message.StreamFunc) {
	if !mt.active {
		return
	}

	if handler != nil {
		endData := &message.EventMessageEndData{
			MessageID:  mt.messageID,
			Type:       string(mt.messageType),
			Timestamp:  time.Now().UnixMilli(),
			DurationMs: time.Now().UnixMilli() - mt.startTime,
			ChunkCount: mt.chunkCount,
			Status:     "completed",
		}
		if mt.toolCallInfo != nil {
			endData.ToolCall = mt.toolCallInfo
		}
		if endJSON, err := jsoniter.Marshal(endData); err == nil {
			handler(message.ChunkMessageEnd, endJSON)
		}
	}

	mt.active = false
	mt.messageID = ""
	mt.toolCallInfo = nil
}

// Provider OpenAI-compatible provider with capability adapters
// Supports: vision, tool calls, streaming, JSON mode, reasoning
type Provider struct {
	*base.Provider
	adapters []adapters.CapabilityAdapter
}

// buildAPIURL builds the complete API URL from host and endpoint
// If host ends with /, it's used as-is (user has specified full path)
// Otherwise, /v1 prefix is added automatically (standard for OpenAI-compatible APIs)
func buildAPIURL(host, endpoint string) string {
	// If host ends with /, use it as-is (user has specified full path like /v1/ or /api/)
	// Otherwise, add /v1 prefix (standard for OpenAI-compatible APIs)
	if !strings.HasSuffix(host, "/") {
		endpoint = "/v1" + endpoint
	}
	host = strings.TrimSuffix(host, "/")
	return host + endpoint
}

// New create a new OpenAI provider with capability adapters
func New(conn connector.Connector, capabilities *gouOpenAI.Capabilities) *Provider {
	return &Provider{
		Provider: base.NewProvider(conn, capabilities),
		adapters: buildAdapters(capabilities),
	}
}

// buildAdapters builds capability adapters based on model capabilities
func buildAdapters(cap *gouOpenAI.Capabilities) []adapters.CapabilityAdapter {
	if cap == nil {
		return []adapters.CapabilityAdapter{}
	}

	result := make([]adapters.CapabilityAdapter, 0)

	// Tool call adapter
	result = append(result, adapters.NewToolCallAdapter(cap.ToolCalls))

	// Vision adapter
	visionSupport, visionFormat := context.GetVisionSupport(cap)
	if visionSupport {
		result = append(result, adapters.NewVisionAdapter(true, visionFormat))
	} else if cap.Vision != nil {
		// Vision explicitly disabled, add adapter to remove image content
		result = append(result, adapters.NewVisionAdapter(false, context.VisionFormatNone))
	}

	// Audio adapter
	result = append(result, adapters.NewAudioAdapter(cap.Audio))

	// Reasoning adapter (always add to handle reasoning_effort and temperature parameters)
	// Even if the model doesn't support reasoning, we need the adapter to strip reasoning_effort
	if cap.Reasoning {
		// Detect reasoning format based on capabilities
		format := detectReasoningFormat(cap)
		result = append(result, adapters.NewReasoningAdapter(format, cap))
	} else {
		// Model doesn't support reasoning, use None format to strip reasoning parameters
		result = append(result, adapters.NewReasoningAdapter(adapters.ReasoningFormatNone, cap))
	}

	return result
}

// detectReasoningFormat detects the reasoning format based on capabilities
func detectReasoningFormat(cap *gouOpenAI.Capabilities) adapters.ReasoningFormat {
	// TODO: Implement better detection logic
	// For now, default to OpenAI o1 format if reasoning is supported
	if cap.Reasoning {
		return adapters.ReasoningFormatOpenAI
	}
	return adapters.ReasoningFormatNone
}

// Stream stream completion from OpenAI API
func (p *Provider) Stream(ctx *context.Context, messages []context.Message, options *context.CompletionOptions, handler message.StreamFunc) (*context.CompletionResponse, error) {
	// Add debug log
	trace, _ := ctx.Trace()
	if trace != nil {
		trace.Debug("OpenAI Stream: Starting stream request", map[string]any{
			"message_count": len(messages),
		})
	}

	maxRetries := 3
	var lastErr error

	// Get Go context for cancellation support
	// Read from Stack.Options if available (call-level override)
	goCtx := ctx.Context
	if ctx.Stack != nil && ctx.Stack.Options != nil && ctx.Stack.Options.Context != nil {
		goCtx = ctx.Stack.Options.Context
	}
	if goCtx == nil {
		goCtx = gocontext.Background()
	}

	// Make a copy of messages to avoid modifying the original
	currentMessages := make([]context.Message, len(messages))
	copy(currentMessages, messages)

	// Outer loop: handle network/API errors with exponential backoff
	for attempt := 0; attempt < maxRetries; attempt++ {
		// Check if context is cancelled before retry
		select {
		case <-goCtx.Done():
			return nil, fmt.Errorf("context cancelled: %w", goCtx.Err())
		default:
		}

		// Check for force interrupt before retry
		if ctx.Interrupt != nil {
			if signal := ctx.Interrupt.Peek(); signal != nil && signal.Type == context.InterruptForce {
				return nil, fmt.Errorf("force interrupted by user")
			}
		}

		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second

			// Add debug log to trace
			if trace != nil {
				trace.Warn("OpenAI stream request failed, retrying", map[string]any{
					"backoff":     backoff.String(),
					"attempt":     attempt + 1,
					"max_retries": maxRetries,
					"error":       lastErr.Error(),
				})
			}

			// Sleep with context cancellation support
			timer := time.NewTimer(backoff)
			interruptTicker := time.NewTicker(100 * time.Millisecond) // Check interrupt every 100ms
			defer interruptTicker.Stop()

		backoffLoop:
			for {
				select {
				case <-timer.C:
					// Backoff completed, continue to retry
					break backoffLoop
				case <-goCtx.Done():
					timer.Stop()
					return nil, fmt.Errorf("context cancelled during backoff: %w", goCtx.Err())
				case <-interruptTicker.C:
					// Check for force interrupt during backoff
					if ctx.Interrupt != nil {
						if signal := ctx.Interrupt.Peek(); signal != nil && signal.Type == context.InterruptForce {
							timer.Stop()
							return nil, fmt.Errorf("force interrupted by user during backoff")
						}
					}
				}
			}
		}

		response, err := p.streamWithRetry(ctx, currentMessages, options, handler)
		log.Trace("[LLM] streamWithRetry returned: err=%v", err)
		if err == nil {
			if trace != nil && goCtx.Err() == nil {
				trace.Debug("OpenAI Stream: Request completed successfully")
			}
			return response, nil
		}
		lastErr = err
		log.Trace("[LLM] Checking context after error: goCtx.Err()=%v", goCtx.Err())

		// Check for context cancellation before logging (trace calls may block if context is cancelled)
		if goCtx.Err() != nil {
			log.Trace("[LLM] Context cancelled in retry loop, returning")
			return nil, fmt.Errorf("context cancelled: %w", goCtx.Err())
		}

		if trace != nil {
			trace.Debug("OpenAI Stream: Request failed", map[string]any{
				"error": err.Error(),
			})
		}

		// Note: Tool call validation errors should not reach here anymore
		// because we now pass through validation failures to Agent layer
		// This check is kept for safety but should not trigger
		if isToolCallValidationError(err) {
			if trace != nil {
				trace.Debug("Tool call validation error (unexpected, should be handled differently)", map[string]any{
					"error": err.Error(),
				})
			}
		}

		// Check if error is retryable (network errors, rate limits, etc.)
		if !isRetryableError(err) {
			return nil, fmt.Errorf("non-retryable error: %w", err)
		}
	}

	return nil, fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

// streamWithRetry performs a single streaming request attempt
func (p *Provider) streamWithRetry(ctx *context.Context, messages []context.Message, options *context.CompletionOptions, handler message.StreamFunc) (*context.CompletionResponse, error) {
	streamStartTime := time.Now()
	requestID := fmt.Sprintf("req_%d", streamStartTime.UnixNano())

	// Add debug log
	trace, _ := ctx.Trace()
	if trace != nil {
		trace.Debug("OpenAI Stream: streamWithRetry starting", map[string]any{
			"request_id": requestID,
		})
	}

	// Get Go context for cancellation support
	goCtx := ctx.Context
	if goCtx == nil {
		goCtx = gocontext.Background()
	}

	// Check if context is already cancelled
	select {
	case <-goCtx.Done():
		return nil, fmt.Errorf("context cancelled before stream start: %w", goCtx.Err())
	default:
	}

	// Check for force interrupt before stream start
	if ctx.Interrupt != nil {
		if signal := ctx.Interrupt.Peek(); signal != nil && signal.Type == context.InterruptForce {
			return nil, fmt.Errorf("force interrupted by user before stream start")
		}
	}

	// Note: ChunkStreamStart/End are now sent at Agent level, not LLM level
	// This is because an agent may make multiple LLM calls in one stream

	// Preprocess messages and options through adapters
	processedMessages := messages
	processedOptions := options
	for _, adapter := range p.adapters {
		// Preprocess messages
		newMessages, err := adapter.PreprocessMessages(processedMessages)
		if err != nil {
			if handler != nil {
				handler(message.ChunkError, []byte(fmt.Sprintf("adapter %s message preprocessing failed: %v", adapter.Name(), err)))
			}
			return nil, fmt.Errorf("adapter %s message preprocessing failed: %w", adapter.Name(), err)
		}
		processedMessages = newMessages

		// Preprocess options
		newOpts, err := adapter.PreprocessOptions(processedOptions)
		if err != nil {
			// Send error to handler
			if handler != nil {
				handler(message.ChunkError, []byte(fmt.Sprintf("adapter %s option preprocessing failed: %v", adapter.Name(), err)))
			}
			return nil, fmt.Errorf("adapter %s option preprocessing failed: %w", adapter.Name(), err)
		}
		processedOptions = newOpts
	}

	// Build request body
	requestBody, err := p.buildRequestBody(processedMessages, processedOptions, true)
	if err != nil {
		return nil, fmt.Errorf("failed to build request body: %w", err)
	}

	// Get connector settings
	setting := p.Connector.Setting()
	host, ok := setting["host"].(string)
	if !ok || host == "" {
		return nil, fmt.Errorf("no host found in connector settings")
	}

	key, ok := setting["key"].(string)
	if !ok || key == "" {
		return nil, fmt.Errorf("API key is not set")
	}

	// Build URL
	url := buildAPIURL(host, "/chat/completions")

	if trace != nil {
		trace.Debug("OpenAI Stream: Sending request", map[string]any{
			"url": url,
		})
	}

	// Create HTTP request with proxy support
	req := http.New(url).
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", key)).
		SetHeader("Accept", "text/event-stream").
		SetHeader("User-Agent", "YaoAgent/1.0 (+https://yaoagents.com)")

	// Accumulate response data
	accumulator := &streamAccumulator{
		toolCalls: make(map[int]*accumulatedToolCall),
	}

	// Message tracker for lifecycle events (tracks individual messages like thinking, text, tool_call)
	messageTracker := &messageTracker{
		idGenerator: ctx.IDGenerator,
	}

	// Stream handler
	streamHandler := func(data []byte) int {
		// Check for context cancellation
		select {
		case <-goCtx.Done():
			if trace != nil {
				trace.Warn("Stream cancelled by context")
			}
			return http.HandlerReturnBreak
		default:
		}

		// Check for force interrupt signal
		if ctx.Interrupt != nil {
			if signal := ctx.Interrupt.Peek(); signal != nil && signal.Type == context.InterruptForce {
				if trace != nil {
					trace.Warn("Stream cancelled by force interrupt")
				}
				return http.HandlerReturnBreak
			}
		}

		if len(data) == 0 {
			return http.HandlerReturnOk
		}

		// Record LLM raw output to trace
		if trace != nil {
			trace.Debug("LLM Raw Output", map[string]any{
				"data": string(data),
			})
		}

		// Parse SSE data
		dataStr := string(data)
		if !strings.HasPrefix(dataStr, "data: ") {
			return http.HandlerReturnOk
		}

		dataStr = strings.TrimPrefix(dataStr, "data: ")
		dataStr = strings.TrimSpace(dataStr)

		// Check for [DONE] marker
		if dataStr == "[DONE]" {
			return http.HandlerReturnOk
		}

		// Parse JSON chunk
		var chunk StreamChunk
		if err := jsoniter.UnmarshalFromString(dataStr, &chunk); err != nil {
			if trace != nil {
				trace.Warn("Failed to parse stream chunk", map[string]any{
					"error": err.Error(),
				})
			}
			return http.HandlerReturnOk
		}

		// Process chunk
		if len(chunk.Choices) > 0 {
			choice := chunk.Choices[0]
			delta := choice.Delta

			// Update accumulator metadata
			if accumulator.id == "" {
				accumulator.id = chunk.ID
				accumulator.model = chunk.Model
				accumulator.created = chunk.Created
			}

			// Handle role
			if delta.Role != "" {
				accumulator.role = delta.Role
			}

			// Handle reasoning content (DeepSeek R1)
			if delta.ReasoningContent != "" {
				// Start thinking message if not active
				if !messageTracker.active || messageTracker.messageType != message.ChunkThinking {
					messageTracker.startMessage(message.ChunkThinking, handler)
				}

				accumulator.reasoningContent += delta.ReasoningContent
				if handler != nil {
					handler(message.ChunkThinking, []byte(delta.ReasoningContent))
					messageTracker.incrementChunk()
				}
			}

			// Handle content
			if delta.Content != "" {
				// Start text message if not active
				if !messageTracker.active || messageTracker.messageType != message.ChunkText {
					messageTracker.startMessage(message.ChunkText, handler)
				}

				accumulator.content += delta.Content
				if handler != nil {
					handler(message.ChunkText, []byte(delta.Content))
					messageTracker.incrementChunk()
				}
			}

			// Handle refusal
			if delta.Refusal != "" {
				// Start refusal message if not active
				if !messageTracker.active || messageTracker.messageType != message.ChunkRefusal {
					messageTracker.startMessage(message.ChunkRefusal, handler)
				}

				accumulator.refusal += delta.Refusal
				if handler != nil {
					handler(message.ChunkRefusal, []byte(delta.Refusal))
					messageTracker.incrementChunk()
				}
			}

			// Handle tool calls
			if len(delta.ToolCalls) > 0 {
				for _, tc := range delta.ToolCalls {
					if _, exists := accumulator.toolCalls[tc.Index]; !exists {
						accumulator.toolCalls[tc.Index] = &accumulatedToolCall{}

						// Start new tool call message when we first see this tool call
						if tc.ID != "" {
							toolCallInfo := &message.EventToolCallInfo{
								ID:    tc.ID,
								Name:  tc.Function.Name, // May be partial or empty initially
								Index: tc.Index,
							}
							messageTracker.startToolCallMessage(toolCallInfo, handler)
						}
					}
					accTC := accumulator.toolCalls[tc.Index]

					if tc.ID != "" {
						accTC.id = tc.ID
					}
					if tc.Type != "" {
						accTC.typ = tc.Type
					}
					if tc.Function.Name != "" {
						accTC.functionName = tc.Function.Name
						// Update tool call info in tracker
						if messageTracker.active && messageTracker.toolCallInfo != nil {
							messageTracker.toolCallInfo.Name = tc.Function.Name
						}
					}
					if tc.Function.Arguments != "" {
						accTC.functionArgs += tc.Function.Arguments
						// Update tool call info in tracker
						if messageTracker.active && messageTracker.toolCallInfo != nil {
							messageTracker.toolCallInfo.Arguments = accTC.functionArgs
						}
					}
				}

				// Notify handler of tool call progress
				// Send the raw delta from OpenAI (as JSON bytes)
				// Handler will convert to object for frontend merge
				if handler != nil {
					toolCallData, _ := jsoniter.Marshal(delta.ToolCalls)
					handler(message.ChunkToolCall, toolCallData)
					messageTracker.incrementChunk()
				}
			}

			// Handle finish reason
			if choice.FinishReason != nil && *choice.FinishReason != "" {
				accumulator.finishReason = *choice.FinishReason
			}

			// Handle usage (in choices, for older API versions)
			if chunk.Usage != nil {
				accumulator.usage = &message.UsageInfo{
					PromptTokens:     chunk.Usage.PromptTokens,
					CompletionTokens: chunk.Usage.CompletionTokens,
					TotalTokens:      chunk.Usage.TotalTokens,
				}
			}
		}

		// Check for usage at the top level (newer API versions with stream_options)
		if chunk.Usage != nil && accumulator.usage == nil {
			accumulator.usage = &message.UsageInfo{
				PromptTokens:     chunk.Usage.PromptTokens,
				CompletionTokens: chunk.Usage.CompletionTokens,
				TotalTokens:      chunk.Usage.TotalTokens,
			}
		}

		return http.HandlerReturnOk
	}

	// Log request for debugging
	if trace != nil {
		if requestBodyJSON, marshalErr := jsoniter.Marshal(requestBody); marshalErr == nil {
			trace.Debug("OpenAI Stream Request", map[string]any{
				"url":  url,
				"body": string(requestBodyJSON),
			})
		}
	}

	// Buffer to capture non-SSE error responses
	var errorBuffer strings.Builder
	errorDetected := false

	// Wrap streamHandler to detect JSON error responses
	// Note: API error responses are raw JSON without "data: " prefix
	// Normal SSE data always starts with "data: " prefix
	wrappedHandler := func(data []byte) int {
		dataStr := string(data)
		trimmed := strings.TrimSpace(dataStr)

		// Skip empty lines
		if trimmed == "" {
			return http.HandlerReturnOk
		}

		// Normal SSE data starts with "data: " - pass to streamHandler
		if strings.HasPrefix(dataStr, "data: ") {
			return streamHandler(data)
		}

		// Detect if this looks like a JSON error response (raw JSON without "data: " prefix)
		// API errors are returned as raw JSON: {"error": {...}}
		if strings.HasPrefix(trimmed, "{") && strings.Contains(dataStr, `"error"`) {
			errorDetected = true
		}

		// If error detected, accumulate all data for parsing
		if errorDetected {
			errorBuffer.Write(data)
			errorBuffer.WriteString("\n")
			return http.HandlerReturnOk
		}

		// Unknown format, pass to streamHandler (it will skip non-SSE data)
		return streamHandler(data)
	}

	// Make streaming request (goCtx already set at function start)
	log.Trace("[LLM] Starting HTTP Stream request: url=%s", url)
	err = req.Stream(goCtx, "POST", requestBody, wrappedHandler)
	log.Trace("[LLM] HTTP Stream request returned: err=%v", err)

	// Check if we captured an error response
	if errorDetected && errorBuffer.Len() > 0 {
		errorJSON := errorBuffer.String()
		if trace != nil {
			trace.Error(i18n.T(ctx.Locale, "llm.openai.stream.api_error"), map[string]any{"response": errorJSON}) // "OpenAI API returned error response"
		}

		// Try to parse error
		var apiError struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Param   string `json:"param"`
				Code    string `json:"code"`
			} `json:"error"`
		}

		if parseErr := jsoniter.UnmarshalFromString(errorJSON, &apiError); parseErr == nil && apiError.Error.Message != "" {
			err = fmt.Errorf("OpenAI API error: %s (type: %s, param: %s, code: %s)",
				apiError.Error.Message, apiError.Error.Type, apiError.Error.Param, apiError.Error.Code)
		} else {
			err = fmt.Errorf("OpenAI API error: %s", strings.TrimSpace(errorJSON))
		}
	}

	// Check if error is due to context cancellation FIRST (before logging)
	// This prevents blocking on trace operations when context is cancelled
	if err != nil && goCtx.Err() != nil {
		log.Trace("[LLM] Context cancelled detected, skipping handler calls and returning")
		// NOTE: Do NOT call handler or groupTracker.endGroup here
		// The connection is already closed, calling handler may block indefinitely
		// Just return the error immediately
		return nil, fmt.Errorf("stream cancelled: %w", goCtx.Err())
	}

	// Log any error from streaming (only if not cancelled)
	if err != nil && trace != nil {
		trace.Error(i18n.T(ctx.Locale, "llm.openai.stream.error"), map[string]any{"error": err.Error()}) // "OpenAI Stream Error"
	}

	if err != nil {
		// End current message if active
		messageTracker.endMessage(handler)

		// Notify handler of error if provided
		if handler != nil {
			errData := []byte(err.Error())
			handler(message.ChunkError, errData)
		}
		return nil, fmt.Errorf("streaming request failed: %w", err)
	}

	// Check if we received any data
	if accumulator.id == "" {
		if trace != nil {
			trace.Warn("OpenAI stream completed but no data was received")

			// Log request details for debugging
			if requestBodyJSON, err := jsoniter.Marshal(requestBody); err == nil {
				trace.Error(i18n.T(ctx.Locale, "llm.openai.stream.no_data"), map[string]any{"body": string(requestBodyJSON)}) // "Request body that caused empty response"
			}
			trace.Error(i18n.T(ctx.Locale, "llm.openai.stream.no_data_info"), map[string]any{ // "Request details"
				"url":     url,
				"model":   accumulator.model,
				"created": accumulator.created,
			})
		}

		err := fmt.Errorf("no data received from OpenAI API")

		// End current message if active
		messageTracker.endMessage(handler)

		// Notify handler of error if provided
		if handler != nil {
			errData := []byte(err.Error())
			handler(message.ChunkError, errData)
		}
		return nil, err
	}

	// Build final response
	response := &context.CompletionResponse{
		ID:               accumulator.id,
		Object:           "chat.completion",
		Created:          accumulator.created,
		Model:            accumulator.model,
		Role:             accumulator.role,
		Content:          accumulator.content,
		ReasoningContent: accumulator.reasoningContent,
		Refusal:          accumulator.refusal,
		FinishReason:     accumulator.finishReason,
		Usage:            accumulator.usage,
	}

	// Convert accumulated tool calls to ToolCall slice
	if len(accumulator.toolCalls) > 0 {
		toolCalls := make([]context.ToolCall, 0, len(accumulator.toolCalls))
		for i := 0; i < len(accumulator.toolCalls); i++ {
			if tc, exists := accumulator.toolCalls[i]; exists {
				toolCalls = append(toolCalls, context.ToolCall{
					ID:   tc.id,
					Type: context.ToolCallType(tc.typ),
					Function: context.Function{
						Name:      tc.functionName,
						Arguments: tc.functionArgs,
					},
				})
			}
		}
		response.ToolCalls = toolCalls

		// Validate tool call results if schema is provided
		// Note: If validation fails, we log the error but DO NOT return error
		// Instead, we let the response through so Agent layer can handle it
		// Agent layer will re-validate and provide better error feedback to LLM
		if err := p.validateToolCallResults(options, toolCalls); err != nil {
			// Log validation error
			if trace, _ := ctx.Trace(); trace != nil {
				trace.Warn("Tool call validation failed at LLM layer, passing to Agent layer for handling", map[string]any{
					"error": err.Error(),
				})
			}
			// End current message
			messageTracker.endMessage(handler)

			// Continue and return response (don't return error)
			// Agent layer will handle validation and retry
		}
	}

	// End final message if still active
	messageTracker.endMessage(handler)

	return response, nil
}

// Post post completion request to OpenAI API
func (p *Provider) Post(ctx *context.Context, messages []context.Message, options *context.CompletionOptions) (*context.CompletionResponse, error) {
	// Add debug log
	trace, _ := ctx.Trace()
	if trace != nil {
		trace.Debug("OpenAI Post: Starting non-stream request", map[string]any{
			"message_count": len(messages),
		})
	}

	maxRetries := 3
	var lastErr error

	// Get Go context for cancellation support
	// Read from Stack.Options if available (call-level override)
	goCtx := ctx.Context
	if ctx.Stack != nil && ctx.Stack.Options != nil && ctx.Stack.Options.Context != nil {
		goCtx = ctx.Stack.Options.Context
	}
	if goCtx == nil {
		goCtx = gocontext.Background()
	}

	// Make a copy of messages to avoid modifying the original
	currentMessages := make([]context.Message, len(messages))
	copy(currentMessages, messages)

	// Outer loop: handle network/API errors with exponential backoff
	for attempt := 0; attempt < maxRetries; attempt++ {
		// Check if context is cancelled before retry
		select {
		case <-goCtx.Done():
			return nil, fmt.Errorf("context cancelled: %w", goCtx.Err())
		default:
		}

		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			if trace != nil {
				trace.Warn("OpenAI post request failed, retrying", map[string]any{
					"backoff":     backoff.String(),
					"attempt":     attempt + 1,
					"max_retries": maxRetries,
					"error":       lastErr.Error(),
				})
			}

			// Sleep with context cancellation support
			timer := time.NewTimer(backoff)
			select {
			case <-timer.C:
				// Continue to retry
			case <-goCtx.Done():
				timer.Stop()
				return nil, fmt.Errorf("context cancelled during backoff: %w", goCtx.Err())
			}
		}

		response, err := p.postWithRetry(ctx, currentMessages, options)
		if err == nil {
			return response, nil
		}
		lastErr = err

		// Note: Tool call validation errors should not reach here anymore
		// because we now pass through validation failures to Agent layer
		// This check is kept for safety but should not trigger
		if isToolCallValidationError(err) {
			if trace != nil {
				trace.Debug("Tool call validation error in Post (unexpected, should be handled differently)", map[string]any{
					"error": err.Error(),
				})
			}
		}

		// Check if error is retryable (network errors, rate limits, etc.)
		if !isRetryableError(err) {
			return nil, fmt.Errorf("non-retryable error: %w", err)
		}
	}

	return nil, fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

// postWithRetry performs a single POST request attempt
func (p *Provider) postWithRetry(ctx *context.Context, messages []context.Message, options *context.CompletionOptions) (*context.CompletionResponse, error) {
	// Get trace from context
	trace, _ := ctx.Trace()

	// Preprocess messages and options through adapters
	processedMessages := messages
	processedOptions := options
	for _, adapter := range p.adapters {
		// Preprocess messages
		newMessages, err := adapter.PreprocessMessages(processedMessages)
		if err != nil {
			return nil, fmt.Errorf("adapter %s message preprocessing failed: %w", adapter.Name(), err)
		}
		processedMessages = newMessages

		// Preprocess options
		newOpts, err := adapter.PreprocessOptions(processedOptions)
		if err != nil {
			return nil, fmt.Errorf("adapter %s option preprocessing failed: %w", adapter.Name(), err)
		}
		processedOptions = newOpts
	}

	// Build request body
	requestBody, err := p.buildRequestBody(processedMessages, processedOptions, false)
	if err != nil {
		return nil, fmt.Errorf("failed to build request body: %w", err)
	}

	// Get connector settings
	setting := p.Connector.Setting()
	host, ok := setting["host"].(string)
	if !ok || host == "" {
		return nil, fmt.Errorf("no host found in connector settings")
	}

	key, ok := setting["key"].(string)
	if !ok || key == "" {
		return nil, fmt.Errorf("API key is not set")
	}

	// Build URL
	url := buildAPIURL(host, "/chat/completions")

	// Create HTTP request with proxy support
	req := http.New(url).
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", key)).
		SetHeader("User-Agent", "YaoAgent/1.0 (+https://yaoagents.com)")

	// Make request
	resp := req.Post(requestBody)
	if resp.Code != 200 {
		// Try to get detailed error message from response
		errorMsg := resp.Message
		if resp.Data != nil {
			if errorData, ok := resp.Data.(map[string]interface{}); ok {
				if errObj, ok := errorData["error"]; ok {
					if errMap, ok := errObj.(map[string]interface{}); ok {
						if msg, ok := errMap["message"].(string); ok {
							errorMsg = msg
						}
					}
				}
			}
			// Log full response data for debugging
			if trace != nil {
				if respJSON, err := jsoniter.Marshal(resp.Data); err == nil {
					trace.Error(i18n.T(ctx.Locale, "llm.openai.post.api_error"), map[string]any{"response": string(respJSON)}) // "OpenAI API error response"
				}
			}
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.Code, errorMsg)
	}

	// Parse response
	var fullResp CompletionResponseFull
	respData, err := jsoniter.Marshal(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	if err := jsoniter.Unmarshal(respData, &fullResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(fullResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	choice := fullResp.Choices[0]

	// Convert content interface{} to string
	content := ""
	if choice.Message.Content != nil {
		switch v := choice.Message.Content.(type) {
		case string:
			content = v
		default:
			// For complex content (arrays), marshal to JSON
			if contentBytes, err := jsoniter.Marshal(v); err == nil {
				content = string(contentBytes)
			}
		}
	}

	response := &context.CompletionResponse{
		ID:                fullResp.ID,
		Object:            fullResp.Object,
		Created:           fullResp.Created,
		Model:             fullResp.Model,
		Role:              string(choice.Message.Role),
		Content:           content,
		ReasoningContent:  choice.Message.ReasoningContent,
		ToolCalls:         choice.Message.ToolCalls,
		FinishReason:      choice.FinishReason,
		Usage:             fullResp.Usage,
		SystemFingerprint: fullResp.SystemFingerprint,
	}

	if choice.Message.Refusal != nil {
		response.Refusal = *choice.Message.Refusal
	}

	// Validate tool call results if present
	if len(response.ToolCalls) > 0 {
		if err := p.validateToolCallResults(options, response.ToolCalls); err != nil {
			return nil, fmt.Errorf("tool call validation failed: %w", err)
		}
	}

	return response, nil
}

// buildRequestBody builds the request body for OpenAI API
func (p *Provider) buildRequestBody(messages []context.Message, options *context.CompletionOptions, streaming bool) (map[string]interface{}, error) {
	if options == nil {
		return nil, fmt.Errorf("options are required")
	}

	// Get model and other settings from connector
	setting := p.Connector.Setting()
	model, ok := setting["model"].(string)
	if !ok || model == "" {
		return nil, fmt.Errorf("model is not set in connector")
	}

	// Get thinking setting from connector (for models that support reasoning/thinking mode)
	var thinkingSetting interface{}
	if thinking, exists := setting["thinking"]; exists {
		thinkingSetting = thinking
	}

	// Convert messages to API format
	apiMessages := make([]map[string]interface{}, 0, len(messages))
	for _, msg := range messages {
		apiMsg := map[string]interface{}{
			"role": string(msg.Role),
		}

		if msg.Content != nil {
			// Check if Content is []context.ContentPart and convert to API format
			if parts, ok := msg.Content.([]context.ContentPart); ok {
				apiParts := make([]map[string]interface{}, 0, len(parts))
				for _, part := range parts {
					apiPart := map[string]interface{}{
						"type": string(part.Type),
					}
					switch part.Type {
					case context.ContentText:
						apiPart["text"] = part.Text
					case context.ContentImageURL:
						if part.ImageURL != nil {
							apiPart["image_url"] = map[string]interface{}{
								"url": part.ImageURL.URL,
							}
							if part.ImageURL.Detail != "" {
								apiPart["image_url"].(map[string]interface{})["detail"] = part.ImageURL.Detail
							}
						}
					case context.ContentInputAudio:
						if part.InputAudio != nil {
							apiPart["input_audio"] = part.InputAudio
						}
					}
					apiParts = append(apiParts, apiPart)
				}
				apiMsg["content"] = apiParts
			} else {
				// Content is string or already in map format, use as is
				apiMsg["content"] = msg.Content
			}
		}

		if msg.Name != nil {
			apiMsg["name"] = *msg.Name
		}

		if msg.ToolCallID != nil {
			apiMsg["tool_call_id"] = *msg.ToolCallID
		}

		if len(msg.ToolCalls) > 0 {
			apiMsg["tool_calls"] = msg.ToolCalls
		}

		if msg.Refusal != nil {
			apiMsg["refusal"] = *msg.Refusal
		}

		apiMessages = append(apiMessages, apiMsg)
	}

	// Build request body
	body := map[string]interface{}{
		"model":    model,
		"messages": apiMessages,
		"stream":   streaming,
	}

	// Add optional parameters
	if options.Temperature != nil {
		body["temperature"] = *options.Temperature
	}

	// Use max_completion_tokens (modern API parameter for GPT-5+)
	// GPT-5 models only support max_completion_tokens (not max_tokens)
	if options.MaxCompletionTokens != nil {
		body["max_completion_tokens"] = *options.MaxCompletionTokens
	} else if options.MaxTokens != nil {
		// Fallback: convert MaxTokens to max_completion_tokens for compatibility
		body["max_completion_tokens"] = *options.MaxTokens
	}

	if options.TopP != nil {
		body["top_p"] = *options.TopP
	}

	if options.N != nil {
		body["n"] = *options.N
	}

	if options.Stop != nil {
		body["stop"] = options.Stop
	}

	if options.PresencePenalty != nil {
		body["presence_penalty"] = *options.PresencePenalty
	}

	if options.FrequencyPenalty != nil {
		body["frequency_penalty"] = *options.FrequencyPenalty
	}

	if len(options.LogitBias) > 0 {
		body["logit_bias"] = options.LogitBias
	}

	if options.User != "" {
		body["user"] = options.User
	}

	if options.ResponseFormat != nil {
		// Build response_format according to OpenAI API requirements
		responseFormat := map[string]interface{}{
			"type": options.ResponseFormat.Type,
		}

		// For json_schema type, include the schema details
		if options.ResponseFormat.Type == context.ResponseFormatJSONSchema && options.ResponseFormat.JSONSchema != nil {
			responseFormat["json_schema"] = options.ResponseFormat.JSONSchema
		}

		body["response_format"] = responseFormat
	}

	if options.Seed != nil {
		body["seed"] = *options.Seed
	}

	if len(options.Tools) > 0 {
		body["tools"] = options.Tools
	}

	if options.ToolChoice != nil {
		body["tool_choice"] = options.ToolChoice
	}

	// Reasoning effort (o1 and GPT-5 models)
	if options.ReasoningEffort != nil {
		body["reasoning_effort"] = *options.ReasoningEffort
	}

	// For streaming, include usage info by default
	if streaming {
		if options.StreamOptions != nil {
			body["stream_options"] = options.StreamOptions
		} else {
			// Default: include usage info in streaming response
			body["stream_options"] = map[string]interface{}{
				"include_usage": true,
			}
		}
	}

	if options.Audio != nil {
		body["audio"] = options.Audio
	}

	// Add thinking parameter for models that support reasoning/thinking mode
	if thinkingSetting != nil {
		body["thinking"] = thinkingSetting
	}

	return body, nil
}

// validateToolCallResults validates tool call arguments against JSON schema
func (p *Provider) validateToolCallResults(options *context.CompletionOptions, toolCalls []context.ToolCall) error {
	if options == nil || options.Tools == nil || len(options.Tools) == 0 {
		return nil
	}

	// Build tool schema map for quick lookup
	toolSchemas := make(map[string]interface{})
	for _, tool := range options.Tools {
		if function, ok := tool["function"].(map[string]interface{}); ok {
			if name, ok := function["name"].(string); ok {
				if parameters, ok := function["parameters"]; ok {
					toolSchemas[name] = parameters
				}
			}
		}
	}

	// Validate each tool call
	for _, tc := range toolCalls {
		schema, hasSchema := toolSchemas[tc.Function.Name]
		if !hasSchema {
			continue // No schema to validate against
		}

		// Parse arguments JSON
		var args interface{}
		if err := jsoniter.UnmarshalFromString(tc.Function.Arguments, &args); err != nil {
			return fmt.Errorf("tool call %s has invalid JSON arguments: %w", tc.Function.Name, err)
		}

		// Validate against schema
		if err := jsonschema.ValidateData(schema, args); err != nil {
			return fmt.Errorf("tool call %s arguments validation failed: %w", tc.Function.Name, err)
		}
	}

	return nil
}

// isToolCallValidationError checks if an error is a tool call validation error
func isToolCallValidationError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "tool call validation failed") ||
		strings.Contains(errStr, "arguments validation failed")
}

// isRetryableError checks if an error is retryable
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Retryable: network errors, timeouts, rate limits, server errors
	retryablePatterns := []string{
		"timeout",
		"connection refused",
		"connection reset",
		"EOF",
		"HTTP 429", // Rate limit
		"HTTP 500", // Internal server error
		"HTTP 502", // Bad gateway
		"HTTP 503", // Service unavailable
		"HTTP 504", // Gateway timeout
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(strings.ToLower(errStr), strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}
