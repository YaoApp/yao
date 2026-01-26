// Package llm provides the LLM JSAPI implementation
package llm

import (
	"fmt"

	"github.com/yaoapp/gou/connector"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
)

// JSAPI implements LlmAPI interface for ctx.llm.* methods
type JSAPI struct {
	ctx *agentContext.Context
}

// Ensure JSAPI implements both interfaces
var _ agentContext.LlmAPI = (*JSAPI)(nil)
var _ agentContext.LlmAPIWithCallback = (*JSAPI)(nil)

// NewJSAPI creates a new JSAPI for the given context
func NewJSAPI(ctx *agentContext.Context) *JSAPI {
	return &JSAPI{ctx: ctx}
}

// SetJSAPIFactory registers the JSAPI factory with the context package
// This should be called during initialization
func SetJSAPIFactory() {
	agentContext.LlmAPIFactory = func(ctx *agentContext.Context) agentContext.LlmAPI {
		return NewJSAPI(ctx)
	}
}

// Stream implements LlmAPI.Stream - calls LLM with streaming output to ctx.Writer
func (api *JSAPI) Stream(connectorID string, messages []interface{}, opts map[string]interface{}) interface{} {
	return api.StreamWithHandler(connectorID, messages, opts, nil)
}

// StreamWithHandler implements LlmAPIWithCallback.StreamWithHandler - calls LLM with OnMessage handler
func (api *JSAPI) StreamWithHandler(connectorID string, messages []interface{}, opts map[string]interface{}, handler agentContext.OnMessageFunc) interface{} {
	result := &Result{
		Connector: connectorID,
	}

	// Validate context
	if api.ctx == nil {
		result.Error = "context is nil"
		return result
	}

	// Get connector
	conn, err := connector.Select(connectorID)
	if err != nil {
		result.Error = fmt.Sprintf("failed to select connector %s: %v", connectorID, err)
		return result
	}

	// Parse messages to context.Message format
	ctxMessages, err := parseMessages(messages)
	if err != nil {
		result.Error = fmt.Sprintf("failed to parse messages: %v", err)
		return result
	}

	// Build CompletionOptions from opts
	completionOptions := buildCompletionOptions(conn, opts)

	// Create LLM instance
	llmInstance, err := New(conn, completionOptions)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create LLM instance: %v", err)
		return result
	}

	// Create stream handler with the provided callback
	// Note: We pass handler directly to the stream handler instead of setting ctx.Stack.Options.OnMessage
	// This avoids race conditions in concurrent batch calls where multiple goroutines
	// would otherwise overwrite the same ctx.Stack.Options.OnMessage
	streamHandler := createStreamHandlerWithCallback(api.ctx, handler)

	// Execute LLM stream call
	response, err := llmInstance.Stream(api.ctx, ctxMessages, completionOptions, streamHandler)
	if err != nil {
		result.Error = fmt.Sprintf("LLM stream failed: %v", err)
		return result
	}

	// Set response
	result.Response = response

	// Extract text content from response
	if response != nil {
		result.Content = extractContent(response)
	}

	return result
}

// createStreamHandlerWithCallback creates a stream handler that uses the provided callback directly
// This is used instead of setting ctx.Stack.Options.OnMessage to avoid race conditions
// in concurrent batch calls
func createStreamHandlerWithCallback(ctx *agentContext.Context, handler agentContext.OnMessageFunc) message.StreamFunc {
	// Handle nil context
	if ctx == nil {
		return func(chunkType message.StreamChunkType, data []byte) int {
			return 0 // No-op handler when context is nil
		}
	}

	// Stream state for tracking message groups
	state := &streamState{
		ctx:     ctx,
		buffer:  []byte{},
		handler: handler, // Store the handler directly in state
	}

	return func(chunkType message.StreamChunkType, data []byte) int {
		return state.handleChunk(chunkType, data)
	}
}

