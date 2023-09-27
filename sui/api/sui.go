package api

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
	"github.com/yaoapp/yao/sui/core"
	"github.com/yaoapp/yao/sui/storages/azure"
	"github.com/yaoapp/yao/sui/storages/local"
)

// New create a new sui
func New(dsl *core.DSL) (core.SUI, error) {

	if dsl.Storage == nil {
		return nil, fmt.Errorf("storage is not required")
	}

	switch strings.ToLower(dsl.Storage.Driver) {

	case "local":
		return local.New(dsl)

	case "azure":
		return azure.New(dsl)

	default:
		return nil, fmt.Errorf("%s is not a valid driver", dsl.Storage.Driver)
	}
}

// Load load the sui
func Load(cfg config.Config) error {
	exts := []string{"*.sui.yao", "*.sui.jsonc", "*.sui.json"}
	err := application.App.Walk("suis", func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}

		id := share.ID(root, file)
		_, err := loadFile(file, id)
		if err != nil {
			log.Error("[sui] Load sui %s error: %s", id, err.Error())
			return err
		}
		return nil
	}, exts...)

	if err != nil {
		return err
	}

	return registerAPI()
}

func loadFile(file string, id string) (core.SUI, error) {

	dsl, err := core.Load(file, id)
	if err != nil {
		return nil, err
	}

	sui, err := New(dsl)
	if err != nil {
		return nil, err
	}

	core.SUIs[id] = sui
	return core.SUIs[id], nil
}
