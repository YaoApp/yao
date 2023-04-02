package task

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/task"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

// Load load task
func Load(cfg config.Config) error {
	messages := []string{}
	exts := []string{"*.yao", "*.json", "*.jsonc"}
	err := application.App.Walk("tasks", func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}
		_, err := task.Load(file, share.ID(root, file))
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

// Start tasks
func Start() {
	for name, t := range task.Tasks {
		go t.Start()
		log.Info("[Task] %s start", name)
	}
}

// Stop tasks
func Stop() {
	for name, t := range task.Tasks {
		t.Stop()
		log.Info("[Task] %s stop", name)
	}
}
