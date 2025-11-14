package context

// ModelCapabilities defines the capabilities of a language model
// Used by LLM to select appropriate provider and validate requests
type ModelCapabilities struct {
	Vision     *bool `json:"vision,omitempty"`     // Supports vision/image input
	ToolCalls  *bool `json:"tool_calls,omitempty"` // Supports tool/function calling
	Audio      *bool `json:"audio,omitempty"`      // Supports audio input/output
	Reasoning  *bool `json:"reasoning,omitempty"`  // Supports reasoning/thinking mode (o1, DeepSeek R1)
	Streaming  *bool `json:"streaming,omitempty"`  // Supports streaming responses
	JSON       *bool `json:"json,omitempty"`       // Supports JSON mode
	Multimodal *bool `json:"multimodal,omitempty"` // Supports multimodal input (text + images + audio)
}

// CompletionOptions the completion request options
// These options are extracted from HookCreateResponse and Context, then passed to the LLM connector
// Compatible with OpenAI Chat Completion API: https://platform.openai.com/docs/api-reference/chat/create
type CompletionOptions struct {
	// Model capabilities (used by LLM to select appropriate provider)
	// nil means capabilities are not specified/checked
	Capabilities *ModelCapabilities `json:"capabilities,omitempty"`

	// Wrapper configurations for vision and audio processing
	// Format: "agent" (default) or "mcp:mcp_server_id"
	VisionWrapper string `json:"vision_wrapper,omitempty"` // Vision processing wrapper (for image/video description)
	AudioWrapper  string `json:"audio_wrapper,omitempty"`  // Audio processing wrapper (for speech-to-text/text-to-speech)

	// Audio configuration (for models that support audio output)
	Audio *AudioConfig `json:"audio,omitempty"`

	// Generation parameters
	Temperature         *float64 `json:"temperature,omitempty"`           // Sampling temperature (0-2), defaults to 1
	MaxTokens           *int     `json:"max_tokens,omitempty"`            // Maximum tokens to generate (deprecated, use MaxCompletionTokens)
	MaxCompletionTokens *int     `json:"max_completion_tokens,omitempty"` // Maximum tokens in completion
	TopP                *float64 `json:"top_p,omitempty"`                 // Nucleus sampling parameter (0-1), alternative to temperature
	N                   *int     `json:"n,omitempty"`                     // Number of chat completion choices to generate

	// Control parameters
	Stop             interface{}        `json:"stop,omitempty"`              // Up to 4 sequences where the API will stop generating (string or []string)
	PresencePenalty  *float64           `json:"presence_penalty,omitempty"`  // Presence penalty (-2.0 to 2.0)
	FrequencyPenalty *float64           `json:"frequency_penalty,omitempty"` // Frequency penalty (-2.0 to 2.0)
	LogitBias        map[string]float64 `json:"logit_bias,omitempty"`        // Modify likelihood of specified tokens appearing

	// User and response format
	User           string                 `json:"user,omitempty"`            // Unique identifier representing end-user
	ResponseFormat map[string]interface{} `json:"response_format,omitempty"` // Format of the response (e.g., {"type": "json_object"})
	Seed           *int                   `json:"seed,omitempty"`            // Seed for deterministic sampling

	// Tool calling
	Tools      []map[string]interface{} `json:"tools,omitempty"`       // List of tools the model may call
	ToolChoice interface{}              `json:"tool_choice,omitempty"` // Controls which tool is called ("none", "auto", "required", or specific tool)

	// Streaming configuration
	Stream        *bool          `json:"stream,omitempty"`         // If true, stream partial message deltas
	StreamOptions *StreamOptions `json:"stream_options,omitempty"` // Options for streaming response

	// CUI Context information (from Context)
	Route    string                 `json:"route,omitempty"`    // Route of the request for CUI context
	Metadata map[string]interface{} `json:"metadata,omitempty"` // Metadata to pass to the page for CUI context
}

