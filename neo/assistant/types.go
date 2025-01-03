package assistant

import (
	"context"
	"io"
	"mime/multipart"

	"github.com/yaoapp/gou/rag/driver"
	v8 "github.com/yaoapp/gou/runtime/v8"
	api "github.com/yaoapp/yao/openai"
)

// API the assistant API interface
type API interface {
	Chat(ctx context.Context, messages []map[string]interface{}, option map[string]interface{}, cb func(data []byte) int) error
	Upload(ctx context.Context, file *multipart.FileHeader, reader io.Reader, option map[string]interface{}) (*File, error)
	Download(ctx context.Context, fileID string) (*FileResponse, error)
	ReadBase64(ctx context.Context, fileID string) (string, error)
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
	Flows       []map[string]interface{} `json:"flows,omitempty"`       // Assistant Flows
	Script      *v8.Script               `json:"-" yaml:"-"`            // Assistant Script
	CreatedAt   int64                    `json:"created_at"`            // Creation timestamp
	UpdatedAt   int64                    `json:"updated_at"`            // Last update timestamp
	openai      *api.OpenAI              // OpenAI API
}

// File the file
type File struct {
	ID          string `json:"file_id"`
	Bytes       int    `json:"bytes"`
	CreatedAt   int    `json:"created_at"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
}

// FileResponse represents a file download response
type FileResponse struct {
	Reader      io.ReadCloser
	ContentType string
	Extension   string
}
