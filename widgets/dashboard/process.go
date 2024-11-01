package dashboard

import (
	"fmt"
	"net/url"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/widgets/app"
)

// Export process
func exportProcess() {
	process.Register("yao.dashboard.setting", processSetting)
	process.Register("yao.dashboard.xgen", processXgen)
	process.Register("yao.dashboard.component", processComponent)
	process.Register("yao.dashboard.data", processData)
}

func processXgen(process *process.Process) interface{} {

	dashboard := MustGet(process)
	data := process.ArgsMap(1, map[string]interface{}{})
	excludes := app.Permissions(process, "dashboards", dashboard.ID)
	setting, err := dashboard.Xgen(data, excludes)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	return setting
}

func processComponent(process *process.Process) interface{} {

	process.ValidateArgNums(3)
	dashboard := MustGet(process)
	xpath, _ := url.QueryUnescape(process.ArgsString(1))
	method, _ := url.QueryUnescape(process.ArgsString(2))
	key := fmt.Sprintf("%s.$%s", xpath, method)

	// get cloud props
	cProp, has := dashboard.CProps[key]
	if !has {
		exception.New("%s does not exist", 400, key).Throw()
	}

	// :query
	query := map[string]interface{}{}
	if process.NumOfArgsIs(4) {
		query = process.ArgsMap(3)
	}

	// execute query
	res, err := cProp.ExecQuery(process, query)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	return res
}

func processSetting(process *process.Process) interface{} {
	dashboard := MustGet(process)
	process.Args = append(process.Args, process.Args[0]) // dashboard name
	return dashboard.Action.Setting.MustExec(process)
}

func processData(process *process.Process) interface{} {
	dashboard := MustGet(process)
	return dashboard.Action.Data.MustExec(process)
}
