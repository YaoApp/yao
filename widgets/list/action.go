package list

import (
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/widgets/action"
	"github.com/yaoapp/yao/widgets/hook"
	"github.com/yaoapp/yao/widgets/table"
)

var processActionDefaults = map[string]*action.Process{

	"Setting": {
		Name:    "yao.list.Setting",
		Guard:   "bearer-jwt",
		Process: "yao.list.Xgen",
		Default: []interface{}{nil, nil},
	},
	"Component": {
		Name:    "yao.list.Component",
		Guard:   "bearer-jwt",
		Default: []interface{}{nil, nil, nil},
	},
	"Upload": {
		Name:    "yao.list.Upload",
		Guard:   "bearer-jwt",
		Default: []interface{}{nil, nil, nil},
	},
	"Download": {
		Name:    "yao.list.Download",
		Guard:   "-",
		Process: "fs.system.Download",
		Default: []interface{}{nil},
	},
	"Get": {
		Name:    "yao.list.Get",
		Guard:   "bearer-jwt",
		Default: []interface{}{nil},
	},
	"Save": {
		Name:    "yao.list.Save",
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

	act.Upload = action.ProcessOf(act.Upload).
		Merge(processActionDefaults["Upload"]).
		SetHandler(processHandler)

	act.Download = action.ProcessOf(act.Download).
		Merge(processActionDefaults["Download"]).
		SetHandler(processHandler)

	act.Save = action.ProcessOf(act.Save).
		WithBefore(act.BeforeSave).WithAfter(act.AfterSave).
		Merge(processActionDefaults["Save"]).
		SetHandler(processHandler)

	act.Get = action.ProcessOf(act.Get).
		WithBefore(act.BeforeSave).WithAfter(act.AfterSave).
		Merge(processActionDefaults["Get"]).
		SetHandler(processHandler)
}

// BindModel bind model
func (act *ActionDSL) BindModel(m *model.Model) error {
	return nil
}

// BindTable bind table
func (act *ActionDSL) BindTable(tab *table.DSL) error {

	// Copy Hooks
	hook.CopyBefore(act.BeforeSave, tab.Action.BeforeSave)

	hook.CopyAfter(act.AfterSave, tab.Action.AfterSave)

	// Merge Actions
	act.Save.Merge(tab.Action.Save)

	return nil
}
