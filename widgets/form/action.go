package form

import (
	"fmt"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
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

	act.Setting = action.NewProcess("Setting", act.Setting).
		SetDefault(processActionDefaults).
		SetHandler(processHandler)

	act.Component = action.NewProcess("Component", act.Component).
		SetDefault(processActionDefaults).
		SetHandler(processHandler)

	act.Find = action.NewProcess("Find", act.Find).
		WithBefore(act.BeforeFind).
		WithAfter(act.AfterFind).
		SetDefault(processActionDefaults).
		SetHandler(processHandler)

	act.Save = action.NewProcess("Save", act.Save).
		WithBefore(act.BeforeSave).WithAfter(act.AfterSave).
		SetDefault(processActionDefaults).
		SetHandler(processHandler)

	act.Create = action.NewProcess("Create", act.Create).
		WithBefore(act.BeforeCreate).WithAfter(act.AfterCreate).
		SetDefault(processActionDefaults).
		SetHandler(processHandler)

	act.Update = action.NewProcess("Update", act.Update).
		WithBefore(act.BeforeUpdate).WithAfter(act.AfterUpdate).
		SetDefault(processActionDefaults).
		SetHandler(processHandler)

	act.Delete = action.NewProcess("Delete", act.Delete).
		WithBefore(act.BeforeDelete).WithAfter(act.AfterDelete).
		SetDefault(processActionDefaults).
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
		case map[string]interface{}:
			data := args[0].(map[string]interface{})
			err := form.computeSave(process, data)
			if err != nil {
				log.Error("[form] %s %s -> %s %s", form.ID, p.Name, name, err.Error())
			}
			args[0] = data
		}
		break

	case "yao.form.Update":
		switch args[1].(type) {
		case map[string]interface{}:
			data := args[1].(map[string]interface{})
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

	return res, nil
}
