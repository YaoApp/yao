package openai

import (
	gocontext "context"
	"fmt"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/http"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/llm/adapters"
	"github.com/yaoapp/yao/agent/llm/providers/base"
	"github.com/yaoapp/yao/utils/jsonschema"
)

// startGroup starts a new group and sends group_start event
func (gt *groupTracker) startGroup(groupType context.StreamChunkType, handler context.StreamFunc) {
	if gt.active {
		// End previous group first
		gt.endGroup(handler)
	}

	gt.active = true
	gt.groupID = fmt.Sprintf("grp_%d", time.Now().UnixNano())
	gt.groupType = groupType
	gt.startTime = time.Now().UnixMilli()
	gt.chunkCount = 0
	gt.toolCallInfo = nil

	if handler != nil {
		startData := &context.GroupStartData{
			GroupID:   gt.groupID,
			Type:      string(groupType),
			Timestamp: gt.startTime,
		}
		if startJSON, err := jsoniter.Marshal(startData); err == nil {
			handler(context.ChunkGroupStart, startJSON)
		}
	}
}

// startToolCallGroup starts a new tool call group with tool call info
func (gt *groupTracker) startToolCallGroup(toolCallInfo *context.GroupToolCallInfo, handler context.StreamFunc) {
	if gt.active {
		gt.endGroup(handler)
	}

	gt.active = true
	gt.groupID = fmt.Sprintf("grp_tool_%d", time.Now().UnixNano())
	gt.groupType = context.ChunkToolCall
	gt.startTime = time.Now().UnixMilli()
	gt.chunkCount = 0
	gt.toolCallInfo = toolCallInfo

	if handler != nil {
		startData := &context.GroupStartData{
			GroupID:   gt.groupID,
			Type:      string(context.ChunkToolCall),
			Timestamp: gt.startTime,
			ToolCall:  toolCallInfo,
		}
		if startJSON, err := jsoniter.Marshal(startData); err == nil {
			handler(context.ChunkGroupStart, startJSON)
		}
	}
}

// incrementChunk increments the chunk count for the current group
func (gt *groupTracker) incrementChunk() {
	if gt.active {
		gt.chunkCount++
	}
}

// endGroup ends the current group and sends group_end event
func (gt *groupTracker) endGroup(handler context.StreamFunc) {
	if !gt.active {
		return
	}

	if handler != nil {
		endData := &context.GroupEndData{
			GroupID:    gt.groupID,
			Type:       string(gt.groupType),
			Timestamp:  time.Now().UnixMilli(),
			DurationMs: time.Now().UnixMilli() - gt.startTime,
			ChunkCount: gt.chunkCount,
			Status:     "completed",
		}
		if gt.toolCallInfo != nil {
			endData.ToolCall = gt.toolCallInfo
		}
		if endJSON, err := jsoniter.Marshal(endData); err == nil {
			handler(context.ChunkGroupEnd, endJSON)
		}
	}

	gt.active = false
	gt.groupID = ""
	gt.toolCallInfo = nil
}

// Provider OpenAI-compatible provider with capability adapters
// Supports: vision, tool calls, streaming, JSON mode, reasoning
type Provider struct {
	*base.Provider
	adapters []adapters.CapabilityAdapter
}

// New create a new OpenAI provider with capability adapters
func New(conn connector.Connector, capabilities *context.ModelCapabilities) *Provider {
	return &Provider{
		Provider: base.NewProvider(conn, capabilities),
		adapters: buildAdapters(capabilities),
	}
}

// buildAdapters builds capability adapters based on model capabilities
func buildAdapters(cap *context.ModelCapabilities) []adapters.CapabilityAdapter {
	if cap == nil {
		return []adapters.CapabilityAdapter{}
	}

	result := make([]adapters.CapabilityAdapter, 0)

	// Tool call adapter
	if cap.ToolCalls != nil {
		result = append(result, adapters.NewToolCallAdapter(*cap.ToolCalls))
	}

	// Vision adapter
	if cap.Vision != nil {
		result = append(result, adapters.NewVisionAdapter(*cap.Vision))
	}

	// Audio adapter
	if cap.Audio != nil {
		result = append(result, adapters.NewAudioAdapter(*cap.Audio))
	}

	// Reasoning adapter (always add to handle reasoning_effort and temperature parameters)
	// Even if the model doesn't support reasoning, we need the adapter to strip reasoning_effort
	if cap.Reasoning != nil {
		if *cap.Reasoning {
			// Detect reasoning format based on capabilities
			format := detectReasoningFormat(cap)
			result = append(result, adapters.NewReasoningAdapter(format, cap))
		} else {
			// Model doesn't support reasoning, use None format to strip reasoning parameters
			result = append(result, adapters.NewReasoningAdapter(adapters.ReasoningFormatNone, cap))
		}
	}

	return result
}

