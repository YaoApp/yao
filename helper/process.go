package helper

import (
	"fmt"
	"reflect"

	"github.com/yaoapp/gou"
	"github.com/yaoapp/gou/query/share"
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
			exception.New("参数错误: 第1个参数不是数组", 400).Ctx(fmt.Sprintf("%#v", process.Args[0])).Throw()
		}
		break
	case []share.Record:
		for _, v := range args.([]share.Record) {
			records = append(records, v)
		}
		break
	case []map[string]interface{}:
		records = args.([]map[string]interface{})
		break
	default:
		fmt.Printf("%#v %s\n", args, reflect.TypeOf(args).Kind())
		exception.New("参数错误: 第1个参数不是数组", 400).Ctx(fmt.Sprintf("%#v", process.Args[0])).Throw()
		break
	}
	columns, values := ArraySplit(records)
	return map[string]interface{}{
		"columns": columns,
		"values":  values,
	}
}
