package form

import (
	"fmt"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/yao/widgets/action"
)

var processActionDefaults = map[string]*action.Process{

	"Setting": {
		Name:    "yao.form.Setting",
		Guard:   "bearer-jwt",
		Process: "yao.form.Xgen",
		Default: []interface{}{nil},
	},
	"Component": {
		Name:    "yao.form.Component",
		Guard:   "bearer-jwt",
		Default: []interface{}{nil, nil, nil},
	},
	"Find": {
		Name:    "yao.form.Find",
		Guard:   "bearer-jwt",
		Default: []interface{}{nil, nil},
	},
	"Save": {
		Name:    "yao.form.Save",
		Guard:   "bearer-jwt",
		Default: []interface{}{nil},
	},
	"Create": {
		Name:    "yao.form.Create",
		Guard:   "bearer-jwt",
		Default: []interface{}{nil},
	},
	"Update": {
		Name:    "yao.form.Update",
		Guard:   "bearer-jwt",
		Default: []interface{}{nil, nil},
	},
	"Delete": {
		Name:    "yao.table.Delete",
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
