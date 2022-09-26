package table

import (
	"fmt"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
)

var processActionDefaults = map[string]*ProcessActionDSL{

	"Setting": {
		Name:    "yao.table.Setting",
		Process: "yao.table.Xgen",
		Default: []interface{}{nil},
	},
	"Component": {
		Name:    "yao.table.Component",
		Default: []interface{}{nil, nil, nil},
	},
	"Search": {
		Name:    "yao.table.Search",
		Default: []interface{}{nil, 1, 20},
	},
	"Get": {
		Name:    "yao.table.Get",
		Default: []interface{}{nil},
	},
	"Find": {
		Name:    "yao.table.Find",
		Default: []interface{}{nil, nil},
	},
	"Save": {
		Name:    "yao.table.Save",
		Default: []interface{}{nil},
	},
	"Create": {
		Name:    "yao.table.Create",
		Default: []interface{}{nil},
	},
	"Insert": {
		Name:    "yao.table.Insert",
		Default: []interface{}{nil, nil},
	},
	"Update": {
		Name:    "yao.table.Update",
		Default: []interface{}{nil, nil},
	},
	"UpdateWhere": {
		Name:    "yao.table.UpdateWhere",
		Default: []interface{}{nil, nil},
	},
	"UpdateIn": {
		Name:    "yao.table.UpdateIn",
		Default: []interface{}{nil, nil},
	},
	"Delete": {
		Name:    "yao.table.Delete",
		Default: []interface{}{nil},
	},
	"DeleteWhere": {
		Name:    "yao.table.DeleteWhere",
		Default: []interface{}{nil},
	},
	"DeleteIn": {
		Name:    "yao.table.DeleteIn",
		Default: []interface{}{nil},
	},
}

// SetDefaultProcess set the default value of action
func (action *ActionDSL) SetDefaultProcess() {
	action.Setting = action.newProcessAction("Setting", action.Setting, nil, nil)
	action.Component = action.newProcessAction("Component", action.Component, nil, nil)
	action.Search = action.newProcessAction("Search", action.Search, action.BeforeSearch, action.AfterSearch)
	action.Get = action.newProcessAction("Get", action.Get, action.BeforeGet, action.AfterGet)
	action.Find = action.newProcessAction("Find", action.Find, action.BeforeFind, action.AfterFind)
	action.Save = action.newProcessAction("Save", action.Save, action.BeforeSave, action.AfterSave)
	action.Create = action.newProcessAction("Create", action.Create, action.BeforeCreate, action.AfterCreate)
	action.Insert = action.newProcessAction("Insert", action.Insert, action.BeforeInsert, action.AfterInsert)
	action.Update = action.newProcessAction("Update", action.Update, action.BeforeUpdate, action.AfterUpdate)
	action.UpdateWhere = action.newProcessAction("UpdateWhere", action.UpdateWhere, action.BeforeUpdateWhere, action.AfterUpdateWhere)
	action.UpdateIn = action.newProcessAction("UpdateIn", action.UpdateIn, action.BeforeUpdateIn, action.AfterUpdateIn)
	action.Delete = action.newProcessAction("Delete", action.Delete, action.BeforeDelete, action.AfterDelete)
	action.DeleteWhere = action.newProcessAction("DeleteWhere", action.DeleteWhere, action.BeforeDeleteWhere, action.AfterDeleteWhere)
	action.DeleteIn = action.newProcessAction("DeleteIn", action.DeleteIn, action.BeforeDeleteIn, action.AfterDeleteIn)
}

// BindModel bind model
func (action *ActionDSL) BindModel(m *gou.Model) {
	name := m.ID // should be id
	action.Search.Bind(fmt.Sprintf("models.%s.Paginate", name))
	action.Get.Bind(fmt.Sprintf("models.%s.Get", name))
	action.Find.Bind(fmt.Sprintf("models.%s.Find", name))
	action.Save.Bind(fmt.Sprintf("models.%s.Save", name))
	action.Create.Bind(fmt.Sprintf("models.%s.Create", name))
	action.Insert.Bind(fmt.Sprintf("models.%s.Insert", name))
	action.Update.Bind(fmt.Sprintf("models.%s.Update", name))
	action.UpdateWhere.Bind(fmt.Sprintf("models.%s.UpdateWhere", name))
	action.UpdateIn.Bind(fmt.Sprintf("models.%s.UpdateWhere", name))
	action.Delete.Bind(fmt.Sprintf("models.%s.Delete", name))
	action.DeleteWhere.Bind(fmt.Sprintf("models.%s.DeleteWhere", name))
	action.DeleteIn.Bind(fmt.Sprintf("models.%s.DeleteWhere", name))

	// bind options
	if action.Bind.Option != nil {
		action.Search.Default[0] = action.Bind.Option
		action.Get.Default[0] = action.Bind.Option
		action.Find.Default[1] = action.Bind.Option
	}
}

// setDefault Set the process action disabled
func (action *ActionDSL) newProcessAction(name string, p *ProcessActionDSL, before *BeforeHookActionDSL, after *AfterHookActionDSL) *ProcessActionDSL {

	if p == nil {
		p = &ProcessActionDSL{}
	}

	p.After = after
	p.Before = before

	if defaultProcess, has := processActionDefaults[name]; has {

		p.Name = defaultProcess.Name

		if p.Process == "" {
			p.Process = defaultProcess.Process
		}

		if p.Guard == "" {
			p.Guard = defaultProcess.Guard
		}

		if p.Default == nil {
			p.Default = defaultProcess.Default
		}

		// format defaults
		if len(p.Default) != len(defaultProcess.Default) {
			defauts := defaultProcess.Default
			nums := len(p.Default)
			if nums > len(defaultProcess.Default) {
				nums = len(defaultProcess.Default)
			}
			for i := 0; i < nums; i++ {
				defauts[i] = p.Default[i]
			}
			p.Default = defauts
		}

	}

	return p
}

