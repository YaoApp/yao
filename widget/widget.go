package widget

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou/api"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/share"
)

// LoadInstances load the widget instances
func (widget *DSL) LoadInstances() error {

	messages := []string{}
	err := widget.FS.Walk(func(id string, source map[string]interface{}) {
		instance := NewInstance(widget.ID, id, source, widget.Loader)
		err := instance.Load()
		if err != nil {
			messages = append(messages, fmt.Sprintf("%v %s", id, err.Error()))
			return
		}
		widget.Instances.Store(id, instance)
	})

	if len(messages) > 0 {
		return fmt.Errorf("widgets.%s Load: %s", widget.ID, strings.Join(messages, ";"))
	}

	return err
}

// ReloadInstances reload the widget instances
func (widget *DSL) ReloadInstances() error {

	messages := []string{}

	// Reload the remote widget
	widget.Instances.Range(func(key, value interface{}) bool {
		if instance, ok := value.(*Instance); ok {
			err := instance.Reload()
			if err != nil {
				messages = append(messages, fmt.Sprintf("%v %s", key, err.Error()))
			}
		}
		return true
	})

	if len(messages) > 0 {
		return fmt.Errorf("widgets.%s Reload: %s", widget.ID, strings.Join(messages, ";"))
	}

	return nil
}

// UnloadInstances unload the widget instances
func (widget *DSL) UnloadInstances() error {

	messages := []string{}

	// Unload the remote widget
	widget.Instances.Range(func(key, value interface{}) bool {
		if instance, ok := value.(*Instance); ok {
			err := instance.Unload()
			if err != nil {
				messages = append(messages, fmt.Sprintf("%v %s", key, err.Error()))
			}
			widget.Instances.Delete(key)
		}

		return true
	})

	if len(messages) > 0 {
		return fmt.Errorf("widgets.%s Unload: %s", widget.ID, strings.Join(messages, ";"))
	}

	return nil
}

// RegisterProcess register the widget process
func (widget *DSL) RegisterProcess() error {
	if widget.Process == nil {
		return nil
	}

	handlers := map[string]process.Handler{}
	for name, processName := range widget.Process {

		if processName == "" {
			continue
		}
		handlers[name] = widget.handler(processName)
	}

	process.RegisterGroup(fmt.Sprintf("widgets.%s", widget.ID), handlers)
	return nil
}

// RegisterAPI register the widget API
func (widget *DSL) RegisterAPI() error {

	if widget.API == nil {
		return nil
	}

	id := fmt.Sprintf("__yao.widget.%s", widget.ID)
	widget.API.Group = fmt.Sprintf("/__yao/widget/%s", widget.ID)

	//  Register the widget API
	api.APIs[id] = &api.API{
		ID:   fmt.Sprintf("__yao.widget.%s", widget.ID),
		File: widget.File,
		HTTP: *widget.API,
		Type: "http",
	}

	return nil
}

// Register the process handler
func (widget *DSL) handler(processName string) process.Handler {

	return func(p *process.Process) interface{} {

		p.ValidateArgNums(1)
		instanceID := p.ArgsString(0)
		instance, ok := widget.Instances.Load(instanceID)
		if !ok {
			exception.New("The widget %s instance %s not found", 404, widget.ID, instanceID).Throw()
		}

		args := []interface{}{}
		args = append(args, p.Args...)
		args = append(args, instance.(*Instance).dsl)

		return process.New(processName, args...).Run()
	}
}

// Save the widget source to file
func (widget *DSL) Save(file string, source map[string]interface{}) error {

	err := widget.FS.Save(file, source)
	if err != nil {
		return err
	}

	id := share.ID("", file)
	instance := NewInstance(widget.ID, id, source, widget.Loader)

	// new instance
	old, ok := widget.Instances.Load(id)
	if !ok {
		err := instance.Load()
		if err != nil {
			return err
		}

		widget.Instances.Store(id, instance)
		return nil
	}

	// Reload the instance
	if widget.Remote != nil && widget.Remote.Reload {
		instance.dsl = old.(*Instance).dsl
		err = instance.Reload()
		if err != nil {
			return err
		}
	}

	widget.Instances.Store(id, instance)
	return nil
}

// Remove the widget source file
func (widget *DSL) Remove(file string) error {

	err := widget.FS.Remove(file)
	if err != nil {
		return err
	}

	id := share.ID("", file)
	widget.Instances.Delete(id)
	return nil
}
