package model

import (
	"fmt"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

// Load 加载数据模型
func Load(cfg config.Config) error {

	model.WithCrypt([]byte(fmt.Sprintf(`{"key":"%s"}`, cfg.DB.AESKey)), "AES")
	model.WithCrypt([]byte(`{}`), "PASSWORD")

	exts := []string{"*.mod.yao", "*.mod.json", "*.mod.jsonc"}
	return application.App.Walk("models", func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}
		_, err := model.Load(file, share.ID(root, file))
		return err
	}, exts...)
}
