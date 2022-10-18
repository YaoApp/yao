package form

import (
	"fmt"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/i18n"
	"github.com/yaoapp/yao/widgets/action"
)

var processActionDefaults = map[string]*action.Process{

	"Setting": {
		Name:    "yao.form.Setting",
		Process: "yao.form.Xgen",
		Default: []interface{}{nil},
	},
	"Component": {
		Name:    "yao.form.Component",
		Default: []interface{}{nil, nil, nil},
	},
	"Find": {
		Name:    "yao.form.Find",
		Default: []interface{}{nil, nil},
	},
	"Save": {
		Name:    "yao.form.Save",
		Default: []interface{}{nil},
	},
	"Create": {
		Name:    "yao.form.Create",
		Default: []interface{}{nil},
	},
	"Update": {
		Name:    "yao.form.Update",
		Default: []interface{}{nil, nil},
	},
	"Delete": {
		Name:    "yao.table.Delete",
		Default: []interface{}{nil},
	},
}

// SetDefaultProcess set the default value of action
func (act *ActionDSL) SetDefaultProcess() {

	act.Setting = action.ProcessOf(act.Setting).
		Merge(processActionDefaults["Setting"]).
		SetHandler(processHandler)

	act.Component = action.ProcessOf(act.Component).
		Merge(processActionDefaults["Component"]).
		SetHandler(processHandler)

	act.Find = action.ProcessOf(act.Find).
		WithBefore(act.BeforeFind).WithAfter(act.AfterFind).
		Merge(processActionDefaults["Find"]).
		SetHandler(processHandler)

	act.Save = action.ProcessOf(act.Save).
		WithBefore(act.BeforeSave).WithAfter(act.AfterSave).
		Merge(processActionDefaults["Save"]).
		SetHandler(processHandler)

	act.Create = action.ProcessOf(act.Create).
		WithBefore(act.BeforeCreate).WithAfter(act.AfterCreate).
		Merge(processActionDefaults["Create"]).
		SetHandler(processHandler)

	act.Update = action.ProcessOf(act.Update).
		WithBefore(act.BeforeUpdate).WithAfter(act.AfterUpdate).
		Merge(processActionDefaults["Update"]).
		SetHandler(processHandler)

	act.Delete = action.ProcessOf(act.Delete).
		WithBefore(act.BeforeDelete).WithAfter(act.AfterDelete).
		Merge(processActionDefaults["Delete"]).
		SetHandler(processHandler)

}

// BindModel bind model
func (act *ActionDSL) BindModel(m *gou.Model) {

	name := m.ID
	act.Find.Bind(fmt.Sprintf("models.%s.Find", name))
	act.Save.Bind(fmt.Sprintf("models.%s.Save", name))
	act.Create.Bind(fmt.Sprintf("models.%s.Create", name))
	act.Update.Bind(fmt.Sprintf("models.%s.Update", name))
	act.Delete.Bind(fmt.Sprintf("models.%s.Delete", name))

	// bind options
	if act.Bind.Option != nil {
		act.Find.Default[1] = act.Bind.Option
	}
}

func processHandler(p *action.Process, process *gou.Process) (interface{}, error) {

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

	// Compute In
	switch p.Name {
	case "yao.form.Save", "yao.form.Create":
		switch args[0].(type) {
		case map[string]interface{}, maps.MapStr:
			data := any.Of(args[0]).Map().MapStrAny
			err := form.computeSave(process, data)
			if err != nil {
				log.Error("[form] %s %s -> %s %s", form.ID, p.Name, name, err.Error())
			}
			args[0] = data
		}
		break

	case "yao.form.Update":
		switch args[1].(type) {
		case map[string]interface{}, maps.MapStr:
			data := any.Of(args[1]).Map().MapStrAny
			err := form.computeSave(process, data)
			if err != nil {
				log.Error("[form] %s %s -> %s %s", form.ID, p.Name, name, err.Error())
			}
			args[1] = data
		}
		break
	}

	// Execute Process
	act, err := gou.ProcessOf(name, args...)
	if err != nil {
		log.Error("[form] %s %s -> %s %s", form.ID, p.Name, name, err.Error())
		return nil, fmt.Errorf("[form] %s %s -> %s %s", form.ID, p.Name, name, err.Error())
	}

	res, err := act.WithGlobal(process.Global).WithSID(process.Sid).Exec()
	if err != nil {
		log.Error("[form] %s %s -> %s %s", form.ID, p.Name, name, err.Error())
		return nil, fmt.Errorf("[form] %s %s -> %s %s", form.ID, p.Name, name, err.Error())
	}

	// Compute Out
	switch p.Name {

	case "yao.form.Find":
		switch res.(type) {
		case map[string]interface{}, maps.MapStr:
			data := any.MapOf(res).MapStrAny
			err := form.computeFind(process, data)
			if err != nil {
				log.Error("[form] %s %s -> %s %s", form.ID, p.Name, name, err.Error())
			}
			res = data
		}
		break
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

	// Tranlate
	if p.Name == "yao.form.Setting" {

		widgets := []string{}
		if form.Action.Bind.Model != "" {
			m := gou.Select(form.Action.Bind.Model)
			widgets = append(widgets, fmt.Sprintf("model.%s", m.ID))
		}

		if form.Action.Bind.Table != "" {
			widgets = append(widgets, fmt.Sprintf("table.%s", form.Action.Bind.Table))
		}

		widgets = append(widgets, fmt.Sprintf("form.%s", form.ID))
		res, err = i18n.Trans(process.Lang(config.Conf.Lang), widgets, res)
		if err != nil {
			return nil, fmt.Errorf("[form] Trans.table.%s %s", form.ID, err.Error())
		}

	}

	return res, nil
}
