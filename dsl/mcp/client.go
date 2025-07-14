package mcp

import (
	"context"

	"github.com/yaoapp/yao/dsl/types"
)

// YaoMCPClient is the MCP client DSL manager
type YaoMCPClient struct {
	root string // The relative path of the MCP client DSL
}

// NewClient returns a new MCP client DSL manager
func NewClient(root string) types.Manager {
	return New(root)
}

// New returns a new connector DSL manager
func New(root string) types.Manager {
	return &YaoMCPClient{root: root}
}

// Loaded return all loaded DSLs
func (client *YaoMCPClient) Loaded(ctx context.Context) (map[string]*types.Info, error) {
	return nil, nil
}

// Load will unload the DSL first, then load the DSL from DB or file system
func (client *YaoMCPClient) Load(ctx context.Context, id string, options interface{}) error {
	return nil
}

// Unload will unload the DSL from memory
func (client *YaoMCPClient) Unload(ctx context.Context, id string, options interface{}) error {
	return nil
}

// Reload will unload the DSL first, then reload the DSL from DB or file system
func (client *YaoMCPClient) Reload(ctx context.Context, id string, options interface{}) error {
	return nil
}

// Validate will validate the DSL from source
func (client *YaoMCPClient) Validate(ctx context.Context, source string) (bool, []types.LintMessage) {
	return false, nil
}

// Execute will execute the DSL
func (client *YaoMCPClient) Execute(ctx context.Context, method string, args ...any) (any, error) {
	return nil, nil
}
