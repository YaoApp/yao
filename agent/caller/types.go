// Package caller provides types and utilities for agent-to-agent calls
package caller

import (
	agentContext "github.com/yaoapp/yao/agent/context"
)

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
