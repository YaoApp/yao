package chart

import (
	"fmt"
	"net/url"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/widgets/app"
)

// Export process
func exportProcess() {
	process.Register("yao.chart.setting", processSetting)
	process.Register("yao.chart.xgen", processXgen)
	process.Register("yao.chart.component", processComponent)
	process.Register("yao.chart.data", processData)
}

func processXgen(process *process.Process) interface{} {

	chart := MustGet(process)
	data := process.ArgsMap(1, map[string]interface{}{})
	excludes := app.Permissions(process, "charts", chart.ID)
	setting, err := chart.Xgen(data, excludes)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	return setting
}

func processComponent(process *process.Process) interface{} {

	process.ValidateArgNums(3)
	chart := MustGet(process)
	xpath, _ := url.QueryUnescape(process.ArgsString(1))
	method, _ := url.QueryUnescape(process.ArgsString(2))
	key := fmt.Sprintf("%s.$%s", xpath, method)

	// get cloud props
	cProp, has := chart.CProps[key]
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
	chart := MustGet(process)
	process.Args = append(process.Args, process.Args[0]) // chart name
	return chart.Action.Setting.MustExec(process)
}

func processData(process *process.Process) interface{} {
	chart := MustGet(process)
	return chart.Action.Data.MustExec(process)
}
