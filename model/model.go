package model

import (
	"fmt"
	"path/filepath"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

// Load 加载数据模型
func Load(cfg config.Config) error {
	if share.BUILDIN {
		return LoadBuildIn("models", "")
	}
	return LoadFrom(filepath.Join(cfg.Root, "models"), "")
}

// LoadFrom 从特定目录加载
func LoadFrom(dir string, prefix string) error {

	if share.DirNotExists(dir) {
		return fmt.Errorf("%s does not exists", dir)
	}

	err := share.Walk(dir, ".json", func(root, filename string) {
		name := prefix + share.SpecName(root, filename)
		content := share.ReadFile(filename)
		_, err := gou.LoadModelReturn(string(content), name)
		if err != nil {
			log.With(log.F{"root": root, "file": filename}).Error(err.Error())
			return
		}
	})

	return err
}

// LoadBuildIn 从制品中读取
func LoadBuildIn(dir string, prefix string) error {
	return nil
}
