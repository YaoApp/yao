package server

import (
	"fmt"
	"path/filepath"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

// Load 加载API
func Load(cfg config.Config) {
	var root = filepath.Join(cfg.Root, "servers")
	LoadFrom(root, "")
}

// LoadFrom 从特定目录加载
func LoadFrom(dir string, prefix string) error {

	if share.DirNotExists(dir) {
		return fmt.Errorf("%s does not exists", dir)
	}

	err := share.Walk(dir, ".sock.json", func(root, filename string) {
		name := prefix + share.SpecName(root, filename)
		content := share.ReadFile(filename)
		_, err := gou.LoadServer(string(content), name)
		if err != nil {
			log.With(log.F{"root": root, "file": filename}).Error(err.Error())
		}
	})

	return err
}