// parseMessages converts JS message array to context.Message slice
func parseMessages(messages []interface{}) ([]agentContext.Message, error) {
	result := make([]agentContext.Message, 0, len(messages))

	for i, msg := range messages {
		msgMap, ok := msg.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("message %d is not an object", i)
		}

		ctxMsg := agentContext.Message{}

		// Required: role
		if role, ok := msgMap["role"].(string); ok {
			ctxMsg.Role = agentContext.MessageRole(role)
		} else {
			return nil, fmt.Errorf("message %d missing role", i)
		}

		// Optional: content (can be string or array for multimodal)
		if content, ok := msgMap["content"]; ok {
			ctxMsg.Content = content
		}

		// Optional: name
		if name, ok := msgMap["name"].(string); ok {
			ctxMsg.Name = &name
		}

		// Optional: tool_calls
		if toolCalls, ok := msgMap["tool_calls"]; ok {
			if tcArray, ok := toolCalls.([]interface{}); ok {
				ctxMsg.ToolCalls = parseToolCalls(tcArray)
			}
		}

		// Optional: tool_call_id (for tool response messages)
		if toolCallID, ok := msgMap["tool_call_id"].(string); ok {
			ctxMsg.ToolCallID = &toolCallID
		}

		result = append(result, ctxMsg)
	}

	return result, nil
}

// parseToolCalls converts JS tool_calls array to context.ToolCall slice
func parseToolCalls(toolCalls []interface{}) []agentContext.ToolCall {
	result := make([]agentContext.ToolCall, 0, len(toolCalls))

	for _, tc := range toolCalls {
		tcMap, ok := tc.(map[string]interface{})
		if !ok {
			continue
		}

		toolCall := agentContext.ToolCall{}

		if id, ok := tcMap["id"].(string); ok {
			toolCall.ID = id
		}
		if typ, ok := tcMap["type"].(string); ok {
			toolCall.Type = agentContext.ToolCallType(typ)
		}
		if fn, ok := tcMap["function"].(map[string]interface{}); ok {
			toolCall.Function = agentContext.Function{}
			if name, ok := fn["name"].(string); ok {
				toolCall.Function.Name = name
			}
			if args, ok := fn["arguments"].(string); ok {
				toolCall.Function.Arguments = args
			}
		}

		result = append(result, toolCall)
	}

	return result
}

// buildCompletionOptions creates CompletionOptions from JS opts map
func buildCompletionOptions(conn connector.Connector, opts map[string]interface{}) *agentContext.CompletionOptions {
	// Get capabilities from connector
	capabilities := GetCapabilitiesFromConn(conn, nil)

	completionOptions := &agentContext.CompletionOptions{
		Capabilities: capabilities,
	}

	if opts == nil {
		return completionOptions
	}

	// Temperature
	if temp, ok := opts["temperature"].(float64); ok {
		completionOptions.Temperature = &temp
	}

	// Max tokens
	if maxTokens, ok := opts["max_tokens"].(float64); ok {
		mt := int(maxTokens)
		completionOptions.MaxTokens = &mt
	}
	if maxCompletionTokens, ok := opts["max_completion_tokens"].(float64); ok {
		mct := int(maxCompletionTokens)
		completionOptions.MaxCompletionTokens = &mct
	}

	// Top P
	if topP, ok := opts["top_p"].(float64); ok {
		completionOptions.TopP = &topP
	}

	// Presence penalty
	if presencePenalty, ok := opts["presence_penalty"].(float64); ok {
		completionOptions.PresencePenalty = &presencePenalty
	}

	// Frequency penalty
	if frequencyPenalty, ok := opts["frequency_penalty"].(float64); ok {
		completionOptions.FrequencyPenalty = &frequencyPenalty
	}

	// Stop sequences
	if stop, ok := opts["stop"]; ok {
		completionOptions.Stop = stop
	}

	// User
	if user, ok := opts["user"].(string); ok {
		completionOptions.User = user
	}

	// Seed
	if seed, ok := opts["seed"].(float64); ok {
		s := int(seed)
		completionOptions.Seed = &s
	}

	// Tools
	if tools, ok := opts["tools"].([]interface{}); ok {
		completionOptions.Tools = make([]map[string]interface{}, 0, len(tools))
		for _, tool := range tools {
			if toolMap, ok := tool.(map[string]interface{}); ok {
				completionOptions.Tools = append(completionOptions.Tools, toolMap)
			}
		}
	}

	// Tool choice
	if toolChoice, ok := opts["tool_choice"]; ok {
		completionOptions.ToolChoice = toolChoice
	}

	// Response format
	if responseFormat, ok := opts["response_format"].(map[string]interface{}); ok {
		rf := &agentContext.ResponseFormat{}
		if rfType, ok := responseFormat["type"].(string); ok {
			rf.Type = agentContext.ResponseFormatType(rfType)
		}
		if jsonSchema, ok := responseFormat["json_schema"].(map[string]interface{}); ok {
			rf.JSONSchema = &agentContext.JSONSchema{}
			if name, ok := jsonSchema["name"].(string); ok {
				rf.JSONSchema.Name = name
			}
			if desc, ok := jsonSchema["description"].(string); ok {
				rf.JSONSchema.Description = desc
			}
			if schema, ok := jsonSchema["schema"]; ok {
				rf.JSONSchema.Schema = schema
			}
			if strict, ok := jsonSchema["strict"].(bool); ok {
				rf.JSONSchema.Strict = &strict
			}
		}
		completionOptions.ResponseFormat = rf
	}

	// Reasoning effort (for reasoning models)
	if reasoningEffort, ok := opts["reasoning_effort"].(string); ok {
		completionOptions.ReasoningEffort = &reasoningEffort
	}

	return completionOptions
}

