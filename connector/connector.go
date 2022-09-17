package connector

import (
	"fmt"
	"path/filepath"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

// Load load store
func Load(cfg config.Config) error {
	var root = filepath.Join(cfg.Root, "connectors")
	return LoadFrom(root, "")
}

// LoadFrom load from dir
func LoadFrom(dir string, prefix string) error {

	if share.DirNotExists(dir) {
		return fmt.Errorf("%s does not exists", dir)
	}

	err := share.Walk(dir, ".json", func(root, filename string) {
		name := prefix + share.SpecName(root, filename)
		content := share.ReadFile(filename)
		_, err := connector.Load(string(content), name)
		if err != nil {
			log.With(log.F{"root": root, "file": filename}).Error(err.Error())
		}
	})

	return err
}
