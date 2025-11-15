package cui

import (
	"github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/output/message"
)

// Factory is the factory for creating CUI writers and adapters
type Factory struct{}

// NewFactory creates a new CUI factory
func NewFactory() *Factory {
	return &Factory{}
}

// CreateWriter creates a CUI writer
func (f *Factory) CreateWriter(ctx *context.Context) (message.Writer, error) {
	return NewWriter(ctx)
}

// CreateAdapter creates a CUI adapter
func (f *Factory) CreateAdapter(ctx *context.Context) (message.Adapter, error) {
	return NewAdapter(), nil
}
