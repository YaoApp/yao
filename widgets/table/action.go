package table

import (
	"fmt"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/widgets/action"
	"github.com/yaoapp/yao/widgets/hook"
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
	"Upload": {
		Name:    "yao.table.Upload",
		Guard:   "bearer-jwt",
		Default: []interface{}{nil, nil, nil},
	},
	"Download": {
		Name:    "yao.table.Download",
		Guard:   "-",
		Process: "fs.system.Download",
		Default: []interface{}{nil},
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

func (act *ActionDSL) getDefaults() map[string]*action.Process {
	defaults := map[string]*action.Process{}
	for key, action := range processActionDefaults {
		new := *action
		if act.Guard != "" {
			new.Guard = act.Guard
		}
		defaults[key] = &new
	}
	return defaults
}

// SetDefaultProcess set the default value of action
func (act *ActionDSL) SetDefaultProcess() {
	defaults := act.getDefaults()
	act.Setting = action.ProcessOf(act.Setting).
		Merge(defaults["Setting"]).
		SetHandler(processHandler)

	act.Component = action.ProcessOf(act.Component).
		Merge(defaults["Component"]).
		SetHandler(processHandler)

	act.Upload = action.ProcessOf(act.Upload).
		Merge(defaults["Upload"]).
		SetHandler(processHandler)

	act.Download = action.ProcessOf(act.Download).
		Merge(defaults["Download"]).
		SetHandler(processHandler)

	act.Search = action.ProcessOf(act.Search).
		WithBefore(act.BeforeSearch).WithAfter(act.AfterSearch).
		Merge(defaults["Search"]).
		SetHandler(processHandler)

	act.Get = action.ProcessOf(act.Get).
		WithBefore(act.BeforeGet).WithAfter(act.AfterGet).
		Merge(defaults["Get"]).
		SetHandler(processHandler)

	act.Find = action.ProcessOf(act.Find).
		WithBefore(act.BeforeFind).
		WithAfter(act.AfterFind).
		Merge(defaults["Find"]).
		SetHandler(processHandler)

	act.Save = action.ProcessOf(act.Save).
		WithBefore(act.BeforeSave).WithAfter(act.AfterSave).
		Merge(defaults["Save"]).
		SetHandler(processHandler)

	act.Create = action.ProcessOf(act.Create).
		WithBefore(act.BeforeCreate).WithAfter(act.AfterCreate).
		Merge(defaults["Create"]).
		SetHandler(processHandler)

	act.Insert = action.ProcessOf(act.Insert).
		WithBefore(act.BeforeInsert).WithAfter(act.AfterInsert).
		Merge(defaults["Insert"]).
		SetHandler(processHandler)

	act.Update = action.ProcessOf(act.Update).
		WithBefore(act.BeforeUpdate).WithAfter(act.AfterUpdate).
		Merge(defaults["Update"]).
		SetHandler(processHandler)

	act.UpdateWhere = action.ProcessOf(act.UpdateWhere).
		WithBefore(act.BeforeUpdateWhere).WithAfter(act.AfterUpdateWhere).
		Merge(defaults["UpdateWhere"]).
		SetHandler(processHandler)

	act.UpdateIn = action.ProcessOf(act.UpdateIn).
		WithBefore(act.BeforeUpdateIn).WithAfter(act.AfterUpdateIn).
		Merge(defaults["UpdateIn"]).
		SetHandler(processHandler)

	act.Delete = action.ProcessOf(act.Delete).
		WithBefore(act.BeforeDelete).WithAfter(act.AfterDelete).
		Merge(defaults["Delete"]).
		SetHandler(processHandler)

	act.DeleteWhere = action.ProcessOf(act.DeleteWhere).
		WithBefore(act.BeforeDeleteWhere).WithAfter(act.AfterDeleteWhere).
		Merge(defaults["DeleteWhere"]).
		SetHandler(processHandler)

	act.DeleteIn = action.ProcessOf(act.DeleteIn).
		WithBefore(act.BeforeDeleteIn).WithAfter(act.AfterDeleteIn).
		Merge(defaults["DeleteIn"]).
		SetHandler(processHandler)
}

// BindModel bind model
func (act *ActionDSL) BindModel(m *model.Model) error {

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

	return nil
}

// BindTable bind table
func (act *ActionDSL) BindTable(tab *DSL) error {

	// Copy Hooks
	hook.CopyBefore(act.BeforeSearch, tab.Action.BeforeSearch)
	hook.CopyBefore(act.BeforeGet, tab.Action.BeforeGet)
	hook.CopyBefore(act.BeforeFind, tab.Action.BeforeFind)
	hook.CopyBefore(act.BeforeSave, tab.Action.BeforeSave)
	hook.CopyBefore(act.BeforeCreate, tab.Action.BeforeCreate)
	hook.CopyBefore(act.BeforeInsert, tab.Action.BeforeInsert)
	hook.CopyBefore(act.BeforeUpdate, tab.Action.BeforeUpdate)
	hook.CopyBefore(act.BeforeUpdateWhere, tab.Action.BeforeUpdateWhere)
	hook.CopyBefore(act.BeforeUpdateIn, tab.Action.BeforeUpdateIn)
	hook.CopyBefore(act.BeforeDelete, tab.Action.BeforeDelete)
	hook.CopyBefore(act.BeforeDeleteWhere, tab.Action.BeforeDeleteWhere)
	hook.CopyBefore(act.BeforeDeleteIn, tab.Action.BeforeDeleteIn)
	hook.CopyAfter(act.AfterSearch, tab.Action.AfterSearch)
	hook.CopyAfter(act.AfterGet, tab.Action.AfterGet)
	hook.CopyAfter(act.AfterFind, tab.Action.AfterFind)
	hook.CopyAfter(act.AfterSave, tab.Action.AfterSave)
	hook.CopyAfter(act.AfterCreate, tab.Action.AfterCreate)
	hook.CopyAfter(act.AfterInsert, tab.Action.AfterInsert)
	hook.CopyAfter(act.AfterUpdate, tab.Action.AfterUpdate)
	hook.CopyAfter(act.AfterUpdateWhere, tab.Action.AfterUpdateWhere)
	hook.CopyAfter(act.AfterUpdateIn, tab.Action.AfterUpdateIn)
	hook.CopyAfter(act.AfterDelete, tab.Action.AfterDelete)
	hook.CopyAfter(act.AfterDeleteWhere, tab.Action.AfterDeleteWhere)
	hook.CopyAfter(act.AfterDeleteIn, tab.Action.AfterDeleteIn)

	// Merge Actions
	act.Search.Merge(tab.Action.Search)
	act.Get.Merge(tab.Action.Get)
	act.Find.Merge(tab.Action.Find)
	act.Save.Merge(tab.Action.Save)
	act.Create.Merge(tab.Action.Create)
	act.Insert.Merge(tab.Action.Insert)
	act.Update.Merge(tab.Action.Update)
	act.UpdateWhere.Merge(tab.Action.UpdateWhere)
	act.UpdateIn.Merge(tab.Action.UpdateIn)
	act.Delete.Merge(tab.Action.Delete)
	act.DeleteWhere.Merge(tab.Action.DeleteWhere)
	act.DeleteIn.Merge(tab.Action.DeleteIn)

	return nil
}
