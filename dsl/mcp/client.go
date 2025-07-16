package mcp

import (
	"context"
	"fmt"

	goumcp "github.com/yaoapp/gou/mcp"
	"github.com/yaoapp/yao/dsl/types"
)

// YaoMCPClient is the MCP client DSL manager
type YaoMCPClient struct {
	root string   // The relative path of the MCP client DSL
	fs   types.IO // The file system IO interface
	db   types.IO // The database IO interface
}

// NewClient returns a new MCP client DSL manager
func NewClient(root string, fs types.IO, db types.IO) types.Manager {
	return &YaoMCPClient{root: root, fs: fs, db: db}
}

// Loaded return all loaded DSLs
func (client *YaoMCPClient) Loaded(ctx context.Context) (map[string]*types.Info, error) {
	infos := map[string]*types.Info{}

	// Get all loaded MCP clients
	clientIDs := goumcp.ListClients()

	for _, id := range clientIDs {
		// Get the client
		mcpClient, err := goumcp.Select(id)
		if err != nil {
			continue // Skip if client not found
		}

		// Get meta info from the client
		meta := mcpClient.GetMetaInfo()

		infos[id] = &types.Info{
			ID:          id,
			Path:        types.ToPath(types.TypeMCPClient, id),
			Type:        types.TypeMCPClient,
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
func (client *YaoMCPClient) Load(ctx context.Context, options *types.LoadOptions) error {
	if options == nil {
		return fmt.Errorf("load options is required")
	}

	if options.ID == "" {
		return fmt.Errorf("load options id is required")
	}

	var err error

	// Case 1: If Source is provided, use LoadClientSource
	if options.Source != "" {
		_, err = goumcp.LoadClientSource(options.Source, options.ID)
		if err != nil {
			return err
		}
	} else if options.Path != "" && options.Store == types.StoreTypeFile {
		// Case 2: If Path is provided and Store is file, use LoadClient with Path
		_, err = goumcp.LoadClient(options.Path, options.ID)
		if err != nil {
			return err
		}
	} else if options.Store == types.StoreTypeDB {
		// Case 3: If Store is db, get Source from DB first
		if client.db == nil {
			return fmt.Errorf("db io is required for store type db")
		}
		source, exists, err := client.db.Source(options.ID)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("mcp client %s not found in database", options.ID)
		}
		_, err = goumcp.LoadClientSource(source, options.ID)
		if err != nil {
			return err
		}
	} else {
		// Case 4: Default case, use LoadClient with ID
		path := types.ToPath(types.TypeMCPClient, options.ID)
		_, err = goumcp.LoadClient(path, options.ID)
		if err != nil {
			return err
		}
	}

	return nil
}

// Unload will unload the DSL from memory
func (client *YaoMCPClient) Unload(ctx context.Context, options *types.UnloadOptions) error {
	if options == nil {
		return fmt.Errorf("unload options is required")
	}

	if options.ID == "" {
		return fmt.Errorf("unload options id is required")
	}

	// Use the UnloadClient function from gou/mcp package
	goumcp.UnloadClient(options.ID)
	return nil
}

// Reload will unload the DSL first, then reload the DSL from DB or file system
func (client *YaoMCPClient) Reload(ctx context.Context, options *types.ReloadOptions) error {
	if options == nil {
		return fmt.Errorf("reload options is required")
	}

	if options.ID == "" {
		return fmt.Errorf("reload options id is required")
	}

	// First unload
	goumcp.UnloadClient(options.ID)

	// Then load
	var err error
	if options.Source != "" {
		_, err = goumcp.LoadClientSource(options.Source, options.ID)
		if err != nil {
			return err
		}
	} else if options.Path != "" && options.Store == types.StoreTypeFile {
		_, err = goumcp.LoadClient(options.Path, options.ID)
		if err != nil {
			return err
		}
	} else if options.Store == types.StoreTypeDB {
		if client.db == nil {
			return fmt.Errorf("db io is required for store type db")
		}
		source, exists, err := client.db.Source(options.ID)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("mcp client %s not found in database", options.ID)
		}
		_, err = goumcp.LoadClientSource(source, options.ID)
		if err != nil {
			return err
		}
	} else {
		path := types.ToPath(types.TypeMCPClient, options.ID)
		_, err = goumcp.LoadClient(path, options.ID)
		if err != nil {
			return err
		}
	}

	return nil
}

// Validate will validate the DSL from source
func (client *YaoMCPClient) Validate(ctx context.Context, source string) (bool, []types.LintMessage) {
	return true, []types.LintMessage{}
}

// Execute will execute the DSL
func (client *YaoMCPClient) Execute(ctx context.Context, id string, method string, args ...any) (any, error) {
	return nil, fmt.Errorf("Not implemented")
}
