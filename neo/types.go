package neo

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/neo/assistant"
	"github.com/yaoapp/yao/neo/conversation"
)

// DSL AI assistant
type DSL struct {
	ID                  string                         `json:"-" yaml:"-"`
	Name                string                         `json:"name,omitempty" yaml:"name,omitempty"`
	Use                 string                         `json:"use,omitempty" yaml:"use,omitempty"` // Which assistant to use default
	Guard               string                         `json:"guard,omitempty" yaml:"guard,omitempty"`
	Connector           string                         `json:"connector" yaml:"connector"`
	ConversationSetting conversation.Setting           `json:"conversation" yaml:"conversation"`
	Option              map[string]interface{}         `json:"option" yaml:"option"`
	Prepare             string                         `json:"prepare,omitempty" yaml:"prepare,omitempty"`
	Create              string                         `json:"create,omitempty" yaml:"create,omitempty"`
	Write               string                         `json:"write,omitempty" yaml:"write,omitempty"`
	AssistantListHook   string                         `json:"assistants,omitempty" yaml:"assistants,omitempty"` // Get the assistant list from the hook
	Prompts             []assistant.Prompt             `json:"prompts,omitempty" yaml:"prompts,omitempty"`
	Allows              []string                       `json:"allows,omitempty" yaml:"allows,omitempty"`
	Assistant           assistant.API                  `json:"-" yaml:"-"` // The default assistant
	Conversation        conversation.Conversation      `json:"-" yaml:"-"`
	GuardHandlers       []gin.HandlerFunc              `json:"-" yaml:"-"`
	AssistantList       []assistant.Assistant          `json:"-" yaml:"-"`
	AssistantMaps       map[string]assistant.Assistant `json:"-" yaml:"-"`
}

// Context the context
type Context struct {
	Sid             string                 `json:"sid" yaml:"-"`           // Session ID
	ChatID          string                 `json:"chat_id,omitempty"`      // Chat ID, use to select chat
	AssistantID     string                 `json:"assistant_id,omitempty"` // Assistant ID, use to select assistant
	Stack           string                 `json:"stack,omitempty"`
	Path            string                 `json:"pathname,omitempty"`
	FormData        map[string]interface{} `json:"formdata,omitempty"`
	Field           *Field                 `json:"field,omitempty"`
	Namespace       string                 `json:"namespace,omitempty"`
	Config          map[string]interface{} `json:"config,omitempty"`
	Signal          interface{}            `json:"signal,omitempty"`
	context.Context `json:"-" yaml:"-"`
}

// Field the context field
type Field struct {
	Name string `json:"name,omitempty"`
	Bind string `json:"bind,omitempty"`
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

// Prompt a prompt
