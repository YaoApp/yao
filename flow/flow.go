package flow

import (
	"github.com/yaoapp/gou"
	"github.com/yaoapp/xiang/config"
	"github.com/yaoapp/xiang/share"
)

// Load 加载业务逻辑编排
func Load(cfg config.Config) {
	LoadFrom(cfg.RootFLow, "")
}

// LoadFrom 从特定目录加载
func LoadFrom(dir string, prefix string) {

	if share.DirNotExists(dir) {
		return
	}

	share.Walk(dir, ".json", func(root, filename string) {
		name := share.SpecName(root, filename)
		content := share.ReadFile(filename)
		gou.LoadFlow(string(content), prefix+name)
	})
}
