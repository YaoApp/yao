package llm

import (
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/llm/providers"
)

// New create a new LLM instance
// conn: connector object from connector.Select()
// options: completion options containing capabilities and other settings
func New(conn connector.Connector, options *context.CompletionOptions) (LLM, error) {
	// Select appropriate provider based on capabilities
	return providers.SelectProvider(conn, options)
}
