package api

import (
	"github.com/yaoapp/gou/api"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

// Load apis
func Load(cfg config.Config) error {
	exts := []string{"*.http.yao", "*.http.json", "*.http.jsonc"}
	return application.App.Walk("apis", func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}
		_, err := api.Load(file, share.ID(root, file))
		return err
	}, exts...)
}
