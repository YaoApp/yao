package chart

import (
	"fmt"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/exception"
)

// Export process
func exportProcess() {
	gou.RegisterProcessHandler("yao.chart.setting", processSetting)
	gou.RegisterProcessHandler("yao.chart.xgen", processXgen)
	gou.RegisterProcessHandler("yao.chart.component", processComponent)
	gou.RegisterProcessHandler("yao.chart.data", processData)
}

func processXgen(process *gou.Process) interface{} {

	chart := MustGet(process)
	setting, err := chart.Xgen()
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	return setting
}

func processComponent(process *gou.Process) interface{} {

	process.ValidateArgNums(3)
	chart := MustGet(process)
	xpath := process.ArgsString(1)
	method := process.ArgsString(2)
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

func processSetting(process *gou.Process) interface{} {
	form := MustGet(process)
	process.Args = append(process.Args, process.Args[0]) // chart name
	return form.Action.Setting.MustExec(process)
}

func processData(process *gou.Process) interface{} {
	form := MustGet(process)
	return form.Action.Data.MustExec(process)
}
