package caller

import (
	agentContext "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
)

// JSAPI implements context.AgentAPI and context.AgentAPIWithCallback interfaces
// Provides ctx.agent.Call(), ctx.agent.All(), ctx.agent.Any(), ctx.agent.Race()
// and their *WithHandler variants for streaming callback support
type JSAPI struct {
	ctx          *agentContext.Context
	orchestrator *Orchestrator
}

// Ensure JSAPI implements AgentAPIWithCallback
var _ agentContext.AgentAPIWithCallback = (*JSAPI)(nil)

// NewJSAPI creates a new agent JSAPI instance
func NewJSAPI(ctx *agentContext.Context) *JSAPI {
	return &JSAPI{
		ctx:          ctx,
		orchestrator: NewOrchestrator(ctx),
	}
}

// Call executes a single agent call
// Usage: ctx.agent.Call("assistant-id", messages, options?)
// Returns: { agent_id, response, content, error }
// Note: For sub-agent calls, skip.history = true is automatically set
// to prevent A2A messages from being saved to chat history.
// Sub-agents output normally with ThreadID for SSE stream isolation.
func (api *JSAPI) Call(agentID string, messages []interface{}, opts map[string]interface{}) interface{} {
	req := api.buildRequest(agentID, messages, opts)
	// Force skip options for sub-agent calls
	api.forceSkipForSubAgent(req)
	result := api.orchestrator.callAgent(req)
	return result
}

// All executes all agent calls and waits for all to complete (like Promise.all)
// Each request should have:
//   - agent: string - target agent ID
//   - messages: array - messages to send
//   - options?: object - call options
func (api *JSAPI) All(requests []interface{}) []interface{} {
	reqs := api.parseRequests(requests)
	results := api.orchestrator.All(reqs)
	return api.convertResults(results)
}

// Any returns as soon as any agent call succeeds (like Promise.any)
// Each request should have:
//   - agent: string - target agent ID
//   - messages: array - messages to send
//   - options?: object - call options
func (api *JSAPI) Any(requests []interface{}) []interface{} {
	reqs := api.parseRequests(requests)
	results := api.orchestrator.Any(reqs)
	return api.convertResults(results)
}

// Race returns as soon as any agent call completes (like Promise.race)
// Each request should have:
//   - agent: string - target agent ID
//   - messages: array - messages to send
//   - options?: object - call options
func (api *JSAPI) Race(requests []interface{}) []interface{} {
	reqs := api.parseRequests(requests)
	results := api.orchestrator.Race(reqs)
	return api.convertResults(results)
}

// ============================================================================
// AgentAPIWithCallback Implementation
// ============================================================================

// CallWithHandler executes a single agent call with an OnMessage handler
// Note: For sub-agent calls, skip.history = true is automatically set.
// Sub-agents output normally with ThreadID. Use the handler callback
// to receive streaming messages.
func (api *JSAPI) CallWithHandler(agentID string, messages []interface{}, opts map[string]interface{}, handler agentContext.OnMessageFunc) interface{} {
	req := api.buildRequest(agentID, messages, opts)
	req.Handler = handler
	// Force skip options for sub-agent calls
	api.forceSkipForSubAgent(req)
	result := api.orchestrator.callAgent(req)
	return result
}

// AllWithHandler executes all agent calls with handlers
func (api *JSAPI) AllWithHandler(requests []interface{}, globalHandler agentContext.BatchOnMessageFunc) []interface{} {
	reqs := api.parseRequestsWithHandlers(requests, globalHandler)
	results := api.orchestrator.All(reqs)
	return api.convertResults(results)
}

// AnyWithHandler executes agent calls and returns on first success, with handlers
func (api *JSAPI) AnyWithHandler(requests []interface{}, globalHandler agentContext.BatchOnMessageFunc) []interface{} {
	reqs := api.parseRequestsWithHandlers(requests, globalHandler)
	results := api.orchestrator.Any(reqs)
	return api.convertResults(results)
}

// RaceWithHandler executes agent calls and returns on first completion, with handlers
func (api *JSAPI) RaceWithHandler(requests []interface{}, globalHandler agentContext.BatchOnMessageFunc) []interface{} {
	reqs := api.parseRequestsWithHandlers(requests, globalHandler)
	results := api.orchestrator.Race(reqs)
	return api.convertResults(results)
}

// forceSkipForSubAgent ensures proper A2A call behavior:
// - skip.history = true: A2A messages are not saved to chat history
// - skip.output = false: Sub-agents output normally with ThreadID for SSE stream isolation
//
// IMPORTANT: skip.output is explicitly set to false to override any user settings.
// This ensures ThreadID mechanism works correctly for concurrent sub-agent calls.
// Users can use the onChunk callback to receive streaming messages if needed.
func (api *JSAPI) forceSkipForSubAgent(req *Request) {
	if req.Options == nil {
		req.Options = &CallOptions{}
	}
	if req.Options.Skip == nil {
		req.Options.Skip = &agentContext.Skip{}
	}
	req.Options.Skip.History = true
	// Force output to be enabled - this overrides any user settings
	// Sub-agents MUST output with ThreadID for proper SSE stream isolation
	req.Options.Skip.Output = false
}

