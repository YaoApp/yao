package dsl

import (
	"context"
	"fmt"

	"github.com/yaoapp/yao/dsl/api"
	"github.com/yaoapp/yao/dsl/connector"
	"github.com/yaoapp/yao/dsl/io"
	"github.com/yaoapp/yao/dsl/mcp"
	"github.com/yaoapp/yao/dsl/model"
	"github.com/yaoapp/yao/dsl/types"
)

// DSL is the base DSL struct
type DSL struct {
	Type    types.Type
	exts    []string
	root    string
	manager types.Manager
	db      types.IO
	fs      types.IO
}

// New returns a new DSL manager
func New(typ types.Type) (types.DSL, error) {
	var manager types.Manager
	var db types.IO = io.NewDB(typ)
	var fs types.IO = io.NewFS(typ)

	// Get the root path and the extensions of the type
	root, exts := types.TypeRootAndExts(typ)

	// Create the manager
	switch typ {
	case types.TypeConnector:
		exts = []string{".conn.yao", ".conn.jsonc", ".conn.json"}
		manager = connector.New(root, fs, db)

	case types.TypeModel:
		exts = []string{".mod.yao", ".mod.jsonc", ".mod.json"}
		manager = model.New(root, fs, db)

	case types.TypeMCPClient:
		exts = []string{".mcp.yao", ".mcp.jsonc", ".mcp.json"}
		manager = mcp.NewClient(root, fs, db)

	// case types.TypeMCPServer:
	// 	exts = []string{".mcp.yao", ".mcp.jsonc", ".mcp.json"}
	// 	manager = mcp.NewServer(root)

	case types.TypeAPI:
		exts = []string{".http.yao", ".http.jsonc", ".http.json"}
		manager = api.New(root, fs, db)

	default:
		return nil, fmt.Errorf("dsl manager is not initialized, %s not supported", typ)
	}

	return &DSL{Type: typ, manager: manager, root: root, exts: exts, db: db, fs: fs}, nil
}

// Inspect DSL
func (dsl *DSL) Inspect(ctx context.Context, id string) (*types.Info, error) {

	// Get the info from the db
	info, exists, err := dsl.db.Inspect(id)
	if err != nil {
		return nil, err
	}

	if !exists {
		// Get the info from the file
		info, exists, err = dsl.fs.Inspect(id)
		if err != nil {
			return nil, err
		}

		if !exists {
			return nil, fmt.Errorf("%s not found, %s", dsl.Type, id)
		}
	}

	// Merge the status from the manager
	loaded, err := dsl.manager.Loaded(ctx)
	if err != nil {
		fmt.Printf("DEBUG: manager.Loaded failed: %v\n", err)
		return info, err
	}

	// Check if the DSL is loaded
	if _, ok := loaded[id]; ok {
		info.Status = types.StatusLoaded
	}

	return info, nil
}

// Path Get Path by id, ( If the DSL is saved as file, return the file path )
func (dsl *DSL) Path(ctx context.Context, id string) (string, error) {
	return types.ToPath(dsl.Type, id), nil
}

// Source Get Source by id
func (dsl *DSL) Source(ctx context.Context, id string) (string, error) {

	// Get the source from the db
	source, exists, err := dsl.db.Source(id)
	if err != nil {
		return "", err
	}

	if !exists {
		// Get the source from the file
		source, exists, err = dsl.fs.Source(id)
		if err != nil {
			return "", err
		}

		if !exists {
			return "", fmt.Errorf("%s DSL not found, %s", dsl.Type, id)
		}
	}

	return source, nil
}

// List DSLs
func (dsl *DSL) List(ctx context.Context, opts *types.ListOptions) ([]*types.Info, error) {
	// Get the list from the db
	var dbList []*types.Info
	var fileList []*types.Info
	var err error

	// If StoreType is not specified or is DB, get from db
	if opts.Store == "" || opts.Store == types.StoreTypeDB {
		dbList, err = dsl.db.List(opts)
		if err != nil {
			return nil, err
		}
	}

	// If StoreType is not specified or is File, get from file
	if opts.Store == "" || opts.Store == types.StoreTypeFile {
		fileList, err = dsl.fs.List(opts)
		if err != nil {
			return nil, err
		}
	}

	// Merge the list and unique
	list := []*types.Info{}
	unique := make(map[string]bool)
	for _, info := range dbList {
		if _, ok := unique[info.ID]; !ok {
			list = append(list, info)
			unique[info.ID] = true
		}
	}
	for _, info := range fileList {
		if _, ok := unique[info.ID]; !ok {
			list = append(list, info)
			unique[info.ID] = true
		}
	}

	// Merge the status from the manager
	loaded, err := dsl.manager.Loaded(ctx)
	if err != nil {
		return nil, err
	}

	// Merge the status from the manager
	for _, info := range list {
		if _, ok := loaded[info.ID]; ok {
			info.Status = types.StatusLoaded
		}
	}

	return list, nil
}

