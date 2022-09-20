package widgets

import (
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/widgets/app"
	"github.com/yaoapp/yao/widgets/login"
)

// Load the widgets
func Load(cfg config.Config) error {

	err := login.Load(cfg)
	if err != nil {
		return err
	}

	// app login
	err = app.Load(cfg)
	if err != nil {
		return err
	}

	return nil
}
