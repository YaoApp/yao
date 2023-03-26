package socket

import (
	"github.com/yaoapp/yao/config"
)

// Load 加载API
func Load(cfg config.Config) error {
	// var root = filepath.Join(cfg.Root, "sockets")
	// return LoadFrom(root, "")
	return nil
}

// LoadFrom 从特定目录加载
// func LoadFrom(dir string, prefix string) error {

// 	if share.DirNotExists(dir) {
// 		return fmt.Errorf("%s does not exists", dir)
// 	}

// 	err := share.Walk(dir, ".sock.json", func(root, filename string) {
// 		name := prefix + share.SpecName(root, filename)
// 		content := share.ReadFile(filename)
// 		_, err := gou.LoadSocket(string(content), name)
// 		if err != nil {
// 			log.With(log.F{"root": root, "file": filename}).Error(err.Error())
// 		}
// 	})

// 	return err
// }
