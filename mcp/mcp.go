package mcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/mcp"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/dsl"
	"github.com/yaoapp/yao/dsl/types"
	"github.com/yaoapp/yao/share"
)

// Load load MCP clients
func Load(cfg config.Config) error {
	messages := []string{}

	// Check if mcps directory exists
	exists, err := application.App.Exists("mcps")

	// Load filesystem MCP clients if directory exists
	if err == nil && exists {
		exts := []string{"*.mcp.yao", "*.mcp.json", "*.mcp.jsonc"}
		err = application.App.Walk("mcps", func(root, file string, isdir bool) error {
			if isdir {
				return nil
			}
			_, err := mcp.LoadClient(file, share.ID(root, file))
			if err != nil {
				messages = append(messages, err.Error())
			}
			return err
		}, exts...)

		if len(messages) > 0 {
			for _, message := range messages {
				log.Error("Load filesystem MCP clients error: %s", message)
			}
			return fmt.Errorf("%s", strings.Join(messages, ";\n"))
		}
	}

	// Load database MCP clients (ignore error)
	errs := loadDatabaseMCPs()
	if len(errs) > 0 {
		for _, err := range errs {
			log.Error("Load database MCP clients error: %s", err.Error())
		}
	}
	return err
}

// loadDatabaseMCPs load database MCP clients
func loadDatabaseMCPs() []error {
	var errs []error = []error{}
	manager, err := dsl.New(types.TypeMCPClient)
	if err != nil {
		errs = append(errs, err)
		return errs
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mcps, err := manager.List(ctx, &types.ListOptions{Store: types.StoreTypeDB, Source: true})
	if err != nil {
		errs = append(errs, err)
		return errs
	}

	// Load MCP clients
	for _, info := range mcps {
		_, err := mcp.LoadClientSource(info.Source, info.ID)
		if err != nil {
			errs = append(errs, err)
			continue
		}
	}

	return errs
}
