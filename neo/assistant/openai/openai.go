package openai

import (
	"context"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/yao/neo/assistant"
	api "github.com/yaoapp/yao/openai"
)

// OpenAI the openai assistant
type OpenAI struct {
	ID        string              `json:"assistant_id"` // the assistant id
	Connector connector.Connector `json:"-" yaml:"-"`
	openai    *api.OpenAI
}

// New create a new openai assistant
func New(connector connector.Connector, id string) (*OpenAI, error) {

	setting := connector.Setting()
	openai, err := api.NewOpenAI(setting)
	if err != nil {
		return nil, err
	}

	return &OpenAI{ID: id, Connector: connector, openai: openai}, nil
}

// Current set the current assistant
func (ast *OpenAI) Current(id string) *OpenAI {
	ast.ID = id
	return ast
}

// List list all assistants
func (ast *OpenAI) List(ctx context.Context, param assistant.QueryParam) ([]assistant.Assistant, error) {
	return nil, nil
}

// Create create a new assistant
func (ast *OpenAI) Create() {}

// Delete delete an assistant
func (ast *OpenAI) Delete() {}

// Update update an assistant
func (ast *OpenAI) Update() {}

// Get get an assistant
func (ast *OpenAI) Get() {}
