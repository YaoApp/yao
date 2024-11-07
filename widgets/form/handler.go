package form

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou/model"
	gouProcess "github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/session"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/i18n"
	"github.com/yaoapp/yao/widgets/action"
)

// ********************************
// * Execute the process of form *
// ********************************
// Life-Circle: Before Hook → Compute Edit → Run Process → Compute View → After Hook
// Execute Compute Edit On:    Save, Create, Update
// Execute Compute View On:    Find
func processHandler(p *action.Process, process *gouProcess.Process) (interface{}, error) {

	form, err := Get(process)
	if err != nil {
		return nil, err
	}
	args := p.Args(process)

	// Process
	name := p.Process
	if name == "" {
		name = p.ProcessBind
	}

	if name == "" {
		log.Error("[form] %s %s process is required", form.ID, p.Name)
		return nil, fmt.Errorf("[form] %s %s process is required", form.ID, p.Name)
	}

	// Before Hook
	if p.Before != nil {
		log.Trace("[form] %s %s before: exec(%v)", form.ID, p.Name, args)
		newArgs, err := p.Before.Exec(args, process.Sid, process.Global)
		if err != nil {
			log.Error("[form] %s %s before: %s", form.ID, p.Name, err.Error())
		} else {
			log.Trace("[form] %s %s before: args:%v", form.ID, p.Name, args)
			args = newArgs
		}
	}

	// Compute Edit
	err = form.ComputeEdit(p.Name, process, args, form.getField())
	if err != nil {
		log.Error("[form] %s %s Compute Edit Error: %s", form.ID, p.Name, err.Error())
	}

	// Execute Process
	act, err := gouProcess.Of(name, args...)
	if err != nil {
		log.Error("[form] %s %s -> %s %s", form.ID, p.Name, name, err.Error())
		return nil, fmt.Errorf("[form] %s %s -> %s %s", form.ID, p.Name, name, err.Error())
	}

	err = act.WithGlobal(process.Global).WithSID(process.Sid).Execute()
	if err != nil {
		log.Error("[form] %s %s -> %s %s", form.ID, p.Name, name, err.Error())
		return nil, fmt.Errorf("[form] %s %s -> %s %s", form.ID, p.Name, name, err.Error())
	}
	defer act.Release()
	res := act.Value()

	// Compute View
	err = form.ComputeView(p.Name, process, res, form.getField())
	if err != nil {
		log.Error("[form] %s %s Compute View Error: %s", form.ID, p.Name, err.Error())
	}

	// After hook
	if p.After != nil {
		log.Trace("[form] %s %s after: exec(%v)", form.ID, p.Name, res)
		newRes, err := p.After.Exec(res, process.Sid, process.Global)
		if err != nil {
			log.Error("[form] %s %s after: %s", form.ID, p.Name, err.Error())
		} else {
			log.Trace("[form] %s %s after: %v", form.ID, p.Name, newRes)
			res = newRes
		}
	}

	// Tranlate the result
	newRes, err := form.translate(p.Name, process, res)
	if err != nil {
		return nil, fmt.Errorf("[form] %s %s Translate Error: %s", form.ID, p.Name, err.Error())
	}

	return newRes, nil
}

// translateSetting
func (dsl *DSL) translate(name string, process *gouProcess.Process, data interface{}) (interface{}, error) {

	if strings.ToLower(name) != "yao.form.setting" {
		return data, nil
	}

	widgets := []string{}
	if dsl.Action != nil && dsl.Action.Bind != nil && dsl.Action.Bind.Model != "" {
		m := model.Select(dsl.Action.Bind.Model)
		widgets = append(widgets, fmt.Sprintf("model.%s", m.ID))
	}

	if dsl.Action != nil && dsl.Action.Bind != nil && dsl.Action.Bind.Table != "" {
		widgets = append(widgets, fmt.Sprintf("table.%s", dsl.Action.Bind.Table))
	}

	if dsl.Action != nil && dsl.Action.Bind != nil && dsl.Action.Bind.Form != "" {
		widgets = append(widgets, fmt.Sprintf("form.%s", dsl.Action.Bind.Form))
	}

	widgets = append(widgets, fmt.Sprintf("form.%s", dsl.ID))
	res, err := i18n.Trans(session.Lang(process, config.Conf.Lang), widgets, data)
	if err != nil {
		return nil, err
	}

	return res, nil
}
