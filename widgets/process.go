package widgets

import (
	"github.com/yaoapp/gou/process"
)

// WidgetHandlers Processes
var WidgetHandlers = map[string]process.Handler{
	"apis":    processApis,
	"actions": processActions,
	"models":  processModels,
	"fields":  processFields,
	"filters": processFilters,
}

func init() {
	process.RegisterGroup("widget", WidgetHandlers)
}

// Get the loaded APIs
func processApis(process *process.Process) interface{} {
	return Apis()
}

// Get the actions of each widget
func processActions(process *process.Process) interface{} {
	return Actions()
}

// Get the loaded Models
func processModels(process *process.Process) interface{} {
	return Models()
}

// Get the loaded Fields
func processFields(process *process.Process) interface{} {
	return Fields()
}

// Get the loaded Filters
func processFilters(process *process.Process) interface{} {
	return Filters()
}

// Get the loaded flows
func processFlows() {}

// Get the loaded Models
func processScripts() {}
