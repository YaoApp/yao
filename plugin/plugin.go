package plugin

import (
	"github.com/yaoapp/gou"
	"github.com/yaoapp/xiang/config"
	"github.com/yaoapp/xiang/share"
)

// Load 加载业务插件
func Load(cfg config.Config) {
	LoadFrom(cfg.RootPlugin)
}

// LoadFrom 从特定目录加载
func LoadFrom(dir string) {

	if share.DirNotExists(dir) {
		return
	}

	share.Walk(dir, ".so", func(root, filename string) {
		name := share.SpecName(root, filename)
		gou.LoadPlugin(filename, name)
	})
}
