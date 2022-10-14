package fs

import (
	"os"
	"path/filepath"

	"github.com/yaoapp/gou/fs"
	"github.com/yaoapp/gou/fs/dsl"
	"github.com/yaoapp/gou/fs/system"
	"github.com/yaoapp/yao/config"
)

// Load system fs
func Load(cfg config.Config) error {

	root, err := filepath.Abs(cfg.Root)
	if err != nil {
		return err
	}

	dataRoot, err := Root(cfg)
	if err != nil {
		return err
	}

	scriptRoot := filepath.Join(root, "scripts")
	dslDenyList := []string{scriptRoot, dataRoot}

	if _, err := os.Stat(dataRoot); os.IsNotExist(err) {
		err := os.MkdirAll(dataRoot, os.ModePerm)
		if err != nil {
			return err
		}
	}

	fs.Register("system", system.New(dataRoot))
	// fs.Register("binary", system.New(root))                    // Next
	fs.RootRegister("dsl", dsl.New(root).DenyAbs(dslDenyList...)) // DSL
	fs.RootRegister("script", system.New(scriptRoot))             // Script
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
