package runtime

import (
	"github.com/yaoapp/gou"
	"github.com/yaoapp/gou/runtime"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/fs"
)

// Load runtime
func Load(cfg config.Config) error {
	dataRoot, err := fs.Root(cfg)
	if err != nil {
		return err
	}
	gou.LoadRuntime(runtime.Option{FileRoot: dataRoot})
	return nil
}
