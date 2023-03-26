package task

import (
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/task"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

// Load load task
func Load(cfg config.Config) error {
	exts := []string{"*.yao", "*.json", "*.jsonc"}
	return application.App.Walk("tasks", func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}
		_, err := task.Load(file, share.ID(root, file))
		return err
	}, exts...)
}