// streamState manages stream handler state
type streamState struct {
	ctx            *agentContext.Context
	inMessage      bool
	currentMsgID   string
	currentMsgType string
	buffer         []byte
	msgCounter     int                        // Counter for generating message IDs when IDGenerator is nil
	chunkCounter   int                        // Counter for generating chunk IDs when IDGenerator is nil
	handler        agentContext.OnMessageFunc // Direct handler reference (avoids race condition via ctx.Stack.Options)
}

// generateMessageID generates a unique message ID
func (s *streamState) generateMessageID() string {
	if s.ctx != nil && s.ctx.IDGenerator != nil {
		return s.ctx.IDGenerator.GenerateMessageID()
	}
	s.msgCounter++
	return fmt.Sprintf("M%d", s.msgCounter)
}

// generateChunkID generates a unique chunk ID
func (s *streamState) generateChunkID() string {
	if s.ctx != nil && s.ctx.IDGenerator != nil {
		return s.ctx.IDGenerator.GenerateChunkID()
	}
	s.chunkCounter++
	return fmt.Sprintf("C%d", s.chunkCounter)
}

// handleChunk processes a single stream chunk
func (s *streamState) handleChunk(chunkType message.StreamChunkType, data []byte) int {
	switch chunkType {
	case message.ChunkMessageStart:
		s.inMessage = true
		s.currentMsgID = s.generateMessageID()
		s.buffer = []byte{}
		return 0

	case message.ChunkText:
		if !s.inMessage {
			s.inMessage = true
			s.currentMsgID = s.generateMessageID()
		}
		s.currentMsgType = message.TypeText
		s.buffer = append(s.buffer, data...)

		// Create message
		msg := &message.Message{
			ChunkID:   s.generateChunkID(),
			MessageID: s.currentMsgID,
			Type:      message.TypeText,
			Delta:     true,
			Props: map[string]interface{}{
				"content": string(data),
			},
		}

		// Call handler directly if provided (for batch calls and single calls with callback)
		// We use direct handler instead of ctx.Stack.Options.OnMessage to avoid race conditions
		// in concurrent batch calls where multiple goroutines would overwrite the shared OnMessage
		if s.handler != nil {
			if ret := s.handler(msg); ret != 0 {
				return ret
			}
		}

		// Send to output for actual message delivery to client
		// Note: ctx.Send may also call ctx.Stack.Options.OnMessage if set (for agent calls),
		// but for LLM calls we don't set OnMessage, so no double callback occurs
		if err := s.ctx.Send(msg); err != nil {
			// Log error but continue streaming
			return 0
		}
		return 0

	case message.ChunkThinking:
		if !s.inMessage {
			s.inMessage = true
			s.currentMsgID = s.generateMessageID()
		}
		s.currentMsgType = message.TypeThinking
		s.buffer = append(s.buffer, data...)

		msg := &message.Message{
			ChunkID:   s.generateChunkID(),
			MessageID: s.currentMsgID,
			Type:      message.TypeThinking,
			Delta:     true,
			Props: map[string]interface{}{
				"content": string(data),
			},
		}

		// Call handler directly if provided
		if s.handler != nil {
			if ret := s.handler(msg); ret != 0 {
				return ret
			}
		}

		if err := s.ctx.Send(msg); err != nil {
			return 0
		}
		return 0

	case message.ChunkToolCall:
		if !s.inMessage {
			s.inMessage = true
			s.currentMsgID = s.generateMessageID()
		}
		s.currentMsgType = message.TypeToolCall
		s.buffer = append(s.buffer, data...)

		// Tool call chunks are more complex - parse and forward
		msg := &message.Message{
			ChunkID:   s.generateChunkID(),
			MessageID: s.currentMsgID,
			Type:      message.TypeToolCall,
			Delta:     true,
			Props: map[string]interface{}{
				"raw": string(data),
			},
		}

		// Call handler directly if provided
		if s.handler != nil {
			if ret := s.handler(msg); ret != 0 {
				return ret
			}
		}

		if err := s.ctx.Send(msg); err != nil {
			return 0
		}
		return 0

	case message.ChunkMessageEnd:
		if s.inMessage {
			s.inMessage = false
			s.currentMsgID = ""
			s.buffer = []byte{}
		}
		return 0

	case message.ChunkError:
		// Send error and stop
		msg := &message.Message{
			Type: message.TypeError,
			Props: map[string]interface{}{
				"error": string(data),
			},
		}

		// Call handler directly if provided
		if s.handler != nil {
			s.handler(msg)
		}

		_ = s.ctx.Send(msg) // Ignore error on error message
		return 1            // Stop on error

	default:
		// Other chunk types (stream_start, stream_end, metadata) - ignore
		return 0
	}
}

