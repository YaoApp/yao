package widget

import (
	"github.com/yaoapp/gou/process"
)

// NewInstance create a new widget instance
func NewInstance(widgetID string, instanceID string, source map[string]interface{}, loader LoaderDSL) *Instance {
	return &Instance{id: instanceID, source: source, widget: widgetID, loader: loader}
}

// Load load the widget instance
func (instance *Instance) Load() error {
	if instance.loader.Load == "" {
		return nil
	}
	dsl, err := instance.exec(instance.loader.Load, instance.id, instance.source)
	if err != nil {
		return err
	}

	instance.dsl = dsl
	return nil
}

// Reload reload the widget instance
func (instance *Instance) Reload() error {
	if instance.loader.Reload == "" {
		return nil
	}

	dsl, err := instance.exec(instance.loader.Reload, instance.id, instance.source, instance.dsl)
	if err != nil {
		return err
	}

	instance.dsl = dsl
	return nil
}

// Unload unload the widget instance
func (instance *Instance) Unload() error {
	if instance.loader.Unload == "" {
		return nil
	}
	_, err := instance.exec(instance.loader.Unload, instance.id)
	return err
}

// exec exec the widget process
func (instance *Instance) exec(processName string, args ...interface{}) (interface{}, error) {
	p, err := process.Of(processName, args...)
	if err != nil {
		return nil, err
	}
	return p.Exec()
}
