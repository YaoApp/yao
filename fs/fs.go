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
	seedRoot := filepath.Join(cfg.AppSource, "seeds")
	dslDenyList := []string{scriptRoot, cfg.DataRoot}

	fs.Register("app", system.New(cfg.AppSource))        // App Soruce root path, it's an dangerous operation, be careful to use it.
	fs.Register("data", system.New(cfg.DataRoot))        // Data root
	fs.Register("seed", system.New(seedRoot).ReadOnly()) // Seed read only file system, for initial data seeding

	// Deprecated: DO NOT USE SYSTEM, DSL AND SCRIPT IN THE FUTURE, THEY WILL BE DEPRECATED IN THE FUTURE
	fs.Register("system", system.New(cfg.DataRoot))                        // alias Data
	fs.RootRegister("dsl", dsl.New(cfg.AppSource).DenyAbs(dslDenyList...)) // DSL ( will be deprecated in the future)
	fs.RootRegister("script", system.New(scriptRoot))                      // Script ( will be deprecated in the future)
	return nil
}
