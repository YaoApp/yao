package test

import (
	"os"
	"path/filepath"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/i18n"
	"github.com/yaoapp/yao/model"
	"github.com/yaoapp/yao/runtime"
	"github.com/yaoapp/yao/share"
)

// LoadEngine load engine
func LoadEngine(language ...string) error {

	runtime.Load(config.Conf)
	i18n.Load(config.Conf)
	share.DBConnect(config.Conf.DB) // removed later
	gou.LoadCrypt(`{}`, "PASSWORD")
	gou.LoadCrypt(`{}`, "AES")

	// Session server
	err := share.SessionStart()
	if err != nil {
		return err
	}

	// load engine models
	dev := os.Getenv("YAO_DEV")
	if dev != "" {
		err := model.LoadFrom(filepath.Join(dev, "yao", "models"), "xiang.")
		if err != nil {
			return err
		}
	}

	return nil
}
