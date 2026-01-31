package proxy

// ============================================
// Anthropic API Types
// ============================================

// AnthropicRequest represents a request to the Anthropic Messages API
type AnthropicRequest struct {
	Model         string               `json:"model"`
	Messages      []AnthropicMsg       `json:"messages"`
	System        interface{}          `json:"system,omitempty"` // string or []SystemBlock
	MaxTokens     int                  `json:"max_tokens"`
	Stream        bool                 `json:"stream,omitempty"`
	Temperature   *float64             `json:"temperature,omitempty"`
	TopP          *float64             `json:"top_p,omitempty"`
	TopK          *int                 `json:"top_k,omitempty"`
	StopSequences []string             `json:"stop_sequences,omitempty"`
	Tools         []AnthropicTool      `json:"tools,omitempty"`
	ToolChoice    *AnthropicToolChoice `json:"tool_choice,omitempty"`
	Metadata      map[string]string    `json:"metadata,omitempty"`
}

// AnthropicMsg represents a message in Anthropic format
type AnthropicMsg struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // string or []ContentBlock
}

// ContentBlock represents a content block in Anthropic messages
type ContentBlock struct {
	Type string `json:"type"`

	// For text blocks
	Text string `json:"text,omitempty"`

	// For image blocks
	Source *ImageSource `json:"source,omitempty"`

	// For tool_use blocks
	ID    string      `json:"id,omitempty"`
	Name  string      `json:"name,omitempty"`
	Input interface{} `json:"input,omitempty"`

	// For tool_result blocks
	ToolUseID string      `json:"tool_use_id,omitempty"`
	Content   interface{} `json:"content,omitempty"` // string or []ContentBlock
	IsError   bool        `json:"is_error,omitempty"`
}

// ImageSource represents an image source in Anthropic format
type ImageSource struct {
	Type      string `json:"type"`                 // "base64" or "url"
	MediaType string `json:"media_type,omitempty"` // e.g., "image/jpeg"
	Data      string `json:"data,omitempty"`       // base64 encoded data
	URL       string `json:"url,omitempty"`        // URL for url type
}

// SystemBlock represents a system message block
type SystemBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// AnthropicTool represents a tool definition in Anthropic format
type AnthropicTool struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	InputSchema interface{} `json:"input_schema"`
}

// AnthropicToolChoice represents tool choice in Anthropic format
type AnthropicToolChoice struct {
	Type string `json:"type"` // "auto", "any", "tool"
	Name string `json:"name,omitempty"`
}

// AnthropicResponse represents a response from the Anthropic Messages API
type AnthropicResponse struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	Role         string         `json:"role"`
	Content      []ContentBlock `json:"content"`
	Model        string         `json:"model"`
	StopReason   *string        `json:"stop_reason"`
	StopSequence *string        `json:"stop_sequence,omitempty"`
	Usage        *Usage         `json:"usage"`
}

// Usage represents token usage statistics
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// AnthropicStreamEvent represents an SSE event in Anthropic format
type AnthropicStreamEvent struct {
	Type         string             `json:"type"`
	Index        int                `json:"index,omitempty"`
	Message      *AnthropicResponse `json:"message,omitempty"`
	ContentBlock *ContentBlock      `json:"content_block,omitempty"`
	Delta        *DeltaContent      `json:"delta,omitempty"`
	Usage        *Usage             `json:"usage,omitempty"`
}

// DeltaContent represents delta content in streaming
type DeltaContent struct {
	Type        string  `json:"type,omitempty"`
	Text        string  `json:"text,omitempty"`
	PartialJSON string  `json:"partial_json,omitempty"`
	StopReason  *string `json:"stop_reason,omitempty"`
}

// ============================================
// OpenAI API Types
// ============================================

// OpenAIRequest represents a request to OpenAI Chat Completions API
type OpenAIRequest struct {
	Model         string         `json:"model"`
	Messages      []OpenAIMsg    `json:"messages"`
	MaxTokens     int            `json:"max_tokens,omitempty"`
	Stream        bool           `json:"stream,omitempty"`
	StreamOptions *StreamOptions `json:"stream_options,omitempty"`
	Temperature   *float64       `json:"temperature,omitempty"`
	TopP          *float64       `json:"top_p,omitempty"`
	Stop          []string       `json:"stop,omitempty"`
	Tools         []OpenAITool   `json:"tools,omitempty"`
	ToolChoice    interface{}    `json:"tool_choice,omitempty"` // "auto", "none", "required", or object
}

// StreamOptions represents stream options in OpenAI format
type StreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

// OpenAIMsg represents a message in OpenAI format
type OpenAIMsg struct {
	Role       string           `json:"role"`
	Content    interface{}      `json:"content,omitempty"` // string or []OpenAIContent
	ToolCalls  []OpenAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
	Name       string           `json:"name,omitempty"`
}

// OpenAIContent represents content in OpenAI messages (for multimodal)
type OpenAIContent struct {
	Type     string          `json:"type"`
	Text     string          `json:"text,omitempty"`
	ImageURL *OpenAIImageURL `json:"image_url,omitempty"`
}

// OpenAIImageURL represents an image URL in OpenAI format
type OpenAIImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"` // "auto", "low", "high"
}

// OpenAITool represents a tool definition in OpenAI format
type OpenAITool struct {
	Type     string         `json:"type"`
	Function OpenAIFunction `json:"function"`
}

// OpenAIFunction represents a function in OpenAI tool
type OpenAIFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Parameters  interface{} `json:"parameters"`
}

// OpenAIToolCall represents a tool call in OpenAI format
type OpenAIToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function OpenAIFunctionCall `json:"function"`
	Index    *int               `json:"index,omitempty"` // For streaming
}

// OpenAIFunctionCall represents a function call in OpenAI format
type OpenAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// OpenAIResponse represents a response from OpenAI Chat Completions API
type OpenAIResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []OpenAIChoice `json:"choices"`
	Usage   *OpenAIUsage   `json:"usage,omitempty"`
}

// OpenAIChoice represents a choice in OpenAI response
type OpenAIChoice struct {
	Index        int       `json:"index"`
	Message      OpenAIMsg `json:"message"`
	FinishReason string    `json:"finish_reason"`
}

// OpenAIUsage represents usage statistics in OpenAI format
type OpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// OpenAIStreamChunk represents a streaming chunk from OpenAI
type OpenAIStreamChunk struct {
	ID      string               `json:"id"`
	Object  string               `json:"object"`
	Created int64                `json:"created"`
	Model   string               `json:"model"`
	Choices []OpenAIStreamChoice `json:"choices"`
	Usage   *OpenAIUsage         `json:"usage,omitempty"`
}

// OpenAIStreamChoice represents a choice in OpenAI streaming response
type OpenAIStreamChoice struct {
	Index        int               `json:"index"`
	Delta        OpenAIStreamDelta `json:"delta"`
	FinishReason string            `json:"finish_reason,omitempty"`
}

// OpenAIStreamDelta represents delta content in OpenAI streaming
type OpenAIStreamDelta struct {
	Role      string           `json:"role,omitempty"`
	Content   string           `json:"content,omitempty"`
	ToolCalls []OpenAIToolCall `json:"tool_calls,omitempty"`
}

// ============================================
// Internal Types
// ============================================

// ToolCallAccumulator accumulates tool call data during streaming
type ToolCallAccumulator struct {
	Index int
	ID    string
	Name  string
	Args  string
}