// detectReasoningFormat detects the reasoning format based on capabilities
func detectReasoningFormat(cap *context.ModelCapabilities) adapters.ReasoningFormat {
	// TODO: Implement better detection logic
	// For now, default to OpenAI o1 format if reasoning is supported
	if cap.Reasoning != nil && *cap.Reasoning {
		return adapters.ReasoningFormatOpenAI
	}
	return adapters.ReasoningFormatNone
}

// Stream stream completion from OpenAI API
func (p *Provider) Stream(ctx *context.Context, messages []context.Message, options *context.CompletionOptions, handler context.StreamFunc) (*context.CompletionResponse, error) {
	maxRetries := 3
	maxValidationRetries := 3
	var lastErr error

	// Get Go context for cancellation support
	goCtx := ctx.Context
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
			// Exponential backoff: 1s, 2s, 4s
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			log.Warn("OpenAI stream request failed, retrying in %v (attempt %d/%d): %v", backoff, attempt+1, maxRetries, lastErr)

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

		response, err := p.streamWithRetry(ctx, currentMessages, options, handler)
		if err == nil {
			return response, nil
		}
		lastErr = err

		// Check if error is tool call validation failure
		if isToolCallValidationError(err) {
			// Handle tool call validation retry with feedback to LLM
			validationRetryMessages := currentMessages
			for validationAttempt := 0; validationAttempt < maxValidationRetries; validationAttempt++ {
				log.Warn("Tool call validation failed (attempt %d/%d): %v", validationAttempt+1, maxValidationRetries, err)

				// Add error feedback to conversation history
				validationRetryMessages = append(validationRetryMessages, context.Message{
					Role:    context.RoleSystem,
					Content: fmt.Sprintf("Tool call validation error: %v. Please correct the tool call arguments to match the required schema.", err),
				})

				// Retry with feedback
				response, err = p.streamWithRetry(ctx, validationRetryMessages, options, handler)
				if err == nil {
					return response, nil
				}

				// Check if still validation error
				if !isToolCallValidationError(err) {
					// Different error type, break out of validation retry loop
					lastErr = err
					break
				}
				lastErr = err
			}

			// If we exhausted validation retries, return the error
			if isToolCallValidationError(lastErr) {
				return nil, fmt.Errorf("tool call validation failed after %d retries: %w", maxValidationRetries, lastErr)
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
func (p *Provider) streamWithRetry(ctx *context.Context, messages []context.Message, options *context.CompletionOptions, handler context.StreamFunc) (*context.CompletionResponse, error) {
	streamStartTime := time.Now()
	requestID := fmt.Sprintf("req_%d", streamStartTime.UnixNano())

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

	// Send stream_start event
	if handler != nil {
		model, _ := p.GetModel()
		startData := &context.StreamStartData{
			RequestID: requestID,
			Timestamp: streamStartTime.UnixMilli(),
			Model:     model,
		}
		if startJSON, err := jsoniter.Marshal(startData); err == nil {
			handler(context.ChunkStreamStart, startJSON)
		}
	}

	// Preprocess options through adapters
	processedOptions := options
	for _, adapter := range p.adapters {
		newOpts, err := adapter.PreprocessOptions(processedOptions)
		if err != nil {
			// Send error to handler
			if handler != nil {
				handler(context.ChunkError, []byte(fmt.Sprintf("adapter %s preprocessing failed: %v", adapter.Name(), err)))
			}
			return nil, fmt.Errorf("adapter %s preprocessing failed: %w", adapter.Name(), err)
		}
		processedOptions = newOpts
	}

	// Build request body
	requestBody, err := p.buildRequestBody(messages, processedOptions, true)
	if err != nil {
		// Send stream_end with error
		if handler != nil {
			endData := &context.StreamEndData{
				RequestID:  requestID,
				Timestamp:  time.Now().UnixMilli(),
				DurationMs: time.Since(streamStartTime).Milliseconds(),
				Status:     "error",
				Error:      err.Error(),
			}
			if endJSON, err := jsoniter.Marshal(endData); err == nil {
				handler(context.ChunkStreamEnd, endJSON)
			}
		}
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
	endpoint := "/chat/completions"
	if host == "https://api.openai.com" && !strings.HasPrefix(endpoint, "/v1") {
		endpoint = "/v1" + endpoint
	}
	host = strings.TrimSuffix(host, "/")
	url := host + endpoint

	// Create HTTP request with proxy support
	req := http.New(url).
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", key)).
		SetHeader("Accept", "text/event-stream")

	// Accumulate response data
	accumulator := &streamAccumulator{
		toolCalls: make(map[int]*accumulatedToolCall),
	}

	// Group tracker for lifecycle events
	groupTracker := &groupTracker{}

	// Stream handler
	streamHandler := func(data []byte) int {
		// Check for context cancellation
		select {
		case <-goCtx.Done():
			log.Warn("Stream cancelled by context")
			return http.HandlerReturnBreak
		default:
		}

		if len(data) == 0 {
			return http.HandlerReturnOk
		}

		// Log raw stream data for debugging
		log.Trace("OpenAI Stream Raw Data: %s", string(data))

		// Parse SSE data
		dataStr := string(data)
		if !strings.HasPrefix(dataStr, "data: ") {
			log.Trace("Skipping non-SSE line: %s", dataStr)
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
			log.Warn("Failed to parse stream chunk: %v", err)
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
				// Start thinking group if not active
				if !groupTracker.active || groupTracker.groupType != context.ChunkThinking {
					groupTracker.startGroup(context.ChunkThinking, handler)
				}

				accumulator.reasoningContent += delta.ReasoningContent
				if handler != nil {
					handler(context.ChunkThinking, []byte(delta.ReasoningContent))
					groupTracker.incrementChunk()
				}
			}

			// Handle content
			if delta.Content != "" {
				// Start text group if not active
				if !groupTracker.active || groupTracker.groupType != context.ChunkText {
					groupTracker.startGroup(context.ChunkText, handler)
				}

				accumulator.content += delta.Content
				if handler != nil {
					handler(context.ChunkText, []byte(delta.Content))
					groupTracker.incrementChunk()
				}
			}

			// Handle refusal
			if delta.Refusal != "" {
				// Start refusal group if not active
				if !groupTracker.active || groupTracker.groupType != context.ChunkRefusal {
					groupTracker.startGroup(context.ChunkRefusal, handler)
				}

				accumulator.refusal += delta.Refusal
				if handler != nil {
					handler(context.ChunkRefusal, []byte(delta.Refusal))
					groupTracker.incrementChunk()
				}
			}

			// Handle tool calls
			if len(delta.ToolCalls) > 0 {
				for _, tc := range delta.ToolCalls {
					if _, exists := accumulator.toolCalls[tc.Index]; !exists {
						accumulator.toolCalls[tc.Index] = &accumulatedToolCall{}

						// Start new tool call group when we first see this tool call
						if tc.ID != "" {
							toolCallInfo := &context.GroupToolCallInfo{
								ID:    tc.ID,
								Name:  tc.Function.Name, // May be partial or empty initially
								Index: tc.Index,
							}
							groupTracker.startToolCallGroup(toolCallInfo, handler)
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
						if groupTracker.active && groupTracker.toolCallInfo != nil {
							groupTracker.toolCallInfo.Name = tc.Function.Name
						}
					}
					if tc.Function.Arguments != "" {
						accTC.functionArgs += tc.Function.Arguments
						// Update tool call info in tracker
						if groupTracker.active && groupTracker.toolCallInfo != nil {
							groupTracker.toolCallInfo.Arguments = accTC.functionArgs
						}
					}
				}

				// Notify handler of tool call progress
				if handler != nil {
					toolCallData, _ := jsoniter.Marshal(delta.ToolCalls)
					handler(context.ChunkToolCall, toolCallData)
					groupTracker.incrementChunk()
				}
			}

			// Handle finish reason
			if choice.FinishReason != nil && *choice.FinishReason != "" {
				accumulator.finishReason = *choice.FinishReason
			}

			// Handle usage (in choices, for older API versions)
			if chunk.Usage != nil {
				accumulator.usage = &context.UsageInfo{
					PromptTokens:     chunk.Usage.PromptTokens,
					CompletionTokens: chunk.Usage.CompletionTokens,
					TotalTokens:      chunk.Usage.TotalTokens,
				}
			}
		}

		// Check for usage at the top level (newer API versions with stream_options)
		if chunk.Usage != nil && accumulator.usage == nil {
			accumulator.usage = &context.UsageInfo{
				PromptTokens:     chunk.Usage.PromptTokens,
				CompletionTokens: chunk.Usage.CompletionTokens,
				TotalTokens:      chunk.Usage.TotalTokens,
			}
		}

		return http.HandlerReturnOk
	}

	// Log request for debugging
	if requestBodyJSON, marshalErr := jsoniter.Marshal(requestBody); marshalErr == nil {
		log.Debug("OpenAI Stream Request - URL: %s, Body: %s", url, string(requestBodyJSON))
	}

	// Buffer to capture non-SSE error responses
	var errorBuffer strings.Builder
	errorDetected := false

	// Wrap streamHandler to detect JSON error responses
	wrappedHandler := func(data []byte) int {
		dataStr := string(data)

		// Detect if this looks like a JSON error response (starts with "{" or contains "error")
		if strings.Contains(dataStr, `"error"`) || (strings.TrimSpace(dataStr) == "{" && !errorDetected) {
			errorDetected = true
		}

		// If error detected, accumulate all data for parsing
		if errorDetected {
			errorBuffer.Write(data)
			errorBuffer.WriteString("\n")
			return http.HandlerReturnOk
		}

		// Otherwise, use normal handler
		return streamHandler(data)
	}

	// Make streaming request (goCtx already set at function start)
	err = req.Stream(goCtx, "POST", requestBody, wrappedHandler)

	// Check if we captured an error response
	if errorDetected && errorBuffer.Len() > 0 {
		errorJSON := errorBuffer.String()
		log.Error("OpenAI API returned error response: %s", errorJSON)

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

	// Log any error from streaming
	if err != nil {
		log.Error("OpenAI Stream Error: %v", err)
	}

	// Check if error is due to context cancellation
	if err != nil && goCtx.Err() != nil {
		// End current group if active
		groupTracker.endGroup(handler)

		// Send stream_end with cancellation status
		if handler != nil {
			endData := &context.StreamEndData{
				RequestID:  requestID,
				Timestamp:  time.Now().UnixMilli(),
				DurationMs: time.Since(streamStartTime).Milliseconds(),
				Status:     "cancelled",
				Error:      goCtx.Err().Error(),
			}
			if endJSON, err := jsoniter.Marshal(endData); err == nil {
				handler(context.ChunkStreamEnd, endJSON)
			}
		}
		return nil, fmt.Errorf("stream cancelled: %w", goCtx.Err())
	}

	if err != nil {
		// End current group if active
		groupTracker.endGroup(handler)

		// Notify handler of error if provided
		if handler != nil {
			errData := []byte(err.Error())
			handler(context.ChunkError, errData)

			// Send stream_end with error
			endData := &context.StreamEndData{
				RequestID:  requestID,
				Timestamp:  time.Now().UnixMilli(),
				DurationMs: time.Since(streamStartTime).Milliseconds(),
				Status:     "error",
				Error:      err.Error(),
			}
			if endJSON, err := jsoniter.Marshal(endData); err == nil {
				handler(context.ChunkStreamEnd, endJSON)
			}
		}
		return nil, fmt.Errorf("streaming request failed: %w", err)
	}

	// Check if we received any data
	if accumulator.id == "" {
		log.Warn("OpenAI stream completed but no data was received (accumulator.id is empty)")

		// Log request details for debugging
		if requestBodyJSON, err := jsoniter.Marshal(requestBody); err == nil {
			log.Error("Request body that caused empty response: %s", string(requestBodyJSON))
		}
		log.Error("Request URL: %s", url)
		log.Error("Model in accumulator: %s, Created: %d", accumulator.model, accumulator.created)

		err := fmt.Errorf("no data received from OpenAI API")

		// End current group if active
		groupTracker.endGroup(handler)

		// Notify handler of error if provided
		if handler != nil {
			errData := []byte(err.Error())
			handler(context.ChunkError, errData)

			// Send stream_end with error
			endData := &context.StreamEndData{
				RequestID:  requestID,
				Timestamp:  time.Now().UnixMilli(),
				DurationMs: time.Since(streamStartTime).Milliseconds(),
				Status:     "error",
				Error:      err.Error(),
			}
			if endJSON, err := jsoniter.Marshal(endData); err == nil {
				handler(context.ChunkStreamEnd, endJSON)
			}
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
		if err := p.validateToolCallResults(options, toolCalls); err != nil {
			// End current group
			groupTracker.endGroup(handler)

			// Send stream_end with validation error
			if handler != nil {
				endData := &context.StreamEndData{
					RequestID:  requestID,
					Timestamp:  time.Now().UnixMilli(),
					DurationMs: time.Since(streamStartTime).Milliseconds(),
					Status:     "error",
					Error:      "tool call validation failed",
				}
				if endJSON, err := jsoniter.Marshal(endData); err == nil {
					handler(context.ChunkStreamEnd, endJSON)
				}
			}

			// Tool call validation failed, need to retry with error feedback
			return nil, fmt.Errorf("tool call validation failed: %w", err)
		}
	}

	// End final group if still active
	groupTracker.endGroup(handler)

	// Send stream_end event (success)
	if handler != nil {
		endData := &context.StreamEndData{
			RequestID:  requestID,
			Timestamp:  time.Now().UnixMilli(),
			DurationMs: time.Since(streamStartTime).Milliseconds(),
			Status:     "completed",
			Usage:      response.Usage,
		}
		if endJSON, err := jsoniter.Marshal(endData); err == nil {
			handler(context.ChunkStreamEnd, endJSON)
		}
	}

	return response, nil
}

// Post post completion request to OpenAI API
func (p *Provider) Post(ctx *context.Context, messages []context.Message, options *context.CompletionOptions) (*context.CompletionResponse, error) {
	maxRetries := 3
	maxValidationRetries := 3
	var lastErr error

	// Get Go context for cancellation support
	goCtx := ctx.Context
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
			log.Warn("OpenAI post request failed, retrying in %v (attempt %d/%d): %v", backoff, attempt+1, maxRetries, lastErr)

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

		// Check if error is tool call validation failure
		if isToolCallValidationError(err) {
			// Handle tool call validation retry with feedback to LLM
			validationRetryMessages := currentMessages
			for validationAttempt := 0; validationAttempt < maxValidationRetries; validationAttempt++ {
				log.Warn("Tool call validation failed (attempt %d/%d): %v", validationAttempt+1, maxValidationRetries, err)

				// Add error feedback to conversation history
				validationRetryMessages = append(validationRetryMessages, context.Message{
					Role:    context.RoleSystem,
					Content: fmt.Sprintf("Tool call validation error: %v. Please correct the tool call arguments to match the required schema.", err),
				})

				// Retry with feedback
				response, err = p.postWithRetry(ctx, validationRetryMessages, options)
				if err == nil {
					return response, nil
				}

				// Check if still validation error
				if !isToolCallValidationError(err) {
					// Different error type, break out of validation retry loop
					lastErr = err
					break
				}
				lastErr = err
			}

			// If we exhausted validation retries, return the error
			if isToolCallValidationError(lastErr) {
				return nil, fmt.Errorf("tool call validation failed after %d retries: %w", maxValidationRetries, lastErr)
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
	// Preprocess options through adapters
	processedOptions := options
	for _, adapter := range p.adapters {
		newOpts, err := adapter.PreprocessOptions(processedOptions)
		if err != nil {
			return nil, fmt.Errorf("adapter %s preprocessing failed: %w", adapter.Name(), err)
		}
		processedOptions = newOpts
	}

	// Build request body
	requestBody, err := p.buildRequestBody(messages, processedOptions, false)
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
	endpoint := "/chat/completions"
	if host == "https://api.openai.com" && !strings.HasPrefix(endpoint, "/v1") {
		endpoint = "/v1" + endpoint
	}
	host = strings.TrimSuffix(host, "/")
	url := host + endpoint

	// Create HTTP request with proxy support
	req := http.New(url).
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", key))

	// Make request
	resp := req.Post(requestBody)
	if resp.Code != 200 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.Code, resp.Message)
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

	// Get model from connector settings
	setting := p.Connector.Setting()
	model, ok := setting["model"].(string)
	if !ok || model == "" {
		return nil, fmt.Errorf("model is not set in connector")
	}

	// Convert messages to API format
	apiMessages := make([]map[string]interface{}, 0, len(messages))
	for _, msg := range messages {
		apiMsg := map[string]interface{}{
			"role": string(msg.Role),
		}

		if msg.Content != nil {
			apiMsg["content"] = msg.Content
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
