package model

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

// Load 加载数据模型
func Load(cfg config.Config) error {

	messages := []string{}

	model.WithCrypt([]byte(fmt.Sprintf(`{"key":"%s"}`, cfg.DB.AESKey)), "AES")
	model.WithCrypt([]byte(`{}`), "PASSWORD")

	exts := []string{"*.mod.yao", "*.mod.json", "*.mod.jsonc"}
	err := application.App.Walk("models", func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}
		_, err := model.Load(file, share.ID(root, file))
		if err != nil {
			messages = append(messages, err.Error())
		}
		return err
	}, exts...)

	if len(messages) > 0 {
		return fmt.Errorf(strings.Join(messages, ";\n"))
	}

	return err
}
