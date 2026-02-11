package anthropic

import (
	gocontext "context"
	"fmt"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/connector"
	gouAnthropicConn "github.com/yaoapp/gou/connector/anthropic"
	gouOpenAI "github.com/yaoapp/gou/connector/openai"
	"github.com/yaoapp/gou/http"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/i18n"
	"github.com/yaoapp/yao/agent/llm/adapters"
	"github.com/yaoapp/yao/agent/llm/providers/base"
	"github.com/yaoapp/yao/agent/output/message"
)

// Provider Anthropic Messages API provider
type Provider struct {
	*base.Provider
	adapters []adapters.CapabilityAdapter
}

// New create a new Anthropic provider
func New(conn connector.Connector, capabilities *gouOpenAI.Capabilities) *Provider {
	return &Provider{
		Provider: base.NewProvider(conn, capabilities),
		adapters: buildAdapters(capabilities),
	}
}

// NewFromAnthropicCaps create a new Anthropic provider from Anthropic capabilities
func NewFromAnthropicCaps(conn connector.Connector, caps *gouAnthropicConn.Capabilities) *Provider {
	// Convert anthropic capabilities to openai capabilities for base provider compatibility
	openaiCaps := &gouOpenAI.Capabilities{
		Vision:                caps.Vision,
		Audio:                 caps.Audio,
		ToolCalls:             caps.ToolCalls,
		Reasoning:             caps.Reasoning,
		Streaming:             caps.Streaming,
		JSON:                  caps.JSON,
		Multimodal:            caps.Multimodal,
		TemperatureAdjustable: caps.TemperatureAdjustable,
	}
	return &Provider{
		Provider: base.NewProvider(conn, openaiCaps),
		adapters: buildAdapters(openaiCaps),
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
		result = append(result, adapters.NewVisionAdapter(false, context.VisionFormatNone))
	}

	// Audio adapter
	result = append(result, adapters.NewAudioAdapter(cap.Audio))

	// Reasoning adapter
	if cap.Reasoning {
		result = append(result, adapters.NewReasoningAdapter(adapters.ReasoningFormatOpenAI, cap))
	} else {
		result = append(result, adapters.NewReasoningAdapter(adapters.ReasoningFormatNone, cap))
	}

	return result
}

