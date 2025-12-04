package message

import (
	"net/http"

	"github.com/yaoapp/gou/connector/openai"
	traceTypes "github.com/yaoapp/yao/trace/types"
)

// Options are the options for the writer
type Options struct {
	BaseURL      string
	Accept       string
	Writer       http.ResponseWriter
	Trace        traceTypes.Manager
	Capabilities *openai.Capabilities
	Locale       string
}

// Message represents a universal message structure (DSL)
// All messages are expressed through Type + Props, without predefining specific types
type Message struct {
	// Core fields
	Type  string                 `json:"type"`            // Message type (frontend decides how to render)
	Props map[string]interface{} `json:"props,omitempty"` // Message properties (passed to frontend component)

	// Streaming control - Hierarchical structure for Agent/LLM/MCP streaming
	// See STREAMING.md for detailed explanation of the streaming architecture
	ChunkID   string `json:"chunk_id,omitempty"`   // Unique chunk ID (auto-generated: C1, C2, C3...; for dedup/ordering/debugging)
	MessageID string `json:"message_id,omitempty"` // Logical message ID (delta merge target; multiple chunks combine into one message)
	BlockID   string `json:"block_id,omitempty"`   // Output block ID (Agent-level control: one LLM call, one MCP call, etc.; for UI rendering blocks/sections)
	ThreadID  string `json:"thread_id,omitempty"`  // Thread ID (optional; for concurrent Agent/LLM/MCP calls to distinguish output streams)

	// Delta control
	Delta       bool   `json:"delta,omitempty"`        // Whether this is an incremental update
	DeltaPath   string `json:"delta_path,omitempty"`   // Update path (e.g., "content", "data", "items.0.name")
	DeltaAction string `json:"delta_action,omitempty"` // Update action (append, replace, merge, set)

	// Type correction (for streaming scenarios)
	TypeChange bool `json:"type_change,omitempty"` // Marks this as a type correction message

	// Metadata
	Metadata *Metadata `json:"metadata,omitempty"` // Additional metadata
}

// Metadata represents message metadata
type Metadata struct {
	Timestamp int64  `json:"timestamp,omitempty"` // Timestamp in nanoseconds
	Sequence  int    `json:"sequence,omitempty"`  // Sequence number (for ordering)
	TraceID   string `json:"trace_id,omitempty"`  // Trace ID (for debugging)
}

// Group represents a semantically complete group of messages
type Group struct {
	ID       string     `json:"id"`                 // Message group ID
	Messages []*Message `json:"messages"`           // List of messages
	Metadata *Metadata  `json:"metadata,omitempty"` // Metadata
}

// Built-in message types that all adapters must support
// These types have standardized Props structures
const (
	// User interaction types
	TypeUserInput = "user_input" // User input message (frontend display only)

	// Content types
	TypeText     = "text"      // Plain text or Markdown content
	TypeThinking = "thinking"  // Reasoning/thinking process (e.g., o1 models)
	TypeLoading  = "loading"   // Loading/processing indicator (preprocessing, knowledge base search, etc.)
	TypeToolCall = "tool_call" // LLM tool/function call
	TypeError    = "error"     // Error message

	// Media types (with OpenAI support)
	TypeImage = "image" // Image content
	TypeAudio = "audio" // Audio content
	TypeVideo = "video" // Video content

	// System types (not visible in standard chat clients)
	TypeAction = "action" // System action (open panel, navigate, etc.) - silent in OpenAI clients
	TypeEvent  = "event"  // Lifecycle event (stream_start, stream_end, etc.) - CUI only, silent in OpenAI clients
)

// Event types for TypeEvent messages
// Hierarchical structure: Stream > Thread > Block > Message > Chunk
const (
	// Stream level events (Agent layer - overall conversation stream)
	EventStreamStart = "stream_start" // Stream started event
	EventStreamEnd   = "stream_end"   // Stream ended event

	// Thread level events (optional - for concurrent scenarios)
	EventThreadStart = "thread_start" // Thread started event
	EventThreadEnd   = "thread_end"   // Thread ended event

	// Block level events (Agent layer - logical output sections)
	EventBlockStart = "block_start" // Block started event
	EventBlockEnd   = "block_end"   // Block ended event

	// Message level events (LLM layer - individual logical messages)
	EventMessageStart = "message_start" // Message started event
	EventMessageEnd   = "message_end"   // Message ended event
)

// Standard Props structures for built-in types

// UserInputProps defines the standard structure for user input messages
// Type: "user_input"
// Props: {"content": string | ContentPart[], "role": string, "name": string}
type UserInputProps struct {
	Content interface{} `json:"content"`        // User input (text string or multimodal ContentPart[])
	Role    string      `json:"role,omitempty"` // User role: "user", "system", "developer" (default: "user")
	Name    string      `json:"name,omitempty"` // Optional participant name
}

