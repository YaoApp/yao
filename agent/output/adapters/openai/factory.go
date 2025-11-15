package openai

import (
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
)

// Factory is the factory for creating OpenAI writers and adapters
type Factory struct {
	options []Option
}

// NewFactory creates a new OpenAI factory with options
func NewFactory(options ...Option) *Factory {
	return &Factory{
		options: options,
	}
}

// CreateWriter creates an OpenAI writer
func (f *Factory) CreateWriter(ctx *context.Context) (message.Writer, error) {
	return NewWriter(ctx)
}

// CreateAdapter creates an OpenAI adapter
func (f *Factory) CreateAdapter(ctx *context.Context) (message.Adapter, error) {
	return NewAdapter(f.options...), nil
}
