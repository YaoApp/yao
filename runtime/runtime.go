package runtime

import (
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/yao/config"
)

// Start v8 runtime
func Start(cfg config.Config) error {

	option := &v8.Option{
		MinSize:           cfg.Runtime.MinSize,
		MaxSize:           cfg.Runtime.MaxSize,
		HeapSizeLimit:     cfg.Runtime.HeapSizeLimit,
		HeapAvailableSize: cfg.Runtime.HeapAvailableSize,
		HeapSizeRelease:   cfg.Runtime.HeapSizeRelease,
		Precompile:        cfg.Runtime.Precompile,
		DataRoot:          cfg.DataRoot,
		Mode:              cfg.Runtime.Mode,
		DefaultTimeout:    cfg.Runtime.DefaultTimeout,
		ContextTimeout:    cfg.Runtime.ContextTimeout,
		Import:            cfg.Runtime.Import,
	}

	err := v8.Start(option)
	if err != nil {
		return err
	}

	return nil
}

// Stop v8 runtime
func Stop() error {
	v8.Stop()
	return nil
}
