package schedule

import (
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/schedule"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

// Load load schedule
func Load(cfg config.Config) error {
	exts := []string{"*.sch.yao", "*.sch.json", "*.sch.jsonc"}
	return application.App.Walk("schedules", func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}
		_, err := schedule.Load(file, share.ID(root, file))
		return err
	}, exts...)
}
