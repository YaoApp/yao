package llm

import "github.com/yaoapp/yao/agent/context"

// LLM the LLM interface
type LLM interface {
	Stream(ctx *context.Context, messages []context.Message, options *CompletionOptions, handler context.StreamFunc) (*context.ResponseCompletion, error)
	Post(ctx *context.Context, messages []context.Message, options *CompletionOptions) (*context.ResponseCompletion, error)
}
