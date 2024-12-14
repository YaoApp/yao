package assistant

import (
	"context"
)

// API the assistant API interface
type API interface {
	Chat(ctx context.Context, messages []map[string]interface{}, option map[string]interface{}, cb func(data []byte) int) error
	List(ctx context.Context, param QueryParam) ([]Assistant, error)
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
	ID          string                 `json:"assistant_id"`          // Assistant ID
	Name        string                 `json:"name,omitempty"`        // Assistant Name
	Connector   string                 `json:"connector"`             // AI Connector
	Description string                 `json:"description,omitempty"` // Assistant Description
	Option      map[string]interface{} `json:"option,omitempty"`      // AI Option
	Prompts     []Prompt               `json:"prompts,omitempty"`     // AI Prompts
	API         API                    `json:"-" yaml:"-"`            // Assistant API
}
