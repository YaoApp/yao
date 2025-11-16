package message

// Message represents a universal message structure (DSL)
// All messages are expressed through Type + Props, without predefining specific types
type Message struct {
	// Core fields
	Type  string                 `json:"type"`            // Message type (frontend decides how to render)
	Props map[string]interface{} `json:"props,omitempty"` // Message properties (passed to frontend component)

	// Streaming control
	ID    string `json:"id,omitempty"`    // Message ID (used for merging messages in streaming scenarios)
	Delta bool   `json:"delta,omitempty"` // Whether this is an incremental update
	Done  bool   `json:"done,omitempty"`  // Whether the message is complete

	// Delta update control
	DeltaPath   string `json:"delta_path,omitempty"`   // Update path (e.g., "content", "data", "items.0.name")
	DeltaAction string `json:"delta_action,omitempty"` // Update action (append, replace, merge, set)

	// Type correction (for streaming scenarios)
	TypeChange bool `json:"type_change,omitempty"` // Marks this as a type correction message

	// Message group
	GroupID    string `json:"group_id,omitempty"`    // Parent message group ID
	GroupStart bool   `json:"group_start,omitempty"` // Marks the start of a message group
	GroupEnd   bool   `json:"group_end,omitempty"`   // Marks the end of a message group

	// Metadata
	Metadata *Metadata `json:"metadata,omitempty"` // Additional metadata
}

// Metadata represents message metadata
type Metadata struct {
	Timestamp int64  `json:"timestamp,omitempty"` // Timestamp in nanoseconds
	Sequence  int    `json:"sequence,omitempty"`  // Sequence number (for ordering)
	TraceID   string `json:"trace_id,omitempty"`  // Trace ID (for debugging)
}

// MessageGroup represents a semantically complete group of messages
type MessageGroup struct {
	ID       string     `json:"id"`                 // Message group ID
	Messages []*Message `json:"messages"`           // List of messages
	Metadata *Metadata  `json:"metadata,omitempty"` // Metadata
}

// Built-in message types that all adapters must support
// These types have standardized Props structures
const (
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

// Standard Props structures for built-in types

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
