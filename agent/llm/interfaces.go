package llm

import "github.com/yaoapp/yao/agent/context"

// LLM the LLM interface
type LLM interface {
	Stream(ctx *context.Context, messages []context.Message, options *context.CompletionOptions, handler context.StreamFunc) (*context.CompletionResponse, error)
	Post(ctx *context.Context, messages []context.Message, options *context.CompletionOptions) (*context.CompletionResponse, error)
}
