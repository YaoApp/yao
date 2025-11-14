package handlers

import (
	"github.com/yaoapp/yao/agent/context"
)

// DefaultStreamHandler creates a default stream handler that sends messages via context
// This handler is used when no custom handler is provided
func DefaultStreamHandler(ctx *context.Context) context.StreamFunc {
	return func(data []byte) int {
		// TODO: Implement default stream handling
		// - Parse streaming chunk data
		// - Extract content from chunk
		// - Send message via ctx (SSE, WebSocket, etc.)
		// - Handle different chunk types (content, tool_calls, reasoning)
		// - Return 1 to continue streaming, 0 to stop
		return 1
	}
}

// SendStreamChunk sends a stream chunk via context
// Used internally by DefaultStreamHandler
func SendStreamChunk(ctx *context.Context, chunk *StreamChunk) error {
	// TODO: Implement sending stream chunk
	// - Format chunk for transport (SSE, WebSocket)
	// - Send via ctx's connection
	// - Handle errors and retries
	return nil
}

// StreamChunk represents a parsed streaming chunk
type StreamChunk struct {
	Type    ChunkType `json:"type"`              // Type of chunk (content, reasoning, tool_call, etc.)
	Content string    `json:"content,omitempty"` // Text content

	// For reasoning chunks
	ReasoningContent string `json:"reasoning_content,omitempty"`

	// For tool call chunks
	ToolCallID       string `json:"tool_call_id,omitempty"`
	ToolCallFunction string `json:"tool_call_function,omitempty"`
	ToolCallArgs     string `json:"tool_call_args,omitempty"`

	// Metadata
	Done         bool   `json:"done"`                    // Whether this is the final chunk
	FinishReason string `json:"finish_reason,omitempty"` // Reason for completion (if done)
}

// ChunkType represents the type of streaming chunk
type ChunkType string

const (
	ChunkTypeContent   ChunkType = "content"   // Regular text content
	ChunkTypeReasoning ChunkType = "reasoning" // Reasoning/thinking content
	ChunkTypeToolCall  ChunkType = "tool_call" // Tool call chunk
	ChunkTypeDone      ChunkType = "done"      // Final chunk (completion)
	ChunkTypeError     ChunkType = "error"     // Error chunk
)

// ParseStreamChunk parses raw streaming data into StreamChunk
func ParseStreamChunk(data []byte) (*StreamChunk, error) {
	// TODO: Implement stream chunk parsing
	// - Parse SSE format (data: {...})
	// - Handle different provider formats (OpenAI, DeepSeek, etc.)
	// - Extract content, reasoning, tool calls
	// - Detect completion (done: true)
	return nil, nil
}

// FormatSSE formats a StreamChunk as Server-Sent Events format
func FormatSSE(chunk *StreamChunk) string {
	// TODO: Implement SSE formatting
	// - Format as "data: {...}\n\n"
	// - Handle special cases (done, error)
	// - Ensure proper JSON encoding
	return ""
}

// FormatWebSocket formats a StreamChunk as WebSocket message
func FormatWebSocket(chunk *StreamChunk) []byte {
	// TODO: Implement WebSocket formatting
	// - Format as JSON message
	// - Add message type/metadata
	// - Handle binary vs text frames
	return nil
}
