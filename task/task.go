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
		_, err := task.Load(file, share.ID(root, file))
		return err
	}, exts...)
}

// // LoadFrom load from dir
// func LoadFrom(dir string, prefix string) error {

// 	if share.DirNotExists(dir) {
// 		return fmt.Errorf("%s does not exists", dir)
// 	}

// 	err := share.Walk(dir, ".json", func(root, filename string) {
// 		name := prefix + share.SpecName(root, filename)
// 		content := share.ReadFile(filename)
// 		_, err := gou.LoadTask(string(content), name)
// 		if err != nil {
// 			log.With(log.F{"root": root, "file": filename}).Error(err.Error())
// 		}
// 	})

// 	return err
// }
