package aigc

import "github.com/yaoapp/kun/exception"

// DSL the connector DSL
type DSL struct {
	ID        string   `json:"-"`
	Name      string   `json:"name,omitempty"`
	Connector string   `json:"connector"`
	Process   string   `json:"process,omitempty"`
	Prompts   []Prompt `json:"prompts"`
	Optional  Optional `json:"optional,omitempty"`
	AI        AI       `json:"-"`
}

// Prompt a prompt
type Prompt struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	User    string `json:"user,omitempty"`
}

// Optional optional
type Optional struct {
	Autopilot bool `json:"autopilot,omitempty"`
	JSON      bool `json:"json,omitempty"`
}

// AI the AI interface
type AI interface {
	ChatCompletions(messages []map[string]interface{}, option map[string]interface{}, cb func(data []byte) int) (interface{}, *exception.Exception)
	GetContent(response interface{}) (string, *exception.Exception)
	Embeddings(input interface{}, user string) (interface{}, *exception.Exception)
	Tiktoken(input string) (int, error)
	MaxToken() int
}
