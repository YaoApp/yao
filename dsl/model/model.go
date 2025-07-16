package model

import (
	"context"
	"fmt"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/dsl/types"
)

// YaoModel is the MCP client DSL manager
type YaoModel struct {
	root string   // The relative path of the model DSL
	fs   types.IO // The file system IO interface
	db   types.IO // The database IO interface
}

// New returns a new connector DSL manager
func New(root string, fs types.IO, db types.IO) types.Manager {
	return &YaoModel{root: root, fs: fs, db: db}
}

// Loaded return all loaded DSLs
func (m *YaoModel) Loaded(ctx context.Context) (map[string]*types.Info, error) {

	infos := map[string]*types.Info{}
	for id, mod := range model.Models {
		meta := mod.GetMetaInfo()
		infos[id] = &types.Info{
			ID:          id,
			Path:        mod.File,
			Type:        types.TypeModel,
			Label:       meta.Label,
			Sort:        meta.Sort,
			Description: meta.Description,
			Tags:        meta.Tags,
			Readonly:    meta.Readonly,
			Builtin:     meta.Builtin,
			Mtime:       meta.Mtime,
			Ctime:       meta.Ctime,
		}
	}

	return infos, nil
}

// Load will unload the DSL first, then load the DSL from DB or file system
func (m *YaoModel) Load(ctx context.Context, options *types.LoadOptions) error {

	if options == nil {
		return fmt.Errorf("load options is required")
	}

	if options.ID == "" {
		return fmt.Errorf("load options id is required")
	}

	var opts map[string]interface{}
	if options.Options != nil {
		opts = options.Options
	}

	var migration bool = false
	if v, ok := opts["migration"]; ok {
		migration = v.(bool)
	}

	var reset bool = false
	if v, ok := opts["reset"]; ok {
		reset = v.(bool)
	}

	var mod *model.Model
	var err error

	// Case 1: If Source is provided, use LoadSource
	if options.Source != "" {
		mod, err = model.LoadSourceSync([]byte(options.Source), options.ID, "")
		if err != nil {
			return err
		}
	} else if options.Path != "" && options.Store == "fs" {
		// Case 2: If Path is provided and Store is fs, use LoadSync with Path
		mod, err = model.LoadSync(options.Path, options.ID)
		if err != nil {
			return err
		}
	} else if options.Store == "db" {
		// Case 3: If Store is db, get Source from DB first
		if m.db == nil {
			return fmt.Errorf("db io is required for store type db")
		}
		source, exists, err := m.db.Source(options.ID)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("model %s not found in database", options.ID)
		}
		mod, err = model.LoadSourceSync([]byte(source), options.ID, "")
		if err != nil {
			return err
		}
	} else {
		// Case 4: Default case, use LoadSync with ID
		path := types.ToPath(types.TypeModel, options.ID)
		mod, err = model.LoadSync(path, options.ID)
		if err != nil {
			return err
		}
	}

	if migration || reset {
		return mod.Migrate(reset, model.WithDonotInsertValues(true))
	}

	return nil
}

// Unload will unload the DSL from memory
func (m *YaoModel) Unload(ctx context.Context, options *types.UnloadOptions) error {

	if options == nil {
		return fmt.Errorf("unload options is required")
	}

	if options.ID == "" {
		return fmt.Errorf("unload options id is required")
	}

	var opts map[string]interface{}
	if options.Options != nil {
		opts = options.Options
	}

	var dropTable bool = false
	if v, ok := opts["dropTable"]; ok {
		dropTable = v.(bool)
	}

	// Try to get model, handle panic
	var mod *model.Model
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				if ex, ok := r.(exception.Exception); ok {
					if ex.Message == fmt.Sprintf("Model:%s; not found", options.ID) {
						err = fmt.Errorf("model %s not found", options.ID)
						return
					}
				}
				panic(r)
			}
		}()
		mod = model.Select(options.ID)
	}()

	if err != nil {
		return err
	}

	if mod == nil {
		return fmt.Errorf("model %s not found", options.ID)
	}

	if dropTable {
		return mod.DropTable()
	}

	return nil
}

// Reload will unload the DSL first, then reload the DSL from DB or file system
func (m *YaoModel) Reload(ctx context.Context, options *types.ReloadOptions) error {

	if options == nil {
		return fmt.Errorf("reload options is required")
	}

	if options.ID == "" {
		return fmt.Errorf("reload options id is required")
	}

	var opts map[string]interface{}
	if options.Options != nil {
		opts = options.Options
	}

	var migrate bool = false
	if v, ok := opts["migrate"]; ok {
		migrate = v.(bool)
	}

	var reset bool = false
	if v, ok := opts["reset"]; ok {
		reset = v.(bool)
	}

	var mod *model.Model
	var err error

	// Case 1: If Source is provided, use LoadSource
	if options.Source != "" {
		mod, err = model.LoadSourceSync([]byte(options.Source), options.ID, "")
		if err != nil {
			return err
		}
	} else if options.Path != "" && options.Store == "fs" {
		// Case 2: If Path is provided and Store is fs, use LoadSync with Path
		mod, err = model.LoadSync(options.Path, options.ID)
		if err != nil {
			return err
		}
	} else if options.Store == "db" {
		// Case 3: If Store is db, get Source from DB first
		if m.db == nil {
			return fmt.Errorf("db io is required for store type db")
		}
		source, exists, err := m.db.Source(options.ID)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("model %s not found in database", options.ID)
		}
		mod, err = model.LoadSourceSync([]byte(source), options.ID, "")
		if err != nil {
			return err
		}
	} else {
		// Case 4: Default case, use LoadSync with ID
		path := types.ToPath(types.TypeModel, options.ID)
		mod, err = model.LoadSync(path, options.ID)
		if err != nil {
			return err
		}
	}

	if migrate || reset {
		return mod.Migrate(reset, model.WithDonotInsertValues(true))
	}

	return nil
}

// Validate will validate the DSL from source
func (m *YaoModel) Validate(ctx context.Context, source string) (bool, []types.LintMessage) {
	return true, []types.LintMessage{}
}

// Execute will execute the DSL
func (m *YaoModel) Execute(ctx context.Context, id string, method string, args ...any) (any, error) {
	return nil, fmt.Errorf("Not implemented")
}
