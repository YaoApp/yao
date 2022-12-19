package widgets

import (
	"fmt"
	"strings"

	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/widgets/app"
	"github.com/yaoapp/yao/widgets/chart"
	"github.com/yaoapp/yao/widgets/component"
	"github.com/yaoapp/yao/widgets/dashboard"
	"github.com/yaoapp/yao/widgets/expression"
	"github.com/yaoapp/yao/widgets/field"
	"github.com/yaoapp/yao/widgets/form"
	"github.com/yaoapp/yao/widgets/list"
	"github.com/yaoapp/yao/widgets/login"
	"github.com/yaoapp/yao/widgets/table"
)

// Load the widgets
func Load(cfg config.Config) error {

	messages := []string{}

	// load expression
	err := expression.Export()
	if err != nil {
		messages = append(messages, err.Error())
	}

	// load component
	err = component.Export()
	if err != nil {
		messages = append(messages, err.Error())
	}

	// load field transform
	err = field.LoadAndExport(config.Conf)
	if err != nil {
		messages = append(messages, err.Error())
	}

	// login widget
	err = login.LoadAndExport(cfg)
	if err != nil {
		messages = append(messages, err.Error())
	}

	// app widget
	err = app.LoadAndExport(cfg)
	if err != nil {
		messages = append(messages, err.Error())
	}

	// table widget
	err = table.LoadAndExport(cfg)
	if err != nil {
		messages = append(messages, err.Error())
	}

	// list widget
	err = list.LoadAndExport(cfg)
	if err != nil {
		messages = append(messages, err.Error())
	}

	// form widget
	err = form.LoadAndExport(cfg)
	if err != nil {
		messages = append(messages, err.Error())
	}

	// chart widget
	err = chart.LoadAndExport(cfg)
	if err != nil {
		messages = append(messages, err.Error())
	}

	// dashboard widget
	err = dashboard.LoadAndExport(cfg)
	if err != nil {
		messages = append(messages, err.Error())
	}

	if len(messages) > 0 {
		err = fmt.Errorf(strings.Join(messages, ";\n"))
		return err
	}

	return nil
}
