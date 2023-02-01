package store

import (
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

// Load load store
func Load(cfg config.Config) error {
	exts := []string{"*.yao", "*.json", "*.jsonc"}
	return application.App.Walk("stores", func(root, file string, isdir bool) error {
		_, err := store.Load(file, share.ID(root, file))
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
// 		_, err := gou.LoadStore(string(content), name)
// 		if err != nil {
// 			log.With(log.F{"root": root, "file": filename}).Error(err.Error())
// 		}
// 	})

// 	return err
// }
