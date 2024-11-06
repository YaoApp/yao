package api

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou/api"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

// Load apis
func Load(cfg config.Config) error {
	messages := []string{}

	exts := []string{"*.http.yao", "*.http.json", "*.http.jsonc"}
	err := application.App.Walk("apis", func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}
		_, err := api.Load(file, share.ID(root, file))
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
