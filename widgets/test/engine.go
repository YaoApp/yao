package test

import (
	"os"
	"path/filepath"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/lang"
	"github.com/yaoapp/yao/model"
	"github.com/yaoapp/yao/share"
)

// LoadEngine load engine
func LoadEngine(language ...string) error {

	// langs
	if len(language) < 1 {
		os.Unsetenv("YAO_LANG")
	} else {
		os.Setenv("YAO_LANG", language[0])
	}
	lang.Load(config.Conf)

	share.DBConnect(config.Conf.DB) // removed later
	gou.LoadCrypt(`{}`, "PASSWORD")
	gou.LoadCrypt(`{}`, "AES")

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
