package store

import (
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

// Load load store
func Load(cfg config.Config) error {
	exts := []string{"*.yao", "*.json", "*.jsonc"}
	return application.App.Walk("stores", func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}

		_, err := store.Load(file, share.ID(root, file))
		return err
	}, exts...)
}
