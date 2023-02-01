package cert

import (
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/ssl"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

// Load 加载API
func Load(cfg config.Config) error {
	exts := []string{"*.pem", "*.key", "*.pub"}
	return application.App.Walk("certs", func(root, file string, isdir bool) error {
		_, err := ssl.Load(file, share.ID(root, file))
		return err
	}, exts...)
}
