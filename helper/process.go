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
	gou.RegisterProcessHandler("xiang.helper.ArrayColumn", ProcessArrayColumn)
	gou.RegisterProcessHandler("xiang.helper.ArrayKeep", ProcessArrayKeep)
	gou.RegisterProcessHandler("xiang.helper.MapKeys", ProcessMapKeys)
	gou.RegisterProcessHandler("xiang.helper.MapValues", ProcessMapValues)
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

// ProcessArrayColumn  xiang.helper.ArrayColumn  返回多条数据记录，指定字段数值。
func ProcessArrayColumn(process *gou.Process) interface{} {
	process.ValidateArgNums(2)
	args := process.Args[0]
	records := []map[string]interface{}{}
	name := process.ArgsString(1)

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
	values := ArrayColumn(records, name)
	return values
}

// ProcessArrayKeep  xiang.helper.ArrayKeep  仅保留指定键名的数据
func ProcessArrayKeep(process *gou.Process) interface{} {
	process.ValidateArgNums(2)
	args := process.Args[0]
	columnsAny := process.Args[1]
	records := []map[string]interface{}{}
	columns := []string{}

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

	switch columnsAny.(type) {
	case []interface{}:
		for _, v := range columnsAny.([]interface{}) {
			value, ok := v.(string)
			if ok {
				columns = append(columns, value)
				continue
			}
			exception.New("参数错误: 第2个参数不是字符串数组", 400).Ctx(process.Args[0]).Throw()
		}
	case []string:
	default:
		exception.New("参数错误: 第2个参数不是字符串数组", 400).Ctx(process.Args[0]).Throw()
		break
	}

	return ArrayKeep(records, columns)
}

// ProcessMapValues  xiang.helper.MapValues 返回映射的数值
func ProcessMapValues(process *gou.Process) interface{} {
	process.ValidateArgNums(1)
	record := process.ArgsMap(0)
	return MapValues(record)
}

// ProcessMapKeys  xiang.helper.MapKeys 返回映射的键
func ProcessMapKeys(process *gou.Process) interface{} {
	process.ValidateArgNums(1)
	record := process.ArgsMap(0)
	return MapKeys(record)
}
