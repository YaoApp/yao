package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/yaoapp/gou/connector"
	gouHTTP "github.com/yaoapp/gou/http"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
)

func init() {
	process.Register("llm.ChatCompletions", ProcessChatCompletions)
}

// ProcessChatCompletions implements the llm.ChatCompletions Process.
// A universal replacement for openai.chat.Completions that auto-detects
// connector type (openai, anthropic, etc.) and routes accordingly.
//
// Usage:
//
//	Process("llm.ChatCompletions", connector, messages)
//	Process("llm.ChatCompletions", connector, messages, opts)
//	Process("llm.ChatCompletions", connector, messages, opts, callback)
//
// Args:
//   - connector (string): Connector ID, any type (openai / anthropic / ...)
//   - messages  ([]map):  Message array, supports multimodal content (image_url, etc.)
//   - opts      (map):    Optional. temperature, max_tokens, etc.
//   - callback  (func):   Optional. Streaming callback func(data []byte) int
//
// Returns: OpenAI-compatible format { choices: [{ message: { role, content } }], ... }
func ProcessChatCompletions(p *process.Process) interface{} {
	p.ValidateArgNums(2)

	// 1. Parse connector ID
	connectorID := p.ArgsString(0)
	if connectorID == "" {
		return newErrorResponse("llm.ChatCompletions: connector is required")
	}

	// 2. Parse messages
	rawMessages := p.ArgsArray(1)
	messages := make([]map[string]interface{}, 0, len(rawMessages))
	for i, v := range rawMessages {
		msg, ok := v.(map[string]interface{})
		if !ok {
			return newErrorResponse(fmt.Sprintf("llm.ChatCompletions: message %d is not an object", i))
		}
		messages = append(messages, msg)
	}

	// 3. Parse optional opts
	var opts map[string]interface{}
	if p.NumOfArgs() > 2 && p.Args[2] != nil {
		if o, ok := p.Args[2].(map[string]interface{}); ok {
			opts = o
		}
	}

	// 4. Parse optional callback (for streaming)
	var callback func(data []byte) int
	if p.NumOfArgs() > 3 && p.Args[3] != nil {
		switch cb := p.Args[3].(type) {
		case func(data []byte) int:
			callback = cb
		case bridge.FunctionT:
			callback = func(data []byte) int {
				v, err := cb.Call(string(data))
				if err != nil {
					return gouHTTP.HandlerReturnError
				}
				ret, ok := v.(int)
				if !ok {
					return gouHTTP.HandlerReturnError
				}
				return ret
			}
		}
	}

	// 5. Select connector
	conn, err := connector.Select(connectorID)
	if err != nil {
		return newErrorResponse(fmt.Sprintf("llm.ChatCompletions: connector %s not found: %v", connectorID, err))
	}

	// 6. Build completion options (reuse jsapi.go logic)
	completionOptions := buildCompletionOptions(conn, opts)

	// 7. Create LLM instance (auto-selects openai/anthropic provider)
	llmInstance, err := New(conn, completionOptions)
	if err != nil {
		return newErrorResponse(fmt.Sprintf("llm.ChatCompletions: failed to create LLM: %v", err))
	}

	// 8. Parse messages to context.Message format (reuse jsapi.go logic)
	interfaceMessages := make([]interface{}, len(messages))
	for i, m := range messages {
		interfaceMessages[i] = m
	}
	ctxMessages, err := parseMessages(interfaceMessages)
	if err != nil {
		return newErrorResponse(fmt.Sprintf("llm.ChatCompletions: invalid messages: %v", err))
	}

	// 8.1 Normalize multimodal content: convert []interface{} maps to []ContentPart
	//     so that providers (especially Anthropic) can type-assert correctly.
	for i := range ctxMessages {
		if parts, ok := ctxMessages[i].Content.([]interface{}); ok {
			ctxMessages[i].Content = normalizeContentParts(parts)
		}
	}

	// 9. Build a minimal headless context for LLM call
	parent := p.Context
	if parent == nil {
		parent = context.Background()
	}
	authInfo := authorized.ProcessAuthInfo(p)
	chatID := agentContext.GenChatID()
	ctx := agentContext.New(parent, authInfo, chatID)
	defer ctx.Release()

	// 10. Create stream handler
	var streamHandler message.StreamFunc
	if callback != nil {
		// With callback: forward raw chunks to caller
		streamHandler = func(chunkType message.StreamChunkType, data []byte) int {
			if chunkType == message.ChunkText || chunkType == message.ChunkThinking {
				return callback(data)
			}
			return 0
		}
	} else {
		// No callback: no-op handler, just collect final response
		streamHandler = func(chunkType message.StreamChunkType, data []byte) int {
			return 0
		}
	}

	// 11. Execute LLM stream call
	response, err := llmInstance.Stream(ctx, ctxMessages, completionOptions, streamHandler)
	if err != nil {
		return newErrorResponse(fmt.Sprintf("llm.ChatCompletions: LLM call failed: %v", err))
	}

	// 12. Convert CompletionResponse to OpenAI-compatible format
	//     { choices: [{ message: { role, content } }], id, model, ... }
	return toOpenAIFormat(response)
}

// toOpenAIFormat converts CompletionResponse to OpenAI chat.completions format
// for backward compatibility with code that consumed openai.chat.Completions.
func toOpenAIFormat(resp *agentContext.CompletionResponse) map[string]interface{} {
	if resp == nil {
		return map[string]interface{}{
			"choices": []interface{}{},
		}
	}

	msgMap := map[string]interface{}{
		"role":    resp.Role,
		"content": resp.Content,
	}
	if len(resp.ToolCalls) > 0 {
		msgMap["tool_calls"] = resp.ToolCalls
	}

	choice := map[string]interface{}{
		"index":         0,
		"message":       msgMap,
		"finish_reason": "stop",
	}

	result := map[string]interface{}{
		"id":      resp.ID,
		"object":  "chat.completion",
		"created": resp.Created,
		"model":   resp.Model,
		"choices": []interface{}{choice},
	}

	if resp.Usage != nil {
		result["usage"] = resp.Usage
	}

	return result
}

// newErrorResponse creates an error response in OpenAI-compatible format
func newErrorResponse(errMsg string) map[string]interface{} {
	return map[string]interface{}{
		"error": map[string]interface{}{
			"message": errMsg,
			"type":    "invalid_request_error",
		},
	}
}

// normalizeContentParts converts []interface{} (raw maps from Process args)
// to []agentContext.ContentPart (strongly typed) via JSON round-trip.
// This is essential for providers (e.g. Anthropic) that type-assert on
// []ContentPart to apply format-specific conversions (image_url → image).
func normalizeContentParts(parts []interface{}) []agentContext.ContentPart {
	raw, err := json.Marshal(parts)
	if err != nil {
		return nil
	}
	var typed []agentContext.ContentPart
	if err := json.Unmarshal(raw, &typed); err != nil {
		return nil
	}
	return typed
}
