package driver

import (
	"fmt"
	"strings"
	"sync"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/yao/share"
)

// Source the application source driver
type Source struct {
	Path       string
	Extensions []string
	Instances  sync.Map
}

// NewSource create a new local driver
func NewSource(path string, exts []string) *Source {
	return &Source{
		Path:       path,
		Extensions: exts,
	}
}

// Walk load the widget instances
func (app *Source) Walk(cb func(string, map[string]interface{})) error {

	if app.Path == "" {
		return fmt.Errorf("The widget path is empty")
	}

	if app.Extensions == nil || len(app.Extensions) == 0 {
		app.Extensions = []string{"*.yao", "*.json", "*.jsonc"}
	}

	messages := []string{}
	err := application.App.Walk(app.Path, func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}

		id := share.ID(root, file)
		source := map[string]interface{}{}

		data, err := application.App.Read(file)
		if err != nil {
			messages = append(messages, err.Error())
			return nil
		}

		err = application.Parse(file, data, &source)
		if err != nil {
			messages = append(messages, err.Error())
			return nil
		}

		cb(id, source)
		return nil
	}, app.Extensions...)

	if err != nil {
		return err
	}

	if len(messages) > 0 {
		return fmt.Errorf("%s", strings.Join(messages, ";\n"))
	}

	return nil
}

// Save save the widget DSL
func (app *Source) Save(file string, source map[string]interface{}) error {
	return fmt.Errorf("The widget source driver is read-only, using Studio API instead")
}

// Remove remove the widget DSL
func (app *Source) Remove(file string) error {
	return fmt.Errorf("The widget source driver is read-only, using Studio API instead")
}