// CompletionResponse represents the unified completion response
// Compatible with OpenAI chat completion response format
type CompletionResponse struct {
	// Response metadata
	ID      string `json:"id"`      // Unique identifier for the completion
	Object  string `json:"object"`  // Object type (e.g., "chat.completion")
	Created int64  `json:"created"` // Unix timestamp of creation
	Model   string `json:"model"`   // Model used for completion

	// Completion content (these fields can coexist)
	Content          string           `json:"content"`                     // Text content (regular response text)
	ReasoningContent string           `json:"reasoning_content,omitempty"` // Reasoning/thinking content (for o1, DeepSeek R1, etc.)
	ToolCalls        []ToolCallResult `json:"tool_calls,omitempty"`        // Tool calls made by the model
	Refusal          string           `json:"refusal,omitempty"`           // Refusal message if model refused to answer
	ContentTypes     []ContentType    `json:"content_types"`               // Types of content present (can have multiple simultaneously)

	// Raw response data
	Raw interface{} `json:"raw,omitempty"` // Original raw response from the LLM provider (for debugging and special cases)

	// Completion metadata
	FinishReason string `json:"finish_reason"` // Reason for completion (stop, length, tool_calls, content_filter, etc.)

	// Usage statistics
	Usage *UsageInfo `json:"usage,omitempty"` // Token usage statistics

	// Additional metadata
	SystemFingerprint string                 `json:"system_fingerprint,omitempty"` // System fingerprint for reproducibility
	Metadata          map[string]interface{} `json:"metadata,omitempty"`           // Additional metadata
}

// ContentType represents the type of content in the response
// A response can contain multiple content types simultaneously
type ContentType string

// Content type constants - a response can have multiple types simultaneously
// For example: text + reasoning, or text + tool_call, or all three
const (
	ContentTypeText      ContentType = "text"      // Regular text content
	ContentTypeReasoning ContentType = "reasoning" // Reasoning/thinking content (o1, DeepSeek R1, etc.)
	ContentTypeToolCall  ContentType = "tool_call" // Tool/function call
	ContentTypeRefusal   ContentType = "refusal"   // Model refused to answer
	ContentTypeEmpty     ContentType = "empty"     // Empty response (no content)
)

// UsageInfo represents token usage statistics
type UsageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`     // Tokens in the prompt
	CompletionTokens int `json:"completion_tokens"` // Tokens in the completion
	TotalTokens      int `json:"total_tokens"`      // Total tokens used

	// Detailed token breakdown (for models with reasoning)
	PromptTokensDetails     *TokenDetails `json:"prompt_tokens_details,omitempty"`     // Detailed prompt token breakdown
	CompletionTokensDetails *TokenDetails `json:"completion_tokens_details,omitempty"` // Detailed completion token breakdown
}

// TokenDetails provides detailed token usage breakdown
type TokenDetails struct {
	CachedTokens    int `json:"cached_tokens,omitempty"`    // Tokens from cache
	ReasoningTokens int `json:"reasoning_tokens,omitempty"` // Tokens used for reasoning/thinking
	AudioTokens     int `json:"audio_tokens,omitempty"`     // Tokens used for audio
	TextTokens      int `json:"text_tokens,omitempty"`      // Tokens used for text
}

// ToolCallResult represents a tool call result in the completion
type ToolCallResult struct {
	ID       string             `json:"id"`       // Tool call ID
	Type     string             `json:"type"`     // Tool call type (usually "function")
	Function FunctionCallResult `json:"function"` // Function call details
}

// FunctionCallResult represents a function call result
type FunctionCallResult struct {
	Name      string `json:"name"`      // Function name
	Arguments string `json:"arguments"` // Function arguments as JSON string
}

// FinishReason constants
const (
	FinishReasonStop          = "stop"           // Natural stop point
	FinishReasonLength        = "length"         // Max tokens reached
	FinishReasonToolCalls     = "tool_calls"     // Tool calls made
	FinishReasonContentFilter = "content_filter" // Content filtered
	FinishReasonFunctionCall  = "function_call"  // Function call (deprecated)
	FinishReasonError         = "error"          // Error occurred
)
