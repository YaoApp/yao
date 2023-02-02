package script

import (
	"github.com/yaoapp/gou/application"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

// Load 加载共享库
func Load(cfg config.Config) error {
	exts := []string{"*.js"}
	return application.App.Walk("scripts", func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}
		_, err := v8.Load(file, share.ID(root, file))
		return err
	}, exts...)
}
