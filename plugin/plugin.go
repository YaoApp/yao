package plugin

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/yaoapp/gou/plugin"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

// Load 加载业务插件
func Load(cfg config.Config) error {

	root, err := Root(cfg)
	if err != nil {
		return err
	}

	// Ignore if the plugins directory does not exist
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return nil
	}

	messages := []string{}
	err = filepath.Walk(root, func(file string, info fs.FileInfo, err error) error {
		if info == nil || info.IsDir() {
			return nil
		}

		if !strings.HasSuffix(file, ".so") && !strings.HasSuffix(file, ".dll") {
			return nil
		}

		_, err = plugin.Load(file, share.ID(root, file))
		if err != nil {
			messages = append(messages, err.Error())
		}
		return err
	})

	if len(messages) > 0 {
		return fmt.Errorf("%s", strings.Join(messages, ";\n"))
	}

	return err

}

// Root return plugin root
func Root(cfg config.Config) (string, error) {
	root := filepath.Join(cfg.ExtensionRoot, "plugins")
	if cfg.ExtensionRoot == "" {
		root = filepath.Join(cfg.Root, "plugins")
	}

	root, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}

	return root, nil
}
