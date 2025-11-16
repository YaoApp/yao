package adapters

import (
	"github.com/yaoapp/yao/agent/context"
)

// CapabilityAdapter is the interface for capability-specific message and response processing
// Each adapter handles one capability dimension (tool calls, vision, audio, reasoning, etc.)
type CapabilityAdapter interface {
	// Name returns the adapter name for debugging
	Name() string

	// PreprocessMessages preprocesses messages before sending to LLM
	// Returns modified messages or error
	PreprocessMessages(messages []context.Message) ([]context.Message, error)

	// PreprocessOptions preprocesses completion options before sending to LLM
	// Returns modified options or error
	PreprocessOptions(options *context.CompletionOptions) (*context.CompletionOptions, error)

	// PostprocessResponse postprocesses the LLM response
	// Returns modified response or error
	PostprocessResponse(response *context.CompletionResponse) (*context.CompletionResponse, error)

	// ProcessStreamChunk processes a streaming chunk
	// Returns modified chunk type and data, or error
	ProcessStreamChunk(chunkType context.StreamChunkType, data []byte) (context.StreamChunkType, []byte, error)
}

// BaseAdapter provides default implementations for CapabilityAdapter
// Adapters can embed this and override only the methods they need
type BaseAdapter struct {
	name string
}

// NewBaseAdapter creates a new base adapter
func NewBaseAdapter(name string) *BaseAdapter {
	return &BaseAdapter{name: name}
}

// Name returns the adapter name
func (a *BaseAdapter) Name() string {
	return a.name
}

// PreprocessMessages default implementation (no-op)
func (a *BaseAdapter) PreprocessMessages(messages []context.Message) ([]context.Message, error) {
	return messages, nil
}

// PreprocessOptions default implementation (no-op)
func (a *BaseAdapter) PreprocessOptions(options *context.CompletionOptions) (*context.CompletionOptions, error) {
	return options, nil
}

// PostprocessResponse default implementation (no-op)
func (a *BaseAdapter) PostprocessResponse(response *context.CompletionResponse) (*context.CompletionResponse, error) {
	return response, nil
}

// ProcessStreamChunk default implementation (pass through)
func (a *BaseAdapter) ProcessStreamChunk(chunkType context.StreamChunkType, data []byte) (context.StreamChunkType, []byte, error) {
	return chunkType, data, nil
}