// Stream stream completion from Anthropic API
func (p *Provider) Stream(ctx *context.Context, messages []context.Message, options *context.CompletionOptions, handler message.StreamFunc) (*context.CompletionResponse, error) {
	trace, _ := ctx.Trace()
	if trace != nil {
		trace.Debug("Anthropic Stream: Starting stream request", map[string]any{
			"message_count": len(messages),
		})
	}

	maxRetries := 3
	var lastErr error

	goCtx := ctx.Context
	if ctx.Stack != nil && ctx.Stack.Options != nil && ctx.Stack.Options.Context != nil {
		goCtx = ctx.Stack.Options.Context
	}
	if goCtx == nil {
		goCtx = gocontext.Background()
	}

	currentMessages := make([]context.Message, len(messages))
	copy(currentMessages, messages)

	for attempt := 0; attempt < maxRetries; attempt++ {
		select {
		case <-goCtx.Done():
			return nil, fmt.Errorf("context cancelled: %w", goCtx.Err())
		default:
		}

		if ctx.Interrupt != nil {
			if signal := ctx.Interrupt.Peek(); signal != nil && signal.Type == context.InterruptForce {
				return nil, fmt.Errorf("force interrupted by user")
			}
		}

		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			if trace != nil {
				trace.Warn("Anthropic stream request failed, retrying", map[string]any{
					"backoff":     backoff.String(),
					"attempt":     attempt + 1,
					"max_retries": maxRetries,
					"error":       lastErr.Error(),
				})
			}

			timer := time.NewTimer(backoff)
			interruptTicker := time.NewTicker(100 * time.Millisecond)
			defer interruptTicker.Stop()

		backoffLoop:
			for {
				select {
				case <-timer.C:
					break backoffLoop
				case <-goCtx.Done():
					timer.Stop()
					return nil, fmt.Errorf("context cancelled during backoff: %w", goCtx.Err())
				case <-interruptTicker.C:
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
		if err == nil {
			if trace != nil && goCtx.Err() == nil {
				trace.Debug("Anthropic Stream: Request completed successfully")
			}
			return response, nil
		}
		lastErr = err

		if goCtx.Err() != nil {
			return nil, fmt.Errorf("context cancelled: %w", goCtx.Err())
		}

		if !isRetryableError(err) {
			return nil, fmt.Errorf("non-retryable error: %w", err)
		}
	}

	return nil, fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

// streamWithRetry performs a single streaming request to Anthropic API
func (p *Provider) streamWithRetry(ctx *context.Context, messages []context.Message, options *context.CompletionOptions, handler message.StreamFunc) (*context.CompletionResponse, error) {
	streamStartTime := time.Now()
	trace, _ := ctx.Trace()

	goCtx := ctx.Context
	if goCtx == nil {
		goCtx = gocontext.Background()
	}

	select {
	case <-goCtx.Done():
		return nil, fmt.Errorf("context cancelled before stream start: %w", goCtx.Err())
	default:
	}

	if ctx.Interrupt != nil {
		if signal := ctx.Interrupt.Peek(); signal != nil && signal.Type == context.InterruptForce {
			return nil, fmt.Errorf("force interrupted by user before stream start")
		}
	}

	// Preprocess messages and options through adapters
	processedMessages := messages
	processedOptions := options
	for _, adapter := range p.adapters {
		newMessages, err := adapter.PreprocessMessages(processedMessages)
		if err != nil {
			return nil, fmt.Errorf("adapter %s message preprocessing failed: %w", adapter.Name(), err)
		}
		processedMessages = newMessages

		newOpts, err := adapter.PreprocessOptions(processedOptions)
		if err != nil {
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

	version := "2023-06-01"
	if v, ok := setting["version"].(string); ok && v != "" {
		version = v
	}

	// Build URL: host/v1/messages
	url := buildAPIURL(host, "/messages")

	if trace != nil {
		trace.Debug("Anthropic Stream: Sending request", map[string]any{
			"url": url,
		})
	}

	// Create HTTP request with Anthropic auth headers
	req := http.New(url).
		SetHeader("Content-Type", "application/json").
		SetHeader("x-api-key", key).
		SetHeader("anthropic-version", version).
		SetHeader("Accept", "text/event-stream").
		SetHeader("User-Agent", "YaoAgent/1.0 (+https://yaoagents.com)")

	// Accumulate response data
	accumulator := &streamAccumulator{
		toolCalls:         make(map[int]*accumulatedToolCall),
		currentBlockIndex: -1,
	}

	// Message tracker for lifecycle events
	msgTracker := &messageTracker{
		idGenerator: ctx.IDGenerator,
	}

	// Stream handler for Anthropic SSE events
	// Anthropic SSE format:
	//   event: <event_type>
	//   data: <json>
	var currentEventType string

	streamHandler := func(data []byte) int {
		select {
		case <-goCtx.Done():
			return http.HandlerReturnBreak
		default:
		}

		if ctx.Interrupt != nil {
			if signal := ctx.Interrupt.Peek(); signal != nil && signal.Type == context.InterruptForce {
				return http.HandlerReturnBreak
			}
		}

		if len(data) == 0 {
			return http.HandlerReturnOk
		}

		dataStr := string(data)
		trimmed := strings.TrimSpace(dataStr)

		if trimmed == "" {
			return http.HandlerReturnOk
		}

		// Parse event type line
		// Support both "event: type" (with space) and "event:type" (without space) formats
		if strings.HasPrefix(trimmed, "event:") {
			currentEventType = strings.TrimSpace(strings.TrimPrefix(trimmed, "event:"))
			return http.HandlerReturnOk
		}

		// Parse data line
		// Support both "data: {...}" (with space) and "data:{...}" (without space) formats
		if !strings.HasPrefix(trimmed, "data:") {
			// Check for error response
			if strings.HasPrefix(trimmed, "{") && strings.Contains(trimmed, `"error"`) {
				var apiErr APIError
				if err := jsoniter.UnmarshalFromString(trimmed, &apiErr); err == nil && apiErr.Error.Message != "" {
					if handler != nil {
						handler(message.ChunkError, []byte(apiErr.Error.Message))
					}
				}
			}
			return http.HandlerReturnOk
		}

		jsonStr := strings.TrimSpace(strings.TrimPrefix(trimmed, "data:"))

		if jsonStr == "" {
			return http.HandlerReturnOk
		}

		// Process based on event type
		switch currentEventType {
		case "message_start":
			var event MessageStartEvent
			if err := jsoniter.UnmarshalFromString(jsonStr, &event); err == nil {
				accumulator.id = event.Message.ID
				accumulator.model = event.Message.Model
				accumulator.role = event.Message.Role
				if event.Message.Usage != nil {
					accumulator.usage = &message.UsageInfo{
						PromptTokens: event.Message.Usage.InputTokens,
						TotalTokens:  event.Message.Usage.InputTokens,
					}
				}
			}

		case "content_block_start":
			var event ContentBlockStartEvent
			if err := jsoniter.UnmarshalFromString(jsonStr, &event); err == nil {
				accumulator.currentBlockIndex = event.Index
				accumulator.currentBlockType = event.ContentBlock.Type

				switch event.ContentBlock.Type {
				case "thinking":
					startMessage(msgTracker, message.ChunkThinking, handler)
				case "text":
					startMessage(msgTracker, message.ChunkText, handler)
				case "tool_use":
					accumulator.toolCalls[event.Index] = &accumulatedToolCall{
						id:   event.ContentBlock.ID,
						name: event.ContentBlock.Name,
					}
					toolCallInfo := &message.EventToolCallInfo{
						ID:    event.ContentBlock.ID,
						Name:  event.ContentBlock.Name,
						Index: event.Index,
					}
					startToolCallMessage(msgTracker, toolCallInfo, handler)
				}
			}

		case "content_block_delta":
			var event ContentBlockDeltaEvent
			if err := jsoniter.UnmarshalFromString(jsonStr, &event); err == nil {
				switch event.Delta.Type {
				case "thinking_delta":
					if event.Delta.Thinking != "" {
						accumulator.thinkingContent += event.Delta.Thinking
						if handler != nil {
							handler(message.ChunkThinking, []byte(event.Delta.Thinking))
							incrementChunk(msgTracker)
						}
					}

				case "text_delta":
					if event.Delta.Text != "" {
						accumulator.content += event.Delta.Text
						if handler != nil {
							handler(message.ChunkText, []byte(event.Delta.Text))
							incrementChunk(msgTracker)
						}
					}

				case "input_json_delta":
					if event.Delta.PartialJSON != "" {
						if tc, exists := accumulator.toolCalls[event.Index]; exists {
							tc.inputJSON += event.Delta.PartialJSON
							// Update tracker
							if msgTracker.active && msgTracker.toolCallInfo != nil {
								msgTracker.toolCallInfo.Arguments = tc.inputJSON
							}
						}
						if handler != nil {
							// Send tool call delta
							toolCallData, _ := jsoniter.Marshal([]map[string]interface{}{
								{
									"index": event.Index,
									"function": map[string]interface{}{
										"arguments": event.Delta.PartialJSON,
									},
								},
							})
							handler(message.ChunkToolCall, toolCallData)
							incrementChunk(msgTracker)
						}
					}

				case "signature_delta":
					// Handle thinking signature delta (for extended thinking)
					// The signature is accumulated but not sent to handler
					var sigDelta struct {
						Type      string `json:"type"`
						Signature string `json:"signature"`
					}
					if err := jsoniter.UnmarshalFromString(jsonStr, &struct {
						Delta *struct {
							Signature string `json:"signature"`
						} `json:"delta"`
					}{Delta: &struct {
						Signature string `json:"signature"`
					}{}}); err == nil {
						_ = sigDelta // signature tracking if needed
					}
				}
			}

		case "content_block_stop":
			endMessage(msgTracker, handler)

		case "message_delta":
			var event MessageDeltaEvent
			if err := jsoniter.UnmarshalFromString(jsonStr, &event); err == nil {
				accumulator.stopReason = event.Delta.StopReason
				if event.Usage != nil {
					if accumulator.usage == nil {
						accumulator.usage = &message.UsageInfo{}
					}
					accumulator.usage.CompletionTokens = event.Usage.OutputTokens
					accumulator.usage.TotalTokens = accumulator.usage.PromptTokens + event.Usage.OutputTokens
				}
			}

		case "message_stop":
			// Message complete
			endMessage(msgTracker, handler)

		case "ping":
			// Keep-alive, ignore

		case "error":
			var apiErr struct {
				Type  string `json:"type"`
				Error struct {
					Type    string `json:"type"`
					Message string `json:"message"`
				} `json:"error"`
			}
			if err := jsoniter.UnmarshalFromString(jsonStr, &apiErr); err == nil && apiErr.Error.Message != "" {
				if handler != nil {
					handler(message.ChunkError, []byte(apiErr.Error.Message))
				}
			}
		}

		return http.HandlerReturnOk
	}

	// Log request
	if trace != nil {
		if requestBodyJSON, marshalErr := jsoniter.Marshal(requestBody); marshalErr == nil {
			trace.Debug("Anthropic Stream Request", map[string]any{
				"url":  url,
				"body": string(requestBodyJSON),
			})
		}
	}

	// Error buffer for non-SSE error responses
	var errorBuffer strings.Builder
	errorDetected := false

	wrappedHandler := func(data []byte) int {
		dataStr := string(data)
		trimmed := strings.TrimSpace(dataStr)

		if trimmed == "" {
			return http.HandlerReturnOk
		}

		// SSE event/data lines - pass to stream handler
		// Support both "event: type" (with space) and "event:type" (without space) formats
		if strings.HasPrefix(trimmed, "event:") || strings.HasPrefix(trimmed, "data:") {
			return streamHandler(data)
		}

		// Detect JSON error response
		if strings.HasPrefix(trimmed, "{") && strings.Contains(dataStr, `"error"`) {
			errorDetected = true
		}

		if errorDetected {
			errorBuffer.Write(data)
			errorBuffer.WriteString("\n")
			return http.HandlerReturnOk
		}

		return streamHandler(data)
	}

	// Make streaming request
	log.Trace("[LLM] Starting Anthropic Stream request: url=%s", url)
	err = req.Stream(goCtx, "POST", requestBody, wrappedHandler)
	_ = streamStartTime

	// Check for captured error response
	if errorDetected && errorBuffer.Len() > 0 {
		errorJSON := errorBuffer.String()
		if trace != nil {
			trace.Error(i18n.T(ctx.Locale, "llm.anthropic.stream.api_error"), map[string]any{"response": errorJSON})
		}

		var apiErr APIError
		if parseErr := jsoniter.UnmarshalFromString(errorJSON, &apiErr); parseErr == nil && apiErr.Error.Message != "" {
			err = fmt.Errorf("Anthropic API error: %s (type: %s)", apiErr.Error.Message, apiErr.Error.Type)
		} else {
			err = fmt.Errorf("Anthropic API error: %s", strings.TrimSpace(errorJSON))
		}
	}

	// Handle context cancellation
	if err != nil && goCtx.Err() != nil {
		return nil, fmt.Errorf("stream cancelled: %w", goCtx.Err())
	}

	if err != nil {
		endMessage(msgTracker, handler)
		if handler != nil {
			handler(message.ChunkError, []byte(err.Error()))
		}
		return nil, fmt.Errorf("streaming request failed: %w", err)
	}

	// Check for empty response
	if accumulator.id == "" {
		endMessage(msgTracker, handler)
		errMsg := fmt.Errorf("no data received from Anthropic API")
		if handler != nil {
			handler(message.ChunkError, []byte(errMsg.Error()))
		}
		return nil, errMsg
	}

	// Build final response (convert to unified CompletionResponse)
	response := &context.CompletionResponse{
		ID:               accumulator.id,
		Object:           "message",
		Model:            accumulator.model,
		Role:             accumulator.role,
		Content:          accumulator.content,
		ReasoningContent: accumulator.thinkingContent,
		FinishReason:     mapStopReason(accumulator.stopReason),
		Usage:            accumulator.usage,
	}

	// Convert accumulated tool calls
	if len(accumulator.toolCalls) > 0 {
		toolCalls := make([]context.ToolCall, 0, len(accumulator.toolCalls))
		for i := 0; i < len(accumulator.toolCalls); i++ {
			if tc, exists := accumulator.toolCalls[i]; exists {
				toolCalls = append(toolCalls, context.ToolCall{
					ID:   tc.id,
					Type: "function",
					Function: context.Function{
						Name:      tc.name,
						Arguments: tc.inputJSON,
					},
				})
			}
		}
		response.ToolCalls = toolCalls
	}

	endMessage(msgTracker, handler)
	return response, nil
}

// Post non-streaming completion request to Anthropic API
func (p *Provider) Post(ctx *context.Context, messages []context.Message, options *context.CompletionOptions) (*context.CompletionResponse, error) {
	trace, _ := ctx.Trace()

	maxRetries := 3
	var lastErr error

	goCtx := ctx.Context
	if ctx.Stack != nil && ctx.Stack.Options != nil && ctx.Stack.Options.Context != nil {
		goCtx = ctx.Stack.Options.Context
	}
	if goCtx == nil {
		goCtx = gocontext.Background()
	}

	currentMessages := make([]context.Message, len(messages))
	copy(currentMessages, messages)

	for attempt := 0; attempt < maxRetries; attempt++ {
		select {
		case <-goCtx.Done():
			return nil, fmt.Errorf("context cancelled: %w", goCtx.Err())
		default:
		}

		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			if trace != nil {
				trace.Warn("Anthropic post request failed, retrying", map[string]any{
					"backoff": backoff.String(),
					"attempt": attempt + 1,
					"error":   lastErr.Error(),
				})
			}
			timer := time.NewTimer(backoff)
			select {
			case <-timer.C:
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

		if !isRetryableError(err) {
			return nil, fmt.Errorf("non-retryable error: %w", err)
		}
	}

	return nil, fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

// postWithRetry performs a single POST request to Anthropic API
func (p *Provider) postWithRetry(ctx *context.Context, messages []context.Message, options *context.CompletionOptions) (*context.CompletionResponse, error) {
	trace, _ := ctx.Trace()

	// Preprocess through adapters
	processedMessages := messages
	processedOptions := options
	for _, adapter := range p.adapters {
		newMessages, err := adapter.PreprocessMessages(processedMessages)
		if err != nil {
			return nil, fmt.Errorf("adapter %s message preprocessing failed: %w", adapter.Name(), err)
		}
		processedMessages = newMessages

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

	version := "2023-06-01"
	if v, ok := setting["version"].(string); ok && v != "" {
		version = v
	}

	url := buildAPIURL(host, "/messages")

	// Create HTTP request
	req := http.New(url).
		SetHeader("Content-Type", "application/json").
		SetHeader("x-api-key", key).
		SetHeader("anthropic-version", version).
		SetHeader("User-Agent", "YaoAgent/1.0 (+https://yaoagents.com)")

	resp := req.Post(requestBody)
	if resp.Code != 200 {
		errorMsg := resp.Message
		if resp.Data != nil {
			if respJSON, err := jsoniter.Marshal(resp.Data); err == nil {
				if trace != nil {
					trace.Error(i18n.T(ctx.Locale, "llm.anthropic.post.api_error"), map[string]any{"response": string(respJSON)})
				}
				// Try to extract error message
				var apiErr APIError
				if err := jsoniter.Unmarshal(respJSON, &apiErr); err == nil && apiErr.Error.Message != "" {
					errorMsg = apiErr.Error.Message
				}
			}
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.Code, errorMsg)
	}

	// Parse response
	var fullResp NonStreamResponse
	respData, err := jsoniter.Marshal(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	if err := jsoniter.Unmarshal(respData, &fullResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract content from content blocks
	var content string
	var reasoningContent string
	var toolCalls []context.ToolCall

	for _, block := range fullResp.Content {
		switch block.Type {
		case "text":
			content += block.Text
		case "thinking":
			reasoningContent += block.Thinking
		case "tool_use":
			inputJSON := ""
			if block.Input != nil {
				if inputBytes, err := jsoniter.Marshal(block.Input); err == nil {
					inputJSON = string(inputBytes)
				}
			}
			toolCalls = append(toolCalls, context.ToolCall{
				ID:   block.ID,
				Type: "function",
				Function: context.Function{
					Name:      block.Name,
					Arguments: inputJSON,
				},
			})
		}
	}

	// Build unified response
	response := &context.CompletionResponse{
		ID:               fullResp.ID,
		Object:           "message",
		Model:            fullResp.Model,
		Role:             fullResp.Role,
		Content:          content,
		ReasoningContent: reasoningContent,
		ToolCalls:        toolCalls,
		FinishReason:     mapStopReason(fullResp.StopReason),
	}

	if fullResp.Usage != nil {
		response.Usage = &message.UsageInfo{
			PromptTokens:     fullResp.Usage.InputTokens,
			CompletionTokens: fullResp.Usage.OutputTokens,
			TotalTokens:      fullResp.Usage.InputTokens + fullResp.Usage.OutputTokens,
		}
	}

	return response, nil
}

// buildRequestBody builds the Anthropic Messages API request body
func (p *Provider) buildRequestBody(messages []context.Message, options *context.CompletionOptions, streaming bool) (map[string]interface{}, error) {
	if options == nil {
		return nil, fmt.Errorf("options are required")
	}

	setting := p.Connector.Setting()
	model, ok := setting["model"].(string)
	if !ok || model == "" {
		return nil, fmt.Errorf("model is not set in connector")
	}

	// Separate system messages from conversation messages
	var systemContent string
	var apiMessages []map[string]interface{}

	for _, msg := range messages {
		if msg.Role == "system" {
			// Anthropic: system prompt is a top-level field, not in messages
			if contentStr, ok := msg.Content.(string); ok {
				if systemContent != "" {
					systemContent += "\n\n"
				}
				systemContent += contentStr
			}
			continue
		}

		apiMsg := map[string]interface{}{
			"role": string(msg.Role),
		}

		// Handle content
		if msg.Content != nil {
			if parts, ok := msg.Content.([]context.ContentPart); ok {
				// Convert multimodal content parts to Anthropic format
				apiParts := make([]map[string]interface{}, 0, len(parts))
				for _, part := range parts {
					switch part.Type {
					case context.ContentText:
						apiParts = append(apiParts, map[string]interface{}{
							"type": "text",
							"text": part.Text,
						})
					case context.ContentImageURL:
						if part.ImageURL != nil {
							// Convert OpenAI image_url to Anthropic image format
							apiParts = append(apiParts, convertImagePart(part))
						}
					}
				}
				apiMsg["content"] = apiParts
			} else {
				apiMsg["content"] = msg.Content
			}
		}

		// Handle tool_result role (Anthropic uses different format)
		if msg.Role == "tool" && msg.ToolCallID != nil {
			apiMsg["role"] = "user"
			apiMsg["content"] = []map[string]interface{}{
				{
					"type":        "tool_result",
					"tool_use_id": *msg.ToolCallID,
					"content":     msg.Content,
				},
			}
		}

		// Handle assistant messages with tool_use
		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			contentBlocks := make([]map[string]interface{}, 0)

			// Add text content if present
			if contentStr, ok := msg.Content.(string); ok && contentStr != "" {
				contentBlocks = append(contentBlocks, map[string]interface{}{
					"type": "text",
					"text": contentStr,
				})
			}

			// Add tool_use blocks
			for _, tc := range msg.ToolCalls {
				var input interface{}
				if tc.Function.Arguments != "" {
					jsoniter.UnmarshalFromString(tc.Function.Arguments, &input)
				}
				if input == nil {
					input = map[string]interface{}{}
				}
				contentBlocks = append(contentBlocks, map[string]interface{}{
					"type":  "tool_use",
					"id":    tc.ID,
					"name":  tc.Function.Name,
					"input": input,
				})
			}

			apiMsg["content"] = contentBlocks
		}

		apiMessages = append(apiMessages, apiMsg)
	}

	// Build request body
	body := map[string]interface{}{
		"model": model,
	}

	if len(apiMessages) > 0 {
		body["messages"] = apiMessages
	}

	if systemContent != "" {
		body["system"] = systemContent
	}

	if streaming {
		body["stream"] = true
	}

	// max_tokens is required for Anthropic
	maxTokens := 4096 // default
	if options.MaxTokens != nil {
		maxTokens = *options.MaxTokens
	} else if options.MaxCompletionTokens != nil {
		maxTokens = *options.MaxCompletionTokens
	} else if mt, ok := setting["max_tokens"].(int); ok && mt > 0 {
		maxTokens = mt
	}
	body["max_tokens"] = maxTokens

	// Temperature
	if options.Temperature != nil {
		body["temperature"] = *options.Temperature
	}

	if options.TopP != nil {
		body["top_p"] = *options.TopP
	}

	if options.Stop != nil {
		body["stop_sequences"] = options.Stop
	}

	// Tools (convert from OpenAI format to Anthropic format)
	if len(options.Tools) > 0 {
		anthropicTools := convertTools(options.Tools)
		if len(anthropicTools) > 0 {
			body["tools"] = anthropicTools
		}
	}

	if options.ToolChoice != nil {
		body["tool_choice"] = convertToolChoice(options.ToolChoice)
	}

	// Thinking configuration from connector settings
	if thinking, exists := setting["thinking"]; exists && thinking != nil {
		body["thinking"] = thinking
	}

	return body, nil
}

// convertTools converts OpenAI-format tools to Anthropic format
func convertTools(tools []map[string]interface{}) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(tools))
	for _, tool := range tools {
		function, ok := tool["function"].(map[string]interface{})
		if !ok {
			continue
		}

		anthropicTool := map[string]interface{}{
			"name": function["name"],
		}
		if desc, ok := function["description"]; ok {
			anthropicTool["description"] = desc
		}
		if params, ok := function["parameters"]; ok {
			anthropicTool["input_schema"] = params
		}

		result = append(result, anthropicTool)
	}
	return result
}

// convertToolChoice converts OpenAI tool_choice to Anthropic format
func convertToolChoice(choice interface{}) interface{} {
	switch v := choice.(type) {
	case string:
		switch v {
		case "auto":
			return map[string]interface{}{"type": "auto"}
		case "none":
			return map[string]interface{}{"type": "none"}
		case "required":
			return map[string]interface{}{"type": "any"}
		}
	case map[string]interface{}:
		if fn, ok := v["function"].(map[string]interface{}); ok {
			if name, ok := fn["name"].(string); ok {
				return map[string]interface{}{
					"type": "tool",
					"name": name,
				}
			}
		}
	}
	return map[string]interface{}{"type": "auto"}
}

// convertImagePart converts an OpenAI image_url content part to Anthropic image format
func convertImagePart(part context.ContentPart) map[string]interface{} {
	if part.ImageURL == nil {
		return map[string]interface{}{"type": "text", "text": "[image not available]"}
	}

	url := part.ImageURL.URL

	// Check if it's a base64 data URL
	if strings.HasPrefix(url, "data:") {
		// Parse data URL: data:image/jpeg;base64,<data>
		parts := strings.SplitN(url, ",", 2)
		if len(parts) == 2 {
			mediaInfo := strings.TrimPrefix(parts[0], "data:")
			mediaInfo = strings.TrimSuffix(mediaInfo, ";base64")
			return map[string]interface{}{
				"type": "image",
				"source": map[string]interface{}{
					"type":       "base64",
					"media_type": mediaInfo,
					"data":       parts[1],
				},
			}
		}
	}

	// URL-based image (Anthropic supports URL images)
	return map[string]interface{}{
		"type": "image",
		"source": map[string]interface{}{
			"type": "url",
			"url":  url,
		},
	}
}

// buildAPIURL builds the API URL for Anthropic
func buildAPIURL(host, endpoint string) string {
	return connector.BuildAPIURL(host, endpoint)
}

// mapStopReason maps Anthropic stop_reason to OpenAI finish_reason
func mapStopReason(stopReason string) string {
	switch stopReason {
	case "end_turn":
		return "stop"
	case "max_tokens":
		return "length"
	case "tool_use":
		return "tool_calls"
	case "stop_sequence":
		return "stop"
	default:
		return stopReason
	}
}

// Message tracker helper functions

func startMessage(mt *messageTracker, messageType message.StreamChunkType, handler message.StreamFunc) {
	if mt.active {
		endMessage(mt, handler)
	}

	mt.active = true
	if mt.idGenerator != nil {
		mt.messageID = mt.idGenerator.GenerateMessageID()
	} else {
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

func startToolCallMessage(mt *messageTracker, toolCallInfo *message.EventToolCallInfo, handler message.StreamFunc) {
	if mt.active {
		endMessage(mt, handler)
	}

	mt.active = true
	if mt.idGenerator != nil {
		mt.messageID = mt.idGenerator.GenerateMessageID()
	} else {
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

func incrementChunk(mt *messageTracker) {
	if mt.active {
		mt.chunkCount++
	}
}

func endMessage(mt *messageTracker, handler message.StreamFunc) {
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

// isRetryableError checks if an error is retryable
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	retryablePatterns := []string{
		"timeout",
		"connection refused",
		"connection reset",
		"EOF",
		"HTTP 429",
		"HTTP 500",
		"HTTP 502",
		"HTTP 503",
		"HTTP 504",
		"overloaded",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(strings.ToLower(errStr), strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}