// Bind the process name
func (p *ProcessActionDSL) Bind(name string) {
	p.ProcessBind = name
}

// Args get the process args
func (p *ProcessActionDSL) Args(process *gou.Process) []interface{} {
	process.ValidateArgNums(1)
	args := p.Default
	nums := len(process.Args[1:])
	if nums > len(args) {
		nums = len(args)
	}

	for i := 0; i < nums; i++ {
		args[i] = p.deepMergeDefault(process.Args[i+1], args[i])
	}
	return args
}

// Exec exec the process
func (p *ProcessActionDSL) Exec(process *gou.Process) (interface{}, error) {

	tab, err := Get(process)
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
		log.Error("[table] %s %s process is required", tab.ID, p.Name)
		return nil, fmt.Errorf("[table] %s %s process is required", tab.ID, p.Name)
	}

	// Before Hook
	if p.Before != nil {
		log.Trace("[table] %s %s before: exec(%v)", tab.ID, p.Name, args)
		newArgs, err := p.Before.Exec(args, process.Sid, process.Global)
		if err != nil {
			log.Error("[table] %s %s before: %s", tab.ID, p.Name, err.Error())
		} else {
			log.Trace("[table] %s %s before: args:%v", tab.ID, p.Name, args)
			args = newArgs
		}
	}

	// Compute In
	switch p.Name {
	case "yao.table.Save", "yao.table.Create":
		switch args[0].(type) {
		case map[string]interface{}:
			data := args[0].(map[string]interface{})
			err := tab.computeSave(process, data)
			if err != nil {
				log.Error("[table] %s %s -> %s %s", tab.ID, p.Name, name, err.Error())
			}
			args[0] = data
		}
		break
	case "yao.table.Update", "yao.table.UpdateWhere", "yao.table.UpdateIn":
		switch args[1].(type) {
		case map[string]interface{}:
			data := args[1].(map[string]interface{})
			err := tab.computeSave(process, data)
			if err != nil {
				log.Error("[table] %s %s -> %s %s", tab.ID, p.Name, name, err.Error())
			}
			args[1] = data
		}
		break
	case "yao.table.Insert":
		break
	}

	// Execute Process
	act, err := gou.ProcessOf(name, args...)
	if err != nil {
		log.Error("[table] %s %s -> %s %s", tab.ID, p.Name, name, err.Error())
		return nil, fmt.Errorf("[table] %s %s -> %s %s", tab.ID, p.Name, name, err.Error())
	}

	res, err := act.WithGlobal(process.Global).WithSID(process.Sid).Exec()
	if err != nil {
		log.Error("[table] %s %s -> %s %s", tab.ID, p.Name, name, err.Error())
		return nil, fmt.Errorf("[table] %s %s -> %s %s", tab.ID, p.Name, name, err.Error())
	}

	// Compute Out
	switch p.Name {

	case "yao.table.Search":
		if newMap, ok := res.(map[string]interface{}); ok {
			err := tab.computeSearch(process, newMap, "data")
			if err != nil {
				log.Error("[table] %s %s -> %s %s", tab.ID, p.Name, name, err.Error())
			}
			res = newMap
		}
		break

	case "yao.table.Get":

		if _, ok := res.([]maps.MapStr); ok {
			data := []interface{}{}
			for _, v := range res.([]maps.MapStr) {
				data = append(data, map[string]interface{}(v))
			}
			res = data
		}

		if data, ok := res.([]interface{}); ok {

			err := tab.computeGet(process, data)
			if err != nil {
				log.Error("[table] %s %s -> %s %s", tab.ID, p.Name, name, err.Error())
			}
			res = data
		}
		break

	case "yao.table.Find":
		switch res.(type) {
		case map[string]interface{}, maps.MapStr:
			data := any.MapOf(res).MapStrAny
			err := tab.computeFind(process, data)
			if err != nil {
				log.Error("[table] %s %s -> %s %s", tab.ID, p.Name, name, err.Error())
			}
			res = data
		}
		break
	}

	// After hook
	if p.After != nil {
		log.Trace("[table] %s %s after: exec(%v)", tab.ID, p.Name, res)
		newRes, err := p.After.Exec(res, process.Sid, process.Global)
		if err != nil {
			log.Error("[table] %s %s after: %s", tab.ID, p.Name, err.Error())
		} else {
			log.Trace("[table] %s %s after: %v", tab.ID, p.Name, newRes)
			res = newRes
		}
	}

	return res, nil
}

// MustExec exec the process
func (p *ProcessActionDSL) MustExec(process *gou.Process) interface{} {
	res, err := p.Exec(process)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return res
}

// deepMergeDefault deep merge args
func (p *ProcessActionDSL) deepMergeDefault(value interface{}, defaults interface{}) interface{} {

	if value == nil {
		return defaults
	}

	switch defaults.(type) {

	case map[string]interface{}:
		defaultMap := defaults.(map[string]interface{})
		valueMap := any.Of(value).MapStr()
		for key, v := range defaultMap {
			valueMap[key] = p.deepMergeDefault(valueMap[key], v)
		}
		return valueMap

	case []interface{}:
		defaultArr := defaults.([]interface{})
		valueArr := any.Of(value).CArray()

		// pad
		nums := len(defaultArr) - len(valueArr)
		for i := 0; i < nums; i++ {
			valueArr = append(valueArr, nil)
		}

		// set default
		for idx, v := range defaultArr {
			valueArr[idx] = p.deepMergeDefault(valueArr[idx], v)
		}

		return valueArr
	}

	return value
}
