package handlers

import (
	"github.com/yaoapp/yao/agent/context"
)

// Handler interface for stream handlers
type Handler interface {
	OnChunk(chunk *StreamChunk) error
	OnComplete() error
	OnError(err error) error
}

// NewDefaultHandler creates a default handler that sends chunks via context
func NewDefaultHandler(ctx *context.Context) Handler {
	return &DefaultHandler{
		ctx: ctx,
	}
}

// DefaultHandler default stream handler implementation
type DefaultHandler struct {
	ctx *context.Context
}

// OnChunk handles a streaming chunk
func (h *DefaultHandler) OnChunk(chunk *StreamChunk) error {
	// TODO: Implement chunk handling
	// - Send chunk via ctx
	// - Handle different chunk types
	// - Aggregate content for final response
	return SendStreamChunk(h.ctx, chunk)
}

// OnComplete handles stream completion
func (h *DefaultHandler) OnComplete() error {
	// TODO: Implement completion handling
	// - Send final message
	// - Close stream
	// - Return aggregated response
	return nil
}

// OnError handles stream errors
func (h *DefaultHandler) OnError(err error) error {
	// TODO: Implement error handling
	// - Send error message to client
	// - Log error
	// - Clean up resources
	return err
}
