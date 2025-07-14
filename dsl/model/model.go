package model

import (
	"context"
	"fmt"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/dsl/types"
)

// YaoModel is the MCP client DSL manager
type YaoModel struct {
	root string // The relative path of the MCP client DSL
}

// NewClient returns a new MCP client DSL manager
func NewClient(root string) types.Manager {
	return New(root)
}

// New returns a new connector DSL manager
func New(root string) types.Manager {
	return &YaoModel{root: root}
}

// Loaded return all loaded DSLs
func (m *YaoModel) Loaded(ctx context.Context) (map[string]*types.Info, error) {

	infos := map[string]*types.Info{}
	for id, mod := range model.Models {
		infos[id] = &types.Info{
			ID:          id,
			Type:        types.TypeModel,
			Label:       mod.MetaData.Name,
			Path:        mod.File,
			Sort:        999,
			Tags:        []string{},
			Description: "Description",
		}
	}

	return infos, nil
}

// Load will unload the DSL first, then load the DSL from DB or file system
func (m *YaoModel) Load(ctx context.Context, id string, options interface{}) error {
	var opts map[string]interface{}
	if v, ok := options.(map[string]interface{}); ok {
		opts = v
	}

	var migration bool = false
	if v, ok := opts["migration"]; ok {
		migration = v.(bool)
	}

	var reset bool = false
	if v, ok := opts["reset"]; ok {
		reset = v.(bool)
	}

	path := types.ToPath(types.TypeModel, id)
	mod, err := model.LoadSync(path, id)
	if err != nil {
		return err
	}

	if migration || reset {
		return mod.Migrate(reset, model.WithDonotInsertValues(true))
	}

	return nil
}

// Unload will unload the DSL from memory
func (m *YaoModel) Unload(ctx context.Context, id string, options interface{}) error {

	var opts map[string]interface{}
	if v, ok := options.(map[string]interface{}); ok {
		opts = v
	}

	var dropTable bool = false
	if v, ok := opts["dropTable"]; ok {
		dropTable = v.(bool)
	}

	mod := model.Select(id)
	if mod == nil {
		return fmt.Errorf("model %s not found", id)
	}

	if dropTable {
		return mod.DropTable()
	}

	return nil
}

// Reload will unload the DSL first, then reload the DSL from DB or file system
func (m *YaoModel) Reload(ctx context.Context, id string, options interface{}) error {

	var opts map[string]interface{}
	if v, ok := options.(map[string]interface{}); ok {
		opts = v
	}

	var migration bool = false
	if v, ok := opts["migration"]; ok {
		migration = v.(bool)
	}

	var reset bool = false
	if v, ok := opts["reset"]; ok {
		reset = v.(bool)
	}

	// Reload the model
	path := types.ToPath(types.TypeModel, id)
	mod, err := model.LoadSync(path, id)
	if err != nil {
		return err
	}

	if migration || reset {
		return mod.Migrate(reset, model.WithDonotInsertValues(true))
	}
	return nil
}

// Validate will validate the DSL from source
func (m *YaoModel) Validate(ctx context.Context, source string) (bool, []types.LintMessage) {
	return true, []types.LintMessage{}
}

// Execute will execute the DSL
func (m *YaoModel) Execute(ctx context.Context, method string, args ...any) (any, error) {
	return nil, fmt.Errorf("Not implemented")
}
