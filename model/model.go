package model

import (
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

// Load 加载数据模型
func Load(cfg config.Config) error {
	exts := []string{"*.mod.yao", "*.mod.json", "*.mod.jsonc"}
	return application.App.Walk("apis", func(root, file string, isdir bool) error {
		_, err := model.Load(file, share.ID(root, file))
		return err
	}, exts...)
}
