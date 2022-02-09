package script

import (
	"fmt"
	"path/filepath"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

// Load 加载共享库
func Load(cfg config.Config) error {
	if share.BUILDIN {
		return LoadBuildIn("scripts")
	}
	return LoadFrom(filepath.Join(cfg.Root, "scripts"))
}

// LoadBuildIn 从制品中读取
func LoadBuildIn(dir string) error {
	return nil
}

// LoadFrom 从特定目录加载共享库
func LoadFrom(dir string) error {

	if share.DirNotExists(dir) {
		return fmt.Errorf("%s does not exists", dir)
	}

	// 加载共享脚本
	err := share.Walk(dir, ".js", func(root, filename string) {
		name := share.SpecName(root, filename)
		err := gou.Yao.Load(filename, name)
		if err != nil {
			log.Error("加载脚本失败 %s", err.Error())
		}
	})
	return err
}
