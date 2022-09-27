package form

import (
	"fmt"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/exception"
)

// Export process
func exportProcess() {
	gou.RegisterProcessHandler("yao.form.setting", processSetting)
	gou.RegisterProcessHandler("yao.form.xgen", processXgen)
	gou.RegisterProcessHandler("yao.form.component", processComponent)
	gou.RegisterProcessHandler("yao.form.find", processFind)
	gou.RegisterProcessHandler("yao.form.save", processSave)
	gou.RegisterProcessHandler("yao.form.create", processCreate)
	gou.RegisterProcessHandler("yao.form.update", processUpdate)
	gou.RegisterProcessHandler("yao.form.delete", processDelete)
}

func processXgen(process *gou.Process) interface{} {

	form := MustGet(process)
	setting, err := form.Xgen()
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	return setting
}

func processComponent(process *gou.Process) interface{} {

	process.ValidateArgNums(3)
	form := MustGet(process)
	xpath := process.ArgsString(1)
	method := process.ArgsString(2)
	key := fmt.Sprintf("%s.$%s", xpath, method)

	// get cloud props
	cProp, has := form.CProps[key]
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
	process.Args = append(process.Args, process.Args[0]) // formle name
	return form.Action.Setting.MustExec(process)
}

func processSave(process *gou.Process) interface{} {
	form := MustGet(process)
	return form.Action.Save.MustExec(process)
}

func processCreate(process *gou.Process) interface{} {
	form := MustGet(process)
	return form.Action.Create.MustExec(process)
}

func processFind(process *gou.Process) interface{} {
	form := MustGet(process)
	return form.Action.Find.MustExec(process)
}

func processUpdate(process *gou.Process) interface{} {
	form := MustGet(process)
	return form.Action.Update.MustExec(process)
}

func processDelete(process *gou.Process) interface{} {
	form := MustGet(process)
	return form.Action.Delete.MustExec(process)
}
