package assistant

import (
	"context"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/agent/assistant/hook"
	chatctx "github.com/yaoapp/yao/agent/context"
	"github.com/yaoapp/yao/agent/message"
	store "github.com/yaoapp/yao/agent/store/types"
	api "github.com/yaoapp/yao/openai"
)

const (
	// HookErrorMethodNotFound is the error message for method not found
	HookErrorMethodNotFound = "method not found"
)

// API the assistant API interface
type API interface {
	Chat(ctx context.Context, messages []message.Message, option map[string]interface{}, cb func(data []byte) int) error
	GetPlaceholder(locale string) *store.Placeholder
	Execute(c *gin.Context, ctx chatctx.Context, input interface{}, options map[string]interface{}, callback ...interface{}) (interface{}, error)
	Call(c *gin.Context, payload APIPayload) (interface{}, error)
}

// APIPayload the API payload
type APIPayload struct {
	Sid  string        `json:"sid"`
	Name string        `json:"name"`
	Args []interface{} `json:"args,omitempty"`
}

// ResHookInit the response of the init hook
type ResHookInit struct {
	AssistantID string                 `json:"assistant_id,omitempty"`
	ChatID      string                 `json:"chat_id,omitempty"`
	Next        *NextAction            `json:"next,omitempty"`
	Input       []message.Message      `json:"input,omitempty"`
	Options     map[string]interface{} `json:"options,omitempty"`
	Result      any                    `json:"result,omitempty"`
}

// ResHookStream the response of the stream hook
type ResHookStream struct {
	Silent bool           `json:"silent,omitempty"` // Whether to suppress the output
	Next   *NextAction    `json:"next,omitempty"`   // The next action
	Output []message.Data `json:"output,omitempty"` // The output
}

// ResHookDone the response of the done hook
type ResHookDone struct {
	Next   *NextAction       `json:"next,omitempty"`
	Input  []message.Message `json:"input,omitempty"`
	Output []message.Data    `json:"output,omitempty"`
	Result any               `json:"result,omitempty"`
}

// ResHookFail the response of the fail hook
type ResHookFail struct {
	Next   *NextAction       `json:"next,omitempty"`
	Input  []message.Message `json:"input,omitempty"`
	Output string            `json:"output,omitempty"`
	Error  string            `json:"error,omitempty"`
}

// NextAction the next action
type NextAction struct {
	Action  string                 `json:"action"`
	Payload map[string]interface{} `json:"payload,omitempty"`
}

// SearchOption the search option
type SearchOption struct {
	WebSearch *bool `json:"web_search,omitempty" yaml:"web_search,omitempty"` // Whether to search the web
	Knowledge *bool `json:"knowledge,omitempty" yaml:"knowledge,omitempty"`   // Whether to search the knowledge
}

// Prompt a prompt
type Prompt struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

// QueryParam the assistant query param
type QueryParam struct {
	Limit  uint   `json:"limit"`
	Order  string `json:"order"`
	After  string `json:"after"`
	Before string `json:"before"`
}

// Assistant the assistant
type Assistant struct {
	store.AssistantModel
	Search *SearchOption `json:"search,omitempty" yaml:"search,omitempty"` // Whether this assistant supports search
	Script *hook.Script  `json:"-" yaml:"-"`                               // Assistant Script

	// Internal
	// ===============================
	openai *api.OpenAI // OpenAI API
	search bool        // Whether this assistant supports search
	vision bool        // Whether this assistant supports vision
	// toolCalls    bool        // Whether this assistant supports tool_calls
	initHook     bool   // Whether this assistant has an init hook
	runtimeTools []Tool // Converted tools for business logic (OpenAI format)
}

// ModelCapabilities defines the capabilities of a language model
// This configuration is loaded from agent/models.yml
type ModelCapabilities struct {
	Vision     interface{} `json:"vision,omitempty" yaml:"vision,omitempty"`         // Supports vision/image input: bool or VisionFormat string ("openai", "claude"/"base64", "default")
	Tools      bool        `json:"tools,omitempty" yaml:"tools,omitempty"`           // Supports tool/function calling (deprecated, use ToolCalls)
	ToolCalls  bool        `json:"tool_calls,omitempty" yaml:"tool_calls,omitempty"` // Supports tool/function calling
	Audio      bool        `json:"audio,omitempty" yaml:"audio,omitempty"`           // Supports audio input/output
	Reasoning  bool        `json:"reasoning,omitempty" yaml:"reasoning,omitempty"`   // Supports reasoning/thinking mode (o1, DeepSeek R1)
	Streaming  bool        `json:"streaming,omitempty" yaml:"streaming,omitempty"`   // Supports streaming responses
	JSON       bool        `json:"json,omitempty" yaml:"json,omitempty"`             // Supports JSON mode
	Multimodal bool        `json:"multimodal,omitempty" yaml:"multimodal,omitempty"` // Supports multimodal input
}

// VisionCapableModels list of LLM models that support vision capabilities
var VisionCapableModels = map[string]bool{
	// OpenAI Models
	"gpt-4-vision-preview": true,
	"gpt-4v":               true, // Alias for gpt-4-vision-preview

	// Anthropic Models
	"claude-3-opus":   true, // Most capable Claude model
	"claude-3-sonnet": true, // Balanced Claude model
	"claude-3-haiku":  true, // Fast and efficient Claude model

	// Google Models
	"gemini-pro-vision": true,

	// Open Source Models
	"llava-13b": true,
	"cogvlm":    true,
	"qwen-vl":   true,
	"yi-vl":     true,

	// Custom Models
	"gpt-4o":      true, // Custom OpenAI compatible model
	"gpt-4o-mini": true, // Custom OpenAI compatible model - mini version
}

// File the file
type File struct {
	ID          string   `json:"file_id"`
	Bytes       int      `json:"bytes"`
	CreatedAt   int      `json:"created_at"`
	Filename    string   `json:"filename"`
	ContentType string   `json:"content_type"`
	Description string   `json:"description,omitempty"` // Vision analysis result or other description
	URL         string   `json:"url,omitempty"`         // Vision URL for vision-capable models
	DocIDs      []string `json:"doc_ids,omitempty"`     // RAG document IDs
}

// FileResponse represents a file download response
type FileResponse struct {
	Reader      io.ReadCloser
	ContentType string
	Extension   string
}
