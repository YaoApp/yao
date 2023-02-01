package plugin

import (
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/plugin"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

// Load 加载业务插件
func Load(cfg config.Config) error {
	exts := []string{"*.so"}
	return application.App.Walk("apis", func(root, file string, isdir bool) error {
		_, err := plugin.Load(file, share.ID(root, file))
		return err
	}, exts...)
}