// TextProps defines the standard structure for text messages
// Type: "text"
// Props: {"content": string}
type TextProps struct {
	Content string `json:"content"` // Text content (supports Markdown)
}

// ThinkingProps defines the standard structure for thinking messages
// Type: "thinking"
// Props: {"content": string}
type ThinkingProps struct {
	Content string `json:"content"` // Reasoning/thinking content
}

// LoadingProps defines the standard structure for loading messages
// Type: "loading"
// Props: {"message": string}
type LoadingProps struct {
	Message string `json:"message"` // Loading message (e.g., "Searching knowledge base...")
}

// ToolCallProps defines the standard structure for tool_call messages
// Type: "tool_call"
// Props: {"id": string, "name": string, "arguments": string}
type ToolCallProps struct {
	ID        string `json:"id"`                  // Tool call ID
	Name      string `json:"name"`                // Function/tool name
	Arguments string `json:"arguments,omitempty"` // JSON string of arguments
}

// ErrorProps defines the standard structure for error messages
// Type: "error"
// Props: {"message": string, "code": string}
type ErrorProps struct {
	Message string `json:"message"`           // Error message
	Code    string `json:"code,omitempty"`    // Error code
	Details string `json:"details,omitempty"` // Additional error details
}

// ActionProps defines the standard structure for action messages
// Type: "action"
// Props: {"name": string, "payload": map}
type ActionProps struct {
	Name    string                 `json:"name"`              // Action name (e.g., "open_panel", "navigate")
	Payload map[string]interface{} `json:"payload,omitempty"` // Action payload/parameters
}

// EventProps defines the standard structure for event messages
// Type: "event"
// Props: {"event": string, "message": string, "data": map}
type EventProps struct {
	Event   string                 `json:"event"`             // Event type (e.g., "stream_start", "stream_end", "connecting")
	Message string                 `json:"message,omitempty"` // Human-readable message (e.g., "Connecting...")
	Data    map[string]interface{} `json:"data,omitempty"`    // Additional event data
}

// ImageProps defines the standard structure for image messages
// Type: "image"
// Props: {"url": string, "alt": string, "width": int, "height": int, "detail": string}
type ImageProps struct {
	URL    string `json:"url"`              // Required: Image URL or base64 encoded data
	Alt    string `json:"alt,omitempty"`    // Alternative text
	Width  int    `json:"width,omitempty"`  // Image width in pixels
	Height int    `json:"height,omitempty"` // Image height in pixels
	Detail string `json:"detail,omitempty"` // OpenAI detail level: "auto", "low", "high"
}

// AudioProps defines the standard structure for audio messages
// Type: "audio"
// Props: {"url": string, "format": string, "duration": float64, "transcript": string, "autoplay": bool}
type AudioProps struct {
	URL        string  `json:"url"`                  // Required: Audio URL or base64 encoded data
	Format     string  `json:"format,omitempty"`     // Audio format: "mp3", "wav", "ogg", etc.
	Duration   float64 `json:"duration,omitempty"`   // Duration in seconds
	Transcript string  `json:"transcript,omitempty"` // Audio transcript text
	Autoplay   bool    `json:"autoplay,omitempty"`   // Whether to autoplay
	Controls   bool    `json:"controls,omitempty"`   // Whether to show controls (default: true)
}

// VideoProps defines the standard structure for video messages
// Type: "video"
// Props: {"url": string, "format": string, "duration": float64, "thumbnail": string, "width": int, "height": int, "autoplay": bool}
type VideoProps struct {
	URL       string  `json:"url"`                 // Required: Video URL
	Format    string  `json:"format,omitempty"`    // Video format: "mp4", "webm", etc.
	Duration  float64 `json:"duration,omitempty"`  // Duration in seconds
	Thumbnail string  `json:"thumbnail,omitempty"` // Thumbnail/poster image URL
	Width     int     `json:"width,omitempty"`     // Video width in pixels
	Height    int     `json:"height,omitempty"`    // Video height in pixels
	Autoplay  bool    `json:"autoplay,omitempty"`  // Whether to autoplay
	Controls  bool    `json:"controls,omitempty"`  // Whether to show controls (default: true)
	Loop      bool    `json:"loop,omitempty"`      // Whether to loop
}

