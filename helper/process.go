package helper

import (
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/utils"
)

func init() {
	// 注册处理器
	gou.RegisterProcessHandler("xiang.helper.ArrayPluck", ProcessArrayPluck)
	gou.RegisterProcessHandler("xiang.helper.ArraySplit", ProcessArraySplit)
	gou.RegisterProcessHandler("xiang.helper.ArrayColumn", ProcessArrayColumn)
	gou.RegisterProcessHandler("xiang.helper.ArrayKeep", ProcessArrayKeep)
	gou.RegisterProcessHandler("xiang.helper.ArrayTree", ProcessArrayTree)
	gou.RegisterProcessHandler("xiang.helper.MapKeys", ProcessMapKeys)
	gou.RegisterProcessHandler("xiang.helper.MapValues", ProcessMapValues)
	gou.RegisterProcessHandler("xiang.helper.For", ProcessFor)
	gou.RegisterProcessHandler("xiang.helper.Each", ProcessEach)
	gou.RegisterProcessHandler("xiang.helper.Print", ProcessPrint)
}

// ProcessArrayPluck  xiang.helper.ArrayPluck 将多个数据记录集合，合并为一个数据记录集合
func ProcessArrayPluck(process *gou.Process) interface{} {
	process.ValidateArgNums(2)
	columns := process.ArgsStrings(0)
	pluck := process.ArgsMap(1)
	return ArrayPluck(columns, pluck)
}

// ProcessArraySplit  xiang.helper.ArraySplit 将多条数记录集合，分解为一个 columns:[]string 和 values: [][]interface{}
func ProcessArraySplit(process *gou.Process) interface{} {
	process.ValidateArgNums(1)
	records := process.ArgsRecords(0)
	columns, values := ArraySplit(records)
	return map[string]interface{}{
		"columns": columns,
		"values":  values,
	}
}

// ProcessArrayColumn  xiang.helper.ArrayColumn  返回多条数据记录，指定字段数值。
func ProcessArrayColumn(process *gou.Process) interface{} {
	process.ValidateArgNums(2)
	records := process.ArgsRecords(0)
	name := process.ArgsString(1)
	values := ArrayColumn(records, name)
	return values
}

// ProcessArrayKeep  xiang.helper.ArrayKeep  仅保留指定键名的数据
func ProcessArrayKeep(process *gou.Process) interface{} {
	process.ValidateArgNums(2)
	records := process.ArgsRecords(0)
	columns := process.ArgsStrings(1)
	return ArrayKeep(records, columns)
}

// ProcessArrayTree  xiang.helper.ArrayTree  转换为属性结构
func ProcessArrayTree(process *gou.Process) interface{} {
	process.ValidateArgNums(2)
	records := process.ArgsRecords(0)
	setting := process.ArgsMap(1)
	return ArrayTree(records, setting)
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

// ProcessEach  xiang.helper.Each 循环过程控制
func ProcessEach(process *gou.Process) interface{} {
	process.ValidateArgNums(2)
	v := process.Args[0]
	p := ProcessOf(process.ArgsMap(1))
	Range(v, p)
	return nil
}

// ProcessFor xiang.helper.For 循环过程控制
func ProcessFor(process *gou.Process) interface{} {
	process.ValidateArgNums(3)
	from := process.ArgsInt(0)
	to := process.ArgsInt(1)
	p := ProcessOf(process.ArgsMap(2))
	For(from, to, p)
	return nil
}

// ProcessPrint xiang.helper.Print 打印语句
func ProcessPrint(process *gou.Process) interface{} {
	process.ValidateArgNums(1)
	utils.Dump(process.Args...)
	return nil
}
