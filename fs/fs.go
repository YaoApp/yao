package fs

import (
	"os"
	"path/filepath"

	"github.com/yaoapp/gou/fs"
	"github.com/yaoapp/gou/fs/system"
	"github.com/yaoapp/yao/config"
)

// Load system fs
func Load(cfg config.Config) error {

	root, err := Root(cfg)
	if err != nil {
		return err
	}

	if _, err := os.Stat(root); os.IsNotExist(err) {
		err := os.MkdirAll(root, os.ModePerm)
		if err != nil {
			return err
		}
	}

	fs.Register("system", system.New(root))
	fs.Register("binary", system.New(root)) // next
	return nil
}

// Root return data root
func Root(cfg config.Config) (string, error) {
	root := cfg.DataRoot
	if root == "" {
		root = filepath.Join(cfg.Root, "data")
	}

	root, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}

	return root, nil
}
