package model

import (
	"github.com/yaoapp/gou"
	"github.com/yaoapp/xiang/config"
	"github.com/yaoapp/xiang/share"
)

// Load 加载数据模型
func Load(cfg config.Config) {
	LoadFrom(cfg.RootModel, "")
}

// LoadFrom 从特定目录加载
func LoadFrom(dir string, prefix string) {

	if share.DirNotExists(dir) {
		return
	}

	share.Walk(dir, ".json", func(root, filename string) {
		name := prefix + share.SpecName(root, filename)
		content := share.ReadFile(filename)
		gou.LoadModel(string(content), name)
	})
}
