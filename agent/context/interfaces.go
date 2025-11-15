package context

import "net/http"

// StreamChunkType represents the type of content in a streaming chunk
type StreamChunkType string

// Stream chunk type constants - indicates what type of content is in the current chunk
const (
	// Content chunk types - actual data from the LLM
	ChunkText     StreamChunkType = "text"      // Regular text content
	ChunkThinking StreamChunkType = "thinking"  // Reasoning/thinking content (o1, DeepSeek R1)
	ChunkToolCall StreamChunkType = "tool_call" // Tool/function call
	ChunkRefusal  StreamChunkType = "refusal"   // Model refusal
	ChunkMetadata StreamChunkType = "metadata"  // Metadata (usage, finish_reason, etc.)
	ChunkError    StreamChunkType = "error"     // Error chunk
	ChunkUnknown  StreamChunkType = "unknown"   // Unknown/unrecognized chunk type

	// Lifecycle event types - stream and group boundaries
	ChunkStreamStart StreamChunkType = "stream_start" // Stream begins (entire request starts)
	ChunkStreamEnd   StreamChunkType = "stream_end"   // Stream ends (entire request completes)
	ChunkGroupStart  StreamChunkType = "group_start"  // Message group begins (text/tool_call/thinking group starts)
	ChunkGroupEnd    StreamChunkType = "group_end"    // Message group ends (text/tool_call/thinking group completes)
)

// StreamFunc the streaming function callback
// Parameters:
//   - chunkType: the type of content in this chunk (text, thinking, tool_call, etc.)
//   - data: the actual chunk data (could be text, JSON, or other format)
//
// Returns:
//   - int: status code (0 = continue, non-zero = stop streaming)
type StreamFunc func(chunkType StreamChunkType, data []byte) int

// Writer is an alias for http.ResponseWriter interface used by an agent to construct a response.
// A Writer may not be used after the agent execution has completed.
type Writer = http.ResponseWriter

// Agent the agent interface
type Agent interface {

	// Stream stream the agent
	Stream(ctx *Context, messages []Message, handler StreamFunc) error

	// Run run the agent
	Run(ctx *Context, messages []Message) (*Response, error)
}
