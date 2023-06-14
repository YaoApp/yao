package widget

import (
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

func init() {
	process.RegisterGroup("widget", map[string]process.Handler{
		"Save":   ProcessSave,
		"Remove": ProcessRemove,
	})
}

// ProcessSave process the widget save
func ProcessSave(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	name := process.ArgsString(0)
	file := process.ArgsString(1)
	source := process.ArgsMap(2)

	widget, ok := Widgets[name]
	if !ok {
		exception.New("The widget %s not found", 404, name).Throw()
	}

	err := widget.Save(file, source)
	if err != nil {
		exception.New(err.Error(), 500, name, err).Throw()
	}

	return nil
}

// ProcessRemove process the widget save
func ProcessRemove(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	name := process.ArgsString(0)
	file := process.ArgsString(1)
	widget, ok := Widgets[name]
	if !ok {
		exception.New("The widget %s not found", 404, name).Throw()
	}

	err := widget.Remove(file)
	if err != nil {
		exception.New(err.Error(), 500, name).Throw()
	}
	return nil
}
