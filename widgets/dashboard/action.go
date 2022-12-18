package dashboard

import (
	"github.com/yaoapp/yao/widgets/action"
)

var processActionDefaults = map[string]*action.Process{

	"Setting": {
		Name:    "yao.dashboard.Setting",
		Guard:   "bearer-jwt",
		Process: "yao.dashboard.Xgen",
		Default: []interface{}{nil},
	},
	"Component": {
		Name:    "yao.dashboard.Component",
		Guard:   "bearer-jwt",
		Default: []interface{}{nil, nil, nil},
	},
	"Data": {
		Name:    "yao.dashboard.Data",
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

	act.Data = action.ProcessOf(act.Data).
		WithBefore(act.BeforeData).WithAfter(act.AfterData).
		Merge(processActionDefaults["Data"]).
		SetHandler(processHandler)
}
