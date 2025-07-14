package dsl

import (
	"context"
	"fmt"

	"github.com/yaoapp/yao/dsl/api"
	"github.com/yaoapp/yao/dsl/connector"
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
}

// New returns a new DSL manager
func New(typ types.Type) (types.DSL, error) {
	var manager types.Manager

	// Get the root path and the extensions of the type
	root, exts := types.TypeRootAndExts(typ)

	// Create the manager
	switch typ {
	case types.TypeConnector:
		exts = []string{".conn.yao", ".conn.jsonc", ".conn.json"}
		manager = connector.New(root)

	case types.TypeModel:
		exts = []string{".mod.yao", ".mod.jsonc", ".mod.json"}
		manager = model.New(root)

	case types.TypeMCPClient:
		exts = []string{".mcp.yao", ".mcp.jsonc", ".mcp.json"}
		manager = mcp.NewClient(root)

	// case types.TypeMCPServer:
	// 	exts = []string{".mcp.yao", ".mcp.jsonc", ".mcp.json"}
	// 	manager = mcp.NewServer(root)

	case types.TypeAPI:
		exts = []string{".http.yao", ".http.jsonc", ".http.json"}
		manager = api.New(root)

	default:
		return nil, fmt.Errorf("dsl manager is not initialized, %s not supported", typ)
	}

	return &DSL{Type: typ, manager: manager, root: root, exts: exts}, nil
}

// Inspect DSL
func (dsl *DSL) Inspect(ctx context.Context, id string) (*types.Info, error) {

	// Get the info from the db
	info, exists, err := dsl.dbInspect(id)
	if err != nil {
		return nil, err
	}

	if !exists {
		// Get the info from the file
		info, exists, err = dsl.fsInspect(types.ToPath(dsl.Type, id))
		if err != nil {
			return nil, err
		}

		if !exists {
			return nil, fmt.Errorf("dsl not found, %s", id)
		}
	}

	// Merge the status from the manager
	loaded, err := dsl.manager.Loaded(ctx)
	if err != nil {
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
	source, exists, err := dsl.dbSource(id)
	if err != nil {
		return "", err
	}

	if !exists {
		// Get the source from the file
		source, exists, err = dsl.fsSource(types.ToPath(dsl.Type, id))
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
	dbList, err := dsl.dbList(opts)
	if err != nil {
		return nil, err
	}

	// Get the list from the file
	fileList, err := dsl.fsList(opts)
	if err != nil {
		return nil, err
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
	if options.Store == types.StoreTypeDB {
		err := dsl.dbCreate(options)
		if err != nil {
			return err
		}
	}

	if options.Store == types.StoreTypeFile {
		err := dsl.fsCreate(options)
		if err != nil {
			return err
		}
	}

	// Load the DSL
	err := dsl.Load(ctx, options.ID, options.LoadOptions)
	if err != nil {
		return err
	}

	return nil
}

// Exists Check if the DSL exists
func (dsl *DSL) Exists(ctx context.Context, id string) (bool, error) {
	// Check if the DSL exists in the db
	exists, err := dsl.dbExists(id)
	if err != nil {
		return false, err
	}

	if exists {
		return true, nil
	}

	// Check if the DSL exists in the file
	return dsl.fsExists(id)
}

// Update DSL
func (dsl *DSL) Update(ctx context.Context, options *types.UpdateOptions) error {

	// Exists
	info, exists, err := dsl.dbInspect(options.ID)
	if err != nil {
		return err
	}

	if !exists {
		info, exists, err = dsl.fsInspect(types.ToPath(dsl.Type, options.ID))
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("%s DSL not found, %s", dsl.Type, options.ID)
		}
	}

	// Update the DSL in the db
	if info.Store == types.StoreTypeDB {
		err := dsl.dbUpdate(options)
		if err != nil {
			return err
		}

		// Reload the DSL
		return dsl.manager.Reload(ctx, options.ID, options.ReloadOptions)
	}

	// Update the DSL in the file
	err = dsl.fsUpdate(options)
	if err != nil {
		return err
	}

	// Reload the DSL
	return dsl.manager.Reload(ctx, options.ID, options.ReloadOptions)
}

// Delete DSL
func (dsl *DSL) Delete(ctx context.Context, id string, options ...interface{}) error {
	// Exists
	info, exists, err := dsl.dbInspect(id)
	if err != nil {
		return err
	}

	if !exists {
		info, exists, err = dsl.fsInspect(types.ToPath(dsl.Type, id))
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("%s DSL not found, %s", dsl.Type, id)
		}
	}

	var unloadOptions interface{}
	if len(options) > 0 {
		unloadOptions = options[0]
	}

	if info.Store == types.StoreTypeDB {
		err = dsl.dbDelete(id)
		if err != nil {
			return err
		}

		// Unload the DSL
		return dsl.manager.Unload(ctx, id, unloadOptions)
	}

	err = dsl.fsDelete(id)
	if err != nil {
		return err
	}

	// Unload the DSL
	return dsl.manager.Unload(ctx, id, unloadOptions)

}

// Load DSL
func (dsl *DSL) Load(ctx context.Context, id string, options interface{}) error {
	return dsl.manager.Load(ctx, id, options)
}

// Unload DSL
func (dsl *DSL) Unload(ctx context.Context, id string, options ...interface{}) error {
	var unloadOptions interface{}
	if len(options) > 0 {
		unloadOptions = options[0]
	}
	return dsl.manager.Unload(ctx, id, unloadOptions)
}

// Reload DSL
func (dsl *DSL) Reload(ctx context.Context, id string, options interface{}) error {
	return dsl.manager.Reload(ctx, id, options)
}

// Execute DSL (Some DSLs can be executed)
func (dsl *DSL) Execute(ctx context.Context, method string, args ...any) (any, error) {
	return dsl.manager.Execute(ctx, method, args...)
}

// Validate DSL
func (dsl *DSL) Validate(ctx context.Context, source string) (bool, []types.LintMessage) {
	return dsl.manager.Validate(ctx, source)
}
