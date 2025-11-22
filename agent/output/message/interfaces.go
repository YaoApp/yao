package message

import "github.com/yaoapp/yao/agent/context"

// Writer is the interface for writing output messages
// Different writers handle different output formats (SSE, WebSocket, Standard, etc.)
type Writer interface {
	// Write writes a single message
	Write(msg *Message) error

	// WriteGroup writes a group of messages
	WriteGroup(group *Group) error

	// Flush flushes any buffered data
	Flush() error

	// Close closes the writer and releases resources
	Close() error
}

// Adapter is the interface for adapting messages to different formats
// Adapters transform messages from the universal DSL to specific client formats
type Adapter interface {
	// Adapt transforms a message to the target format
	// Returns a slice of output chunks (some messages may be split into multiple chunks)
	Adapt(msg *Message) ([]interface{}, error)

	// SupportsType checks if this adapter supports a specific message type
	SupportsType(msgType string) bool
}

// WriterFactory creates writers based on context
type WriterFactory interface {
	// NewWriter creates a writer for the given context
	NewWriter(ctx *context.Context, adapter Adapter) (Writer, error)
}

// AdapterFactory creates adapters based on context
type AdapterFactory interface {
	// NewAdapter creates an adapter for the given context
	NewAdapter(ctx *context.Context) (Adapter, error)
}

// StreamHandler handles streaming message processing
// It bridges between LLM streaming chunks and output messages
type StreamHandler interface {
	// Handle processes a streaming chunk from LLM
	Handle(chunkType context.StreamChunkType, data []byte) error

	// Flush flushes any pending messages
	Flush() error

	// Close closes the handler
	Close() error
}
