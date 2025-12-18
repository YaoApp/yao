// Package caller provides a shared interface for calling agents
// This package is used by both content and search packages to avoid circular dependencies
package caller

import (
	agentContext "github.com/yaoapp/yao/agent/context"
)

// AgentCaller interface for calling agents (to avoid circular dependency)
// Used by content handlers (vision, audio, etc.) and search handlers (agent mode)
type AgentCaller interface {
	Stream(ctx *agentContext.Context, messages []agentContext.Message, options ...*agentContext.Options) (*agentContext.Response, error)
}

// AgentGetterFunc is a function type that gets an agent by ID
// This should be set by the assistant package during initialization
var AgentGetterFunc func(agentID string) (AgentCaller, error)
