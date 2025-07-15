package connector

import (
	"context"

	"github.com/yaoapp/yao/dsl/types"
)

// YaoConnector is the connector DSL manager
type YaoConnector struct {
	root string   // The relative path of the connector DSL
	fs   types.IO // The file system IO interface
	db   types.IO // The database IO interface
}

// New returns a new connector DSL manager
func New(root string, fs types.IO, db types.IO) types.Manager {
	return &YaoConnector{root: root, fs: fs, db: db}
}

// Loaded return all loaded DSLs
func (c *YaoConnector) Loaded(ctx context.Context) (map[string]*types.Info, error) {
	return nil, nil
}

// Load will unload the DSL first, then load the DSL from DB or file system
func (c *YaoConnector) Load(ctx context.Context, options *types.LoadOptions) error {
	return nil
}

// Unload will unload the DSL from memory
func (c *YaoConnector) Unload(ctx context.Context, options *types.UnloadOptions) error {
	return nil
}

// Reload will unload the DSL first, then reload the DSL from DB or file system
func (c *YaoConnector) Reload(ctx context.Context, options *types.ReloadOptions) error {
	return nil
}

// Validate will validate the DSL from source
func (c *YaoConnector) Validate(ctx context.Context, source string) (bool, []types.LintMessage) {
	return false, nil
}

// Execute will execute the DSL
func (c *YaoConnector) Execute(ctx context.Context, id string, method string, args ...any) (any, error) {
	return nil, nil
}
