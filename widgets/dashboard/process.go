package dashboard

import (
	"fmt"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/widgets/app"
)

// Export process
func exportProcess() {
	gou.RegisterProcessHandler("yao.dashboard.setting", processSetting)
	gou.RegisterProcessHandler("yao.dashboard.xgen", processXgen)
	gou.RegisterProcessHandler("yao.dashboard.component", processComponent)
	gou.RegisterProcessHandler("yao.dashboard.data", processData)
}

func processXgen(process *gou.Process) interface{} {

	dashboard := MustGet(process)
	data := process.ArgsMap(1, map[string]interface{}{})
	excludes := app.Permissions(process, "dashboards", dashboard.ID)
	setting, err := dashboard.Xgen(data, excludes)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	return setting
}

func processComponent(process *gou.Process) interface{} {

	process.ValidateArgNums(3)
	dashboard := MustGet(process)
	xpath := process.ArgsString(1)
	method := process.ArgsString(2)
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

func processSetting(process *gou.Process) interface{} {
	dashboard := MustGet(process)
	process.Args = append(process.Args, process.Args[0]) // dashboard name
	return dashboard.Action.Setting.MustExec(process)
}

func processData(process *gou.Process) interface{} {
	dashboard := MustGet(process)
	return dashboard.Action.Data.MustExec(process)
}