// extractContent extracts text content from CompletionResponse
func extractContent(response *agentContext.CompletionResponse) string {
	if response == nil || response.Content == nil {
		return ""
	}

	switch content := response.Content.(type) {
	case string:
		return content
	case []interface{}:
		// Multimodal response - extract text parts
		var text string
		for _, part := range content {
			if partMap, ok := part.(map[string]interface{}); ok {
				if partMap["type"] == "text" {
					if t, ok := partMap["text"].(string); ok {
						text += t
					}
				}
			}
		}
		return text
	default:
		return ""
	}
}

// ============================================================================
// Batch LLM Methods: All, Any, Race
// ============================================================================

// All executes all LLM requests concurrently and returns all results
func (api *JSAPI) All(requests []interface{}) []interface{} {
	return api.AllWithHandler(requests, nil)
}

// Any executes LLM requests concurrently and returns first successful result
func (api *JSAPI) Any(requests []interface{}) []interface{} {
	return api.AnyWithHandler(requests, nil)
}

// Race executes LLM requests concurrently and returns first completed result
func (api *JSAPI) Race(requests []interface{}) []interface{} {
	return api.RaceWithHandler(requests, nil)
}

// AllWithHandler executes all LLM requests with global handler
func (api *JSAPI) AllWithHandler(requests []interface{}, globalHandler agentContext.LlmBatchOnMessageFunc) []interface{} {
	parsedRequests := api.parseRequests(requests, globalHandler)
	return api.executeAll(parsedRequests)
}

// AnyWithHandler executes LLM requests and returns first success with handler
func (api *JSAPI) AnyWithHandler(requests []interface{}, globalHandler agentContext.LlmBatchOnMessageFunc) []interface{} {
	parsedRequests := api.parseRequests(requests, globalHandler)
	return api.executeAny(parsedRequests)
}

// RaceWithHandler executes LLM requests and returns first completion with handler
func (api *JSAPI) RaceWithHandler(requests []interface{}, globalHandler agentContext.LlmBatchOnMessageFunc) []interface{} {
	parsedRequests := api.parseRequests(requests, globalHandler)
	return api.executeRace(parsedRequests)
}

// parseRequests converts JS request array to internal Request slice
func (api *JSAPI) parseRequests(requests []interface{}, globalHandler agentContext.LlmBatchOnMessageFunc) []*Request {
	result := make([]*Request, 0, len(requests))

	for i, req := range requests {
		reqMap, ok := req.(map[string]interface{})
		if !ok {
			continue
		}

		request := &Request{}

		// Required: connector
		if connector, ok := reqMap["connector"].(string); ok {
			request.Connector = connector
		} else {
			continue // Skip invalid request
		}

		// Required: messages
		if messages, ok := reqMap["messages"].([]interface{}); ok {
			request.Messages = messages
		} else {
			continue // Skip invalid request
		}

		// Optional: options
		if options, ok := reqMap["options"].(map[string]interface{}); ok {
			// Remove onChunk from options if present (handled via globalHandler)
			delete(options, "onChunk")
			request.Options = options
		}

		// Set handler based on globalHandler
		if globalHandler != nil {
			index := i
			connectorID := request.Connector
			request.Handler = func(msg *message.Message) int {
				return globalHandler(connectorID, index, msg)
			}
		}

		result = append(result, request)
	}

	return result
}

