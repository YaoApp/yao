package llm

import (
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/llm/handlers"
)

// DefaultStreamHandler creates a default stream handler
// This is a convenience function that wraps handlers.DefaultStreamHandler
func DefaultStreamHandler(ctx *context.Context) context.StreamFunc {
	return handlers.DefaultStreamHandler(ctx)
}
