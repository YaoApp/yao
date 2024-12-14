package base

import (
	"context"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/yao/neo/assistant"
	"github.com/yaoapp/yao/openai"
)

// Base the base assistant
type Base struct {
	ID        string              `json:"assistant_id"`
	Prompts   []assistant.Prompt  `json:"prompts,omitempty"`
	Connector connector.Connector `json:"-" yaml:"-"`
	openai    *openai.OpenAI
}

// New create a new base assistant
func New(connector connector.Connector, prompts []assistant.Prompt, id string) (*Base, error) {

	setting := connector.Setting()
	api, err := openai.NewOpenAI(setting)
	if err != nil {
		return nil, err
	}

	return &Base{Connector: connector, ID: id, Prompts: prompts, openai: api}, nil
}

// List list all assistants
func (ast *Base) List(ctx context.Context, param assistant.QueryParam) ([]assistant.Assistant, error) {
	return nil, nil
}
