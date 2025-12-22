package store

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/data"
	"github.com/yaoapp/yao/share"
)

var systemStores = map[string]string{
	"__yao.store":                "yao/stores/store.xun.yao",                // for common data store
	"__yao.cache":                "yao/stores/cache.lru.yao",                // for common cache store
	"__yao.oauth.store":          "yao/stores/oauth/store.xun.yao",          // for OAuth data store
	"__yao.oauth.cache":          "yao/stores/oauth/cache.lru.yao",          // for OAuth cache store
	"__yao.oauth.client":         "yao/stores/oauth/client.xun.yao",         // for OAuth client store
	"__yao.agent.memory.user":    "yao/stores/agent/memory/user.xun.yao",    // for agent user-level memory
	"__yao.agent.memory.team":    "yao/stores/agent/memory/team.xun.yao",    // for agent team-level memory
	"__yao.agent.memory.chat":    "yao/stores/agent/memory/chat.xun.yao",    // for agent chat-level memory
	"__yao.agent.memory.context": "yao/stores/agent/memory/context.xun.yao", // for agent context-level memory
	"__yao.agent.cache":          "yao/stores/agent/cache.lru.yao",          // for agent cache store
	"__yao.kb.store":             "yao/stores/kb/store.xun.yao",             // for knowledge base store
	"__yao.kb.cache":             "yao/stores/kb/cache.lru.yao",             // for knowledge base cache store
}

// replaceVars replaces template variables in the JSON string
// Supports {{ VAR_NAME }} syntax
func replaceVars(jsonStr string, vars map[string]string) string {
	result := jsonStr
	for key, value := range vars {
		// Replace both {{ KEY }} and {{KEY}} patterns
		patterns := []string{
			"{{ " + key + " }}",
			"{{" + key + "}}",
		}
		for _, pattern := range patterns {
			result = strings.ReplaceAll(result, pattern, value)
		}
	}
	return result
}

// Load load store
func Load(cfg config.Config) error {

	// Load system stores
	err := loadSystemStores(cfg)
	if err != nil {
		return err
	}

	// Ignore if the stores directory does not exist
	exists, err := application.App.Exists("stores")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	messages := []string{}
	exts := []string{"*.yao", "*.json", "*.jsonc"}
	err = application.App.Walk("stores", func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}
		_, err := store.Load(file, share.ID(root, file))
		if err != nil {
			messages = append(messages, err.Error())
		}
		return err
	}, exts...)

	if len(messages) > 0 {
		return fmt.Errorf("%s", strings.Join(messages, ";\n"))
	}
	return err
}

// loadSystemStores load system stores
func loadSystemStores(cfg config.Config) error {
	for id, path := range systemStores {
		raw, err := data.Read(path)
		if err != nil {
			return err
		}

		// Replace template variables in the JSON string
		source := string(raw)
		if strings.Contains(source, "YAO_APP_ROOT") || strings.Contains(source, "YAO_DATA_ROOT") {
			vars := map[string]string{
				"YAO_APP_ROOT":  cfg.Root,
				"YAO_DATA_ROOT": cfg.DataRoot,
			}
			source = replaceVars(source, vars)
		}

		// Load store with the processed source
		_, err = store.LoadSource([]byte(source), id, filepath.Join("__system", path))
		if err != nil {
			log.Error("load system store %s error: %s", id, err.Error())
			return err
		}
	}
	return nil
}
