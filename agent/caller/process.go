package caller

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
)

func init() {
	process.Register("agent.Call", processAgentCall)
}

// processAgentCall implements the agent.Call Process handler.
// Enables agent-to-agent calls from contexts without agent.Context (e.g., YaoJob).
//
// Usage: Process("agent.Call", { assistant_id, messages, model?, ... })
// Returns: *Result (same structure as ctx.agent.Call in JSAPI)
func processAgentCall(p *process.Process) interface{} {

	// 1. Parse parameters via struct — fail fast on invalid input
	if len(p.Args) == 0 {
		exception.New("agent.Call: argument is required", 400).Throw()
	}

	var req ProcessCallRequest
	raw, err := json.Marshal(p.Args[0])
	if err != nil {
		exception.New("agent.Call: invalid argument: %s", 400, err.Error()).Throw()
	}
	if err := json.Unmarshal(raw, &req); err != nil {
		exception.New("agent.Call: failed to parse request: %s", 400, err.Error()).Throw()
	}

	if req.AssistantID == "" {
		exception.New("agent.Call: assistant_id is required", 400).Throw()
	}
	if len(req.Messages) == 0 {
		exception.New("agent.Call: messages is required", 400).Throw()
	}

	// 2. Auto-inject authorization info from process context
	authInfo := authorized.ProcessAuthInfo(p)

	// 3. Build timeout context — LLM calls can take minutes (tool use, multi-turn)
	//    Default: 10 minutes (DefaultProcessTimeout). Caller can override via `timeout` field.
	timeoutSec := req.Timeout
	if timeoutSec <= 0 {
		timeoutSec = DefaultProcessTimeout
	}
	parent := p.Context
	if parent == nil {
		parent = context.Background()
	}
	timeoutCtx, cancel := context.WithTimeout(parent, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	// 4. Build headless context + options (encapsulated in context.go)
	ctx, opts := NewHeadlessContext(timeoutCtx, authInfo, &req)
	defer ctx.Release()

	// 5. Parse messages from []map[string]interface{} to []agentContext.Message
	messages := ParseMessages(req.Messages)

	// 6. Get agent and execute
	if AgentGetterFunc == nil {
		return NewResult(req.AssistantID, nil, fmt.Errorf("agent getter not initialized"))
	}

	agent, err := AgentGetterFunc(req.AssistantID)
	if err != nil {
		return NewResult(req.AssistantID, nil, fmt.Errorf("failed to get agent: %w", err))
	}

	resp, err := agent.Stream(ctx, messages, opts)
	if err != nil {
		return NewResult(req.AssistantID, nil, fmt.Errorf("agent call failed: %w", err))
	}

	// 7. Return *Result — shared with ctx.agent.Call() via NewResult()
	return NewResult(req.AssistantID, resp, nil)
}

// ParseMessages converts []map[string]interface{} to []agentContext.Message.
// Extracted as a package-level function so it can be reused by both
// processAgentCall and JSAPI.parseMessages.
func ParseMessages(raw []map[string]interface{}) []agentContext.Message {
	result := make([]agentContext.Message, 0, len(raw))
	for _, msg := range raw {
		ctxMsg := agentContext.Message{}

		// Parse role
		if role, ok := msg["role"].(string); ok {
			ctxMsg.Role = agentContext.MessageRole(role)
		}

		// Parse content (can be string or array of content parts)
		ctxMsg.Content = msg["content"]

		// Parse name
		if name, ok := msg["name"].(string); ok {
			ctxMsg.Name = &name
		}

		// Parse tool_call_id
		if toolCallID, ok := msg["tool_call_id"].(string); ok {
			ctxMsg.ToolCallID = &toolCallID
		}

		// Parse tool_calls
		if toolCalls, ok := msg["tool_calls"].([]interface{}); ok {
			ctxMsg.ToolCalls = parseToolCalls(toolCalls)
		}

		// Parse refusal
		if refusal, ok := msg["refusal"].(string); ok {
			ctxMsg.Refusal = &refusal
		}

		result = append(result, ctxMsg)
	}
	return result
}

// parseToolCalls converts []interface{} to []agentContext.ToolCall
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
		if tcType, ok := tcMap["type"].(string); ok {
			toolCall.Type = agentContext.ToolCallType(tcType)
		}
		if fn, ok := tcMap["function"].(map[string]interface{}); ok {
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
