package widgets

import (
	"github.com/yaoapp/gou"
)

// WidgetHandlers Processes
var WidgetHandlers = map[string]gou.ProcessHandler{
	"apis":    processApis,
	"actions": processActions,
	"models":  processModels,
	"fields":  processFields,
	"filters": processFilters,
}

func init() {
	gou.RegisterProcessGroup("widget", WidgetHandlers)
}

// Get the loaded APIs
func processApis(process *gou.Process) interface{} {
	return Apis()
}

// Get the actions of each widget
func processActions(process *gou.Process) interface{} {
	return Actions()
}

// Get the loaded Models
func processModels(process *gou.Process) interface{} {
	return Models()
}

// Get the loaded Fields
func processFields(process *gou.Process) interface{} {
	return Fields()
}

// Get the loaded Filters
func processFilters(process *gou.Process) interface{} {
	return Filters()
}

// Get the loaded flows
func processFlows() {}

// Get the loaded Models
func processScripts() {}
