package schedule

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/schedule"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

// Load load schedule
func Load(cfg config.Config) error {
	messages := []string{}
	exts := []string{"*.sch.yao", "*.sch.json", "*.sch.jsonc"}
	err := application.App.Walk("schedules", func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}
		_, err := schedule.Load(file, share.ID(root, file))
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

// Start schedules
func Start() {
	for name, sch := range schedule.Schedules {
		sch.Start()
		log.Info("[Schedule] %s start", name)
	}
}

// Stop schedules
func Stop() {
	for name, sch := range schedule.Schedules {
		sch.Stop()
		log.Info("[Schedule] %s stop", name)
	}
}
