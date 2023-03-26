package helper

import (
	"github.com/yaoapp/gou/process"
)

// IF 条件判断
func IF(param CaseParam, paramElse ...CaseParam) interface{} {
	if When(param.When) {
		return process.New(param.Process, param.Args...).Run()
	} else if len(paramElse) > 0 && When(paramElse[0].When) {
		return process.New(paramElse[0].Process, paramElse[0].Args...).Run()
	}
	return nil
}

// ProcessIF xiang.helper.IF IF条件判断
func ProcessIF(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	params := []CaseParam{}
	for _, v := range process.Args {
		params = append(params, CaseParamOf(v))
	}
	if len(params) > 1 {
		IF(params[0], params[1])
	}
	return IF(params[0])
}
