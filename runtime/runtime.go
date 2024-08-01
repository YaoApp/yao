package runtime

import (
	"fmt"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/application"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/yao/config"
)

// Start v8 runtime
func Start(cfg config.Config) error {

	debug := false
	if cfg.Mode == "development" {
		debug = true
	}

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
		Debug:             debug,
	}

	// Read the tsconfig.json
	if cfg.Runtime.Import && application.App != nil {
		if exist, _ := application.App.Exists("tsconfig.json"); exist {
			var tsconfig v8.TSConfig
			raw, err := application.App.Read("tsconfig.json")
			if err != nil {
				return fmt.Errorf("tsconfig.json is not a valid json file %s", err)
			}

			err = jsoniter.Unmarshal(raw, &tsconfig)
			if err != nil {
				return fmt.Errorf("tsconfig.json is not a valid json file %s", err)
			}
			option.TSConfig = &tsconfig
		}
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
