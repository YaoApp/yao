package cert

import (
	"path/filepath"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/ssl"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

// Load 加载API
func Load(cfg config.Config) error {
	exts := []string{"*.pem", "*.key", "*.pub"}
	return application.App.Walk("certs", func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}
		_, err := ssl.Load(file, share.ID(root, file)+filepath.Ext(file))
		return err
	}, exts...)
}
