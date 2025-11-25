package llm

import (
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
)

// LLM the LLM interface
type LLM interface {
	Stream(ctx *context.Context, messages []context.Message, options *context.CompletionOptions, handler message.StreamFunc) (*context.CompletionResponse, error)
	Post(ctx *context.Context, messages []context.Message, options *context.CompletionOptions) (*context.CompletionResponse, error)
}
