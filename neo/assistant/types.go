package assistant

import (
	"context"
	"io"
	"mime/multipart"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/rag/driver"
	v8 "github.com/yaoapp/gou/runtime/v8"
	chatctx "github.com/yaoapp/yao/neo/context"
	"github.com/yaoapp/yao/neo/message"
	api "github.com/yaoapp/yao/openai"
)

const (
	// HookErrorMethodNotFound is the error message for method not found
	HookErrorMethodNotFound = "method not found"
)

// API the assistant API interface
type API interface {
	Chat(ctx context.Context, messages []message.Message, option map[string]interface{}, cb func(data []byte) int) error
	Upload(ctx context.Context, file *multipart.FileHeader, reader io.Reader, option map[string]interface{}) (*File, error)
	Download(ctx context.Context, fileID string) (*FileResponse, error)
	ReadBase64(ctx context.Context, fileID string) (string, error)
	Execute(c *gin.Context, ctx chatctx.Context, input string, options map[string]interface{}) error
	HookInit(c *gin.Context, ctx chatctx.Context, input []message.Message, options map[string]interface{}) (*ResHookInit, error)
}

// ResHookInit the response of the init hook
type ResHookInit struct {
	AssistantID string                 `json:"assistant_id,omitempty"`
	ChatID      string                 `json:"chat_id,omitempty"`
	Next        *NextAction            `json:"next,omitempty"`
	Input       []message.Message      `json:"input,omitempty"`
	Options     map[string]interface{} `json:"options,omitempty"`
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

// RAG the RAG interface
type RAG struct {
	Engine     driver.Engine
	Uploader   driver.FileUpload
	Vectorizer driver.Vectorizer
	Setting    RAGSetting
}

// RAGSetting the RAG setting
type RAGSetting struct {
	IndexPrefix string `json:"index_prefix" yaml:"index_prefix"`
}

// Prompt a prompt
type Prompt struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

// Function a function
type Function struct {
	Type     string `json:"type"`
	Function struct {
		Name        string                 `json:"name"`
		Description string                 `json:"description"`
		Parameters  map[string]interface{} `json:"parameters"`
	} `json:"function"`
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
	ID          string                   `json:"assistant_id"`          // Assistant ID
	Type        string                   `json:"type,omitempty"`        // Assistant Type, default is assistant
	Name        string                   `json:"name,omitempty"`        // Assistant Name
	Avatar      string                   `json:"avatar,omitempty"`      // Assistant Avatar
	Connector   string                   `json:"connector"`             // AI Connector
	Path        string                   `json:"path,omitempty"`        // Assistant Path
	BuiltIn     bool                     `json:"built_in,omitempty"`    // Whether this is a built-in assistant
	Sort        int                      `json:"sort,omitempty"`        // Assistant Sort
	Description string                   `json:"description,omitempty"` // Assistant Description
	Tags        []string                 `json:"tags,omitempty"`        // Assistant Tags
	Readonly    bool                     `json:"readonly,omitempty"`    // Whether this assistant is readonly
	Mentionable bool                     `json:"mentionable,omitempty"` // Whether this assistant is mentionable
	Automated   bool                     `json:"automated,omitempty"`   // Whether this assistant is automated
	Options     map[string]interface{}   `json:"options,omitempty"`     // AI Options
	Prompts     []Prompt                 `json:"prompts,omitempty"`     // AI Prompts
	Functions   []Function               `json:"functions,omitempty"`   // Assistant Functions
	Flows       []map[string]interface{} `json:"flows,omitempty"`       // Assistant Flows
	Script      *v8.Script               `json:"-" yaml:"-"`            // Assistant Script
	CreatedAt   int64                    `json:"created_at"`            // Creation timestamp
	UpdatedAt   int64                    `json:"updated_at"`            // Last update timestamp
	openai      *api.OpenAI              // OpenAI API
	vision      bool                     // Whether this assistant supports vision
	initHook    bool                     // Whether this assistant has an init hook
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
