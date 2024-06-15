package aigc

import (
	"context"

	"github.com/yaoapp/kun/exception"
)

// DSL the connector DSL
type DSL struct {
	ID        string   `json:"-" yaml:"-"`
	Name      string   `json:"name,omitempty"`
	Connector string   `json:"connector,omitempty"`
	Process   string   `json:"process,omitempty"`
	Prompts   []Prompt `json:"prompts"`
	Optional  Optional `json:"optional,omitempty"`
	AI        AI       `json:"-" yaml:"-"`
}

// Prompt a prompt
type Prompt struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

// Optional optional
type Optional struct {
	Autopilot bool `json:"autopilot,omitempty"`
	JSON      bool `json:"json,omitempty"`
}

// AI the AI interface
type AI interface {
	ChatCompletions(messages []map[string]interface{}, option map[string]interface{}, cb func(data []byte) int) (interface{}, *exception.Exception)
	ChatCompletionsWith(ctx context.Context, messages []map[string]interface{}, option map[string]interface{}, cb func(data []byte) int) (interface{}, *exception.Exception)
	GetContent(response interface{}) (string, *exception.Exception)
	Embeddings(input interface{}, user string) (interface{}, *exception.Exception)
	Tiktoken(input string) (int, error)
	MaxToken() int
}
