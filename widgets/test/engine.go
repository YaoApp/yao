package test

import (
	"os"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/i18n"
	"github.com/yaoapp/yao/share"
)

// LoadEngine load engine
func LoadEngine(language ...string) error {

	// runtime.Load(config.Conf)
	i18n.Load(config.Conf)
	share.DBConnect(config.Conf.DB) // removed later
	model.WithCrypt([]byte(`{}`), "PASSWORD")
	model.WithCrypt([]byte(`{}`), "AES")

	// Session server
	err := share.SessionStart()
	if err != nil {
		return err
	}

	// load engine models
	dev := os.Getenv("YAO_DEV")
	if dev != "" {
		// err := model.LoadFrom(filepath.Join(dev, "yao", "models"), "xiang.")
		// if err != nil {
		// return err
		// }
	}

	return nil
}
