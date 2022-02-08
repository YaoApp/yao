package plugin

import (
	"fmt"
	"path/filepath"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/xiang/config"
	"github.com/yaoapp/xiang/share"
)

// Load 加载业务插件
func Load(cfg config.Config) error {
	return LoadFrom(filepath.Join(cfg.Root, "plugins"))
}

// LoadFrom 从特定目录加载
func LoadFrom(dir string) error {

	if share.DirNotExists(dir) {
		return fmt.Errorf("%s does not exists", dir)
	}

	err := share.Walk(dir, ".so", func(root, filename string) {
		name := share.SpecName(root, filename)
		_, err := gou.LoadPluginReturn(filename, name)
		if err != nil {
			log.With(log.F{"root": root, "file": filename}).Error(err.Error())
		}
	})
	return err
}
