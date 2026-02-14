// Package caller provides types and utilities for agent-to-agent calls
package caller

import (
	agentContext "github.com/yaoapp/yao/agent/context"
)

// DefaultProcessTimeout is the default timeout (in seconds) for agent.Call Process.
// LLM calls with tool use can take minutes; 10 minutes provides safe headroom.
const DefaultProcessTimeout = 600

// Request represents a request to call an agent
type Request struct {
	AgentID  string                     `json:"agent"`             // Target agent ID
	Messages []agentContext.Message     `json:"messages"`          // Messages to send
	Options  *CallOptions               `json:"options,omitempty"` // Call options
	Handler  agentContext.OnMessageFunc `json:"-"`                 // OnMessage handler for this request (not serialized)
}

// CallOptions represents options for an agent call
type CallOptions struct {
	Connector string                 `json:"connector,omitempty"` // Override connector
	Mode      string                 `json:"mode,omitempty"`      // Agent mode (chat, etc.)
	Metadata  map[string]interface{} `json:"metadata,omitempty"`  // Custom metadata passed to hooks
	Skip      *agentContext.Skip     `json:"skip,omitempty"`      // Skip configuration (history, trace, output, etc.)
}

// Result represents the result of an agent call
type Result struct {
	AgentID  string                 `json:"agent_id"`           // Agent ID that was called
	Response *agentContext.Response `json:"response,omitempty"` // Full response from agent
	Content  string                 `json:"content,omitempty"`  // Final text content (extracted from completion)
	Error    string                 `json:"error,omitempty"`    // Error message if call failed
}

// ProcessCallRequest is the parameter structure for the agent.Call Process.
// Fields mirror CompletionRequest + HTTP header semantics, enabling headless
// agent calls from contexts without agent.Context (e.g., YaoJob async tasks).
type ProcessCallRequest struct {
	AssistantID string                   `json:"assistant_id"`       // Required: target assistant ID (maps to X-Yao-Assistant header)
	Messages    []map[string]interface{} `json:"messages"`           // Required: message list (maps to CompletionRequest.Messages)
	Model       string                   `json:"model,omitempty"`    // Optional: connector ID override (maps to CompletionRequest.Model)
	Skip        *agentContext.Skip       `json:"skip,omitempty"`     // Optional: skip config (maps to CompletionRequest.Skip)
	Metadata    map[string]interface{}   `json:"metadata,omitempty"` // Optional: passed to hooks (maps to CompletionRequest.Metadata)
	Locale      string                   `json:"locale,omitempty"`   // Optional (maps to locale query param)
	Route       string                   `json:"route,omitempty"`    // Optional (maps to CompletionRequest.Route)
	ChatID      string                   `json:"chat_id,omitempty"`  // Optional: auto-generated if empty (maps to chat_id query/header)
	Timeout     int                      `json:"timeout,omitempty"`  // Optional: timeout in seconds (default: DefaultProcessTimeout = 600)
}

// NewResult builds a Result from an agent call response.
// Used by both ctx.agent.Call (orchestrator) and Process("agent.Call") to
// ensure consistent result construction.
func NewResult(agentID string, resp *agentContext.Response, err error) *Result {
	result := &Result{AgentID: agentID}
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.Response = resp
	if resp != nil && resp.Completion != nil {
		result.Content = extractContentFromCompletion(resp.Completion)
	}
	return result
}

// ToContextOptions converts CallOptions to context.Options for the agent call
func (o *CallOptions) ToContextOptions() *agentContext.Options {
	if o == nil {
		return nil
	}

	return &agentContext.Options{
		Connector: o.Connector,
		Mode:      o.Mode,
		Metadata:  o.Metadata,
		Skip:      o.Skip,
	}
}
