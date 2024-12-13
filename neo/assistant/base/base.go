package base

import (
	"context"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/yao/neo/assistant"
)

// Base the base assistant
type Base struct {
	ID        string              `json:"assistant_id"`
	Prompts   []assistant.Prompt  `json:"prompts,omitempty"`
	Connector connector.Connector `json:"-" yaml:"-"`
}

// New create a new base assistant
func New(connector connector.Connector, prompts []assistant.Prompt, id ...string) (*Base, error) {
	if len(id) > 0 {
		return &Base{Connector: connector, ID: id[0], Prompts: prompts}, nil
	}
	return &Base{Connector: connector, Prompts: prompts}, nil
}

// List list all assistants
func (ast *Base) List(ctx context.Context, param assistant.QueryParam) ([]assistant.Assistant, error) {
	return nil, nil
}
