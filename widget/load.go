package widget

import (
	"fmt"
	"strings"
	"sync"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
	"github.com/yaoapp/yao/widget/driver"
)

// Widgets the loaded widgets
var Widgets = map[string]*DSL{}

// Load Widgets
func Load(cfg config.Config) error {

	exts := []string{"*.wid.yao", "*.wid.json", "*.wid.jsonc"}
	messages := []string{}

	err := application.App.Walk("widgets", func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}

		id := share.ID(root, file)
		_, err := LoadFile(file, id)
		if err != nil {
			messages = append(messages, err.Error())
		}
		return nil
	}, exts...)

	if err != nil {
		return err
	}

	if len(messages) > 0 {
		return fmt.Errorf("%s", strings.Join(messages, ";\n"))
	}

	return nil
}

// LoadInstances load widget instances
func LoadInstances() error {
	messages := []string{}
	for _, widget := range Widgets {
		err := widget.LoadInstances()
		if err != nil {
			messages = append(messages, err.Error())
		}
	}

	if len(messages) > 0 {
		return fmt.Errorf("%s", strings.Join(messages, ";\n"))
	}
	return nil
}

// LoadFile load widget by file
func LoadFile(file string, id string) (*DSL, error) {

	data, err := application.App.Read(file)
	if err != nil {
		return nil, err
	}

	return LoadSource(data, file, id)
}

// LoadSource load widget by source
func LoadSource(data []byte, file, id string) (*DSL, error) {

	widget := &DSL{ID: id, File: file, Instances: sync.Map{}}
	err := application.Parse(file, data, &widget)
	if err != nil {
		return nil, err
	}

	if widget.Remote != nil {

		widget.FS, err = driver.NewConnector(widget.ID, widget.Remote.Connector, widget.Remote.Table, widget.Remote.Reload)
		if err != nil {
			return nil, err
		}

	} else {
		widget.FS = driver.NewSource(widget.Path, widget.Extensions)
	}

	// register the widget process
	err = widget.RegisterProcess()
	if err != nil {
		return nil, err
	}

	// register the widget api
	err = widget.RegisterAPI()
	if err != nil {
		return nil, err
	}

	Widgets[id] = widget
	return Widgets[id], nil
}
