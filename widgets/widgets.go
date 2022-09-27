package widgets

import (
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/widgets/app"
	"github.com/yaoapp/yao/widgets/form"
	"github.com/yaoapp/yao/widgets/login"
	"github.com/yaoapp/yao/widgets/table"
)

// Load the widgets
func Load(cfg config.Config) error {

	// login widget
	err := login.LoadAndExport(cfg)
	if err != nil {
		return err
	}

	// app widget
	err = app.LoadAndExport(cfg)
	if err != nil {
		return err
	}

	// table widget
	err = table.LoadAndExport(cfg)
	if err != nil {
		return err
	}

	// form widget
	err = form.LoadAndExport(cfg)
	if err != nil {
		return err
	}

	return nil
}
