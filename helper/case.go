package helper

import (
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

// CaseParam 条件参数
type CaseParam struct {
	When    []Condition   `json:"when"`
	Name    string        `json:"name"`
	Process string        `json:"process"`
	Args    []interface{} `json:"args"`
}

// Case 条件判断
func Case(params ...CaseParam) interface{} {
	for _, param := range params {
		if When(param.When) {
			return process.New(param.Process, param.Args...).Run()
		}
	}
	return nil
}

// CaseParamOf 读取参数
func CaseParamOf(v interface{}) CaseParam {
	data, err := jsoniter.Marshal(v)
	if err != nil {
		exception.New("参数错误: %s", 400, err).Throw()
	}
	res := CaseParam{}
	err = jsoniter.Unmarshal(data, &res)
	if err != nil {
		exception.New("参数错误: %s", 400, err).Throw()
	}
	return res
}

// ProcessCase xiang.helper.Case Case条件判断
func ProcessCase(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	params := []CaseParam{}
	for _, v := range process.Args {
		params = append(params, CaseParamOf(v))
	}
	return Case(params...)
}