// executeAll executes all requests concurrently and waits for all to complete
// Each request uses a forked context to avoid race conditions on shared state
func (api *JSAPI) executeAll(requests []*Request) []interface{} {
	if len(requests) == 0 {
		return []interface{}{}
	}

	results := make([]interface{}, len(requests))
	done := make(chan struct{})
	remaining := len(requests)

	for i, req := range requests {
		go func(index int, request *Request) {
			defer func() {
				if err := recover(); err != nil {
					results[index] = &Result{
						Connector: request.Connector,
						Error:     fmt.Sprintf("panic: %v", err),
					}
				}
				done <- struct{}{}
			}()

			// Use forked context to avoid race conditions
			results[index] = api.executeSingleRequestWithForkedContext(request)
		}(i, req)
	}

	// Wait for all to complete
	for remaining > 0 {
		<-done
		remaining--
	}

	return results
}

// executeAny executes requests and returns first successful result
// Each request uses a forked context to avoid race conditions on shared state
func (api *JSAPI) executeAny(requests []*Request) []interface{} {
	if len(requests) == 0 {
		return []interface{}{}
	}

	type indexedResult struct {
		index  int
		result *Result
	}

	resultChan := make(chan indexedResult, len(requests))
	remaining := len(requests)

	for i, req := range requests {
		go func(index int, request *Request) {
			defer func() {
				if err := recover(); err != nil {
					resultChan <- indexedResult{
						index: index,
						result: &Result{
							Connector: request.Connector,
							Error:     fmt.Sprintf("panic: %v", err),
						},
					}
				}
			}()

			// Use forked context to avoid race conditions
			res := api.executeSingleRequestWithForkedContext(request)
			resultChan <- indexedResult{index: index, result: res.(*Result)}
		}(i, req)
	}

	// Wait for first success or all failures
	var firstSuccess *indexedResult
	errors := make([]*indexedResult, 0)

	for remaining > 0 {
		ir := <-resultChan
		remaining--

		if ir.result.Error == "" {
			// Success!
			firstSuccess = &ir
			break
		}
		errors = append(errors, &ir)
	}

	// Drain remaining results in background (don't block)
	if remaining > 0 {
		go func(count int) {
			for i := 0; i < count; i++ {
				<-resultChan
			}
		}(remaining)
	}

	if firstSuccess != nil {
		return []interface{}{firstSuccess.result}
	}

	// All failed - return all errors
	results := make([]interface{}, len(errors))
	for i, e := range errors {
		results[i] = e.result
	}
	return results
}

// executeRace executes requests and returns first completed result (success or failure)
// Each request uses a forked context to avoid race conditions on shared state
func (api *JSAPI) executeRace(requests []*Request) []interface{} {
	if len(requests) == 0 {
		return []interface{}{}
	}

	resultChan := make(chan *Result, len(requests))

	for _, req := range requests {
		go func(request *Request) {
			defer func() {
				if err := recover(); err != nil {
					resultChan <- &Result{
						Connector: request.Connector,
						Error:     fmt.Sprintf("panic: %v", err),
					}
				}
			}()

			// Use forked context to avoid race conditions
			res := api.executeSingleRequestWithForkedContext(request)
			resultChan <- res.(*Result)
		}(req)
	}

	// Return first result
	result := <-resultChan

	// Drain remaining results in background (don't block)
	remaining := len(requests) - 1
	if remaining > 0 {
		go func(count int) {
			for i := 0; i < count; i++ {
				<-resultChan
			}
		}(remaining)
	}

	return []interface{}{result}
}

// executeSingleRequest executes a single LLM request using the original context
// This is used for single calls (not batch)
func (api *JSAPI) executeSingleRequest(request *Request) interface{} {
	return api.StreamWithHandler(request.Connector, request.Messages, request.Options, request.Handler)
}

// executeSingleRequestWithForkedContext executes a single LLM request with a forked context
// This is used by batch operations (All/Any/Race) to avoid race conditions
// when multiple goroutines access shared context state
func (api *JSAPI) executeSingleRequestWithForkedContext(request *Request) interface{} {
	// Fork the context to get independent resources (IDGenerator, Logger, etc.)
	forkedCtx := api.ctx.Fork()

	// Create a temporary JSAPI with the forked context
	forkedAPI := &JSAPI{ctx: forkedCtx}

	return forkedAPI.StreamWithHandler(request.Connector, request.Messages, request.Options, request.Handler)
}
