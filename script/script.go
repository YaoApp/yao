package script

import (
	"fmt"

	"github.com/yaoapp/gou/application"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

// Load load all scripts and services
func Load(cfg config.Config) error {
	v8.CLearModules()
	exts := []string{"*.js", "*.ts"}
	err := application.App.Walk("scripts", func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}
		_, err := v8.Load(file, share.ID(root, file))
		return err
	}, exts...)

	if err != nil {
		return err
	}

	return application.App.Walk("services", func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}
		id := fmt.Sprintf("__yao_service.%s", share.ID(root, file))
		_, err := v8.Load(file, id)
		return err
	}, exts...)
}
