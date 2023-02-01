package script

import (
	"github.com/yaoapp/gou/application"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

// Load 加载共享库
func Load(cfg config.Config) error {
	exts := []string{"*.js"}
	return application.App.Walk("scripts", func(root, file string, isdir bool) error {
		_, err := v8.Load(file, share.ID(root, file))
		return err
	}, exts...)

	// return LoadFrom(filepath.Join(cfg.Root, "services"), "__yao_service.")
}

// // LoadBuildIn 从制品中读取
// func LoadBuildIn(dir string) error {
// 	return nil
// }

// // LoadFrom 从特定目录加载共享库
// func LoadFrom(dir string, prefix string) error {

// 	if share.DirNotExists(dir) {
// 		log.Error("%s does not exists", dir)
// 		return nil
// 	}

// 	// 加载共享脚本
// 	err := share.Walk(dir, ".js", func(root, filename string) {
// 		name := share.SpecName(root, filename)
// 		err := gou.Yao.Load(filename, fmt.Sprintf("%s%s", prefix, name))
// 		if err != nil {
// 			log.Error("加载脚本失败 %s", err.Error())
// 		}
// 	})
// 	return err
// }
