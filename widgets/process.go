package widgets

import (
	"github.com/yaoapp/gou"
)

// WidgetHandlers Processes
var WidgetHandlers = map[string]gou.ProcessHandler{
	"apis": processApis,
}

func init() {
	gou.RegisterProcessGroup("widget", WidgetHandlers)
}

// Get the loaded APIs
func processApis(process *gou.Process) interface{} {
	return Apis()
}

// Get the actions of each widget
func processActions() {
}

// Get the loaded Models
func processModels() {}

// Get the loaded flows
func processFlows() {}

// Get the loaded Models
func processScripts() {}
