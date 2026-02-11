package anthropic

import (
	"github.com/yaoapp/yao/agent/output/message"
)

// ============================================================
// Anthropic Messages API types
// Reference: https://docs.anthropic.com/en/api/messages
// ============================================================

// StreamEvent represents an SSE event from Anthropic streaming API
type StreamEvent struct {
	Type string `json:"type"`
}

// MessageStartEvent represents the message_start SSE event
type MessageStartEvent struct {
	Type    string       `json:"type"`
	Message MessageStart `json:"message"`
}

// MessageStart represents the message object in message_start event
type MessageStart struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	Role         string         `json:"role"`
	Content      []ContentBlock `json:"content"`
	Model        string         `json:"model"`
	StopReason   *string        `json:"stop_reason"`
	StopSequence *string        `json:"stop_sequence"`
	Usage        *UsageInfo     `json:"usage,omitempty"`
}

// ContentBlockStartEvent represents the content_block_start SSE event
type ContentBlockStartEvent struct {
	Type         string       `json:"type"`
	Index        int          `json:"index"`
	ContentBlock ContentBlock `json:"content_block"`
}

// ContentBlockDeltaEvent represents the content_block_delta SSE event
type ContentBlockDeltaEvent struct {
	Type  string     `json:"type"`
	Index int        `json:"index"`
	Delta DeltaBlock `json:"delta"`
}

// ContentBlockStopEvent represents the content_block_stop SSE event
type ContentBlockStopEvent struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
}

// MessageDeltaEvent represents the message_delta SSE event
type MessageDeltaEvent struct {
	Type  string       `json:"type"`
	Delta MessageDelta `json:"delta"`
	Usage *DeltaUsage  `json:"usage,omitempty"`
}

// MessageDelta represents the delta in message_delta event
type MessageDelta struct {
	StopReason   string  `json:"stop_reason,omitempty"`
	StopSequence *string `json:"stop_sequence,omitempty"`
}

// DeltaUsage represents usage in message_delta event
type DeltaUsage struct {
	OutputTokens int `json:"output_tokens"`
}

// ContentBlock represents a content block in the response
type ContentBlock struct {
	Type      string      `json:"type"`                // "text", "thinking", "tool_use"
	Text      string      `json:"text,omitempty"`      // for type "text"
	Thinking  string      `json:"thinking,omitempty"`  // for type "thinking"
	Signature string      `json:"signature,omitempty"` // for type "thinking"
	ID        string      `json:"id,omitempty"`        // for type "tool_use"
	Name      string      `json:"name,omitempty"`      // for type "tool_use"
	Input     interface{} `json:"input,omitempty"`     // for type "tool_use"
}

// DeltaBlock represents a delta block in streaming
type DeltaBlock struct {
	Type        string `json:"type"`                   // "text_delta", "thinking_delta", "input_json_delta"
	Text        string `json:"text,omitempty"`         // for type "text_delta"
	Thinking    string `json:"thinking,omitempty"`     // for type "thinking_delta"
	PartialJSON string `json:"partial_json,omitempty"` // for type "input_json_delta"
}

// UsageInfo represents token usage information
type UsageInfo struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}

// NonStreamResponse represents the full non-streaming response from Anthropic API
type NonStreamResponse struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	Role         string         `json:"role"`
	Content      []ContentBlock `json:"content"`
	Model        string         `json:"model"`
	StopReason   string         `json:"stop_reason"`
	StopSequence *string        `json:"stop_sequence"`
	Usage        *UsageInfo     `json:"usage,omitempty"`
}

// APIError represents an error response from Anthropic API
type APIError struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// streamAccumulator accumulates streaming response data
type streamAccumulator struct {
	id                string
	model             string
	role              string
	content           string
	thinkingContent   string
	thinkingSignature string
	toolCalls         map[int]*accumulatedToolCall
	stopReason        string
	usage             *message.UsageInfo

	// Current content block tracking
	currentBlockIndex int
	currentBlockType  string
}

// accumulatedToolCall accumulates a single tool call from streaming
type accumulatedToolCall struct {
	id        string
	name      string
	inputJSON string
}

// messageTracker tracks message lifecycle for stream events
type messageTracker struct {
	active       bool
	messageID    string
	messageType  message.StreamChunkType
	startTime    int64
	chunkCount   int
	toolCallInfo *message.EventToolCallInfo
	idGenerator  *message.IDGenerator
}
