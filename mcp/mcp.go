package mcp

import (
	"context"
	"fmt"
	"path/filepath"
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

	// Load MCP clients from assistants
	errsAssistants := loadAssistantMCPs()
	if len(errsAssistants) > 0 {
		for _, err := range errsAssistants {
			log.Error("Load assistant MCP clients error: %s", err.Error())
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

// loadAssistantMCPs load MCP clients from assistants directory
func loadAssistantMCPs() []error {
	var errs []error = []error{}

	// Check if assistants directory exists
	exists, err := application.App.Exists("assistants")
	if err != nil || !exists {
		log.Trace("Assistants directory not found or not accessible")
		return errs
	}

	log.Trace("Loading MCP clients from assistants directory...")

	// Track processed assistants to avoid duplicates
	processedAssistants := make(map[string]bool)

	// Walk through assistants directory to find all valid assistants with mcps
	err = application.App.Walk("assistants", func(root, file string, isdir bool) error {
		if !isdir {
			return nil
		}

		// Check if this is a valid assistant directory (has package.yao)
		// file is relative path from root, so we need to join root + file
		pkgFile := filepath.Join(root, file, "package.yao")
		pkgExists, _ := application.App.Exists(pkgFile)
		if !pkgExists {
			return nil
		}

		// Extract assistant ID from path (e.g., "/assistants/expense" -> "expense")
		// file is like "/tests/mcpload", trim leading "/" and replace "/" with "."
		assistantID := strings.TrimPrefix(file, "/")
		assistantID = strings.ReplaceAll(assistantID, "/", ".")

		// Skip if already processed
		if processedAssistants[assistantID] {
			return nil
		}
		processedAssistants[assistantID] = true

		log.Trace("Found assistant: %s", assistantID)

		// Check if the assistant has an mcps directory
		mcpsDir := filepath.Join(root, file, "mcps")
		mcpsDirExists, _ := application.App.Exists(mcpsDir)
		if !mcpsDirExists {
			log.Trace("Assistant %s has no mcps directory", assistantID)
			return nil
		}

		log.Trace("Loading MCPs from assistant %s", assistantID)

		// Load MCP clients from the assistant's mcps directory
		exts := []string{"*.mcp.yao", "*.mcp.json", "*.mcp.jsonc"}
		err := application.App.Walk(mcpsDir, func(mcpRoot, mcpFile string, mcpIsDir bool) error {
			if mcpIsDir {
				return nil
			}

			// Generate MCP client ID with agents.<assistantID>./ prefix
			// Support nested paths: "mcps/nested/tool.mcp.yao" -> "nested.tool"
			relPath := strings.TrimPrefix(mcpFile, mcpsDir+"/")
			relPath = strings.TrimPrefix(relPath, "/")
			relPath = strings.TrimSuffix(relPath, ".mcp.yao")
			relPath = strings.TrimSuffix(relPath, ".mcp.json")
			relPath = strings.TrimSuffix(relPath, ".mcp.jsonc")
			mcpName := strings.ReplaceAll(relPath, "/", ".")
			clientID := fmt.Sprintf("agents.%s.%s", assistantID, mcpName)

			log.Trace("Loading MCP client %s from file %s", clientID, mcpFile)

			_, err := mcp.LoadClientWithType(mcpFile, clientID, "agent")
			if err != nil {
				log.Error("Failed to load MCP client %s from assistant %s: %s", clientID, assistantID, err.Error())
				errs = append(errs, fmt.Errorf("failed to load MCP client %s: %w", clientID, err))
				return nil // Continue loading other MCPs
			}

			log.Info("Loaded MCP client: %s", clientID)
			return nil
		}, exts...)

		if err != nil {
			errs = append(errs, fmt.Errorf("failed to walk MCPs in assistant %s: %w", assistantID, err))
		}

		return nil
	}, "")

	if err != nil {
		errs = append(errs, fmt.Errorf("failed to walk assistants directory: %w", err))
	}

	return errs
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
