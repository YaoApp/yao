package table

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
		Name:    "yao.table.Setting",
		Guard:   "bearer-jwt",
		Process: "yao.table.Xgen",
		Default: []interface{}{nil},
	},
	"Component": {
		Name:    "yao.table.Component",
		Guard:   "bearer-jwt",
		Default: []interface{}{nil, nil, nil},
	},
	"Search": {
		Name:    "yao.table.Search",
		Guard:   "bearer-jwt",
		Default: []interface{}{nil, 1, 20},
	},
	"Get": {
		Name:    "yao.table.Get",
		Guard:   "bearer-jwt",
		Default: []interface{}{nil},
	},
	"Find": {
		Name:    "yao.table.Find",
		Guard:   "bearer-jwt",
		Default: []interface{}{nil, nil},
	},
	"Save": {
		Name:    "yao.table.Save",
		Guard:   "bearer-jwt",
		Default: []interface{}{nil},
	},
	"Create": {
		Name:    "yao.table.Create",
		Guard:   "bearer-jwt",
		Default: []interface{}{nil},
	},
	"Insert": {
		Name:    "yao.table.Insert",
		Guard:   "bearer-jwt",
		Default: []interface{}{nil, nil},
	},
	"Update": {
		Name:    "yao.table.Update",
		Guard:   "bearer-jwt",
		Default: []interface{}{nil, nil},
	},
	"UpdateWhere": {
		Name:    "yao.table.UpdateWhere",
		Guard:   "bearer-jwt",
		Default: []interface{}{nil, nil},
	},
	"UpdateIn": {
		Name:    "yao.table.UpdateIn",
		Guard:   "bearer-jwt",
		Default: []interface{}{nil, nil},
	},
	"Delete": {
		Name:    "yao.table.Delete",
		Guard:   "bearer-jwt",
		Default: []interface{}{nil},
	},
	"DeleteWhere": {
		Name:    "yao.table.DeleteWhere",
		Guard:   "bearer-jwt",
		Default: []interface{}{nil},
	},
	"DeleteIn": {
		Name:    "yao.table.DeleteIn",
		Guard:   "bearer-jwt",
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

	act.Search = action.ProcessOf(act.Search).
		WithBefore(act.BeforeSearch).WithAfter(act.AfterSearch).
		Merge(processActionDefaults["Search"]).
		SetHandler(processHandler)

	act.Get = action.ProcessOf(act.Get).
		WithBefore(act.BeforeGet).WithAfter(act.AfterGet).
		Merge(processActionDefaults["Get"]).
		SetHandler(processHandler)

	act.Find = action.ProcessOf(act.Find).
		WithBefore(act.BeforeFind).
		WithAfter(act.AfterFind).
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

	act.Insert = action.ProcessOf(act.Insert).
		WithBefore(act.BeforeInsert).WithAfter(act.AfterInsert).
		Merge(processActionDefaults["Insert"]).
		SetHandler(processHandler)

	act.Update = action.ProcessOf(act.Update).
		WithBefore(act.BeforeUpdate).WithAfter(act.AfterUpdate).
		Merge(processActionDefaults["Update"]).
		SetHandler(processHandler)

	act.UpdateWhere = action.ProcessOf(act.UpdateWhere).
		WithBefore(act.BeforeUpdateWhere).WithAfter(act.AfterUpdateWhere).
		Merge(processActionDefaults["UpdateWhere"]).
		SetHandler(processHandler)

	act.UpdateIn = action.ProcessOf(act.UpdateIn).
		WithBefore(act.BeforeUpdateIn).WithAfter(act.AfterUpdateIn).
		Merge(processActionDefaults["UpdateIn"]).
		SetHandler(processHandler)

	act.Delete = action.ProcessOf(act.Delete).
		WithBefore(act.BeforeDelete).WithAfter(act.AfterDelete).
		Merge(processActionDefaults["Delete"]).
		SetHandler(processHandler)

	act.DeleteWhere = action.ProcessOf(act.DeleteWhere).
		WithBefore(act.BeforeDeleteWhere).WithAfter(act.AfterDeleteWhere).
		Merge(processActionDefaults["DeleteWhere"]).
		SetHandler(processHandler)

	act.DeleteIn = action.ProcessOf(act.DeleteIn).
		WithBefore(act.BeforeDeleteIn).WithAfter(act.AfterDeleteIn).
		Merge(processActionDefaults["DeleteIn"]).
		SetHandler(processHandler)
}

// BindModel bind model
func (act *ActionDSL) BindModel(m *gou.Model) {

	name := m.ID
	act.Search.Bind(fmt.Sprintf("models.%s.Paginate", name))
	act.Get.Bind(fmt.Sprintf("models.%s.Get", name))
	act.Find.Bind(fmt.Sprintf("models.%s.Find", name))
	act.Save.Bind(fmt.Sprintf("models.%s.Save", name))
	act.Create.Bind(fmt.Sprintf("models.%s.Create", name))
	act.Insert.Bind(fmt.Sprintf("models.%s.Insert", name))
	act.Update.Bind(fmt.Sprintf("models.%s.Update", name))
	act.UpdateWhere.Bind(fmt.Sprintf("models.%s.UpdateWhere", name))
	act.UpdateIn.Bind(fmt.Sprintf("models.%s.UpdateWhere", name))
	act.Delete.Bind(fmt.Sprintf("models.%s.Delete", name))
	act.DeleteWhere.Bind(fmt.Sprintf("models.%s.DeleteWhere", name))
	act.DeleteIn.Bind(fmt.Sprintf("models.%s.DeleteWhere", name))

	// bind options
	if act.Bind.Option != nil {
		act.Search.DefaultMerge([]interface{}{act.Bind.Option})
		act.Get.DefaultMerge([]interface{}{act.Bind.Option})
		act.Find.DefaultMerge([]interface{}{nil, act.Bind.Option})
	}
}

func processHandler(p *action.Process, process *gou.Process) (interface{}, error) {

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
		case map[string]interface{}, maps.MapStr:
			data := any.Of(args[0]).Map().MapStrAny
			err := tab.computeSave(process, data)
			if err != nil {
				log.Error("[table] %s %s -> %s %s", tab.ID, p.Name, name, err.Error())
			}
			args[0] = data
		}
		break

	case "yao.table.Update", "yao.table.UpdateWhere", "yao.table.UpdateIn":
		switch args[1].(type) {
		case map[string]interface{}, maps.MapStr:
			data := any.Of(args[1]).Map().MapStrAny
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
