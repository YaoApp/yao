package fs

import (
	"path/filepath"

	"github.com/yaoapp/gou/fs"
	"github.com/yaoapp/gou/fs/dsl"
	"github.com/yaoapp/gou/fs/system"
	"github.com/yaoapp/yao/config"
)

// Load system fs
func Load(cfg config.Config) error {

	scriptRoot := filepath.Join(cfg.AppSource, "scripts")
	dslDenyList := []string{scriptRoot, cfg.DataRoot}

	fs.Register("system", system.New(cfg.DataRoot))
	fs.RootRegister("dsl", dsl.New(cfg.AppSource).DenyAbs(dslDenyList...)) // DSL
	fs.RootRegister("script", system.New(scriptRoot))                      // Script
	return nil
}
