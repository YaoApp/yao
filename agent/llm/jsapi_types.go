// Package llm provides types and utilities for LLM JSAPI
package llm

import (
	agentContext "github.com/yaoapp/yao/agent/context"
)

// Request represents a request to call an LLM connector
type Request struct {
	Connector string                     `json:"connector"`         // Target connector ID
	Messages  []interface{}              `json:"messages"`          // Messages to send
	Options   map[string]interface{}     `json:"options,omitempty"` // LLM call options (temperature, max_tokens, etc.)
	Handler   agentContext.OnMessageFunc `json:"-"`                 // OnMessage handler for this request (not serialized)
}

// Result represents the result of a LLM call via JSAPI
type Result struct {
	Connector string                           `json:"connector"`          // Connector ID that was used
	Response  *agentContext.CompletionResponse `json:"response,omitempty"` // Full LLM response
	Content   string                           `json:"content,omitempty"`  // Extracted text content
	Error     string                           `json:"error,omitempty"`    // Error message if call failed
}

// Note: LlmBatchOnMessageFunc is defined in agent/context/jsapi_llm.go
// to avoid circular dependencies
