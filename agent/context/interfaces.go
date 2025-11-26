package context

import (
	"net/http"

	"github.com/yaoapp/yao/agent/output/message"
)

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

	// Lifecycle event types - stream and message boundaries
	ChunkStreamStart  StreamChunkType = "stream_start"  // Stream begins (entire request starts)
	ChunkStreamEnd    StreamChunkType = "stream_end"    // Stream ends (entire request completes)
	ChunkMessageStart StreamChunkType = "message_start" // Message begins (text/tool_call/thinking message starts)
	ChunkMessageEnd   StreamChunkType = "message_end"   // Message ends (text/tool_call/thinking message completes)
)

// Writer is an alias for http.ResponseWriter interface used by an agent to construct a response.
// A Writer may not be used after the agent execution has completed.
type Writer = http.ResponseWriter

// Agent the agent interface
type Agent interface {

	// Stream stream the agent
	Stream(ctx *Context, messages []Message, handler message.StreamFunc) error

	// Run run the agent
	Run(ctx *Context, messages []Message) (*Response, error)
}