// Delta action constants for incremental updates
const (
	DeltaAppend  = "append"  // Append (for arrays, strings)
	DeltaReplace = "replace" // Replace (for any value)
	DeltaMerge   = "merge"   // Merge (for objects)
	DeltaSet     = "set"     // Set (for new fields)
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

// StreamFunc the streaming function callback
// Parameters:
//   - chunkType: the type of content in this chunk (text, thinking, tool_call, etc.)
//   - data: the actual chunk data (could be text, JSON, or other format)
//
// Returns:
//   - int: status code (0 = continue, non-zero = stop streaming)
type StreamFunc func(chunkType StreamChunkType, data []byte) int

// AssistantInfo represents the assistant information structure
type AssistantInfo struct {
	ID          string `json:"assistant_id"`          // Assistant ID
	Type        string `json:"type,omitempty"`        // Assistant Type, default is assistant
	Name        string `json:"name,omitempty"`        // Assistant Name
	Avatar      string `json:"avatar,omitempty"`      // Assistant Avatar
	Description string `json:"description,omitempty"` // Assistant Description
}

// UsageInfo represents token usage statistics
// Structure matches OpenAI API: https://platform.openai.com/docs/api-reference/chat/object#chat-object-usage
type UsageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`     // Number of tokens in the prompt
	CompletionTokens int `json:"completion_tokens"` // Number of tokens in the generated completion
	TotalTokens      int `json:"total_tokens"`      // Total number of tokens used (prompt + completion)

	// Detailed token breakdown
	PromptTokensDetails     *PromptTokensDetails     `json:"prompt_tokens_details,omitempty"`     // Breakdown of tokens used in the prompt
	CompletionTokensDetails *CompletionTokensDetails `json:"completion_tokens_details,omitempty"` // Breakdown of tokens used in the completion
}

// PromptTokensDetails provides detailed breakdown of tokens used in the prompt
type PromptTokensDetails struct {
	AudioTokens  int `json:"audio_tokens,omitempty"`  // Audio input tokens present in the prompt
	CachedTokens int `json:"cached_tokens,omitempty"` // Cached tokens present in the prompt
}

// CompletionTokensDetails provides detailed breakdown of tokens used in the completion
type CompletionTokensDetails struct {
	AcceptedPredictionTokens int `json:"accepted_prediction_tokens,omitempty"` // Tokens from predictions that appeared in the completion
	AudioTokens              int `json:"audio_tokens,omitempty"`               // Audio input tokens generated by the model
	ReasoningTokens          int `json:"reasoning_tokens,omitempty"`           // Tokens generated by the model for reasoning (o1, o1-mini, DeepSeek R1)
	RejectedPredictionTokens int `json:"rejected_prediction_tokens,omitempty"` // Tokens from predictions that did not appear in the completion
}

// ============================================================================
// Stream Lifecycle Event Data Structures
// ============================================================================
// These structures define the data format for stream lifecycle events.
// They provide a standardized way to communicate stream boundaries and metadata
// to the frontend, enabling better UI/UX (progress indicators, timing, etc.).

// EventStreamStartData represents the data for stream_start event
// Sent when a streaming request begins
type EventStreamStartData struct {
	ContextID string                 `json:"context_id"`          // Context ID for the response
	RequestID string                 `json:"request_id"`          // Unique identifier for this request
	Timestamp int64                  `json:"timestamp"`           // Unix timestamp when stream started
	ChatID    string                 `json:"chat_id"`             // Chat ID being used (e.g., "chat-123")
	TraceID   string                 `json:"trace_id"`            // Trace ID being used (e.g., "trace-123")
	Assistant *AssistantInfo         `json:"assistant,omitempty"` // Assistant information
	Metadata  map[string]interface{} `json:"metadata,omitempty"`  // Metadata to pass to the page for CUI context
}

// EventStreamEndData represents the data for stream_end event
// Sent when a streaming request completes (successfully or with error)
type EventStreamEndData struct {
	RequestID  string                 `json:"request_id"`         // Corresponding request ID
	ContextID  string                 `json:"context_id"`         // Context ID for the response
	TraceID    string                 `json:"trace_id"`           // Trace ID being used (e.g., "trace-123")
	Timestamp  int64                  `json:"timestamp"`          // Unix timestamp when stream ended
	DurationMs int64                  `json:"duration_ms"`        // Total duration in milliseconds
	Status     string                 `json:"status"`             // "completed" | "error" | "cancelled"
	Error      string                 `json:"error,omitempty"`    // Error message if status is "error"
	Usage      *UsageInfo             `json:"usage,omitempty"`    // Token usage statistics
	Metadata   map[string]interface{} `json:"metadata,omitempty"` // Metadata to pass to the page for CUI context
}

// EventMessageStartData represents the data for message_start event
// Sent when a logical message begins (text, tool_call, thinking, etc.)
// LLM layer: Marks the beginning of a single logical message output
type EventMessageStartData struct {
	MessageID string                 `json:"message_id"`          // Message ID (M1, M2, M3...)
	Type      string                 `json:"type"`                // Message type: "text" | "thinking" | "tool_call" | "refusal"
	Timestamp int64                  `json:"timestamp"`           // Unix timestamp when message started
	ThreadID  string                 `json:"thread_id,omitempty"` // Thread ID (optional; for concurrent streams)
	ToolCall  *EventToolCallInfo     `json:"tool_call,omitempty"` // Tool call metadata (if type is "tool_call")
	Extra     map[string]interface{} `json:"extra,omitempty"`     // Additional metadata (for custom providers or future extensions)
}

// EventMessageEndData represents the data for message_end event
// Sent when a logical message completes
// LLM layer: Signals that all chunks for this message have been sent, client should merge and process
type EventMessageEndData struct {
	MessageID  string                 `json:"message_id"`          // Message ID (M1, M2, M3...)
	Type       string                 `json:"type"`                // Message type (same as in message_start)
	Timestamp  int64                  `json:"timestamp"`           // Unix timestamp when message ended
	ThreadID   string                 `json:"thread_id,omitempty"` // Thread ID (optional; for concurrent streams)
	DurationMs int64                  `json:"duration_ms"`         // Duration of this message in milliseconds
	ChunkCount int                    `json:"chunk_count"`         // Number of data chunks in this message
	Status     string                 `json:"status"`              // "completed" | "partial" | "error"
	ToolCall   *EventToolCallInfo     `json:"tool_call,omitempty"` // Complete tool call info (if type is "tool_call")
	Extra      map[string]interface{} `json:"extra,omitempty"`     // Additional metadata (e.g., complete content for direct use)
}

// EventToolCallInfo contains tool call information for message events
// Used in both message_start (partial info) and message_end (complete info)
type EventToolCallInfo struct {
	ID        string `json:"id"`                  // Tool call ID (e.g., "call_abc123")
	Name      string `json:"name"`                // Function name (may be partial in message_start)
	Arguments string `json:"arguments,omitempty"` // Complete arguments (only in message_end)
	Index     int    `json:"index"`               // Index in the tool calls array
}

// EventBlockStartData represents the data for block_start event
// Sent when an output block begins (one LLM call, one MCP call, one Agent sub-task, etc.)
// Agent layer: Groups multiple related messages into a logical section
type EventBlockStartData struct {
	BlockID   string                 `json:"block_id"`        // Block ID (B1, B2, B3...)
	Type      string                 `json:"type"`            // Block type: "llm" | "mcp" | "agent" | "tool" | "mixed"
	Timestamp int64                  `json:"timestamp"`       // Unix timestamp when block started
	Label     string                 `json:"label,omitempty"` // Human-readable label (e.g., "Searching knowledge base", "Calling weather API")
	Extra     map[string]interface{} `json:"extra,omitempty"` // Additional metadata
}

// EventBlockEndData represents the data for block_end event
// Sent when an output block completes
// Agent layer: Signals that this logical section is complete
type EventBlockEndData struct {
	BlockID      string                 `json:"block_id"`        // Block ID (B1, B2, B3...)
	Type         string                 `json:"type"`            // Block type (same as in block_start)
	Timestamp    int64                  `json:"timestamp"`       // Unix timestamp when block ended
	DurationMs   int64                  `json:"duration_ms"`     // Duration of this block in milliseconds
	MessageCount int                    `json:"message_count"`   // Number of messages in this block
	Status       string                 `json:"status"`          // "completed" | "partial" | "error"
	Extra        map[string]interface{} `json:"extra,omitempty"` // Additional metadata
}

// EventThreadStartData represents the data for thread_start event
// Sent when a concurrent thread begins (parallel Agent/LLM/MCP calls)
// Used in concurrent scenarios to distinguish multiple parallel output streams
type EventThreadStartData struct {
	ThreadID  string                 `json:"thread_id"`       // Thread ID (T1, T2, T3...)
	Type      string                 `json:"type"`            // Thread type: "agent" | "llm" | "mcp" | "tool"
	Timestamp int64                  `json:"timestamp"`       // Unix timestamp when thread started
	Label     string                 `json:"label,omitempty"` // Human-readable label (e.g., "Parallel search 1", "Background task")
	Extra     map[string]interface{} `json:"extra,omitempty"` // Additional metadata
}

// EventThreadEndData represents the data for thread_end event
// Sent when a concurrent thread completes
type EventThreadEndData struct {
	ThreadID   string                 `json:"thread_id"`       // Thread ID (T1, T2, T3...)
	Type       string                 `json:"type"`            // Thread type (same as in thread_start)
	Timestamp  int64                  `json:"timestamp"`       // Unix timestamp when thread ended
	DurationMs int64                  `json:"duration_ms"`     // Duration of this thread in milliseconds
	BlockCount int                    `json:"block_count"`     // Number of blocks in this thread
	Status     string                 `json:"status"`          // "completed" | "partial" | "error"
	Extra      map[string]interface{} `json:"extra,omitempty"` // Additional metadata
}
