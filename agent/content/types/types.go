package types

import (
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/connector/openai"
	agentContext "github.com/yaoapp/yao/agent/context"
)

// Options represents the options for the content
type Options struct {

	// Connector, Current connector instance
	Connector connector.Connector

	// Capabilities, Current capabilities instance
	Capabilities *openai.Capabilities

	// CompletionOptions, Current completion options instance
	CompletionOptions *agentContext.CompletionOptions

	// StreamOptions, Current stream options instance
	StreamOptions *agentContext.StreamOptions

	// SilentLoading, if true, suppress loading messages (used when called from parent handler)
	SilentLoading bool
}
