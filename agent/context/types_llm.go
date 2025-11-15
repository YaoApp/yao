package context

// Uses represents the wrapper configurations for assistant
// Used to specify which assistant or MCP server to use for vision, audio, search, and fetch operations
type Uses struct {
	Vision string `json:"vision,omitempty"` // Vision processing tool. Format: "agent" or "mcp:server_id"
	Audio  string `json:"audio,omitempty"`  // Audio processing tool. Format: "agent" or "mcp:server_id"
	Search string `json:"search,omitempty"` // Search tool. Format: "agent" or "mcp:server_id"
	Fetch  string `json:"fetch,omitempty"`  // Fetch/retrieval tool. Format: "agent" or "mcp:server_id"
}

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

	// User-specified tools for vision, audio, search, and fetch processing
	Uses *Uses `json:"uses,omitempty"`

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
	User           string          `json:"user,omitempty"`            // Unique identifier representing end-user
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"` // Format of the model's output
	Seed           *int            `json:"seed,omitempty"`            // Seed for deterministic sampling

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

// CompletionResponse represents the unified LLM completion response
// This is Yao's internal representation that works with multiple LLM providers (OpenAI, Claude, DeepSeek, etc.)
type CompletionResponse struct {
	// Response metadata
	ID      string `json:"id"`      // Unique identifier for the completion
	Object  string `json:"object"`  // Object type (e.g., "chat.completion")
	Created int64  `json:"created"` // Unix timestamp of creation
	Model   string `json:"model"`   // Model used for completion

	// Response message (similar to OpenAI's message structure)
	Role    string      `json:"role"`              // Role of the response, typically "assistant"
	Content interface{} `json:"content,omitempty"` // string (text) or []ContentPart (multimodal: text, image, audio)

	// Tool calls (when model calls functions/tools)
	ToolCalls []ToolCall `json:"tool_calls,omitempty"` // Tool calls made by the model

	// Refusal (when model refuses to respond due to policy)
	Refusal string `json:"refusal,omitempty"` // Refusal message if model refused to answer

	// Reasoning content (for reasoning models like o1, DeepSeek R1)
	ReasoningContent string `json:"reasoning_content,omitempty"` // Thinking/reasoning process

	// Completion metadata
	FinishReason string `json:"finish_reason"` // Why generation stopped (stop, length, tool_calls, content_filter, etc.)

	// Usage statistics
	Usage *UsageInfo `json:"usage,omitempty"` // Token usage statistics

	// Additional metadata
	SystemFingerprint string                 `json:"system_fingerprint,omitempty"` // System fingerprint for reproducibility
	Metadata          map[string]interface{} `json:"metadata,omitempty"`           // Additional provider-specific metadata

	// Raw response data (for debugging and special cases)
	Raw interface{} `json:"raw,omitempty"` // Original raw response from the LLM provider
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

// FinishReason constants - why the model stopped generating tokens
const (
	FinishReasonStop          = "stop"           // Natural stop point or provided stop sequence reached
	FinishReasonLength        = "length"         // Max tokens limit reached
	FinishReasonToolCalls     = "tool_calls"     // Model called a tool
	FinishReasonContentFilter = "content_filter" // Content filtered due to safety
	FinishReasonFunctionCall  = "function_call"  // Model called a function (deprecated, use tool_calls)
)

// ResponseFormat specifies the format of the model's output
// Reference: https://platform.openai.com/docs/api-reference/chat/create#chat_create-response_format
type ResponseFormat struct {
	Type       ResponseFormatType `json:"type"`                  // Required: type of response format
	JSONSchema *JSONSchema        `json:"json_schema,omitempty"` // Optional: for type="json_schema", defines the schema
}

// ResponseFormatType represents the type of response format
type ResponseFormatType string

// Response format type constants
const (
	ResponseFormatText       ResponseFormatType = "text"        // Default text format
	ResponseFormatJSON       ResponseFormatType = "json_object" // JSON object format (no schema)
	ResponseFormatJSONSchema ResponseFormatType = "json_schema" // JSON with strict schema validation
)

// JSONSchema defines a JSON schema for structured output
// Used when ResponseFormat.Type is "json_schema"
type JSONSchema struct {
	Name        string      `json:"name"`                  // Required: name of the schema
	Description string      `json:"description,omitempty"` // Optional: description of the schema
	Schema      interface{} `json:"schema"`                // Required: JSON schema (*jsonschema.Schema or map[string]interface{})
	Strict      *bool       `json:"strict,omitempty"`      // Optional: whether to enforce strict schema validation (default: true)
}
