package handlers

import (
	"github.com/yaoapp/yao/agent/context"
)

// DefaultStreamHandler creates a default stream handler that sends messages via context
// This handler is used when no custom handler is provided
func DefaultStreamHandler(ctx *context.Context) context.StreamFunc {
	return func(chunkType context.StreamChunkType, data []byte) int {
		// TODO: Implement default stream handling
		// - Parse streaming chunk data based on chunkType
		// - Extract content from chunk
		// - Send message via ctx (SSE, WebSocket, etc.)
		// - Handle different chunk types (text, thinking, tool_calls, etc.)
		// - Return 0 to continue streaming, non-zero to stop

		switch chunkType {
		case context.ChunkText:
			// Handle text content
		case context.ChunkThinking:
			// Handle reasoning/thinking content
		case context.ChunkToolCall:
			// Handle tool calls
		case context.ChunkMetadata:
			// Handle metadata (usage, finish_reason)
		case context.ChunkError:
			// Handle error
			return 1 // Stop on error
		}

		return 0 // Continue streaming
	}
}

// SendStreamChunk sends a stream chunk via context
// Used internally by DefaultStreamHandler
func SendStreamChunk(ctx *context.Context, chunkType context.StreamChunkType, data []byte) error {
	// TODO: Implement sending stream chunk
	// - Format chunk for transport (SSE, WebSocket)
	// - Send via ctx's connection
	// - Handle errors and retries
	return nil
}

// FormatSSE formats streaming data as Server-Sent Events format
func FormatSSE(chunkType context.StreamChunkType, data []byte) string {
	// TODO: Implement SSE formatting
	// - Format as "data: {...}\n\n"
	// - Include chunk type in the message
	// - Handle special cases (done, error)
	// - Ensure proper JSON encoding
	return ""
}

// FormatWebSocket formats streaming data as WebSocket message
func FormatWebSocket(chunkType context.StreamChunkType, data []byte) []byte {
	// TODO: Implement WebSocket formatting
	// - Format as JSON message with chunk type
	// - Add message type/metadata
	// - Handle binary vs text frames
	return nil
}
