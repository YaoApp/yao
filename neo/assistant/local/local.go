package local

import (
	"context"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/yao/neo/assistant"
	"github.com/yaoapp/yao/openai"
)

// Local the local assistant
type Local struct {
	ID        string              `json:"assistant_id"`
	Prompts   []assistant.Prompt  `json:"prompts,omitempty"`
	Connector connector.Connector `json:"-" yaml:"-"`
	openai    *openai.OpenAI
}

// New create a new local assistant
func New(connector connector.Connector, prompts []assistant.Prompt, id string) (*Local, error) {

	setting := connector.Setting()
	api, err := openai.NewOpenAI(setting)
	if err != nil {
		return nil, err
	}

	return &Local{Connector: connector, ID: id, Prompts: prompts, openai: api}, nil
}

// List list all assistants
func (ast *Local) List(ctx context.Context, param assistant.QueryParam) ([]assistant.Assistant, error) {
	return nil, nil
}
