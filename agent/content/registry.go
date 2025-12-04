package content

import (
	"fmt"

	"github.com/yaoapp/gou/connector/openai"
	agentContext "github.com/yaoapp/yao/agent/context"
)

// Registry holds all registered content handlers
type Registry struct {
	handlers []Handler
}

// NewRegistry creates a new handler registry with default handlers
func NewRegistry() *Registry {
	return &Registry{
		handlers: []Handler{
			&ImageHandler{},
			&AudioHandler{},
			&PDFHandler{},
			&WordHandler{},
			&ExcelHandler{},
			&TextHandler{},
		},
	}
}

// GetHandler finds the appropriate handler for the given content
func (r *Registry) GetHandler(contentType string, fileType FileType) Handler {
	for _, handler := range r.handlers {
		if handler.CanHandle(contentType, fileType) {
			return handler
		}
	}
	return nil
}

// Handle processes content using the appropriate handler
func (r *Registry) Handle(ctx *agentContext.Context, info *Info, capabilities *openai.Capabilities, uses *agentContext.Uses) (*Result, error) {
	handler := r.GetHandler(info.ContentType, info.FileType)
	if handler == nil {
		return nil, fmt.Errorf("no handler found for content type: %s, file type: %s", info.ContentType, info.FileType)
	}

	return handler.Handle(ctx, info, capabilities, uses)
}
