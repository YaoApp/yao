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
	"github.com/yaoapp/yao/agent/llm/providers/base"
	"github.com/yaoapp/yao/utils/jsonschema"
)

// Provider OpenAI-compatible provider
// Supports: vision, tool calls, streaming, JSON mode
type Provider struct {
	*base.Provider
}

// New create a new OpenAI provider
func New(conn connector.Connector, capabilities *context.ModelCapabilities) *Provider {
	return &Provider{
		Provider: base.NewProvider(conn, capabilities),
	}
}

// Stream stream completion from OpenAI API
func (p *Provider) Stream(ctx *context.Context, messages []context.Message, options *context.CompletionOptions, handler context.StreamFunc) (*context.CompletionResponse, error) {
	maxRetries := 3
	maxValidationRetries := 3
	var lastErr error

	// Make a copy of messages to avoid modifying the original
	currentMessages := make([]context.Message, len(messages))
	copy(currentMessages, messages)

	// Outer loop: handle network/API errors with exponential backoff
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			log.Warn("OpenAI stream request failed, retrying in %v (attempt %d/%d): %v", backoff, attempt+1, maxRetries, lastErr)
			time.Sleep(backoff)
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
	// Build request body
	requestBody, err := p.buildRequestBody(messages, options, true)
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
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", key)).
		SetHeader("Accept", "text/event-stream")

	// Accumulate response data
	accumulator := &streamAccumulator{
		toolCalls: make(map[int]*accumulatedToolCall),
	}

	// Stream handler
	streamHandler := func(data []byte) int {
		if len(data) == 0 {
			return http.HandlerReturnOk
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

			// Handle content
			if delta.Content != "" {
				accumulator.content += delta.Content
				if handler != nil {
					handler(context.ChunkText, []byte(delta.Content))
				}
			}

			// Handle refusal
			if delta.Refusal != "" {
				accumulator.refusal += delta.Refusal
				if handler != nil {
					handler(context.ChunkRefusal, []byte(delta.Refusal))
				}
			}

			// Handle tool calls
			if len(delta.ToolCalls) > 0 {
				for _, tc := range delta.ToolCalls {
					if _, exists := accumulator.toolCalls[tc.Index]; !exists {
						accumulator.toolCalls[tc.Index] = &accumulatedToolCall{}
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
					}
					if tc.Function.Arguments != "" {
						accTC.functionArgs += tc.Function.Arguments
					}
				}

				// Notify handler of tool call progress
				if handler != nil {
					toolCallData, _ := jsoniter.Marshal(delta.ToolCalls)
					handler(context.ChunkToolCall, toolCallData)
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

	// Make streaming request
	goCtx := ctx.Context
	if goCtx == nil {
		goCtx = gocontext.Background()
	}

	err = req.Stream(goCtx, "POST", requestBody, streamHandler)
	if err != nil {
		// Notify handler of error if provided
		if handler != nil {
			errData := []byte(err.Error())
			handler(context.ChunkError, errData)
		}
		return nil, fmt.Errorf("streaming request failed: %w", err)
	}

	// Check if we received any data
	if accumulator.id == "" {
		log.Warn("OpenAI stream completed but no data was received (accumulator.id is empty)")
		err := fmt.Errorf("no data received from OpenAI API")
		// Notify handler of error if provided
		if handler != nil {
			errData := []byte(err.Error())
			handler(context.ChunkError, errData)
		}
		return nil, err
	}

	// Build final response
	response := &context.CompletionResponse{
		ID:           accumulator.id,
		Object:       "chat.completion",
		Created:      accumulator.created,
		Model:        accumulator.model,
		Role:         accumulator.role,
		Content:      accumulator.content,
		Refusal:      accumulator.refusal,
		FinishReason: accumulator.finishReason,
		Usage:        accumulator.usage,
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
			// Tool call validation failed, need to retry with error feedback
			return nil, fmt.Errorf("tool call validation failed: %w", err)
		}
	}

	return response, nil
}

// Post post completion request to OpenAI API
func (p *Provider) Post(ctx *context.Context, messages []context.Message, options *context.CompletionOptions) (*context.CompletionResponse, error) {
	maxRetries := 3
	maxValidationRetries := 3
	var lastErr error

	// Make a copy of messages to avoid modifying the original
	currentMessages := make([]context.Message, len(messages))
	copy(currentMessages, messages)

	// Outer loop: handle network/API errors with exponential backoff
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			log.Warn("OpenAI post request failed, retrying in %v (attempt %d/%d): %v", backoff, attempt+1, maxRetries, lastErr)
			time.Sleep(backoff)
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
	// Build request body
	requestBody, err := p.buildRequestBody(messages, options, false)
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
	response := &context.CompletionResponse{
		ID:                fullResp.ID,
		Object:            fullResp.Object,
		Created:           fullResp.Created,
		Model:             fullResp.Model,
		Role:              string(choice.Message.Role),
		Content:           choice.Message.Content,
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

	if options.MaxCompletionTokens != nil {
		body["max_completion_tokens"] = *options.MaxCompletionTokens
	} else if options.MaxTokens != nil {
		body["max_tokens"] = *options.MaxTokens
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
		body["response_format"] = options.ResponseFormat
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
