package connector

import (
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

// Load load store
func Load(cfg config.Config) error {
	exts := []string{"*.yao", "*.json", "*.jsonc"}
	return application.App.Walk("connectors", func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}
		_, err := connector.Load(file, share.ID(root, file))
		return err
	}, exts...)
}
