package neo

import "github.com/yaoapp/yao/aigc"

// Neo AI assistant
type Neo struct {
	ID                  string                 `json:"-"`
	Name                string                 `json:"name,omitempty"`
	Guard               string                 `json:"guard,omitempty"`
	Connector           string                 `json:"connector"`
	ConversationSetting ConversationSetting    `json:"conversation"`
	Option              map[string]interface{} `json:"option"`
	Prompts             []aigc.Prompt          `json:"prompts"`
	Allows              []string               `json:"allows,omitempty"`
	AI                  aigc.AI                `json:"-"`
	Conversation        Conversation           `json:"-"`
	Command             Command                `json:"-"`
}

// ConversationSetting the conversation config
type ConversationSetting struct {
	Connector string `json:"connector,omitempty"`
	Table     string `json:"table,omitempty"`
	MaxSize   int    `json:"max_size,omitempty"`
}

// Conversation the store interface
type Conversation interface {
	GetHistory(sid string) ([]map[string]interface{}, error)
	SaveHistory(sid string, messages []map[string]interface{}) error
}

// Command the command interface
type Command interface {
	Match(messages []map[string]interface{}) (bool, error)
}