// parseRequestsWithHandlers parses requests and attaches handlers
// It checks for per-request _handler fields and wraps globalHandler with agentID/index
// For all calls, this automatically sets:
// - skip.history = true: prevents A2A messages from being saved to chat history
// - skip.output = false: ensures sub-agents output with ThreadID (overrides user settings)
func (api *JSAPI) parseRequestsWithHandlers(requests []interface{}, globalHandler agentContext.BatchOnMessageFunc) []*Request {
	reqs := make([]*Request, 0, len(requests))

	for i, r := range requests {
		reqMap, ok := r.(map[string]interface{})
		if !ok {
			continue
		}

		// Get agent ID
		agentID, ok := reqMap["agent"].(string)
		if !ok {
			continue
		}

		// Get messages
		messages, ok := reqMap["messages"].([]interface{})
		if !ok {
			continue
		}

		// Get options (optional)
		var opts map[string]interface{}
		if o, ok := reqMap["options"].(map[string]interface{}); ok {
			opts = o
		}

		req := api.buildRequest(agentID, messages, opts)

		// Force skip.output = true for all sub-agent calls
		api.forceSkipForSubAgent(req)

		// Check for per-request handler first (takes precedence)
		if handler, ok := reqMap["_handler"].(agentContext.OnMessageFunc); ok && handler != nil {
			req.Handler = handler
		} else if globalHandler != nil {
			// Wrap global handler with agentID and index
			idx := i // Capture index for closure
			aid := agentID
			req.Handler = func(msg *message.Message) int {
				return globalHandler(aid, idx, msg)
			}
		}

		reqs = append(reqs, req)
	}

	return reqs
}

// buildRequest builds a Request from agentID, messages, and options
func (api *JSAPI) buildRequest(agentID string, messages []interface{}, opts map[string]interface{}) *Request {
	req := &Request{
		AgentID:  agentID,
		Messages: api.parseMessages(messages),
	}

	if opts != nil {
		req.Options = api.parseCallOptions(opts)
	}

	return req
}

// parseMessages converts []interface{} to []agentContext.Message
func (api *JSAPI) parseMessages(messages []interface{}) []agentContext.Message {
	result := make([]agentContext.Message, 0, len(messages))
	for _, m := range messages {
		msg, ok := m.(map[string]interface{})
		if !ok {
			continue
		}

		ctxMsg := agentContext.Message{}

		// Parse role
		if role, ok := msg["role"].(string); ok {
			ctxMsg.Role = agentContext.MessageRole(role)
		}

		// Parse content (can be string or array)
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
			ctxMsg.ToolCalls = api.parseToolCalls(toolCalls)
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
func (api *JSAPI) parseToolCalls(toolCalls []interface{}) []agentContext.ToolCall {
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

// parseCallOptions converts map to CallOptions
func (api *JSAPI) parseCallOptions(opts map[string]interface{}) *CallOptions {
	callOpts := &CallOptions{}

	if connector, ok := opts["connector"].(string); ok {
		callOpts.Connector = connector
	}
	if mode, ok := opts["mode"].(string); ok {
		callOpts.Mode = mode
	}
	if metadata, ok := opts["metadata"].(map[string]interface{}); ok {
		callOpts.Metadata = metadata
	}

	// Parse skip configuration
	if skip, ok := opts["skip"].(map[string]interface{}); ok {
		callOpts.Skip = &agentContext.Skip{}
		if history, ok := skip["history"].(bool); ok {
			callOpts.Skip.History = history
		}
		if trace, ok := skip["trace"].(bool); ok {
			callOpts.Skip.Trace = trace
		}
		if output, ok := skip["output"].(bool); ok {
			callOpts.Skip.Output = output
		}
		if keyword, ok := skip["keyword"].(bool); ok {
			callOpts.Skip.Keyword = keyword
		}
		if search, ok := skip["search"].(bool); ok {
			callOpts.Skip.Search = search
		}
		if contentParsing, ok := skip["content_parsing"].(bool); ok {
			callOpts.Skip.ContentParsing = contentParsing
		}
	}

	return callOpts
}

// parseRequests parses an array of request objects into typed Requests
func (api *JSAPI) parseRequests(requests []interface{}) []*Request {
	return api.parseRequestsWithHandlers(requests, nil)
}

// convertResults converts typed Results to interface slice for JS
func (api *JSAPI) convertResults(results []*Result) []interface{} {
	out := make([]interface{}, len(results))
	for i, r := range results {
		out[i] = r
	}
	return out
}

// SetJSAPIFactory sets the factory function for creating AgentAPI instances
// Called by assistant package during initialization
func SetJSAPIFactory() {
	agentContext.AgentAPIFactory = func(ctx *agentContext.Context) agentContext.AgentAPI {
		return NewJSAPI(ctx)
	}
}
