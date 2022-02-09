package flow

import (
	"fmt"
	"path/filepath"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

// Load 加载业务逻辑编排
func Load(cfg config.Config) error {
	if share.BUILDIN {
		return LoadBuildIn("flows", "")
	}
	return LoadFrom(filepath.Join(cfg.Root, "flows"), "")
}

// LoadFrom 从特定目录加载
func LoadFrom(dir string, prefix string) error {

	if share.DirNotExists(dir) {
		return fmt.Errorf("%s does not exists", dir)
	}

	err := share.Walk(dir, ".json", func(root, filename string) {
		name := prefix + share.SpecName(root, filename)
		content := share.ReadFile(filename)
		_, err := gou.LoadFlowReturn(string(content), name)
		if err != nil {
			log.With(log.F{"root": root, "file": filename}).Error(err.Error())
		}
	})

	if err != nil {
		return err
	}

	// Load Script
	err = share.Walk(dir, ".js", func(root, filename string) {
		name := prefix + share.SpecName(root, filename)
		flow := gou.SelectFlow(name)
		if flow != nil {
			script := share.ScriptName(filename)
			content := share.ReadFile(filename)
			flow.LoadScript(string(content), script)
		}
	})

	return err
}

// LoadBuildIn 从制品中读取
func LoadBuildIn(dir string, prefix string) error {
	return nil
}