// Create DSL
func (dsl *DSL) Create(ctx context.Context, options *types.CreateOptions) error {

	if options == nil {
		return fmt.Errorf("create options is required")
	}

	// Set default store type if not specified
	if options.Store == "" {
		options.Store = types.StoreTypeFile
	}

	// Validate store type
	if options.Store != types.StoreTypeDB && options.Store != types.StoreTypeFile {
		return fmt.Errorf("invalid store type: %s", options.Store)
	}

	if options.Store == types.StoreTypeDB {
		err := dsl.db.Create(options)
		if err != nil {
			return err
		}
	} else if options.Store == types.StoreTypeFile {
		err := dsl.fs.Create(options)
		if err != nil {
			return err
		}
	}

	var loadOptions *types.LoadOptions = &types.LoadOptions{
		ID:      options.ID,
		Path:    types.ToPath(dsl.Type, options.ID),
		Source:  options.Source,
		Store:   options.Store,
		Options: options.Load,
	}

	// Load the DSL
	err := dsl.Load(ctx, loadOptions)
	if err != nil {
		return err
	}

	return nil
}

// Exists Check if the DSL exists
func (dsl *DSL) Exists(ctx context.Context, id string) (bool, error) {
	// Check if the DSL exists in the db
	exists, err := dsl.db.Exists(id)
	if err != nil {
		return false, err
	}

	if exists {
		return true, nil
	}

	// Check if the DSL exists in the file
	return dsl.fs.Exists(id)
}

// Update DSL
func (dsl *DSL) Update(ctx context.Context, options *types.UpdateOptions) error {

	if options == nil {
		return fmt.Errorf("update options is required")
	}

	// Exists
	info, exists, err := dsl.db.Inspect(options.ID)
	if err != nil {
		return err
	}

	if !exists {
		info, exists, err = dsl.fs.Inspect(options.ID)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("%s not found, %s", dsl.Type, options.ID)
		}
		// Fix: If store is empty but found in fs, it should be File store
		if info.Store == "" {
			info.Store = types.StoreTypeFile
		}
	} else {
		// Fix: If store is empty but found in db, it should be DB store
		if info.Store == "" {
			info.Store = types.StoreTypeDB
		}
	}

	// Create the reload options
	var reloadOptions *types.ReloadOptions = &types.ReloadOptions{
		ID:      options.ID,
		Path:    info.Path,
		Source:  options.Source,
		Store:   info.Store,
		Options: options.Reload,
	}

	// Update the DSL in the db
	if info.Store == types.StoreTypeDB {
		err := dsl.db.Update(options)
		if err != nil {
			return err
		}

		// Reload the DSL
		return dsl.manager.Reload(ctx, reloadOptions)
	}

	// Update the DSL in the file
	err = dsl.fs.Update(options)
	if err != nil {
		return err
	}

	// Reload the DSL
	return dsl.manager.Reload(ctx, reloadOptions)
}

// Delete DSL
func (dsl *DSL) Delete(ctx context.Context, options *types.DeleteOptions) error {

	if options == nil {
		return fmt.Errorf("delete options is required")
	}

	if options.ID == "" {
		return fmt.Errorf("delete options id is required")
	}

	// Exists
	info, exists, err := dsl.db.Inspect(options.ID)
	if err != nil {
		return err
	}

	if !exists {
		info, exists, err = dsl.fs.Inspect(options.ID)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("%s not found, %s", dsl.Type, options.ID)
		} else {
			// Fix: If store is empty but found in fs, it should be File store
			if info.Store == "" {
				info.Store = types.StoreTypeFile
			}
		}
	} else {
		// Fix: If store is empty but found in db, it should be DB store
		if info.Store == "" {
			info.Store = types.StoreTypeDB
		}
	}

	var opts map[string]interface{}
	if options.Options != nil {
		opts = options.Options
	}

	var unloadOptions *types.UnloadOptions = &types.UnloadOptions{
		ID:      options.ID,
		Path:    info.Path,
		Store:   info.Store,
		Options: opts,
	}

	if info.Store == types.StoreTypeDB {
		err = dsl.db.Delete(options.ID)
		if err != nil {
			return err
		}

		// Unload the DSL
		return dsl.manager.Unload(ctx, unloadOptions)
	}

	err = dsl.fs.Delete(options.ID)
	if err != nil {
		return err
	}

	// Unload the DSL
	return dsl.manager.Unload(ctx, unloadOptions)

}

// Load DSL
func (dsl *DSL) Load(ctx context.Context, options *types.LoadOptions) error {
	return dsl.manager.Load(ctx, options)
}

// Unload DSL
func (dsl *DSL) Unload(ctx context.Context, options *types.UnloadOptions) error {
	return dsl.manager.Unload(ctx, options)
}

// Reload DSL
func (dsl *DSL) Reload(ctx context.Context, options *types.ReloadOptions) error {
	return dsl.manager.Reload(ctx, options)
}

// Execute DSL (Some DSLs can be executed)
func (dsl *DSL) Execute(ctx context.Context, id string, method string, args ...any) (any, error) {
	return dsl.manager.Execute(ctx, id, method, args...)
}

// Validate DSL
func (dsl *DSL) Validate(ctx context.Context, source string) (bool, []types.LintMessage) {
	return dsl.manager.Validate(ctx, source)
}
