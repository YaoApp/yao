package context

import (
	"github.com/yaoapp/gou/connector/openai"
	"github.com/yaoapp/yao/agent/output/message"
)

// Uses represents the wrapper configurations for assistant
// Used to specify which assistant or MCP server to use for vision, audio, search, and fetch operations
type Uses struct {
	Vision string `json:"vision,omitempty"` // Vision processing tool. Format: "agent" or "mcp:server_id"
	Audio  string `json:"audio,omitempty"`  // Audio processing tool. Format: "agent" or "mcp:server_id"
	Search string `json:"search,omitempty"` // Search tool. Format: "builtin", "disabled", "<assistant-id>", "mcp:<server>.<tool>"
	Fetch  string `json:"fetch,omitempty"`  // Fetch/retrieval tool. Format: "agent" or "mcp:server_id"

	// Search-related processing tools (NLP)
	Web      string `json:"web,omitempty"`      // Web search handler: "builtin", "<assistant-id>", "mcp:<server>.<tool>"
	Keyword  string `json:"keyword,omitempty"`  // Keyword extraction: "builtin", "<assistant-id>", "mcp:<server>.<tool>"
	QueryDSL string `json:"querydsl,omitempty"` // QueryDSL generation: "builtin", "<assistant-id>", "mcp:<server>.<tool>"
	Rerank   string `json:"rerank,omitempty"`   // Result reranking: "builtin", "<assistant-id>", "mcp:<server>.<tool>"
}

// VisionFormat specifies the vision input format
type VisionFormat string

// Vision format constants define how image inputs are processed
const (
	// VisionFormatNone indicates no vision support
	VisionFormatNone VisionFormat = ""
	// VisionFormatOpenAI indicates OpenAI format (image_url with URL)
	VisionFormatOpenAI VisionFormat = "openai"
	// VisionFormatClaude indicates Claude/Anthropic format (image with base64)
	VisionFormatClaude VisionFormat = "claude"
	// VisionFormatBase64 forces base64 conversion (alias for claude)
	VisionFormatBase64 VisionFormat = "base64"
	// VisionFormatDefault enables auto-detection of format
	VisionFormatDefault VisionFormat = "default"
)

// GetVisionSupport returns whether vision is supported and the format
func GetVisionSupport(cap *openai.Capabilities) (bool, VisionFormat) {
	if cap == nil || cap.Vision == nil {
		return false, VisionFormatNone
	}

	switch v := cap.Vision.(type) {
	case bool:
		// Legacy bool format
		return v, VisionFormatDefault
	case string:
		// String format
		if v == "" || v == string(VisionFormatNone) {
			return false, VisionFormatNone
		}
		return true, VisionFormat(v)
	case VisionFormat:
		// Direct VisionFormat type
		if v == VisionFormatNone || v == "" {
			return false, VisionFormatNone
		}
		return true, v
	default:
		return false, VisionFormatNone
	}
}

// CompletionOptions the completion request options
// These options are extracted from HookCreateResponse and Context, then passed to the LLM connector
// Compatible with OpenAI Chat Completion API: https://platform.openai.com/docs/api-reference/chat/create
type CompletionOptions struct {
	// Model capabilities (used by LLM to select appropriate provider)
	// nil means capabilities are not specified/checked
	Capabilities *openai.Capabilities `json:"capabilities,omitempty"`

	// User-specified tools for vision, audio, search, and fetch processing
	Uses *Uses `json:"uses,omitempty"`

	// ForceUses controls whether to force using Uses tools even when model has native capabilities
	// When true: Always use tools specified in Uses, ignore model's native multimodal capabilities
	// When false (default): Use model's native capabilities if available, fallback to Uses tools
	// This is useful when you want consistent behavior across different models or prefer specific tools
	ForceUses bool `json:"force_uses,omitempty"`

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

	// Reasoning configuration (for reasoning models like o1, GPT-5)
	ReasoningEffort *string `json:"reasoning_effort,omitempty"` // Reasoning effort level: "low", "medium", "high" (o1 and GPT-5 only)

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
	Usage *message.UsageInfo `json:"usage,omitempty"` // Token usage statistics

	// Additional metadata
	SystemFingerprint string                 `json:"system_fingerprint,omitempty"` // System fingerprint for reproducibility
	Metadata          map[string]interface{} `json:"metadata,omitempty"`           // Additional provider-specific metadata

	// Raw response data (for debugging and special cases)
	Raw interface{} `json:"raw,omitempty"` // Original raw response from the LLM provider
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
