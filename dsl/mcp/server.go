package mcp

import (
	"context"

	"github.com/yaoapp/yao/dsl/types"
)

// YaoMCPServer is the MCP client DSL manager
type YaoMCPServer struct {
	root string // The relative path of the MCP client DSL
}

// NewServer returns a new MCP server DSL manager
func NewServer(root string) types.Manager {
	return &YaoMCPServer{root: root}
}

// Loaded return all loaded DSLs
func (server *YaoMCPServer) Loaded(ctx context.Context) (map[string]*types.Info, error) {
	return nil, nil
}

// Load will unload the DSL first, then load the DSL from DB or file system
func (server *YaoMCPServer) Load(ctx context.Context, options *types.LoadOptions) error {
	return nil
}

// Unload will unload the DSL from memory
func (server *YaoMCPServer) Unload(ctx context.Context, options *types.UnloadOptions) error {
	return nil
}

// Reload will unload the DSL first, then reload the DSL from DB or file system
func (server *YaoMCPServer) Reload(ctx context.Context, options *types.ReloadOptions) error {
	return nil
}

// Validate will validate the DSL from source
func (server *YaoMCPServer) Validate(ctx context.Context, source string) (bool, []types.LintMessage) {
	return false, nil
}

// Execute will execute the DSL
func (server *YaoMCPServer) Execute(ctx context.Context, id string, method string, args ...any) (any, error) {
	return nil, nil
}
