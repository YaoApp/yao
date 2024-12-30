package assistant

import (
	"context"
	"io"
	"mime/multipart"
)

// API the assistant API interface
type API interface {
	Chat(ctx context.Context, messages []map[string]interface{}, option map[string]interface{}, cb func(data []byte) int) error
	Upload(ctx context.Context, file *multipart.FileHeader, reader io.Reader, option map[string]interface{}) (*File, error)
	Download(ctx context.Context, fileID string) (*FileResponse, error)
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
	Description string                   `json:"description,omitempty"` // Assistant Description
	Option      map[string]interface{}   `json:"option,omitempty"`      // AI Option
	Prompts     []Prompt                 `json:"prompts,omitempty"`     // AI Prompts
	Flows       []map[string]interface{} `json:"flows,omitempty"`       // Assistant Flows
	API         API                      `json:"-" yaml:"-"`            // Assistant API
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
