package helper

import (
	"reflect"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/exception"
)

func init() {
	// 注册处理器
	gou.RegisterProcessHandler("xiang.helper.ArrayPluck", ProcessArrayPluck)
	gou.RegisterProcessHandler("xiang.helper.ArraySplit", ProcessArraySplit)

}

// ProcessArrayPluck  xiang.helper.ArrayPluck 将多个数据记录集合，合并为一个数据记录集合
func ProcessArrayPluck(process *gou.Process) interface{} {
	process.ValidateArgNums(2)
	columnsAny := process.Args[0]
	columns := []string{}
	switch columnsAny.(type) {
	case []interface{}:
		for _, v := range columnsAny.([]interface{}) {
			value, ok := v.(string)
			if ok {
				columns = append(columns, value)
				continue
			}
			exception.New("参数错误: 第1个参数不是字符串数组", 400).Ctx(process.Args[0]).Throw()
		}
	case []string:
	default:
		exception.New("参数错误: 第1个参数不是字符串数组", 400).Ctx(process.Args[0]).Throw()
		break
	}
	pluck := process.ArgsMap(1)
	return ArrayPluck(columns, pluck)
}

// ProcessArraySplit  xiang.helper.ArraySplit 将多条数记录集合，分解为一个 columns:[]string 和 values: [][]interface{}
func ProcessArraySplit(process *gou.Process) interface{} {
	process.ValidateArgNums(1)
	args := process.Args[0]
	records := []map[string]interface{}{}

	switch args.(type) {
	case []interface{}:
		for _, v := range args.([]interface{}) {
			value, ok := v.(map[string]interface{})
			if ok {
				records = append(records, value)
				continue
			}
			exception.New("参数错误: 第1个参数不是数组", 400).Ctx(reflect.TypeOf(process.Args[0]).Name()).Throw()
		}
		break
	case []map[string]interface{}:
		records = args.([]map[string]interface{})
		break
	default:
		exception.New("参数错误: 第1个参数不是数组", 400).Ctx(reflect.TypeOf(process.Args[0]).Name()).Throw()
		break
	}
	columns, values := ArraySplit(records)
	return map[string]interface{}{
		"columns": columns,
		"values":  values,
	}
}
