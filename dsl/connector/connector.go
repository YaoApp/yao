package connector

import (
	"context"
	"fmt"

	"github.com/yaoapp/gou/connector"
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
	infos := map[string]*types.Info{}
	for id, conn := range connector.Connectors {
		meta := conn.GetMetaInfo()
		infos[id] = &types.Info{
			ID:          id,
			Path:        conn.ID(),
			Type:        types.TypeConnector,
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
func (c *YaoConnector) Load(ctx context.Context, options *types.LoadOptions) error {
	if options == nil {
		return fmt.Errorf("load options is required")
	}

	if options.ID == "" {
		return fmt.Errorf("load options id is required")
	}

	var err error

	// Case 1: If Source is provided, use LoadSourceSync
	if options.Source != "" {
		connectorPath := types.ToPath(types.TypeConnector, options.ID)
		_, err = connector.LoadSourceSync([]byte(options.Source), options.ID, connectorPath)
		if err != nil {
			return err
		}
	} else if options.Path != "" && options.Store == "fs" {
		// Case 2: If Path is provided and Store is fs, use LoadSync with Path
		_, err = connector.LoadSync(options.Path, options.ID)
		if err != nil {
			return err
		}
	} else if options.Store == "db" {
		// Case 3: If Store is db, get Source from DB first
		if c.db == nil {
			return fmt.Errorf("db io is required for store type db")
		}
		source, exists, err := c.db.Source(options.ID)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("connector %s not found in database", options.ID)
		}
		connectorPath := types.ToPath(types.TypeConnector, options.ID)
		_, err = connector.LoadSourceSync([]byte(source), options.ID, connectorPath)
		if err != nil {
			return err
		}
	} else {
		// Case 4: Default case, use LoadSync with ID
		path := types.ToPath(types.TypeConnector, options.ID)
		_, err = connector.LoadSync(path, options.ID)
		if err != nil {
			return err
		}
	}

	return nil
}

// Unload will unload the DSL from memory
func (c *YaoConnector) Unload(ctx context.Context, options *types.UnloadOptions) error {
	if options == nil {
		return fmt.Errorf("unload options is required")
	}

	if options.ID == "" {
		return fmt.Errorf("unload options id is required")
	}

	return connector.Remove(options.ID)
}

// Reload will unload the DSL first, then reload the DSL from DB or file system
func (c *YaoConnector) Reload(ctx context.Context, options *types.ReloadOptions) error {
	if options == nil {
		return fmt.Errorf("reload options is required")
	}

	if options.ID == "" {
		return fmt.Errorf("reload options id is required")
	}

	// First unload
	err := connector.Remove(options.ID)
	if err != nil {
		return err
	}

	// Then load
	if options.Source != "" {
		connectorPath := types.ToPath(types.TypeConnector, options.ID)
		_, err = connector.LoadSourceSync([]byte(options.Source), options.ID, connectorPath)
		if err != nil {
			return err
		}
	} else if options.Path != "" && options.Store == "fs" {
		_, err = connector.LoadSync(options.Path, options.ID)
		if err != nil {
			return err
		}
	} else if options.Store == "db" {
		if c.db == nil {
			return fmt.Errorf("db io is required for store type db")
		}
		source, exists, err := c.db.Source(options.ID)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("connector %s not found in database", options.ID)
		}
		connectorPath := types.ToPath(types.TypeConnector, options.ID)
		_, err = connector.LoadSourceSync([]byte(source), options.ID, connectorPath)
		if err != nil {
			return err
		}
	} else {
		path := types.ToPath(types.TypeConnector, options.ID)
		_, err = connector.LoadSync(path, options.ID)
		if err != nil {
			return err
		}
	}

	return nil
}

// Validate will validate the DSL from source
func (c *YaoConnector) Validate(ctx context.Context, source string) (bool, []types.LintMessage) {
	return true, []types.LintMessage{}
}

// Execute will execute the DSL
func (c *YaoConnector) Execute(ctx context.Context, id string, method string, args ...any) (any, error) {
	return nil, fmt.Errorf("Not implemented")
}
